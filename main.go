package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func main() {
	fmt.Println("ig this is some shit being done in golang redis yeah")

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ping, err := client.Ping(context.Background()).Result()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println(ping)

	// golang redis set command

	err = client.Set(context.Background(), "name", "Samuel", 0).Err()

	if err != nil {
		fmt.Printf("Failed to set value in the redis instance %s", err.Error())
		return
	}

	// golang redis get commang

	val, err := client.Get(context.Background(), "name").Result()

	if err != nil {
		fmt.Printf("Failed to get value from redis: %s", err.Error())
		return
	}

	fmt.Printf("value retreived from redis: %s\n", val)
}
