package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrier_SuccessOnFirstAttempt(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	attempts := 0
	err := retrier.Retry(context.Background(), func() error {
		attempts++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetrier_SuccessAfterRetries(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	attempts := 0
	err := retrier.Retry(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return domain.ErrRateLimited
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetrier_MaxRetriesExceeded(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	attempts := 0
	err := retrier.Retry(context.Background(), func() error {
		attempts++
		return domain.ErrRateLimited
	})

	require.Error(t, err)
	// Initial attempt + 2 retries = 3 attempts
	assert.Equal(t, 3, attempts)
}

func TestRetrier_NonRetryableError(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})

	attempts := 0
	permanentErr := errors.New("permanent error")
	err := retrier.Retry(context.Background(), func() error {
		attempts++
		return permanentErr
	})

	require.Error(t, err)
	assert.Equal(t, 1, attempts) // Should not retry for non-retryable errors
}

func TestRetrier_ContextCancellation(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	})

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retrier.Retry(ctx, func() error {
		attempts++
		return domain.ErrRateLimited
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || attempts < 10)
}

func TestRetrier_DefaultOptions(t *testing.T) {
	opts := fetcher.DefaultRetrierOptions()

	assert.Equal(t, 3, opts.MaxRetries)
	assert.Equal(t, 1*time.Second, opts.InitialInterval)
	assert.Equal(t, 30*time.Second, opts.MaxInterval)
	assert.Equal(t, 2.0, opts.Multiplier)
}

func TestRetrier_InvalidOptions(t *testing.T) {
	// Test that invalid options are corrected to defaults
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      0, // Invalid
		InitialInterval: 0, // Invalid
		MaxInterval:     0, // Invalid
		Multiplier:      0, // Invalid
	})

	// Should still work with defaults
	attempts := 0
	err := retrier.Retry(context.Background(), func() error {
		attempts++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestRetryWithValue_Success(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.DefaultRetrierOptions())
	ctx := context.Background()

	result, err := fetcher.RetryWithValue(ctx, retrier, func() (string, error) {
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestRetryWithValue_RetryThenSuccess(t *testing.T) {
	retrier := fetcher.NewRetrier(fetcher.RetrierOptions{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
	})
	ctx := context.Background()

	attempts := 0
	result, err := fetcher.RetryWithValue(ctx, retrier, func() (int, error) {
		attempts++
		if attempts < 2 {
			return 0, domain.ErrRateLimited
		}
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 2, attempts)
}

func TestShouldRetryStatus(t *testing.T) {
	tests := []struct {
		status   int
		expected bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true}, // Too Many Requests
		{500, false},
		{501, false},
		{502, true}, // Bad Gateway
		{503, true}, // Service Unavailable
		{504, true}, // Gateway Timeout
		{520, true}, // Cloudflare
		{521, true}, // Cloudflare
		{522, true}, // Cloudflare
		{523, true}, // Cloudflare
		{524, true}, // Cloudflare
		{525, true}, // Cloudflare
		{526, true}, // Cloudflare
		{527, true}, // Cloudflare
		{530, true}, // Cloudflare
	}

	for _, tc := range tests {
		t.Run(string(rune(tc.status)), func(t *testing.T) {
			result := fetcher.ShouldRetryStatus(tc.status)
			assert.Equal(t, tc.expected, result, "Status %d", tc.status)
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		header   string
		expected time.Duration
	}{
		{"", 0},
		{"60", 60 * time.Second},
		{"120", 120 * time.Second},
		{"0", 0},
		{"abc", 0},
	}

	for _, tc := range tests {
		t.Run(tc.header, func(t *testing.T) {
			result := fetcher.ParseRetryAfter(tc.header)
			assert.Equal(t, tc.expected, result)
		})
	}
}
