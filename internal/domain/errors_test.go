package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSentinelErrors verifies sentinel errors are defined
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check string
	}{
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrCacheMiss", ErrCacheMiss, "cache miss"},
		{"ErrCacheExpired", ErrCacheExpired, "cache entry expired"},
		{"ErrRateLimited", ErrRateLimited, "rate limited"},
		{"ErrBlocked", ErrBlocked, "request blocked"},
		{"ErrTimeout", ErrTimeout, "timeout"},
		{"ErrInvalidURL", ErrInvalidURL, "invalid URL"},
		{"ErrNoStrategy", ErrNoStrategy, "no strategy found for URL"},
		{"ErrRenderFailed", ErrRenderFailed, "render failed"},
		{"ErrConversionFailed", ErrConversionFailed, "conversion failed"},
		{"ErrWriteFailed", ErrWriteFailed, "write failed"},
		{"ErrBrowserNotFound", ErrBrowserNotFound, "browser not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.check)
		})
	}
}

// TestLLMSentinelErrors verifies LLM sentinel errors are defined
func TestLLMSentinelErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check string
	}{
		{"ErrLLMNotConfigured", ErrLLMNotConfigured, "not configured"},
		{"ErrLLMMissingAPIKey", ErrLLMMissingAPIKey, "API key is required"},
		{"ErrLLMMissingBaseURL", ErrLLMMissingBaseURL, "base URL is required"},
		{"ErrLLMMissingModel", ErrLLMMissingModel, "model is required"},
		{"ErrLLMInvalidProvider", ErrLLMInvalidProvider, "invalid LLM provider"},
		{"ErrLLMRequestFailed", ErrLLMRequestFailed, "request failed"},
		{"ErrLLMRateLimited", ErrLLMRateLimited, "rate limit exceeded"},
		{"ErrLLMAuthFailed", ErrLLMAuthFailed, "authentication failed"},
		{"ErrLLMContextTooLong", ErrLLMContextTooLong, "context length exceeded"},
		{"ErrLLMCircuitOpen", ErrLLMCircuitOpen, "circuit breaker is open"},
		{"ErrLLMMaxRetriesExceeded", ErrLLMMaxRetriesExceeded, "max retries exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Contains(t, tt.err.Error(), tt.check)
		})
	}
}

