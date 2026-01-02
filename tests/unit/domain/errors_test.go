package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Sentinel Errors Tests
// ============================================================================

func TestSentinelErrors_NotNil(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check func(t *testing.T, err error)
	}{
		{
			name: "ErrNotFound",
			err:  domain.ErrNotFound,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "not found", err.Error())
			},
		},
		{
			name: "ErrCacheMiss",
			err:  domain.ErrCacheMiss,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "cache miss", err.Error())
			},
		},
		{
			name: "ErrCacheExpired",
			err:  domain.ErrCacheExpired,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "cache entry expired", err.Error())
			},
		},
		{
			name: "ErrRateLimited",
			err:  domain.ErrRateLimited,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "rate limited", err.Error())
			},
		},
		{
			name: "ErrBlocked",
			err:  domain.ErrBlocked,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "request blocked", err.Error())
			},
		},
		{
			name: "ErrTimeout",
			err:  domain.ErrTimeout,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "timeout", err.Error())
			},
		},
		{
			name: "ErrInvalidURL",
			err:  domain.ErrInvalidURL,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "invalid URL", err.Error())
			},
		},
		{
			name: "ErrNoStrategy",
			err:  domain.ErrNoStrategy,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "no strategy found for URL", err.Error())
			},
		},
		{
			name: "ErrRenderFailed",
			err:  domain.ErrRenderFailed,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "render failed", err.Error())
			},
		},
		{
			name: "ErrConversionFailed",
			err:  domain.ErrConversionFailed,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "conversion failed", err.Error())
			},
		},
		{
			name: "ErrWriteFailed",
			err:  domain.ErrWriteFailed,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "write failed", err.Error())
			},
		},
		{
			name: "ErrBrowserNotFound",
			err:  domain.ErrBrowserNotFound,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "browser not found", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.err)
		})
	}
}

// ============================================================================
// LLM Sentinel Errors Tests
// ============================================================================

func TestLLMSentinelErrors_NotNil(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check func(t *testing.T, err error)
	}{
		{
			name: "ErrLLMNotConfigured",
			err:  domain.ErrLLMNotConfigured,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM provider not configured", err.Error())
			},
		},
		{
			name: "ErrLLMMissingAPIKey",
			err:  domain.ErrLLMMissingAPIKey,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM API key is required", err.Error())
			},
		},
		{
			name: "ErrLLMMissingBaseURL",
			err:  domain.ErrLLMMissingBaseURL,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM base URL is required", err.Error())
			},
		},
		{
			name: "ErrLLMMissingModel",
			err:  domain.ErrLLMMissingModel,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM model is required", err.Error())
			},
		},
		{
			name: "ErrLLMInvalidProvider",
			err:  domain.ErrLLMInvalidProvider,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "invalid LLM provider", err.Error())
			},
		},
		{
			name: "ErrLLMRequestFailed",
			err:  domain.ErrLLMRequestFailed,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM request failed", err.Error())
			},
		},
		{
			name: "ErrLLMRateLimited",
			err:  domain.ErrLLMRateLimited,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM rate limit exceeded", err.Error())
			},
		},
		{
			name: "ErrLLMAuthFailed",
			err:  domain.ErrLLMAuthFailed,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM authentication failed", err.Error())
			},
		},
		{
			name: "ErrLLMContextTooLong",
			err:  domain.ErrLLMContextTooLong,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM context length exceeded", err.Error())
			},
		},
		{
			name: "ErrLLMCircuitOpen",
			err:  domain.ErrLLMCircuitOpen,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM circuit breaker is open", err.Error())
			},
		},
		{
			name: "ErrLLMMaxRetriesExceeded",
			err:  domain.ErrLLMMaxRetriesExceeded,
			check: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "LLM max retries exceeded", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.err)
		})
	}
}

// ============================================================================
// FetchError Tests
// ============================================================================

