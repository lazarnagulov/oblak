package limiter

import "context"

type RateLimiterConfig struct {
	Capacity int
	Refill   float64
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, config RateLimiterConfig) (bool, error)
}
