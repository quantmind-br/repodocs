package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewTokenBucket tests creating a token bucket
func TestNewTokenBucket(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerMinute int
		burstSize         int
		expectedCapacity  float64
	}{
		{
			name:              "standard values",
			requestsPerMinute: 60,
			burstSize:         10,
			expectedCapacity:  10,
		},
		{
			name:              "zero requests per minute defaults to 60",
			requestsPerMinute: 0,
			burstSize:         5,
			expectedCapacity:  5,
		},
		{
			name:              "negative requests per minute defaults to 60",
			requestsPerMinute: -10,
			burstSize:         5,
			expectedCapacity:  5,
		},
		{
			name:              "zero burst size defaults to 1",
			requestsPerMinute: 60,
			burstSize:         0,
			expectedCapacity:  1,
		},
		{
			name:              "negative burst size defaults to 1",
			requestsPerMinute: 60,
			burstSize:         -5,
			expectedCapacity:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.requestsPerMinute, tt.burstSize)
			assert.NotNil(t, tb)
			assert.Equal(t, tt.expectedCapacity, tb.capacity)
			assert.Equal(t, tt.expectedCapacity, tb.tokens) // Should start full
		})
	}
}

// TestTokenBucket_TryAcquire tests acquiring tokens without blocking
func TestTokenBucket_TryAcquire(t *testing.T) {
	tb := NewTokenBucket(60, 5)

	// Should be able to acquire initial tokens
	for i := 0; i < 5; i++ {
		assert.True(t, tb.TryAcquire(), "Should acquire token %d", i)
	}

	// Should fail when bucket is empty
	assert.False(t, tb.TryAcquire(), "Should fail when bucket is empty")
}

// TestTokenBucket_Wait tests waiting for tokens
func TestTokenBucket_Wait(t *testing.T) {
	tests := []struct {
		name      string
		rate      int
		burst     int
		acquires  int
		expectErr bool
	}{
		{
			name:      "acquire within burst",
			rate:      60,
			burst:     5,
			acquires:  3,
			expectErr: false,
		},
		{
			name:      "acquire all burst tokens",
			rate:      60,
			burst:     3,
			acquires:  3,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := NewTokenBucket(tt.rate, tt.burst)
			ctx := context.Background()

			for i := 0; i < tt.acquires; i++ {
				err := tb.Wait(ctx)
				if tt.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestTokenBucket_Wait_ContextCancellation tests context cancellation
func TestTokenBucket_Wait_ContextCancellation(t *testing.T) {
	tb := NewTokenBucket(60, 1)

	// Drain the bucket
	assert.True(t, tb.TryAcquire())

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return error immediately
	err := tb.Wait(ctx)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestTokenBucket_Available tests checking available tokens
func TestTokenBucket_Available(t *testing.T) {
	tb := NewTokenBucket(60, 10)

	// Should start with full bucket
	assert.Equal(t, float64(10), tb.Available())

	// Acquire some tokens
	tb.TryAcquire()
	tb.TryAcquire()

	// Should have less tokens (allow for floating point precision)
	available := tb.Available()
	assert.InDelta(t, float64(8), available, 0.1)
}

// TestTokenBucket_Refill tests token refill over time
func TestTokenBucket_Refill(t *testing.T) {
	// Create a bucket with high refill rate
	tb := NewTokenBucket(600, 1) // 10 requests per second

	// Drain the bucket
	assert.True(t, tb.TryAcquire())
	assert.False(t, tb.TryAcquire())

	// Wait for refill
	time.Sleep(150 * time.Millisecond)

	// Should have tokens available
	assert.True(t, tb.TryAcquire(), "Should have token after refill")
}

// TestTokenBucket_ConcurrentAccess tests concurrent access safety
func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(600, 10) // 10 requests per second
	ctx := context.Background()

	// Try to acquire many tokens concurrently
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func() {
			tb.Wait(ctx)
			done <- true
		}()
	}

	// Should complete without deadlock
	successCount := 0
	for i := 0; i < 20; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, 20, successCount)
}

// TestNoOpRateLimiter tests the no-op rate limiter
func TestNoOpRateLimiter(t *testing.T) {
	rl := &NoOpRateLimiter{}
	ctx := context.Background()

	// Wait should always succeed
	assert.NoError(t, rl.Wait(ctx))

	// TryAcquire should always succeed
	assert.True(t, rl.TryAcquire())

	// Available should always return 1
	assert.Equal(t, float64(1), rl.Available())
}

// TestRateLimiterInterface tests that both implement the interface
func TestRateLimiterInterface(t *testing.T) {
	var rl RateLimiter

	// Token bucket should implement interface
	rl = NewTokenBucket(60, 10)
	assert.NotNil(t, rl)
	assert.NoError(t, rl.Wait(context.Background()))
	assert.True(t, rl.TryAcquire())
	assert.GreaterOrEqual(t, rl.Available(), float64(0))

	// NoOp should implement interface
	rl = &NoOpRateLimiter{}
	assert.NotNil(t, rl)
	assert.NoError(t, rl.Wait(context.Background()))
	assert.True(t, rl.TryAcquire())
	assert.Equal(t, float64(1), rl.Available())
}