func TestNewFetchError(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		statusCode  int
		err         error
		verify      func(t *testing.T, fe *domain.FetchError)
	}{
		{
			name:       "with status code",
			url:        "https://example.com/docs",
			statusCode: 404,
			err:        errors.New("page not found"),
			verify: func(t *testing.T, fe *domain.FetchError) {
				assert.NotNil(t, fe)
				assert.Equal(t, "https://example.com/docs", fe.URL)
				assert.Equal(t, 404, fe.StatusCode)
				assert.Equal(t, "page not found", fe.Err.Error())
			},
		},
		{
			name:       "with 500 status code",
			url:        "https://example.com/error",
			statusCode: 500,
			err:        errors.New("internal server error"),
			verify: func(t *testing.T, fe *domain.FetchError) {
				assert.NotNil(t, fe)
				assert.Equal(t, "https://example.com/error", fe.URL)
				assert.Equal(t, 500, fe.StatusCode)
				assert.Equal(t, "internal server error", fe.Err.Error())
			},
		},
		{
			name:       "with zero status code",
			url:        "https://example.com/timeout",
			statusCode: 0,
			err:        errors.New("connection timeout"),
			verify: func(t *testing.T, fe *domain.FetchError) {
				assert.NotNil(t, fe)
				assert.Equal(t, "https://example.com/timeout", fe.URL)
				assert.Equal(t, 0, fe.StatusCode)
				assert.Equal(t, "connection timeout", fe.Err.Error())
			},
		},
		{
			name:       "with rate limit status",
			url:        "https://example.com/rate-limited",
			statusCode: 429,
			err:        domain.ErrRateLimited,
			verify: func(t *testing.T, fe *domain.FetchError) {
				assert.NotNil(t, fe)
				assert.Equal(t, 429, fe.StatusCode)
				assert.Same(t, domain.ErrRateLimited, fe.Err)
			},
		},
		{
			name:       "with nil error",
			url:        "https://example.com/unknown",
			statusCode: 520,
			err:        nil,
			verify: func(t *testing.T, fe *domain.FetchError) {
				assert.NotNil(t, fe)
				assert.Equal(t, 520, fe.StatusCode)
				assert.Nil(t, fe.Err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := domain.NewFetchError(tt.url, tt.statusCode, tt.err)
			tt.verify(t, fe)
		})
	}
}

func TestFetchError_Error(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		statusCode int
		err        error
		want       string
	}{
		{
			name:       "with status code > 0",
			url:        "https://example.com/docs",
			statusCode: 404,
			err:        errors.New("not found"),
			want:       "fetch error for https://example.com/docs: status 404: not found",
		},
		{
			name:       "with status code = 0",
			url:        "https://example.com/timeout",
			statusCode: 0,
			err:        errors.New("timeout"),
			want:       "fetch error for https://example.com/timeout: timeout",
		},
		{
			name:       "with Cloudflare error",
			url:        "https://example.com/blocked",
			statusCode: 522,
			err:        errors.New("connection timed out"),
			want:       "fetch error for https://example.com/blocked: status 522: connection timed out",
		},
		{
			name:       "with nil error and status code",
			url:        "https://example.com/unknown",
			statusCode: 500,
			err:        nil,
			want:       "fetch error for https://example.com/unknown: status 500: ",
		},
		{
			name:       "with nil error and no status code",
			url:        "https://example.com/unknown",
			statusCode: 0,
			err:        nil,
			want:       "fetch error for https://example.com/unknown: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fe := domain.NewFetchError(tt.url, tt.statusCode, tt.err)
			assert.Equal(t, tt.want, fe.Error())
		})
	}
}

func TestFetchError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	fe := domain.NewFetchError("https://example.com", 500, baseErr)

	assert.Same(t, baseErr, fe.Unwrap(), "Unwrap should return the original error")
}

func TestFetchError_UnwrapNil(t *testing.T) {
	fe := domain.NewFetchError("https://example.com", 500, nil)

	assert.Nil(t, fe.Unwrap(), "Unwrap should return nil when error is nil")
}

// ============================================================================
// RetryableError Tests
// ============================================================================

func TestRetryableError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryAfter int
		want       string
	}{
		{
			name:       "with retry after > 0",
			err:        errors.New("rate limited"),
			retryAfter: 60,
			want:       "retryable error (retry after 60s): rate limited",
		},
		{
			name:       "with retry after = 0",
			err:        errors.New("timeout"),
			retryAfter: 0,
			want:       "retryable error: timeout",
		},
		{
			name:       "with sentinel error",
			err:        domain.ErrRateLimited,
			retryAfter: 30,
			want:       "retryable error (retry after 30s): rate limited",
		},
		{
			name:       "with nil error",
			err:        nil,
			retryAfter: 10,
			want:       "retryable error (retry after 10s): ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := &domain.RetryableError{
				Err:        tt.err,
				RetryAfter: tt.retryAfter,
			}
			assert.Equal(t, tt.want, re.Error())
		})
	}
}

