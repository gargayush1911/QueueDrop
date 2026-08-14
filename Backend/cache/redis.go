package cache

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func InitRedis() error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6380" // default address if not set
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return RedisClient.Ping(Ctx).Err()
}
