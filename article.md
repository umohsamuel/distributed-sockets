# Building Distributed WebSockets with Go, Redis, and RabbitMQ

![nerd wojak](https://res.cloudinary.com/db6nohcui/image/upload/v1777906963/portfiolio-blog/mjeott5rfzaolfzszk4t.jpg "Yeah, cool image right?")

So you built a WebSocket server. Users connect, they send messages, other users receive them. Life is good. Then someone asks: "What happens when you need to scale to multiple servers?"

And suddenly your nice little `map[*websocket.Conn]bool` is not going to cut it anymore. Because User A is connected to Server 1, User B is connected to Server 2, and your local connection map has no clue the other server even exists.

This is the problem I set out to solve. Let me walk you through how I built a distributed WebSocket system using Go, Redis, and RabbitMQ.

## The Problem

With a single server, WebSocket messaging is straightforward. You keep a map of connections, someone sends a message, you look up the recipient in the map, and write to their connection. Done.

But the moment you have multiple server instances (for load balancing, high availability, or just because your app got popular), you have a problem: Server 1 does not know about connections on Server 2. A message sent from a user on Server 1 to a user on Server 2 will just... disappear into the void.

## The Solution

The architecture is pretty simple once you see it:

1. **Redis** acts as a shared registry. When a user connects to any server, we store `userID -> serverID` in Redis. Now every server can look up where any user is connected.

2. **RabbitMQ** (with topic exchange) handles message routing between servers. Each server subscribes to its own routing key (`server.{serverID}`). When Server 1 needs to send a message to a user on Server 2, it publishes to RabbitMQ with the routing key `server.server-2`. Server 2 picks it up and delivers locally.

3. **Local optimization**: If both users happen to be on the same server, we skip Redis and RabbitMQ entirely and deliver the message directly.

Here is the flow visually:

```
User A (Server 1) sends message to User B (Server 2)

1. Server 1 checks: is User B connected locally? No.
2. Server 1 asks Redis: where is User B? -> "server-2"
3. Server 1 publishes to RabbitMQ: key="server.server-2", body=message
4. Server 2 receives from RabbitMQ (subscribed to "server.server-2")
5. Server 2 finds User B in local connections, writes to their WebSocket
```

## Project Structure

Before we get into the code, here is how I organized the project:

```
cmd/
  main.go              - Entry point, wiring everything together
  api/                 - HTTP routes
  ws/                  - WebSocket handler and message routing logic
config/
  cache/               - Redis client initialization
internal/
  adapter/             - Implementations of domain interfaces
    cache/             - Redis adapter
    queue/             - RabbitMQ adapter
  domain/
    cache/             - Cache interface definition
    queue/             - Queue interface and types
  service/             - Business logic layer
pkg/
  env/                 - Environment variable loading
  util/                - Helpers (server ID, panic on error)
```

I used a clean architecture style where the domain layer defines interfaces and the adapter layer implements them. This way the WebSocket handler does not care whether the queue is RabbitMQ, Kafka, or carrier pigeons. It just calls `Emit()` and `Recieve()`.

## Step 1: Setting Up the Infrastructure

First things first. We need Redis and RabbitMQ running. I used Docker Compose for this:

```yaml
version: "3.9"

services:
  redis:
    image: redis:8-alpine
    container_name: app_redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-secret}
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD:-secret}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - app_network

  rabbitmq:
    image: rabbitmq:4-management-alpine
    container_name: app_rabbitmq
    restart: unless-stopped
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER:-admin}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASSWORD:-secret}
      RABBITMQ_DEFAULT_VHOST: ${RABBITMQ_VHOST:-/}
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - app_network

volumes:
  redis_data:
  rabbitmq_data:

networks:
  app_network:
    driver: bridge
```

Spin it up with:

```bash
docker compose up redis rabbitmq -d
```

RabbitMQ management UI will be at `http://localhost:15672` (guest/guest by default). It is super useful for debugging when you are trying to figure out why your messages are not showing up.

## Step 2: Defining the Domain Interfaces

I like to define my interfaces first so I know exactly what capabilities I need before I write any implementation. For the cache (Redis):

```go
package cache

import (
	"context"
	"time"
)

type Interface interface {
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
```

Simple. Set a key, get a key, delete a key. That is all we need from Redis for this use case.

For the queue (RabbitMQ), there is a bit more going on. First the types:

```go
package queue

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ExchangeDeclare struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

type QueueDeclare struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

type QueueBind struct {
	Name     string
	Key      []string
	Exchange string
	NoWait   bool
	Args     amqp.Table
}

type Consume struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

type Publish struct {
	Ctx       context.Context
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
	Msg       amqp.Publishing
}
```

And the interface itself:

```go
package queue

import amqp "github.com/rabbitmq/amqp091-go"

type MessageHandler func(body []byte, mainMsg amqp.Delivery) error

type Interface interface {
	Emit(exchangeConfig ExchangeDeclare, publishConfig Publish) error
	Recieve(exchangeConfig ExchangeDeclare, queueDeclareConfig QueueDeclare, queueBindConfig QueueBind, consumerConfig Consume, handler MessageHandler) error
}
```

Notice the `MessageHandler` type. This is the callback that gets called for every message the consumer receives. The handler gets both the message body and the raw `amqp.Delivery` so it can Ack or Nack the message.

## Step 3: Implementing the Adapters

### Cache Adapter (Redis)

The Redis adapter is about as simple as it gets:

```go
package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/umohsamuel/distributed-sockets/internal/domain/cache"
)

type Cache struct {
	client *redis.Client
}

func NewCacheClient(client *redis.Client) cache.Interface {
	return &Cache{
		client: client,
	}
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Result()
	return []byte(val), err
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	_, err := c.client.Del(ctx, key).Result()
	return err
}
```

### Queue Adapter (RabbitMQ)

This one is more involved. The `Emit` method declares the exchange and publishes a message. The `Recieve` method declares the exchange, creates a queue, binds it with routing keys, and starts consuming in a goroutine:

```go
package queue

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/umohsamuel/distributed-sockets/internal/domain/queue"
)

type Queue struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewQueueClient(conn *amqp.Connection, channel *amqp.Channel) queue.Interface {
	return &Queue{
		Conn:    conn,
		Channel: channel,
	}
}

func (q *Queue) Emit(exchangeConfig queue.ExchangeDeclare, publishConfig queue.Publish) error {
	err := q.Channel.ExchangeDeclare(
		exchangeConfig.Name,
		exchangeConfig.Kind,
		exchangeConfig.Durable,
		exchangeConfig.AutoDelete,
		exchangeConfig.Internal,
		exchangeConfig.NoWait,
		exchangeConfig.Args,
	)
	if err != nil {
		return err
	}

	return q.Channel.PublishWithContext(
		publishConfig.Ctx,
		publishConfig.Exchange,
		publishConfig.Key,
		publishConfig.Mandatory,
		publishConfig.Immediate,
		publishConfig.Msg,
	)
}

func (q *Queue) Recieve(exchangeConfig queue.ExchangeDeclare, queueDeclareConfig queue.QueueDeclare, queueBindConfig queue.QueueBind, consumerConfig queue.Consume, handler queue.MessageHandler) error {
	err := q.Channel.ExchangeDeclare(
		exchangeConfig.Name,
		exchangeConfig.Kind,
		exchangeConfig.Durable,
		exchangeConfig.AutoDelete,
		exchangeConfig.Internal,
		exchangeConfig.NoWait,
		exchangeConfig.Args,
	)
	if err != nil {
		return err
	}

	declaredQueue, err := q.Channel.QueueDeclare(
		queueDeclareConfig.Name,
		queueDeclareConfig.Durable,
		queueDeclareConfig.AutoDelete,
		queueDeclareConfig.Exclusive,
		queueDeclareConfig.NoWait,
		queueDeclareConfig.Args,
	)
	if err != nil {
		return err
	}

	if len(queueBindConfig.Key) < 1 {
		return fmt.Errorf("Queue bind key []string cannot be lesser than 1")
	}

	for _, key := range queueBindConfig.Key {
		err = q.Channel.QueueBind(
			declaredQueue.Name,
			key,
			queueBindConfig.Exchange,
			queueBindConfig.NoWait,
			queueBindConfig.Args,
		)
		if err != nil {
			return err
		}
	}

	msgs, err := q.Channel.Consume(
		declaredQueue.Name,
		consumerConfig.Consumer,
		consumerConfig.AutoAck,
		consumerConfig.Exclusive,
		consumerConfig.NoLocal,
		consumerConfig.NoWait,
		consumerConfig.Args,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body, msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}()

	return nil
}
```

A few things to note:

- The queue is declared with `Exclusive: true`, meaning it is tied to this connection and will be auto-deleted when the server disconnects. This is exactly what we want because each server instance should have its own ephemeral queue.
- The binding key is a slice of strings because you might want a server to listen on multiple routing keys.
- The consumer runs in a goroutine so it does not block the main thread.

## Step 4: The WebSocket Handler

This is where the magic happens. The `ws` package manages local connections, handles incoming messages, and bridges everything together.

### Connection Management

```go
package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/umohsamuel/distributed-sockets/internal/domain/cache"
	"github.com/umohsamuel/distributed-sockets/internal/domain/queue"
	"github.com/umohsamuel/distributed-sockets/pkg/response"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[string]*websocket.Conn)
var mutex = &sync.Mutex{}

var cacheClient cache.Interface
var queueClient queue.Interface
var serverID string
```

The `clients` map is keyed by `userID` instead of the raw connection. This way we can look up a user's connection by their ID when we need to deliver a message.

### Initialization

```go
func Socket(r *gin.Engine, cache cache.Interface, q queue.Interface, sID string) {
	cacheClient = cache
	queueClient = q
	serverID = sID

	r.GET("/ws", wsHandler)

	startConsumer()
}
```

We register the WebSocket route and immediately start the RabbitMQ consumer. The consumer starts listening for messages destined for this server right away.

### Handling New Connections

```go
func wsHandler(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		response.ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Message:    "user_id required",
		}.Send(ctx)
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Error upgrading connection: ", err)
		return
	}

	mutex.Lock()
	clients[userID] = conn
	mutex.Unlock()

	cacheClient.Set(context.Background(), "user:"+userID, []byte(serverID), 0)

	log.Printf("User %s connected on server %s\n", userID, serverID)

	go handleConnection(userID, conn)
}
```

When a user connects, three things happen:

1. We store their connection in the local `clients` map
2. We store `user:{userID} -> serverID` in Redis so other servers can find them
3. We start a goroutine to read messages from their connection

### Handling Disconnects

```go
func handleConnection(userID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		mutex.Lock()
		delete(clients, userID)
		mutex.Unlock()
		cacheClient.Delete(context.Background(), "user:"+userID)
	}()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			break
		}

		handleIncomingMessage(userID, message)
	}
}
```

When the connection breaks (user closes tab, network dies, whatever), the deferred function cleans up: closes the WebSocket, removes from the local map, and deletes the Redis entry. Now other servers will not try to route messages to a dead connection.

### Routing Messages

This is the core logic. When a user sends a message, we need to figure out where the recipient is and get the message there:

```go
type IncomingMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type QueueMessage struct {
	From string `json:"from"`
	To   string `json:"to"`
	Body string `json:"body"`
}

func handleIncomingMessage(fromUserID string, raw []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Println("Invalid handleIncomingMessage message format:", err)
		return
	}

	mutex.Lock()
	conn, local := clients[msg.To]
	mutex.Unlock()

	if local {
		queueMsg := QueueMessage{From: fromUserID, To: msg.To, Body: msg.Body}
		body, _ := json.Marshal(queueMsg)
		conn.WriteMessage(websocket.TextMessage, body)
		return
	}

	targetServerIDBytes, err := cacheClient.Get(context.Background(), "user:"+msg.To)
	if err != nil {
		log.Printf("User %s not found online\n", msg.To)
		return
	}
	targetServerID := string(targetServerIDBytes)

	queueMsg := QueueMessage{From: fromUserID, To: msg.To, Body: msg.Body}
	body, err := json.Marshal(queueMsg)
	if err != nil {
		log.Println("Failed to Marshall queueMsg:", err)
		return
	}

	err = queueClient.Emit(
		queue.ExchangeDeclare{
			Name:       "messages",
			Kind:       "topic",
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
			Args:       nil,
		},
		queue.Publish{
			Ctx:       context.Background(),
			Exchange:  "messages",
			Key:       "server." + targetServerID,
			Mandatory: false,
			Immediate: false,
			Msg: amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		},
	)
	if err != nil {
		log.Println("Failed to emit message:", err)
	}
}
```

The local shortcut is important. If both users are on the same server, there is zero reason to go through Redis lookup and RabbitMQ. Just deliver directly. Saves network hops and latency.

### The Consumer (Receiving Messages from RabbitMQ)

```go
func startConsumer() {
	err := queueClient.Recieve(
		queue.ExchangeDeclare{
			Name:       "messages",
			Kind:       "topic",
			Durable:    true,
			AutoDelete: false,
			Internal:   false,
			NoWait:     false,
			Args:       nil,
		},
		queue.QueueDeclare{
			Name:       "",
			Durable:    true,
			AutoDelete: false,
			Exclusive:  true,
			NoWait:     false,
			Args:       nil,
		},
		queue.QueueBind{
			Key:      []string{"server." + serverID},
			Exchange: "messages",
			NoWait:   false,
			Args:     nil,
		},
		queue.Consume{
			Consumer:  "",
			AutoAck:   false,
			Exclusive: false,
			NoLocal:   false,
			NoWait:    false,
			Args:      nil,
		},

		func(body []byte, mainMsg amqp.Delivery) error {
			var msg QueueMessage
			if err := json.Unmarshal(body, &msg); err != nil {
				mainMsg.Nack(false, false)
				return err
			}

			mutex.Lock()
			conn, exists := clients[msg.To]
			mutex.Unlock()

			if exists {
				err := conn.WriteMessage(websocket.TextMessage, body)
				mainMsg.Ack(false)
				return err
			}
			log.Printf("User %s not found locally\n", msg.To)
			mainMsg.Nack(false, true)
			return nil
		},
	)
	if err != nil {
		log.Println("Failed to start consumer:", err)
	}
}
```

Each server binds to its own routing key: `server.{serverID}`. When a message arrives, the handler looks up the target user in the local connections map and writes to their WebSocket.

I set `AutoAck: false` so we manually acknowledge messages. If the user is found locally, we Ack. If not (maybe they disconnected between the Redis lookup and delivery), we Nack with `requeue: true` so RabbitMQ can try again or another consumer can pick it up.

## Step 5: Wiring It All Together

The `main.go` ties everything together:

```go
package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/umohsamuel/distributed-sockets/cmd/api"
	"github.com/umohsamuel/distributed-sockets/cmd/ws"
	"github.com/umohsamuel/distributed-sockets/config/cache"
	"github.com/umohsamuel/distributed-sockets/internal/adapter"
	"github.com/umohsamuel/distributed-sockets/internal/adapter/queue"
	"github.com/umohsamuel/distributed-sockets/internal/service"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
	"github.com/umohsamuel/distributed-sockets/pkg/util"
)

var (
	environmentVariables = env.LoadEnvironment()
	serverID             string
)

func init() {
	serverID = util.GetServerID()
	log.Println("Server ID:", serverID)
}

func main() {
	cacheClient := cache.NewCache(environmentVariables)
	defer cacheClient.Close()

	queueConn, err := amqp.Dial(fmt.Sprintf("amqp://guest:%s@%s/", environmentVariables.RabbitMQ.RABBITMQ_PASSWORD, environmentVariables.RabbitMQ.RABBITMQ_ADDR))
	util.FailOnError(err, "Failed to connect to RabbitMQ")
	defer queueConn.Close()

	queueChannel, err := queueConn.Channel()
	util.FailOnError(err, "Failed to open a RabbitMQ channel")
	defer queueChannel.Close()

	queueClient := &queue.Queue{
		Conn:    queueConn,
		Channel: queueChannel,
	}

	adapterDependencies := adapter.AdapterDependencies{
		EnvironmentVariables: environmentVariables,
		CacheClient:          cacheClient,
		QueueClient:          queueClient,
	}

	adapters := adapter.NewAdapter(adapterDependencies)

	serviceDependencies := service.ServiceDependencies{
		Adapter: adapters,
	}

	services := service.NewService(serviceDependencies)

	r := api.API(services, environmentVariables)
	ws.Socket(r.Engine, adapters.CacheImplementation, adapters.QueueImplementation, serverID)
	r.Engine.Run(environmentVariables.Port)
}
```

Each server gets a unique ID via the `SERVER_ID` environment variable (or falls back to the machine hostname):

```go
package util

import "os"

func GetServerID() string {
	id := os.Getenv("SERVER_ID")
	if id != "" {
		return id
	}
	hostname, _ := os.Hostname()
	return hostname
}
```

## Step 6: Testing It

To actually see the distribution working, you need to run multiple server instances. Start Redis and RabbitMQ, then open two terminals.

Terminal 1:

```bash
SERVER_ID=server-1 PORT=:8080 go run cmd/main.go
```

Terminal 2:

```bash
SERVER_ID=server-2 PORT=:8081 go run cmd/main.go
```

Now connect WebSocket clients to different servers. Open a browser console and connect as "alice" on server-1:

```javascript
const ws = new WebSocket("ws://localhost:8080/ws?user_id=alice");
ws.onmessage = (e) => console.log("Received:", e.data);
```

In another tab, connect as "bob" on server-2:

```javascript
const ws = new WebSocket("ws://localhost:8081/ws?user_id=bob");
ws.onmessage = (e) => console.log("Received:", e.data);
```

Now send a message from alice to bob:

```javascript
ws.send(
  JSON.stringify({
    to: "bob",
    body: "hey bob, can you hear me across servers?",
  })
);
```

And bob receives:

```json
{
  "from": "alice",
  "to": "bob",
  "body": "hey bob, can you hear me across servers?"
}
```

That message went from Alice's WebSocket, to Server 1, through Redis (lookup), through RabbitMQ (routing), to Server 2, and finally to Bob's WebSocket. Pretty cool.

## Demo

Here is a video showing the whole thing in action with two server instances running side by side:

<video src="https://res.cloudinary.com/db6nohcui/video/upload/v1777857204/distributed-server_ghx1ne.mp4" controls playsinline width="100%"></video>

## Gotchas I Ran Into

A few things that tripped me up while building this:

**1. Exchange property mismatch.** If you declare an exchange as non-durable and then later change your code to durable, RabbitMQ will reject it with a `PRECONDITION_FAILED` error. You have to delete the old exchange in the management UI and let the app recreate it. Spent a good amount of time confused by this one.

**2. The `set` vs `export` thing on Windows.** If you are using Git Bash on Windows, `set SERVER_ID=server-2` does nothing. You need `export` or inline it: `SERVER_ID=server-2 go run cmd/main.go`. I kept wondering why my second server was using the same ID.

**3. Redis hostname resolution.** When running the app locally but Redis in Docker, you cannot use `redis` as the hostname (that only works inside the Docker network). Use `localhost` instead. Obvious in hindsight but it got me.

**4. Nacking to the right place.** In the consumer, if a user disconnects between the time we looked them up in Redis and the time we try to deliver, the message gets Nacked with requeue. This prevents message loss during that race condition window.

## Where to Go From Here

This is a solid foundation, but there are plenty of things you could add (nothing really, "shes perfect"):

- **Authentication**: Right now we just trust the `user_id` query param. In production you would validate a JWT or session token.
- **Message persistence**: If a user is offline, messages are lost. You could store them in a database and deliver when they reconnect.
- **Presence system**: Use Redis pub/sub or expiring keys to track online/offline status.
- **Multiple message types**: Add a `type` field to messages and route `chat`, `typing`, `notification` etc. differently.

## Final Thoughts

The core idea is honestly pretty simple: use a shared registry (Redis) to know where everyone is, and a message broker (RabbitMQ) to route between servers. The WebSocket part is almost unchanged from a single-server implementation. You just replace the local broadcast channel with a publish to RabbitMQ, and add a consumer that delivers to local connections.

Here's a link to the code on Github [umohsamuel/distributed-sockets](https://github.com/umohsamuel/distributed-sockets), if you want to dig through it or run it yourself. Have fun with it.
