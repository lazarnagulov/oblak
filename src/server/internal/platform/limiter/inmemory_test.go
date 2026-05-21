package limiter

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryLimiter_Allow(t *testing.T) {
	ctx := context.Background()
	limiter := NewInMemoryLimiter()

	cfg := RateLimiterConfig{Capacity: 2, Refill: 0.1}
	key := "test_user_1"

	allowed, err := limiter.Allow(ctx, key, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected first request to be allowed, but it was blocked")
	}

	allowed, err = limiter.Allow(ctx, key, cfg)
	if !allowed {
		t.Error("expected second request to be allowed, but it was blocked")
	}

	allowed, err = limiter.Allow(ctx, key, cfg)
	if allowed {
		t.Error("expected third request to be blocked, but it was allowed")
	}

	anotherKey := "test_user_2"
	allowed, err = limiter.Allow(ctx, anotherKey, cfg)
	if !allowed {
		t.Error("expected request from another user to be allowed, regardless of the first user")
	}
}

func TestInMemoryLimiter_Refill(t *testing.T) {
	ctx := context.Background()
	limiter := NewInMemoryLimiter()

	cfg := RateLimiterConfig{
		Capacity: 1,
		Refill:   100,
	}
	key := "refill_user"
	allowed, _ := limiter.Allow(ctx, key, cfg)
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}

	allowed, _ = limiter.Allow(ctx, key, cfg)
	if allowed {
		t.Error("expected immediate second request to be blocked")
	}

	time.Sleep(15 * time.Millisecond)

	allowed, _ = limiter.Allow(ctx, key, cfg)
	if !allowed {
		t.Error("expected request to be allowed after waiting for refill")
	}
}
