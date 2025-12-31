package database

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

func ConnectRedis(redisURL string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Println("Redis connected successfully")
	return client
}
