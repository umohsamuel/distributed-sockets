package cache

// import (
// 	"context"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// func test() {
// 	fmt.Println("ig this is some shit being done in golang redis yeah")

// 	client := redis.NewClient(&redis.Options{
// 		Addr:     "localhost:6379",
// 		Password: "",
// 		DB:       0,
// 	})

// 	defer client.Close()

// 	ping, err := client.Ping(context.Background()).Result()

// 	if err != nil {
// 		fmt.Println(err.Error())
// 		return
// 	}

// 	fmt.Println(ping)

// 	//how to store more complex data?

// 	type Person struct {
// 		ID         string
// 		Name       string `json:"name"`
// 		Age        int    `json:"age"`
// 		Occupation string `json:"occupation"`
// 	}

// 	samuelID := uuid.NewString()

// 	jsonString, err := json.Marshal(Person{
// 		ID:         samuelID,
// 		Name:       "Samuel",
// 		Age:        24,
// 		Occupation: "Software Engineer",
// 	})

// 	if err != nil {
// 		fmt.Printf("Failed to marshal: %s", err.Error())
// 		return
// 	}

// 	samuelKey := fmt.Sprintf("person:%s", samuelID)

// 	// golang redis set command
// 	err = client.Set(context.Background(), samuelKey, jsonString, 0).Err()

// 	if err != nil {
// 		fmt.Printf("Failed to set value in the redis instance %s", err.Error())
// 		return
// 	}

// 	// golang redis get commang

// 	val, err := client.Get(context.Background(), samuelKey).Result()

// 	if err != nil {
// 		fmt.Printf("Failed to get value from redis: %s", err.Error())
// 		return
// 	}

// 	fmt.Printf("value retreived from redis: %s\n", val)
// }
