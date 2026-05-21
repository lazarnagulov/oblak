package limiter

import (
	"context"

	"github.com/gin-gonic/gin"
)

type LimitHandler func(string) gin.HandlerFunc

type RateLimiterConfig struct {
	Capacity int
	Refill   float64
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, config RateLimiterConfig) (bool, error)
}
