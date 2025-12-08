package app_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchError(t *testing.T) {
	// Test with status code
	originalErr := errors.New("connection refused")
	fetchErr := domain.NewFetchError("https://example.com", 500, originalErr)

	assert.Equal(t, "https://example.com", fetchErr.URL)
	assert.Equal(t, 500, fetchErr.StatusCode)
	assert.Equal(t, originalErr, fetchErr.Err)

	// Test Error() with status code
	expectedMsg := "fetch error for https://example.com: status 500: connection refused"
	assert.Equal(t, expectedMsg, fetchErr.Error())

	// Test Error() without status code
	fetchErrNoStatus := domain.NewFetchError("https://example.com", 0, originalErr)
	expectedMsgNoStatus := "fetch error for https://example.com: connection refused"
	assert.Equal(t, expectedMsgNoStatus, fetchErrNoStatus.Error())

	// Test Unwrap()
	assert.Equal(t, originalErr, fetchErr.Unwrap())
}

func TestRetryableError(t *testing.T) {
	// Test with RetryAfter
	originalErr := errors.New("temporary failure")
	retryableErr := &domain.RetryableError{
		Err:        originalErr,
		RetryAfter: 30,
	}

	assert.Equal(t, originalErr, retryableErr.Err)
	assert.Equal(t, 30, retryableErr.RetryAfter)

	// Test Error() with RetryAfter
	expectedMsg := "retryable error (retry after 30s): temporary failure"
	assert.Equal(t, expectedMsg, retryableErr.Error())

	// Test Error() without RetryAfter
	retryableErrNoDelay := &domain.RetryableError{
		Err:        originalErr,
		RetryAfter: 0,
	}
	expectedMsgNoDelay := "retryable error: temporary failure"
	assert.Equal(t, expectedMsgNoDelay, retryableErrNoDelay.Error())

	// Test Unwrap()
	assert.Equal(t, originalErr, retryableErr.Unwrap())
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Regular error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "RetryableError",
			err:      &domain.RetryableError{Err: errors.New("retry me")},
			expected: true,
		},
		{
			name:     "FetchError with 429",
			err:      domain.NewFetchError("https://example.com", 429, errors.New("too many requests")),
			expected: true,
		},
		{
			name:     "FetchError with 503",
			err:      domain.NewFetchError("https://example.com", 503, errors.New("service unavailable")),
			expected: true,
		},
		{
			name:     "FetchError with 502",
			err:      domain.NewFetchError("https://example.com", 502, errors.New("bad gateway")),
			expected: true,
		},
		{
			name:     "FetchError with 504",
			err:      domain.NewFetchError("https://example.com", 504, errors.New("gateway timeout")),
			expected: true,
		},
		{
			name:     "FetchError with 520 (Cloudflare)",
			err:      domain.NewFetchError("https://example.com", 520, errors.New("cloudflare error")),
			expected: true,
		},
		{
			name:     "FetchError with 525 (Cloudflare)",
			err:      domain.NewFetchError("https://example.com", 525, errors.New("cloudflare error")),
			expected: true,
		},
		{
			name:     "FetchError with 530 (Cloudflare)",
			err:      domain.NewFetchError("https://example.com", 530, errors.New("cloudflare error")),
			expected: true,
		},
		{
			name:     "FetchError with 404 (not retryable)",
			err:      domain.NewFetchError("https://example.com", 404, errors.New("not found")),
			expected: false,
		},
		{
			name:     "FetchError with 200 (not retryable)",
			err:      domain.NewFetchError("https://example.com", 200, errors.New("success")),
			expected: false,
		},
		{
			name:     "FetchError with 0 status (not retryable)",
			err:      domain.NewFetchError("https://example.com", 0, errors.New("some error")),
			expected: false,
		},
		{
			name:     "ErrRateLimited",
			err:      domain.ErrRateLimited,
			expected: true,
		},
		{
			name:     "ErrTimeout",
			err:      domain.ErrTimeout,
			expected: true,
		},
		{
			name:     "ErrNotFound (not retryable)",
			err:      domain.ErrNotFound,
			expected: false,
		},
		{
			name:     "Wrapped RetryableError",
			err:      fmt.Errorf("wrapping: %w", &domain.RetryableError{Err: errors.New("retry me")}),
			expected: true,
		},
		{
			name:     "Wrapped FetchError with retryable status",
			err:      fmt.Errorf("wrapping: %w", domain.NewFetchError("https://example.com", 503, errors.New("service unavailable"))),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.IsRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError(t *testing.T) {
	// Test NewValidationError
	valErr := domain.NewValidationError("url", "invalid format")

	assert.Equal(t, "url", valErr.Field)
	assert.Equal(t, "invalid format", valErr.Message)

	// Test Error()
	expectedMsg := "validation error for url: invalid format"
	assert.Equal(t, expectedMsg, valErr.Error())
}

func TestStrategyError(t *testing.T) {
	// Test with status code
	originalErr := errors.New("execution failed")
	strategyErr := domain.NewStrategyError("crawler", "https://example.com", originalErr)

	assert.Equal(t, "crawler", strategyErr.Strategy)
	assert.Equal(t, "https://example.com", strategyErr.URL)
	assert.Equal(t, originalErr, strategyErr.Err)

	// Test Error()
	expectedMsg := "strategy crawler failed for https://example.com: execution failed"
	assert.Equal(t, expectedMsg, strategyErr.Error())

	// Test Unwrap()
	assert.Equal(t, originalErr, strategyErr.Unwrap())
}

func TestSentinelErrors(t *testing.T) {
	// Test all sentinel errors
	assert.Equal(t, "not found", domain.ErrNotFound.Error())
	assert.Equal(t, "cache miss", domain.ErrCacheMiss.Error())
	assert.Equal(t, "cache entry expired", domain.ErrCacheExpired.Error())
	assert.Equal(t, "rate limited", domain.ErrRateLimited.Error())
	assert.Equal(t, "request blocked", domain.ErrBlocked.Error())
	assert.Equal(t, "timeout", domain.ErrTimeout.Error())
	assert.Equal(t, "invalid URL", domain.ErrInvalidURL.Error())
	assert.Equal(t, "no strategy found for URL", domain.ErrNoStrategy.Error())
	assert.Equal(t, "render failed", domain.ErrRenderFailed.Error())
	assert.Equal(t, "conversion failed", domain.ErrConversionFailed.Error())
	assert.Equal(t, "write failed", domain.ErrWriteFailed.Error())
	assert.Equal(t, "browser not found", domain.ErrBrowserNotFound.Error())
}

func TestErrorUnwrapping(t *testing.T) {
	// Test that errors can be properly unwrapped
	originalErr := errors.New("original error")

	// Test FetchError unwrapping
	fetchErr := domain.NewFetchError("https://example.com", 500, originalErr)
	wrappedFetchErr := fmt.Errorf("fetch failed: %w", fetchErr)
	require.True(t, errors.Is(wrappedFetchErr, originalErr))

	// Test RetryableError unwrapping
	retryableErr := &domain.RetryableError{Err: originalErr}
	wrappedRetryableErr := fmt.Errorf("retry needed: %w", retryableErr)
	require.True(t, errors.Is(wrappedRetryableErr, originalErr))

	// Test StrategyError unwrapping
	strategyErr := domain.NewStrategyError("crawler", "https://example.com", originalErr)
	wrappedStrategyErr := fmt.Errorf("strategy failed: %w", strategyErr)
	require.True(t, errors.Is(wrappedStrategyErr, originalErr))
}
