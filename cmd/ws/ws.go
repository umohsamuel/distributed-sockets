package ws

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// origin := r.Header.Get("Origin")
		//      return origin == "<http://yourdomain.com>"
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)
var mutex = &sync.Mutex{}

func Socket(r *gin.Engine) {
	r.GET("/ws", wsHandler)

	go handleMessages()

}

func wsHandler(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)

	if err != nil {
		fmt.Println("Error upgrading connection: ", err)
		return
	}

	mutex.Lock()
	clients[conn] = true
	mutex.Unlock()

	fmt.Println("ws client connected")

	go handleConnection(conn)
}

func handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()

		if err != nil {
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			break
		}

		broadcast <- message

		fmt.Println("Message received: ", string(message))

		// time.Sleep(3 * time.Second)

		// if err := conn.WriteMessage(messageType, message); err != nil {
		// 	fmt.Println("error writing message: ", err)
		// 	break
		// }
	}
}

func handleMessages() {
	for {
		message := <-broadcast

		mutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mutex.Unlock()
	}
}
