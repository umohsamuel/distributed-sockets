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
		// origin := r.Header.Get("Origin")
		//      return origin == "<http://yourdomain.com>"
		return true
	},
}

// var clients = make(map[*websocket.Conn]bool)
var clients = make(map[string]*websocket.Conn)
var mutex = &sync.Mutex{}

var cacheClient cache.Interface
var queueClient queue.Interface
var serverID string

func Socket(r *gin.Engine, cache cache.Interface, q queue.Interface, sID string) {
	cacheClient = cache
	queueClient = q
	serverID = sID

	r.GET("/ws", wsHandler)

	startConsumer()

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

	mutex.Lock()
	clients[userID] = conn
	mutex.Unlock()

	cacheClient.Set(context.Background(), "user:"+userID, []byte(serverID), 0)

	log.Printf("User %s connected on server %s\n", userID, serverID)

	go handleConnection(userID, conn)
}

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