// TestFetchError tests FetchError methods
func TestFetchError(t *testing.T) {
	t.Run("Error with status code", func(t *testing.T) {
		baseErr := errors.New("connection failed")
		err := &FetchError{
			URL:        "https://example.com",
			StatusCode: 503,
			Err:        baseErr,
		}

		assert.Contains(t, err.Error(), "https://example.com")
		assert.Contains(t, err.Error(), "503")
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("Error without status code", func(t *testing.T) {
		baseErr := errors.New("connection refused")
		err := &FetchError{
			URL: "https://example.com",
			Err: baseErr,
		}

		assert.Contains(t, err.Error(), "https://example.com")
		assert.Contains(t, err.Error(), "connection refused")
		assert.NotContains(t, err.Error(), "status")
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &FetchError{
			URL: "https://example.com",
			Err: baseErr,
		}

		assert.Equal(t, baseErr, errors.Unwrap(err))
	})

	t.Run("NewFetchError creates correct error", func(t *testing.T) {
		baseErr := errors.New("timeout")
		err := NewFetchError("https://example.com", 504, baseErr)

		assert.Equal(t, "https://example.com", err.URL)
		assert.Equal(t, 504, err.StatusCode)
		assert.Equal(t, baseErr, err.Err)
	})
}

// TestRetryableError tests RetryableError methods
func TestRetryableError(t *testing.T) {
	t.Run("Error with retry after", func(t *testing.T) {
		baseErr := errors.New("too many requests")
		err := &RetryableError{
			Err:        baseErr,
			RetryAfter: 120,
		}

		assert.Contains(t, err.Error(), "retry after 120s")
		assert.Contains(t, err.Error(), "too many requests")
	})

	t.Run("Error without retry after", func(t *testing.T) {
		baseErr := errors.New("gateway timeout")
		err := &RetryableError{
			Err:        baseErr,
			RetryAfter: 0,
		}

		assert.Contains(t, err.Error(), "retryable error")
		assert.Contains(t, err.Error(), "gateway timeout")
		assert.NotContains(t, err.Error(), "retry after")
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &RetryableError{
			Err: baseErr,
		}

		assert.Equal(t, baseErr, errors.Unwrap(err))
	})
}

// TestIsRetryable tests the IsRetryable function
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "RetryableError is retryable",
			err:      &RetryableError{Err: errors.New("error")},
			expected: true,
		},
		{
			name: "FetchError with 429 is retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 429,
				Err:        errors.New("too many requests"),
			},
			expected: true,
		},
		{
			name: "FetchError with 502 is retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 502,
				Err:        errors.New("bad gateway"),
			},
			expected: true,
		},
		{
			name: "FetchError with 503 is retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 503,
				Err:        errors.New("service unavailable"),
			},
			expected: true,
		},
		{
			name: "FetchError with 504 is retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 504,
				Err:        errors.New("gateway timeout"),
			},
			expected: true,
		},
		{
			name: "FetchError with 520 is retryable (Cloudflare)",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 520,
				Err:        errors.New("cloudflare error"),
			},
			expected: true,
		},
		{
			name: "FetchError with 525 is retryable (Cloudflare)",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 525,
				Err:        errors.New("cloudflare error"),
			},
			expected: true,
		},
		{
			name: "FetchError with 530 is retryable (Cloudflare)",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 530,
				Err:        errors.New("cloudflare error"),
			},
			expected: true,
		},
		{
			name: "FetchError with 404 is not retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 404,
				Err:        errors.New("not found"),
			},
			expected: false,
		},
		{
			name: "FetchError with 500 is not retryable",
			err: &FetchError{
				URL:        "https://example.com",
				StatusCode: 500,
				Err:        errors.New("internal server error"),
			},
			expected: false,
		},
		{
			name:     "ErrRateLimited is retryable",
			err:      ErrRateLimited,
			expected: true,
		},
		{
			name:     "ErrTimeout is retryable",
			err:      ErrTimeout,
			expected: true,
		},
		{
			name:     "Generic error is not retryable",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "ErrNotFound is not retryable",
			err:      ErrNotFound,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidationError tests ValidationError methods
func TestValidationError(t *testing.T) {
	t.Run("Error method formats correctly", func(t *testing.T) {
		err := &ValidationError{
			Field:   "URL",
			Message: "must be a valid HTTP/HTTPS URL",
		}

		assert.Contains(t, err.Error(), "validation error")
		assert.Contains(t, err.Error(), "URL")
		assert.Contains(t, err.Error(), "must be a valid HTTP/HTTPS URL")
	})

	t.Run("NewValidationError creates correct error", func(t *testing.T) {
		err := NewValidationError("timeout", "must be positive")

		assert.Equal(t, "timeout", err.Field)
		assert.Equal(t, "must be positive", err.Message)
	})
}

// TestStrategyError tests StrategyError methods
func TestStrategyError(t *testing.T) {
	t.Run("Error method formats correctly", func(t *testing.T) {
		baseErr := errors.New("authentication failed")
		err := &StrategyError{
			Strategy: "git",
			URL:      "https://github.com/owner/repo",
			Err:      baseErr,
		}

		errStr := err.Error()
		assert.Contains(t, errStr, "git")
		assert.Contains(t, errStr, "https://github.com/owner/repo")
		assert.Contains(t, errStr, "authentication failed")
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &StrategyError{
			Strategy: "crawler",
			URL:      "https://example.com",
			Err:      baseErr,
		}

		assert.Equal(t, baseErr, errors.Unwrap(err))
	})

	t.Run("NewStrategyError creates correct error", func(t *testing.T) {
		baseErr := errors.New("connection failed")
		err := NewStrategyError("sitemap", "https://example.com/sitemap.xml", baseErr)

		assert.Equal(t, "sitemap", err.Strategy)
		assert.Equal(t, "https://example.com/sitemap.xml", err.URL)
		assert.Equal(t, baseErr, err.Err)
	})
}

// TestLLMError tests LLMError methods
func TestLLMError(t *testing.T) {
	t.Run("Error with status code", func(t *testing.T) {
		baseErr := errors.New("invalid API key")
		err := &LLMError{
			Provider:   "openai",
			StatusCode: 401,
			Message:    "Authentication failed",
			Err:        baseErr,
		}

		errStr := err.Error()
		assert.Contains(t, errStr, "openai")
		assert.Contains(t, errStr, "401")
		assert.Contains(t, errStr, "Authentication failed")
	})

	t.Run("Error without status code", func(t *testing.T) {
		err := &LLMError{
			Provider: "anthropic",
			Message:  "Rate limit exceeded",
		}

		errStr := err.Error()
		assert.Contains(t, errStr, "anthropic")
		assert.Contains(t, errStr, "Rate limit exceeded")
		assert.NotContains(t, errStr, "HTTP")
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := &LLMError{
			Provider: "google",
			Message:  "Error",
			Err:      baseErr,
		}

		assert.Equal(t, baseErr, errors.Unwrap(err))
	})

	t.Run("NewLLMError creates correct error", func(t *testing.T) {
		baseErr := errors.New("timeout")
		err := NewLLMError("openai", 504, "Gateway timeout", baseErr)

		assert.Equal(t, "openai", err.Provider)
		assert.Equal(t, 504, err.StatusCode)
		assert.Equal(t, "Gateway timeout", err.Message)
		assert.Equal(t, baseErr, err.Err)
	})
}

// TestErrorWrapping tests error wrapping and unwrapping
func TestErrorWrapping(t *testing.T) {
	t.Run("FetchError unwraps correctly", func(t *testing.T) {
		baseErr := errors.New("base")
		fetchErr := &FetchError{URL: "http://example.com", Err: baseErr}

		assert.True(t, errors.Is(fetchErr, baseErr))
	})

	t.Run("RetryableError unwraps correctly", func(t *testing.T) {
		baseErr := errors.New("base")
		retryErr := &RetryableError{Err: baseErr}

		assert.True(t, errors.Is(retryErr, baseErr))
	})

	t.Run("StrategyError unwraps correctly", func(t *testing.T) {
		baseErr := errors.New("base")
		strategyErr := &StrategyError{
			Strategy: "test",
			URL:      "http://example.com",
			Err:      baseErr,
		}

		assert.True(t, errors.Is(strategyErr, baseErr))
	})

	t.Run("LLMError unwraps correctly", func(t *testing.T) {
		baseErr := errors.New("base")
		llmErr := &LLMError{
			Provider: "test",
			Message:  "error",
			Err:      baseErr,
		}

		assert.True(t, errors.Is(llmErr, baseErr))
	})
}
