package cache

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func NewCache() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	defer client.Close()

	ping, err := client.Ping(context.Background()).Result()

	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	fmt.Println(ping)

	return client

}
