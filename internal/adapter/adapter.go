package adapter

import (
	"github.com/go-redis/redis/v8"
	cacheA "github.com/umohsamuel/distributed-sockets/internal/adapter/cache"
	queueA "github.com/umohsamuel/distributed-sockets/internal/adapter/queue"
	"github.com/umohsamuel/distributed-sockets/internal/domain/cache"
	"github.com/umohsamuel/distributed-sockets/internal/domain/queue"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
)

type AdapterDependencies struct {
	EnvironmentVariables *env.EnvironmentVariables
	CacheClient          *redis.Client
	QueueClient          *queueA.Queue
}

type Adapters struct {
	EnvironmentVariables *env.EnvironmentVariables
	CacheImplementation  cache.Interface
	QueueImplementation  queue.Interface
}

func NewAdapter(deps AdapterDependencies) *Adapters {
	return &Adapters{
		EnvironmentVariables: deps.EnvironmentVariables,
		CacheImplementation:  cacheA.NewCacheClient(deps.CacheClient),
		QueueImplementation:  queueA.NewQueueClient(deps.QueueClient.Conn, deps.QueueClient.Channel),
	}
}
