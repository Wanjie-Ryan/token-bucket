package ratelimiter

type Limiter interface {
	Allow(Key string) (bool, int)
}
