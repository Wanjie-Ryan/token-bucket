package ratelimiter

import (
	"context"
	"time"

	// "github.com/go-redis/redis"
	"github.com/redis/go-redis/v9"
)

type NaiveRedisLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func NewNaiveRedisLimiterClient(client *redis.Client, limit int, window time.Duration) *NaiveRedisLimiter {
	return &NaiveRedisLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

// Allow is not atomic; get and set are two separate round trips with nothing preventing another groutine's GET from landing in btn
// the gap is the race this phase exists to expose.
func (l *NaiveRedisLimiter) Allow(ctx context.Context, key string) (bool, int) {
	// ctx := context.Background()
	redisKey := "ratelimit:" + key

	count := 0
	val, err := l.client.Get(ctx, redisKey).Int()
	if err == nil {
		count = val
	} else if err != redis.Nil {
		return true, 0
	}

	if count >= l.limit {
		return false, 0
	}

	newCount := count + 1
	l.client.Set(ctx, redisKey, newCount, l.window)
	return true, l.limit - newCount
}
