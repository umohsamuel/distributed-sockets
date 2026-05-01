package main

import (
	"github.com/umohsamuel/distributed-sockets/cmd/api"
	"github.com/umohsamuel/distributed-sockets/cmd/ws"
)

func main() {
	r := api.API()
	ws.Socket(r)

	r.Run(":8080")
}
