package llm

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestDefaultRetryConfig tests default retry config
func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 1*time.Second, cfg.InitialInterval)
	assert.Equal(t, 60*time.Second, cfg.MaxInterval)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Equal(t, 0.1, cfg.JitterFactor)
}

// TestNewRetrier tests creating a retrier
func TestNewRetrier(t *testing.T) {
	tests := []struct {
		name  string
		cfg   RetryConfig
		valid bool
	}{
		{
			name: "valid config",
			cfg: RetryConfig{
				MaxRetries:      3,
				InitialInterval: time.Second,
				MaxInterval:     60 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.1,
			},
			valid: true,
		},
		{
			name: "negative max retries defaults to 0",
			cfg: RetryConfig{
				MaxRetries:      -1,
				InitialInterval: time.Second,
				MaxInterval:     60 * time.Second,
			},
			valid: true,
		},
		{
			name: "zero initial interval defaults to 1s",
			cfg: RetryConfig{
				InitialInterval: 0,
				MaxInterval:     60 * time.Second,
			},
			valid: true,
		},
		{
			name: "zero max interval defaults to 60s",
			cfg: RetryConfig{
				InitialInterval: time.Second,
				MaxInterval:     0,
			},
			valid: true,
		},
		{
			name: "zero multiplier defaults to 2.0",
			cfg: RetryConfig{
				InitialInterval: time.Second,
				Multiplier:      0,
			},
			valid: true,
		},
		{
			name: "negative jitter defaults to 0",
			cfg: RetryConfig{
				InitialInterval: time.Second,
				JitterFactor:    -0.5,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRetrier(tt.cfg, nil)
			assert.NotNil(t, r)
		})
	}
}

