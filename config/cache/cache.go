package cache

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/umohsamuel/distributed-sockets/pkg/env"
)

func NewCache(environmentVariables *env.EnvironmentVariables) *redis.Client {

	client := redis.NewClient(&redis.Options{
		Addr:     environmentVariables.Redis.REDIS_ADDR,
		Password: environmentVariables.Redis.REDIS_PASSWORD,
		DB:       0,
	})

	ping, err := client.Ping(context.Background()).Result()

	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	fmt.Println(ping)

	return client

}