func TestRetryableError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	re := &domain.RetryableError{
		Err:        baseErr,
		RetryAfter: 30,
	}

	assert.Same(t, baseErr, re.Unwrap(), "Unwrap should return the original error")
}

func TestRetryableError_UnwrapNil(t *testing.T) {
	re := &domain.RetryableError{
		Err:        nil,
		RetryAfter: 10,
	}

	assert.Nil(t, re.Unwrap(), "Unwrap should return nil when error is nil")
}

// ============================================================================
// IsRetryable Tests
// ============================================================================

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "RetryableError",
			err: &domain.RetryableError{
				Err:        errors.New("temporary failure"),
				RetryAfter: 10,
			},
			want: true,
		},
		{
			name: "FetchError with 429 status",
			err:  domain.NewFetchError("https://example.com", 429, errors.New("too many requests")),
			want: true,
		},
		{
			name: "FetchError with 503 status",
			err:  domain.NewFetchError("https://example.com", 503, errors.New("service unavailable")),
			want: true,
		},
		{
			name: "FetchError with 502 status",
			err:  domain.NewFetchError("https://example.com", 502, errors.New("bad gateway")),
			want: true,
		},
		{
			name: "FetchError with 504 status",
			err:  domain.NewFetchError("https://example.com", 504, errors.New("gateway timeout")),
			want: true,
		},
		{
			name: "FetchError with 520 status (Cloudflare)",
			err:  domain.NewFetchError("https://example.com", 520, errors.New("unknown error")),
			want: true,
		},
		{
			name: "FetchError with 525 status (Cloudflare)",
			err:  domain.NewFetchError("https://example.com", 525, errors.New("SSL handshake failed")),
			want: true,
		},
		{
			name: "FetchError with 530 status (Cloudflare)",
			err:  domain.NewFetchError("https://example.com", 530, errors.New("origin DNS error")),
			want: true,
		},
		{
			name: "FetchError with 404 status",
			err:  domain.NewFetchError("https://example.com", 404, errors.New("not found")),
			want: false,
		},
		{
			name: "FetchError with 400 status",
			err:  domain.NewFetchError("https://example.com", 400, errors.New("bad request")),
			want: false,
		},
		{
			name: "FetchError with 500 status (not retryable)",
			err:  domain.NewFetchError("https://example.com", 500, errors.New("internal server error")),
			want: false,
		},
		{
			name: "ErrRateLimited sentinel",
			err:  domain.ErrRateLimited,
			want: true,
		},
		{
			name: "ErrTimeout sentinel",
			err:  domain.ErrTimeout,
			want: true,
		},
		{
			name: "ErrNotFound sentinel",
			err:  domain.ErrNotFound,
			want: false,
		},
		{
			name: "ErrInvalidURL sentinel",
			err:  domain.ErrInvalidURL,
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "wrapped RetryableError",
			err:  fmt.Errorf("wrapped: %w", &domain.RetryableError{Err: errors.New("temp"), RetryAfter: 5}),
			want: true,
		},
		{
			name: "wrapped FetchError with retryable status",
			err:  fmt.Errorf("wrapped: %w", domain.NewFetchError("https://example.com", 503, errors.New("unavailable"))),
			want: true,
		},
		{
			name: "wrapped ErrRateLimited",
			err:  fmt.Errorf("wrapped: %w", domain.ErrRateLimited),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.IsRetryable(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================================
// ValidationError Tests
// ============================================================================

func TestNewValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
		verify  func(t *testing.T, ve *domain.ValidationError)
	}{
		{
			name:    "with field and message",
			field:   "url",
			message: "URL must be valid",
			verify: func(t *testing.T, ve *domain.ValidationError) {
				assert.NotNil(t, ve)
				assert.Equal(t, "url", ve.Field)
				assert.Equal(t, "URL must be valid", ve.Message)
			},
		},
		{
			name:    "with empty field",
			field:   "",
			message: "empty field",
			verify: func(t *testing.T, ve *domain.ValidationError) {
				assert.NotNil(t, ve)
				assert.Empty(t, ve.Field)
				assert.Equal(t, "empty field", ve.Message)
			},
		},
		{
			name:    "with empty message",
			field:   "config",
			message: "",
			verify: func(t *testing.T, ve *domain.ValidationError) {
				assert.NotNil(t, ve)
				assert.Equal(t, "config", ve.Field)
				assert.Empty(t, ve.Message)
			},
		},
		{
			name:    "with special characters",
			field:   "path/to/field",
			message: "Error: value 'test' is invalid (must be > 0)",
			verify: func(t *testing.T, ve *domain.ValidationError) {
				assert.NotNil(t, ve)
				assert.Equal(t, "path/to/field", ve.Field)
				assert.Equal(t, "Error: value 'test' is invalid (must be > 0)", ve.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := domain.NewValidationError(tt.field, tt.message)
			tt.verify(t, ve)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		message string
		want    string
	}{
		{
			name:    "standard validation error",
			field:   "url",
			message: "invalid URL format",
			want:    "validation error for url: invalid URL format",
		},
		{
			name:    "with empty field",
			field:   "",
			message: "unknown field",
			want:    "validation error for : unknown field",
		},
		{
			name:    "with empty message",
			field:   "config",
			message: "",
			want:    "validation error for config: ",
		},
		{
			name:    "with special characters",
			field:   "max_retries",
			message: "value must be >= 0",
			want:    "validation error for max_retries: value must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := domain.NewValidationError(tt.field, tt.message)
			assert.Equal(t, tt.want, ve.Error())
		})
	}
}

// ============================================================================
// StrategyError Tests
// ============================================================================

func TestNewStrategyError(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		url      string
		err      error
		verify   func(t *testing.T, se *domain.StrategyError)
	}{
		{
			name:     "full strategy error",
			strategy: "crawler",
			url:      "https://example.com/docs",
			err:      errors.New("failed to fetch"),
			verify: func(t *testing.T, se *domain.StrategyError) {
				assert.NotNil(t, se)
				assert.Equal(t, "crawler", se.Strategy)
				assert.Equal(t, "https://example.com/docs", se.URL)
				assert.Equal(t, "failed to fetch", se.Err.Error())
			},
		},
		{
			name:     "git strategy error",
			strategy: "git",
			url:      "https://github.com/test/repo",
			err:      domain.ErrNotFound,
			verify: func(t *testing.T, se *domain.StrategyError) {
				assert.NotNil(t, se)
				assert.Equal(t, "git", se.Strategy)
				assert.Equal(t, "https://github.com/test/repo", se.URL)
				assert.Same(t, domain.ErrNotFound, se.Err)
			},
		},
		{
			name:     "sitemap strategy error",
			strategy: "sitemap",
			url:      "https://example.com/sitemap.xml",
			err:      errors.New("XML parse error"),
			verify: func(t *testing.T, se *domain.StrategyError) {
				assert.NotNil(t, se)
				assert.Equal(t, "sitemap", se.Strategy)
				assert.Equal(t, "https://example.com/sitemap.xml", se.URL)
				assert.Equal(t, "XML parse error", se.Err.Error())
			},
		},
		{
			name:     "with nil error",
			strategy: "llms",
			url:      "https://example.com/llms.txt",
			err:      nil,
			verify: func(t *testing.T, se *domain.StrategyError) {
				assert.NotNil(t, se)
				assert.Equal(t, "llms", se.Strategy)
				assert.Equal(t, "https://example.com/llms.txt", se.URL)
				assert.Nil(t, se.Err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := domain.NewStrategyError(tt.strategy, tt.url, tt.err)
			tt.verify(t, se)
		})
	}
}

func TestStrategyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		url      string
		err      error
		want     string
	}{
		{
			name:     "standard strategy error",
			strategy: "crawler",
			url:      "https://example.com/docs",
			err:      errors.New("failed to fetch"),
			want:     "strategy crawler failed for https://example.com/docs: failed to fetch",
		},
		{
			name:     "git strategy error",
			strategy: "git",
			url:      "https://github.com/test/repo",
			err:      errors.New("clone failed"),
			want:     "strategy git failed for https://github.com/test/repo: clone failed",
		},
		{
			name:     "with nil error",
			strategy: "pkggo",
			url:      "https://pkg.go.dev/example",
			err:      nil,
			want:     "strategy pkggo failed for https://pkg.go.dev/example: ",
		},
		{
			name:     "with sentinel error",
			strategy: "renderer",
			url:      "https://example.com/spa",
			err:      domain.ErrRenderFailed,
			want:     "strategy renderer failed for https://example.com/spa: render failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := domain.NewStrategyError(tt.strategy, tt.url, tt.err)
			assert.Equal(t, tt.want, se.Error())
		})
	}
}

