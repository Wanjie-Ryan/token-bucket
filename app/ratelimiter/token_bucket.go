package ratelimiter

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

//embed is a compiler directive, special comment that tells Go compiler, "read this file at build time, and bake its content into this variable"

//go:embed scripts/token_bucket.lua
var tokenBucketScript string

type TokenBucketLimiter struct {
	client     *redis.Client
	script     *redis.Script
	capacity   int
	refillRate float64
}

func NewTokenBucketLimiter(client *redis.Client, capacity int, refillRate float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client:     client,
		script:     redis.NewScript(tokenBucketScript),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

func (l *TokenBucketLimiter) Allow(key string) (bool, int) {
	ctx := context.Background()
	redisKey := "ratelimit:token-bucket:" + key
	now := float64(time.Now().UnixNano()) / 1e9

	res, err := l.script.Run(ctx, l.client, []string{redisKey}, l.capacity, l.refillRate, now).Result()
	// the 3 trailing arguments; capacity, refillrate, now become the argv 1,2,3 in the exact order.

	if err != nil {
		return true, 0
	}

	values := res.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	return allowed, remaining
}
