package adapter

import (
	"github.com/go-redis/redis/v8"
	cacheA "github.com/umohsamuel/distributed-sockets/internal/adapter/cache"
	"github.com/umohsamuel/distributed-sockets/internal/domain/cache"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
)

type AdapterDependencies struct {
	EnvironmentVariables *env.EnvironmentVariables
	CacheClient          *redis.Client
}

type Adapters struct {
	EnvironmentVariables *env.EnvironmentVariables
	CacheImplementation  cache.Interface
}

func NewAdapter(deps AdapterDependencies) *Adapters {
	return &Adapters{
		EnvironmentVariables: deps.EnvironmentVariables,
		CacheImplementation:  cacheA.NewCacheClient(deps.CacheClient),
	}
}