func TestStrategyError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	se := domain.NewStrategyError("crawler", "https://example.com", baseErr)

	assert.Same(t, baseErr, se.Unwrap(), "Unwrap should return the original error")
}

func TestStrategyError_UnwrapNil(t *testing.T) {
	se := domain.NewStrategyError("crawler", "https://example.com", nil)

	assert.Nil(t, se.Unwrap(), "Unwrap should return nil when error is nil")
}

// ============================================================================
// LLMError Tests
// ============================================================================

func TestNewLLMError(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		statusCode  int
		message     string
		err         error
		verify      func(t *testing.T, le *domain.LLMError)
	}{
		{
			name:       "full LLM error with status code",
			provider:   "openai",
			statusCode: 401,
			message:    "invalid API key",
			err:        errors.New("authentication failed"),
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Equal(t, "openai", le.Provider)
				assert.Equal(t, 401, le.StatusCode)
				assert.Equal(t, "invalid API key", le.Message)
				assert.Equal(t, "authentication failed", le.Err.Error())
			},
		},
		{
			name:       "anthropic error with status",
			provider:   "anthropic",
			statusCode: 429,
			message:    "rate limit exceeded",
			err:        domain.ErrLLMRateLimited,
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Equal(t, "anthropic", le.Provider)
				assert.Equal(t, 429, le.StatusCode)
				assert.Equal(t, "rate limit exceeded", le.Message)
				assert.Same(t, domain.ErrLLMRateLimited, le.Err)
			},
		},
		{
			name:       "google error with status",
			provider:   "google",
			statusCode: 400,
			message:    "invalid request",
			err:        errors.New("bad request"),
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Equal(t, "google", le.Provider)
				assert.Equal(t, 400, le.StatusCode)
				assert.Equal(t, "invalid request", le.Message)
				assert.Equal(t, "bad request", le.Err.Error())
			},
		},
		{
			name:       "with zero status code",
			provider:   "openai",
			statusCode: 0,
			message:    "network error",
			err:        errors.New("connection failed"),
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Equal(t, "openai", le.Provider)
				assert.Equal(t, 0, le.StatusCode)
				assert.Equal(t, "network error", le.Message)
				assert.Equal(t, "connection failed", le.Err.Error())
			},
		},
		{
			name:       "with nil error",
			provider:   "anthropic",
			statusCode: 500,
			message:    "internal error",
			err:        nil,
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Equal(t, "anthropic", le.Provider)
				assert.Equal(t, 500, le.StatusCode)
				assert.Equal(t, "internal error", le.Message)
				assert.Nil(t, le.Err)
			},
		},
		{
			name:       "with empty provider",
			provider:   "",
			statusCode: 401,
			message:    "unknown provider",
			err:        errors.New("provider not set"),
			verify: func(t *testing.T, le *domain.LLMError) {
				assert.NotNil(t, le)
				assert.Empty(t, le.Provider)
				assert.Equal(t, 401, le.StatusCode)
				assert.Equal(t, "unknown provider", le.Message)
				assert.Equal(t, "provider not set", le.Err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			le := domain.NewLLMError(tt.provider, tt.statusCode, tt.message, tt.err)
			tt.verify(t, le)
		})
	}
}

