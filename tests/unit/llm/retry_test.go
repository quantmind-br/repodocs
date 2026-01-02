package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrier_Execute_SuccessFirstAttempt(t *testing.T) {
	config := llm.DefaultRetryConfig()
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_Execute_SuccessAfterRetry(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return domain.ErrLLMRateLimited
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestRetrier_Execute_MaxRetriesExceeded(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return domain.ErrLLMRateLimited
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMMaxRetriesExceeded)
	assert.Equal(t, 3, callCount)
}

func TestRetrier_Execute_NonRetryableError(t *testing.T) {
	config := llm.DefaultRetryConfig()
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	customErr := errors.New("non-retryable error")
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return customErr
	})

	require.Error(t, err)
	assert.Equal(t, customErr, err)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_Execute_ContextCancelled(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retrier.Execute(ctx, func() error {
		callCount++
		return domain.ErrLLMRateLimited
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRetrier_Execute_ContextDeadlineExceeded(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := retrier.Execute(ctx, func() error {
		return domain.ErrLLMRateLimited
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRetrier_Execute_LLMError429(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return domain.NewLLMError("test", 429, "rate limited", nil)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestRetrier_Execute_LLMError401_NoRetry(t *testing.T) {
	config := llm.DefaultRetryConfig()
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return domain.NewLLMError("test", 401, "unauthorized", nil)
	})

	require.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_BackoffCalculation(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	var delays []time.Duration
	lastCall := time.Now()

	callCount := 0
	_ = retrier.Execute(context.Background(), func() error {
		now := time.Now()
		if callCount > 0 {
			delays = append(delays, now.Sub(lastCall))
		}
		lastCall = now
		callCount++
		if callCount <= 3 {
			return domain.ErrLLMRateLimited
		}
		return nil
	})

	require.Len(t, delays, 3)
	assert.InDelta(t, 100*time.Millisecond, delays[0], float64(50*time.Millisecond))
	assert.InDelta(t, 200*time.Millisecond, delays[1], float64(100*time.Millisecond))
	assert.InDelta(t, 400*time.Millisecond, delays[2], float64(150*time.Millisecond))
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil_error",
			err:      nil,
			expected: false,
		},
		{
			name:     "rate_limited",
			err:      domain.ErrLLMRateLimited,
			expected: true,
		},
		{
			name:     "context_cancelled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context_deadline",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "llm_error_429",
			err:      domain.NewLLMError("test", 429, "rate limited", nil),
			expected: true,
		},
		{
			name:     "llm_error_500",
			err:      domain.NewLLMError("test", 500, "internal error", nil),
			expected: true,
		},
		{
			name:     "llm_error_502",
			err:      domain.NewLLMError("test", 502, "bad gateway", nil),
			expected: true,
		},
		{
			name:     "llm_error_503",
			err:      domain.NewLLMError("test", 503, "unavailable", nil),
			expected: true,
		},
		{
			name:     "llm_error_504",
			err:      domain.NewLLMError("test", 504, "timeout", nil),
			expected: true,
		},
		{
			name:     "llm_error_400",
			err:      domain.NewLLMError("test", 400, "bad request", nil),
			expected: false,
		},
		{
			name:     "llm_error_401",
			err:      domain.NewLLMError("test", 401, "unauthorized", nil),
			expected: false,
		},
		{
			name:     "llm_error_403",
			err:      domain.NewLLMError("test", 403, "forbidden", nil),
			expected: false,
		},
		{
			name:     "llm_error_404",
			err:      domain.NewLLMError("test", 404, "not found", nil),
			expected: false,
		},
		{
			name:     "random_error",
			err:      errors.New("random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llm.IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldRetryStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{200, false},
		{201, false},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			result := llm.ShouldRetryStatusCode(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := llm.DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.InitialInterval)
	assert.Equal(t, 60*time.Second, config.MaxInterval)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 0.1, config.JitterFactor)
}

func TestNewRetrier_DefaultsInvalidConfig(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      -1,
		InitialInterval: -1,
		MaxInterval:     -1,
		Multiplier:      -1,
		JitterFactor:    -1,
	}

	retrier := llm.NewRetrier(config, nil)
	require.NotNil(t, retrier)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestShouldRetry_BackwardCompatibility(t *testing.T) {
	// ShouldRetry is an alias for ShouldRetryStatusCode
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{200, false},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.statusCode)), func(t *testing.T) {
			result := llm.ShouldRetry(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateBackoff_BackwardCompatibility(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.1,
	}

	// Test multiple attempts to see exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		backoff := llm.CalculateBackoff(attempt, config)
		assert.Greater(t, backoff, time.Duration(0))
		assert.LessOrEqual(t, backoff, config.MaxInterval)
	}
}

func TestIsRetryableError_FetchError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "fetch_error_429",
			err:      domain.NewFetchError("test", 429, nil),
			expected: true,
		},
		{
			name:     "fetch_error_500",
			err:      domain.NewFetchError("test", 500, nil),
			expected: true,
		},
		{
			name:     "fetch_error_502",
			err:      domain.NewFetchError("test", 502, nil),
			expected: true,
		},
		{
			name:     "fetch_error_404",
			err:      domain.NewFetchError("test", 404, nil),
			expected: false,
		},
		{
			name:     "fetch_error_401",
			err:      domain.NewFetchError("test", 401, nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llm.IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetrier_Execute_WithLogger(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "info"})
	config := llm.RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, logger)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		if callCount < 2 {
			return domain.ErrLLMRateLimited
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestRetrier_BackoffWithJitter(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.2, // 20% jitter
	}
	retrier := llm.NewRetrier(config, nil)

	var delays []time.Duration
	lastCall := time.Now()

	callCount := 0
	_ = retrier.Execute(context.Background(), func() error {
		now := time.Now()
		if callCount > 0 {
			delays = append(delays, now.Sub(lastCall))
		}
		lastCall = now
		callCount++
		if callCount <= 2 {
			return domain.ErrLLMRateLimited
		}
		return nil
	})

	// With jitter, delays should vary (we can't test exact randomness here)
	// Just verify the retrier was created successfully and delays were recorded
	assert.NotNil(t, retrier)
	assert.Len(t, delays, 2)
}

func TestRetrier_ZeroMaxRetries(t *testing.T) {
	config := llm.RetryConfig{
		MaxRetries:      0,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return domain.ErrLLMRateLimited
	})

	// Should fail on first attempt without retry
	require.Error(t, err)
	assert.Equal(t, 1, callCount)
}

func TestRetrier_NilError(t *testing.T) {
	config := llm.DefaultRetryConfig()
	retrier := llm.NewRetrier(config, nil)

	callCount := 0
	err := retrier.Execute(context.Background(), func() error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}
