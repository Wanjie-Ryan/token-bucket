package connections

import (
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	once        sync.Once
)

// initredis initializes a persistent connection pool. Call once in main before server starts
func InitRedis() {
	once.Do(func() {
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("REDIS_PORT")

		if port == "" {
			port = "6379"
		}

		redisClient = redis.NewClient(&redis.Options{
			Addr:         host + ":" + port,
			PoolSize:     100,
			MinIdleConns: 10,
		})
	})
}

func RedisClient() *redis.Client {
	return redisClient
}