func TestLLMError_Error(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		statusCode int
		message    string
		err        error
		want       string
	}{
		{
			name:       "with status code > 0",
			provider:   "openai",
			statusCode: 401,
			message:    "invalid API key",
			err:        errors.New("auth failed"),
			want:       "openai error (HTTP 401): invalid API key",
		},
		{
			name:       "with status code = 0",
			provider:   "anthropic",
			statusCode: 0,
			message:    "network error",
			err:        errors.New("connection failed"),
			want:       "anthropic error: network error",
		},
		{
			name:       "with rate limit status",
			provider:   "google",
			statusCode: 429,
			message:    "quota exceeded",
			err:        domain.ErrLLMRateLimited,
			want:       "google error (HTTP 429): quota exceeded",
		},
		{
			name:       "with nil error and status",
			provider:   "openai",
			statusCode: 500,
			message:    "server error",
			err:        nil,
			want:       "openai error (HTTP 500): server error",
		},
		{
			name:       "with nil error and no status",
			provider:   "anthropic",
			statusCode: 0,
			message:    "unknown error",
			err:        nil,
			want:       "anthropic error: unknown error",
		},
		{
			name:       "with cloudflare status",
			provider:   "google",
			statusCode: 524,
			message:    "timeout",
			err:        errors.New("request timed out"),
			want:       "google error (HTTP 524): timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			le := domain.NewLLMError(tt.provider, tt.statusCode, tt.message, tt.err)
			assert.Equal(t, tt.want, le.Error())
		})
	}
}

