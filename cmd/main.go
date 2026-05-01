package main

import (
	"github.com/umohsamuel/distributed-sockets/cmd/api"
	"github.com/umohsamuel/distributed-sockets/cmd/ws"
	"github.com/umohsamuel/distributed-sockets/config/cache"
	"github.com/umohsamuel/distributed-sockets/internal/adapter"
	"github.com/umohsamuel/distributed-sockets/internal/service"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
)

var (
	environmentVariables = env.LoadEnvironment()
)

func main() {
	cacheClient := cache.NewCache()

	adapterDependencies := adapter.AdapterDependencies{
		EnvironmentVariables: environmentVariables,
		CacheClient:          cacheClient,
	}

	adapters := adapter.NewAdapter(adapterDependencies)

	serviceDependencies := service.ServiceDependencies{
		Adapter: adapters,
	}

	services := service.NewService(serviceDependencies)

	r := api.API(services, environmentVariables)
	ws.Socket(r.Engine)
	r.Engine.Run(":8080")
}
