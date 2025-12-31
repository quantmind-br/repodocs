package llm

import (
	"context"
	"sync"
	"time"
)

// RateLimiter controls the rate of operations
type RateLimiter interface {
	Wait(ctx context.Context) error
	TryAcquire() bool
	Available() float64
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(requestsPerMinute int, burstSize int) *TokenBucket {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	if burstSize <= 0 {
		burstSize = 1
	}

	return &TokenBucket{
		tokens:     float64(burstSize),
		capacity:   float64(burstSize),
		refillRate: float64(requestsPerMinute) / 60.0,
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastRefill = now
}

// Wait blocks until a token is available or context is cancelled
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		tb.mu.Lock()
		tb.refill()
		if tb.tokens >= 1.0 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}

		tokensNeeded := 1.0 - tb.tokens
		waitDuration := time.Duration(tokensNeeded / tb.refillRate * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			continue
		}
	}
}

// TryAcquire attempts to acquire a token without blocking
func (tb *TokenBucket) TryAcquire() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}
	return false
}

// Available returns the current number of available tokens
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return tb.tokens
}

// NoOpRateLimiter is a rate limiter that doesn't limit
type NoOpRateLimiter struct{}

// Wait always returns immediately
func (n *NoOpRateLimiter) Wait(_ context.Context) error {
	return nil
}

// TryAcquire always returns true
func (n *NoOpRateLimiter) TryAcquire() bool {
	return true
}

// Available always returns 1
func (n *NoOpRateLimiter) Available() float64 {
	return 1.0
}