func TestLLMError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	le := domain.NewLLMError("openai", 401, "auth failed", baseErr)

	assert.Same(t, baseErr, le.Unwrap(), "Unwrap should return the original error")
}

func TestLLMError_UnwrapNil(t *testing.T) {
	le := domain.NewLLMError("openai", 500, "server error", nil)

	assert.Nil(t, le.Unwrap(), "Unwrap should return nil when error is nil")
}

// ============================================================================
// Error Type Assertions Tests
// ============================================================================

func TestErrorTypeAssertions(t *testing.T) {
	t.Run("FetchError type assertion", func(t *testing.T) {
		baseErr := errors.New("base")
		fe := domain.NewFetchError("https://example.com", 404, baseErr)

		var fetchErr *domain.FetchError
		assert.True(t, errors.As(fe, &fetchErr))
		assert.Equal(t, 404, fetchErr.StatusCode)
	})

	t.Run("RetryableError type assertion", func(t *testing.T) {
		re := &domain.RetryableError{
			Err:        errors.New("temp"),
			RetryAfter: 30,
		}

		var retryableErr *domain.RetryableError
		assert.True(t, errors.As(re, &retryableErr))
		assert.Equal(t, 30, retryableErr.RetryAfter)
	})

	t.Run("ValidationError type assertion", func(t *testing.T) {
		ve := domain.NewValidationError("url", "invalid")

		var validationErr *domain.ValidationError
		assert.True(t, errors.As(ve, &validationErr))
		assert.Equal(t, "url", validationErr.Field)
	})

	t.Run("StrategyError type assertion", func(t *testing.T) {
		se := domain.NewStrategyError("crawler", "https://example.com", errors.New("failed"))

		var strategyErr *domain.StrategyError
		assert.True(t, errors.As(se, &strategyErr))
		assert.Equal(t, "crawler", strategyErr.Strategy)
	})

	t.Run("LLMError type assertion", func(t *testing.T) {
		le := domain.NewLLMError("openai", 401, "auth", errors.New("failed"))

		var llmErr *domain.LLMError
		assert.True(t, errors.As(le, &llmErr))
		assert.Equal(t, "openai", llmErr.Provider)
	})

	t.Run("wrapped error type assertion", func(t *testing.T) {
		fe := domain.NewFetchError("https://example.com", 500, errors.New("error"))
		wrapped := fmt.Errorf("wrapped: %w", fe)

		var fetchErr *domain.FetchError
		assert.True(t, errors.As(wrapped, &fetchErr))
		assert.Equal(t, 500, fetchErr.StatusCode)
	})
}

// ============================================================================
// Error Comparison Tests
// ============================================================================

func TestErrorComparison(t *testing.T) {
	t.Run("sentinel errors are identical", func(t *testing.T) {
		assert.Same(t, domain.ErrNotFound, domain.ErrNotFound)
		assert.Same(t, domain.ErrCacheMiss, domain.ErrCacheMiss)
		assert.Same(t, domain.ErrRateLimited, domain.ErrRateLimited)
	})

	t.Run("sentinel LLM errors are identical", func(t *testing.T) {
		assert.Same(t, domain.ErrLLMNotConfigured, domain.ErrLLMNotConfigured)
		assert.Same(t, domain.ErrLLMMissingAPIKey, domain.ErrLLMMissingAPIKey)
		assert.Same(t, domain.ErrLLMRateLimited, domain.ErrLLMRateLimited)
	})

	t.Run("errors.Is with sentinel errors", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", domain.ErrNotFound)
		assert.True(t, errors.Is(err, domain.ErrNotFound))

		err2 := fmt.Errorf("wrapped: %w", domain.ErrRateLimited)
		assert.True(t, errors.Is(err2, domain.ErrRateLimited))
	})
}
