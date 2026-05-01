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
