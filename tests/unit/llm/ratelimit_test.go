package llm_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenBucket_NewTokenBucket(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerMinute int
		burstSize         int
		wantAvailable     float64
	}{
		{
			name:              "normal_values",
			requestsPerMinute: 60,
			burstSize:         10,
			wantAvailable:     10,
		},
		{
			name:              "zero_requests_defaults_to_60",
			requestsPerMinute: 0,
			burstSize:         5,
			wantAvailable:     5,
		},
		{
			name:              "zero_burst_defaults_to_1",
			requestsPerMinute: 60,
			burstSize:         0,
			wantAvailable:     1,
		},
		{
			name:              "negative_values_use_defaults",
			requestsPerMinute: -10,
			burstSize:         -5,
			wantAvailable:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket := llm.NewTokenBucket(tt.requestsPerMinute, tt.burstSize)
			assert.InDelta(t, tt.wantAvailable, bucket.Available(), 0.01)
		})
	}
}

func TestTokenBucket_TryAcquire_Success(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 5)

	for i := 0; i < 5; i++ {
		assert.True(t, bucket.TryAcquire(), "should acquire token %d", i+1)
	}

	assert.False(t, bucket.TryAcquire(), "should fail after burst exhausted")
}

func TestTokenBucket_TryAcquire_Refills(t *testing.T) {
	bucket := llm.NewTokenBucket(6000, 1)

	assert.True(t, bucket.TryAcquire())
	assert.False(t, bucket.TryAcquire())

	time.Sleep(15 * time.Millisecond)
	assert.True(t, bucket.TryAcquire(), "should have refilled after delay")
}

func TestTokenBucket_Wait_Success(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 5)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		err := bucket.Wait(ctx)
		require.NoError(t, err)
	}
}

func TestTokenBucket_Wait_ContextCancelled(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 1)
	ctx, cancel := context.WithCancel(context.Background())

	err := bucket.Wait(ctx)
	require.NoError(t, err)

	cancel()
	err = bucket.Wait(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTokenBucket_Wait_ContextTimeout(t *testing.T) {
	bucket := llm.NewTokenBucket(6, 1)

	err := bucket.Wait(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = bucket.Wait(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestTokenBucket_Wait_Refills(t *testing.T) {
	bucket := llm.NewTokenBucket(6000, 1)
	ctx := context.Background()

	start := time.Now()

	err := bucket.Wait(ctx)
	require.NoError(t, err)
	err = bucket.Wait(ctx)
	require.NoError(t, err)

	elapsed := time.Since(start)
	assert.True(t, elapsed >= 5*time.Millisecond, "should have waited for refill")
	assert.True(t, elapsed < 100*time.Millisecond, "should not wait too long")
}

func TestTokenBucket_Concurrent(t *testing.T) {
	bucket := llm.NewTokenBucket(60000, 100)
	ctx := context.Background()

	var wg sync.WaitGroup
	var acquired int64
	goroutines := 50
	acquiresPerGoroutine := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < acquiresPerGoroutine; j++ {
				if err := bucket.Wait(ctx); err == nil {
					atomic.AddInt64(&acquired, 1)
				}
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(goroutines*acquiresPerGoroutine), acquired)
}

func TestTokenBucket_Available_ReflectsState(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 5)

	assert.InDelta(t, 5.0, bucket.Available(), 0.01)

	bucket.TryAcquire()
	assert.InDelta(t, 4.0, bucket.Available(), 0.01)

	bucket.TryAcquire()
	bucket.TryAcquire()
	assert.InDelta(t, 2.0, bucket.Available(), 0.01)
}

func TestTokenBucket_BurstSize_Capped(t *testing.T) {
	bucket := llm.NewTokenBucket(60000, 5)

	time.Sleep(100 * time.Millisecond)

	available := bucket.Available()
	assert.LessOrEqual(t, available, 5.0, "should not exceed capacity")
}

func TestNoOpRateLimiter(t *testing.T) {
	limiter := &llm.NoOpRateLimiter{}

	assert.NoError(t, limiter.Wait(context.Background()))
	assert.True(t, limiter.TryAcquire())
	assert.Equal(t, 1.0, limiter.Available())
}

func TestTokenBucket_RefillEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerMinute int
		burstSize         int
		maxWaitTime       time.Duration
	}{
		{
			name:              "very_slow_refill",
			requestsPerMinute: 6, // 1 per 10 seconds
			burstSize:         1,
			maxWaitTime:       15 * time.Second,
		},
		{
			name:              "very_fast_refill",
			requestsPerMinute: 60000, // 1000 per second
			burstSize:         10,
			maxWaitTime:       100 * time.Millisecond,
		},
		{
			name:              "burst_equals_rate",
			requestsPerMinute: 60,
			burstSize:         60,
			maxWaitTime:       2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket := llm.NewTokenBucket(tt.requestsPerMinute, tt.burstSize)
			ctx := context.Background()

			// Exhaust burst
			for i := 0; i < tt.burstSize; i++ {
				assert.True(t, bucket.TryAcquire(), "Should acquire token %d", i+1)
			}

			// Should be exhausted
			assert.False(t, bucket.TryAcquire())

			// Wait for refill and verify tokens available
			start := time.Now()
			err := bucket.Wait(ctx)
			elapsed := time.Since(start)

			require.NoError(t, err)
			assert.Greater(t, elapsed.Milliseconds(), int64(0), "Should have waited for refill")
			assert.Less(t, elapsed, tt.maxWaitTime, "Should not wait too long")
		})
	}
}

func TestTokenBucket_WaitPrecision(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 1) // 1 per second
	ctx := context.Background()

	// First token immediate
	start := time.Now()
	err := bucket.Wait(ctx)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 100*time.Millisecond, "First token should be immediate")

	// Second token should wait ~1 second
	start = time.Now()
	err = bucket.Wait(ctx)
	elapsed = time.Since(start)

	require.NoError(t, err)
	assert.Greater(t, elapsed, 900*time.Millisecond, "Should wait for refill")
	assert.Less(t, elapsed, 1500*time.Millisecond, "Should not wait too long")
}

func TestTokenBucket_MultipleWaits(t *testing.T) {
	bucket := llm.NewTokenBucket(600, 1) // 10 per second
	ctx := context.Background()

	// Wait for 3 tokens with timing
	start := time.Now()
	for i := 0; i < 3; i++ {
		err := bucket.Wait(ctx)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)

	// Should wait approximately 200ms total (100ms per additional token)
	assert.Greater(t, elapsed, 150*time.Millisecond)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestTokenBucket_TryAcquireAccuracy(t *testing.T) {
	bucket := llm.NewTokenBucket(60, 3)

	// Should be able to acquire exactly burst size
	acquired := 0
	for acquired < 10 {
		if bucket.TryAcquire() {
			acquired++
		} else {
			break
		}
	}

	assert.Equal(t, 3, acquired, "Should acquire exactly burst size tokens")
	// Use InDelta to account for floating point precision and any minor refills
	assert.InDelta(t, float64(0), bucket.Available(), 0.01, "Available should be approximately 0")
}
