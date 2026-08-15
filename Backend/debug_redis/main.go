package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6380"
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})

	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Println("could not connect to redis at", addr, "-", err)
		os.Exit(1)
	}
	fmt.Println("connected to redis at", addr)

	keys, err := client.Keys(ctx, "event:*").Result()
	if err != nil {
		fmt.Println("KEYS error:", err)
		os.Exit(1)
	}

	if len(keys) == 0 {
		fmt.Println("no event:* keys found in redis")
		return
	}

	fmt.Println("found", len(keys), "event key(s):")
	for _, k := range keys {
		val, err := client.Get(ctx, k).Result()
		if err != nil {
			fmt.Printf("  %s -> (error reading: %v)\n", k, err)
			continue
		}
		ttl, _ := client.TTL(ctx, k).Result()
		fmt.Printf("  %s -> %s (ttl: %s)\n", k, val, ttl)
	}
}
