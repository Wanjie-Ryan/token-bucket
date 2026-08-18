package ratelimiter

import "context"

type Limiter interface {
	Allow(ctx context.Context, Key string) (bool, int)
}