// TestRetrier_Execute_Success tests successful execution
func TestRetrier_Execute_Success(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}
	r := NewRetrier(cfg, nil)
	ctx := context.Background()

	calls := 0
	err := r.Execute(ctx, func() error {
		calls++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

// TestRetrier_Execute_RetrySuccess tests success after retries
func TestRetrier_Execute_RetrySuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	r := NewRetrier(cfg, logger)
	ctx := context.Background()

	calls := 0
	err := r.Execute(ctx, func() error {
		calls++
		if calls < 3 {
			return &domain.LLMError{StatusCode: http.StatusTooManyRequests}
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

// TestRetrier_Execute_MaxRetries tests max retries exceeded
func TestRetrier_Execute_MaxRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}
	r := NewRetrier(cfg, nil)
	ctx := context.Background()

	calls := 0
	err := r.Execute(ctx, func() error {
		calls++
		return &domain.LLMError{StatusCode: http.StatusTooManyRequests}
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMMaxRetriesExceeded)
	assert.Equal(t, 3, calls) // initial + 2 retries
}

// TestRetrier_Execute_HTTPClientTimeout tests that HTTP client timeouts are retried
func TestRetrier_Execute_HTTPClientTimeout(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	r := NewRetrier(cfg, logger)
	ctx := context.Background()

	calls := 0
	err := r.Execute(ctx, func() error {
		calls++
		if calls < 3 {
			return &domain.LLMError{
				Provider: "openai",
				Message:  "request failed",
				Err: &url.Error{
					Op:  "Post",
					URL: "https://api.example.com/v1/chat/completions",
					Err: context.DeadlineExceeded,
				},
			}
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

// TestRetrier_Execute_ContextCancellation tests context cancellation
func TestRetrier_Execute_ContextCancellation(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.0,
	}
	r := NewRetrier(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	calls := 0
	err := r.Execute(ctx, func() error {
		calls++
		return &domain.LLMError{StatusCode: http.StatusTooManyRequests}
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestRetrier_Execute_NonRetryableError tests non-retryable errors
func TestRetrier_Execute_NonRetryableError(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
	}
	r := NewRetrier(cfg, nil)
	ctx := context.Background()

	tests := []struct {
		name  string
		err   error
		calls int
	}{
		{
			name:  "context canceled",
			err:   context.Canceled,
			calls: 1,
		},
		{
			name:  "context deadline exceeded",
			err:   context.DeadlineExceeded,
			calls: 1,
		},
		{
			name:  "bad request",
			err:   &domain.LLMError{StatusCode: http.StatusBadRequest},
			calls: 1,
		},
		{
			name:  "unauthorized",
			err:   &domain.LLMError{StatusCode: http.StatusUnauthorized},
			calls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			err := r.Execute(ctx, func() error {
				calls++
				return tt.err
			})

			assert.Error(t, err)
			assert.Equal(t, tt.calls, calls)
		})
	}
}

// TestIsRetryableError tests retryable error detection
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name: "http client timeout is retryable",
			err: &domain.LLMError{
				Provider: "openai",
				Message:  "request failed",
				Err: &url.Error{
					Op:  "Post",
					URL: "https://api.example.com/v1/chat/completions",
					Err: context.DeadlineExceeded,
				},
			},
			expected: true,
		},
		{
			name: "url error non-timeout is not retryable",
			err: &url.Error{
				Op:  "Get",
				URL: "https://api.example.com",
				Err: errors.New("connection refused"),
			},
			expected: false,
		},
		{
			name:     "rate limited",
			err:      domain.ErrLLMRateLimited,
			expected: true,
		},
		{
			name:     "too many requests",
			err:      &domain.LLMError{StatusCode: http.StatusTooManyRequests},
			expected: true,
		},
		{
			name:     "internal server error",
			err:      &domain.LLMError{StatusCode: http.StatusInternalServerError},
			expected: true,
		},
		{
			name:     "bad gateway",
			err:      &domain.LLMError{StatusCode: http.StatusBadGateway},
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      &domain.LLMError{StatusCode: http.StatusServiceUnavailable},
			expected: true,
		},
		{
			name:     "gateway timeout",
			err:      &domain.LLMError{StatusCode: http.StatusGatewayTimeout},
			expected: true,
		},
		{
			name:     "bad request",
			err:      &domain.LLMError{StatusCode: http.StatusBadRequest},
			expected: false,
		},
		{
			name:     "fetch error with retryable status",
			err:      &domain.FetchError{StatusCode: http.StatusTooManyRequests},
			expected: true,
		},
		{
			name:     "fetch error with non-retryable status",
			err:      &domain.FetchError{StatusCode: http.StatusNotFound},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestShouldRetryStatusCode tests status code retryability
func TestShouldRetryStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			result := ShouldRetryStatusCode(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCalculateBackoff tests backoff calculation
func TestCalculateBackoff(t *testing.T) {
	cfg := RetryConfig{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}

	tests := []struct {
		name        string
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "attempt 0",
			attempt:     0,
			minExpected: 90 * time.Millisecond, // With jitter
			maxExpected: 110 * time.Millisecond,
		},
		{
			name:        "attempt 1",
			attempt:     1,
			minExpected: 180 * time.Millisecond,
			maxExpected: 220 * time.Millisecond,
		},
		{
			name:        "attempt 2",
			attempt:     2,
			minExpected: 360 * time.Millisecond,
			maxExpected: 440 * time.Millisecond,
		},
		{
			name:        "capped at max",
			attempt:     10,
			minExpected: 900 * time.Millisecond,
			maxExpected: 1100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := CalculateBackoff(tt.attempt, cfg)
			assert.GreaterOrEqual(t, backoff, tt.minExpected)
			assert.LessOrEqual(t, backoff, tt.maxExpected)
		})
	}
}

// TestShouldRetry tests backward compatibility function
func TestShouldRetry(t *testing.T) {
	assert.True(t, ShouldRetry(http.StatusTooManyRequests))
	assert.False(t, ShouldRetry(http.StatusBadRequest))
}

// TestRetrier_calculateBackoff tests internal backoff calculation
func TestRetrier_calculateBackoff(t *testing.T) {
	tests := []struct {
		name        string
		cfg         RetryConfig
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name: "no jitter",
			cfg: RetryConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.0,
			},
			attempt:     1,
			minExpected: 200 * time.Millisecond,
			maxExpected: 200 * time.Millisecond,
		},
		{
			name: "with jitter",
			cfg: RetryConfig{
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      2.0,
				JitterFactor:    0.2,
			},
			attempt:     1,
			minExpected: 160 * time.Millisecond,
			maxExpected: 240 * time.Millisecond,
		},
		{
			name: "capped at max",
			cfg: RetryConfig{
				InitialInterval: 500 * time.Millisecond,
				MaxInterval:     1 * time.Second,
				Multiplier:      10.0,
				JitterFactor:    0.0,
			},
			attempt:     10,
			minExpected: 1 * time.Second,
			maxExpected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRetrier(tt.cfg, nil)
			backoff := r.calculateBackoff(tt.attempt)
			assert.GreaterOrEqual(t, backoff, tt.minExpected)
			assert.LessOrEqual(t, backoff, tt.maxExpected)
		})
	}
}
