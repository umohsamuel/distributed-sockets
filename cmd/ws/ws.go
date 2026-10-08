package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/umohsamuel/distributed-sockets/internal/domain/cache"
	"github.com/umohsamuel/distributed-sockets/internal/domain/queue"
	"github.com/umohsamuel/distributed-sockets/pkg/response"
)

const (
	maxMessageSize = 4096

	RouteLocal    = "local"
	RouteRabbitMQ = "rabbitmq"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// origin := r.Header.Get("Origin")
		//      return origin == "<http://yourdomain.com>"
		return true
	},
}

// gorilla/websocket allows only one concurrent writer per connection.
type client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *client) send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

var clients = make(map[string]*client)
var mutex = &sync.Mutex{}

var cacheClient cache.Interface
var queueClient queue.Interface
var serverID string

func Socket(r *gin.Engine, cache cache.Interface, q queue.Interface, sID string) {
	cacheClient = cache
	queueClient = q
	serverID = sID

	r.GET("/ws", wsHandler)

	r.GET("/info", func(ctx *gin.Context) {
		mutex.Lock()
		online := len(clients)
		mutex.Unlock()
		ctx.JSON(http.StatusOK, gin.H{"server_id": serverID, "local_connections": online})
	})

	startConsumer()

}

type Event struct {
	Type     string `json:"type"`
	ID       string `json:"id,omitempty"`
	ServerID string `json:"server_id,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	To       string `json:"to,omitempty"`
	ToServer string `json:"to_server,omitempty"`
	Route    string `json:"route,omitempty"`
	Error    string `json:"error,omitempty"`
}

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
	conn.SetReadLimit(maxMessageSize)

	c := &client{conn: conn}

	mutex.Lock()
	previous := clients[userID]
	clients[userID] = c
	mutex.Unlock()

	if previous != nil {
		previous.conn.Close()
	}

	cacheClient.Set(context.Background(), "user:"+userID, []byte(serverID), 0)

	log.Printf("User %s connected on server %s\n", userID, serverID)

	c.send(Event{Type: "welcome", ServerID: serverID, UserID: userID})

	go handleConnection(userID, c)
}

func handleConnection(userID string, c *client) {
	defer func() {
		c.conn.Close()

		mutex.Lock()
		current := clients[userID] == c
		if current {
			delete(clients, userID)
		}
		mutex.Unlock()

		// The user may have already reconnected on another server.
		if current {
			owner, err := cacheClient.Get(context.Background(), "user:"+userID)
			if err == nil && string(owner) == serverID {
				cacheClient.Delete(context.Background(), "user:"+userID)
			}
		}
	}()

	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			break
		}

		handleIncomingMessage(userID, c, message)
	}
}

type IncomingMessage struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	To   string `json:"to"`
	Body string `json:"body"`
}

type QueueMessage struct {
	Type       string `json:"type"`
	ID         string `json:"id,omitempty"`
	Kind       string `json:"kind,omitempty"`
	From       string `json:"from"`
	To         string `json:"to"`
	Body       string `json:"body"`
	FromServer string `json:"from_server"`
	ToServer   string `json:"to_server"`
	Route      string `json:"route"`
	SentAt     int64  `json:"sent_at"`
}

func handleIncomingMessage(fromUserID string, sender *client, raw []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Println("Invalid handleIncomingMessage message format:", err)
		return
	}

	queueMsg := QueueMessage{
		Type:       "message",
		ID:         msg.ID,
		Kind:       msg.Kind,
		From:       fromUserID,
		To:         msg.To,
		Body:       msg.Body,
		FromServer: serverID,
		SentAt:     time.Now().UnixMilli(),
	}

	mutex.Lock()
	target, local := clients[msg.To]
	mutex.Unlock()

	if local {
		queueMsg.ToServer = serverID
		queueMsg.Route = RouteLocal
		target.send(queueMsg)
		sender.send(Event{Type: "ack", ID: msg.ID, To: msg.To, ToServer: serverID, Route: RouteLocal})
		return
	}

	targetServerIDBytes, err := cacheClient.Get(context.Background(), "user:"+msg.To)
	if err != nil {
		log.Printf("User %s not found online\n", msg.To)
		sender.send(Event{Type: "error", ID: msg.ID, To: msg.To, Error: "user offline"})
		return
	}
	targetServerID := string(targetServerIDBytes)

	queueMsg.ToServer = targetServerID
	queueMsg.Route = RouteRabbitMQ
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
		sender.send(Event{Type: "error", ID: msg.ID, To: msg.To, Error: "failed to route message"})
		return
	}

	sender.send(Event{Type: "ack", ID: msg.ID, To: msg.To, ToServer: targetServerID, Route: RouteRabbitMQ})
}

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
			target, exists := clients[msg.To]
			mutex.Unlock()

			if exists {
				err := target.send(msg)
				mainMsg.Ack(false)
				return err
			}
			// Requeueing would redeliver to this same server indefinitely.
			log.Printf("User %s not found locally\n", msg.To)
			mainMsg.Nack(false, false)
			return nil
		},
	)
	if err != nil {
		log.Println("Failed to start consumer:", err)
	}
}
