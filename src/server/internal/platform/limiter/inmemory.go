package limiter

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type InMemoryLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

func NewInMemoryLimiter() RateLimiter {
	return &InMemoryLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (m *InMemoryLimiter) Allow(ctx context.Context, key string, config RateLimiterConfig) (bool, error) {
	m.mu.RLock()
	limiter, exists := m.limiters[key]
	m.mu.RUnlock()

	if !exists {
		m.mu.Lock()
		if l, ok := m.limiters[key]; ok {
			limiter = l
		} else {
			limiter = rate.NewLimiter(rate.Limit(config.Refill), config.Capacity)
			m.limiters[key] = limiter
		}
		m.mu.Unlock()
	}

	return limiter.Allow(), nil
}
