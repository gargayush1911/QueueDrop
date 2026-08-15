package cache

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func InitRedis() error {
	// Preferred: a full connection URL, e.g. rediss://default:<password>@<host>:<port>
	// This is what Upstash / Redis Cloud / most managed Redis providers give you,
	// and it carries TLS + auth automatically.
	if url := os.Getenv("REDIS_URL"); url != "" {
		opts, err := redis.ParseURL(url)
		if err != nil {
			return err
		}
		RedisClient = redis.NewClient(opts)
		return RedisClient.Ping(Ctx).Err()
	}

	// Fallback: bare host:port for local dev against a plain, unauthenticated Redis.
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6380" // default address if not set
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return RedisClient.Ping(Ctx).Err()
}
