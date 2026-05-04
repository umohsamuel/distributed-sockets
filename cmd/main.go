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
	r.Engine.Run(":8080")
}
