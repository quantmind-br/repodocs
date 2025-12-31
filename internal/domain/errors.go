package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	// ErrNotFound indicates a resource was not found
	ErrNotFound = errors.New("not found")

	// ErrCacheMiss indicates a cache miss
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheExpired indicates the cached entry has expired
	ErrCacheExpired = errors.New("cache entry expired")

	// ErrRateLimited indicates rate limiting was encountered
	ErrRateLimited = errors.New("rate limited")

	// ErrBlocked indicates the request was blocked (e.g., by Cloudflare)
	ErrBlocked = errors.New("request blocked")

	// ErrTimeout indicates a timeout occurred
	ErrTimeout = errors.New("timeout")

	// ErrInvalidURL indicates an invalid URL was provided
	ErrInvalidURL = errors.New("invalid URL")

	// ErrNoStrategy indicates no strategy can handle the URL
	ErrNoStrategy = errors.New("no strategy found for URL")

	// ErrRenderFailed indicates JavaScript rendering failed
	ErrRenderFailed = errors.New("render failed")

	// ErrConversionFailed indicates HTML to Markdown conversion failed
	ErrConversionFailed = errors.New("conversion failed")

	// ErrWriteFailed indicates writing output failed
	ErrWriteFailed = errors.New("write failed")

	// ErrBrowserNotFound indicates Chrome/Chromium was not found
	ErrBrowserNotFound = errors.New("browser not found")
)

// FetchError represents an error during fetching
type FetchError struct {
	URL        string
	StatusCode int
	Err        error
}

func (e *FetchError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("fetch error for %s: status %d: %v", e.URL, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("fetch error for %s: %v", e.URL, e.Err)
}

func (e *FetchError) Unwrap() error {
	return e.Err
}

// NewFetchError creates a new FetchError
func NewFetchError(url string, statusCode int, err error) *FetchError {
	return &FetchError{
		URL:        url,
		StatusCode: statusCode,
		Err:        err,
	}
}

// RetryableError indicates an error that can be retried
type RetryableError struct {
	Err        error
	RetryAfter int // Seconds to wait before retry, 0 if unknown
}

func (e *RetryableError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("retryable error (retry after %ds): %v", e.RetryAfter, e.Err)
	}
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	var fetchErr *FetchError
	if errors.As(err, &fetchErr) {
		// Retry on specific status codes
		switch fetchErr.StatusCode {
		case 429, 503, 502, 504:
			return true
		}
		// Retry on Cloudflare errors
		if fetchErr.StatusCode >= 520 && fetchErr.StatusCode <= 530 {
			return true
		}
	}

	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTimeout)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// StrategyError represents an error in strategy execution
type StrategyError struct {
	Strategy string
	URL      string
	Err      error
}

func (e *StrategyError) Error() string {
	return fmt.Sprintf("strategy %s failed for %s: %v", e.Strategy, e.URL, e.Err)
}

func (e *StrategyError) Unwrap() error {
	return e.Err
}

// NewStrategyError creates a new StrategyError
func NewStrategyError(strategy, url string, err error) *StrategyError {
	return &StrategyError{
		Strategy: strategy,
		URL:      url,
		Err:      err,
	}
}

// =============================================================================
// LLM Errors
// =============================================================================

// LLM sentinel errors
var (
	// ErrLLMNotConfigured indicates LLM provider is not configured
	ErrLLMNotConfigured = errors.New("LLM provider not configured")

	// ErrLLMMissingAPIKey indicates API key is required but not provided
	ErrLLMMissingAPIKey = errors.New("LLM API key is required")

	// ErrLLMMissingBaseURL indicates base URL is required but not provided
	ErrLLMMissingBaseURL = errors.New("LLM base URL is required")

	// ErrLLMMissingModel indicates model is required but not provided
	ErrLLMMissingModel = errors.New("LLM model is required")

	// ErrLLMInvalidProvider indicates an invalid provider type
	ErrLLMInvalidProvider = errors.New("invalid LLM provider")

	// ErrLLMRequestFailed indicates the LLM request failed
	ErrLLMRequestFailed = errors.New("LLM request failed")

	// ErrLLMRateLimited indicates rate limit was exceeded
	ErrLLMRateLimited = errors.New("LLM rate limit exceeded")

	// ErrLLMAuthFailed indicates authentication failed
	ErrLLMAuthFailed = errors.New("LLM authentication failed")

	// ErrLLMContextTooLong indicates context length was exceeded
	ErrLLMContextTooLong = errors.New("LLM context length exceeded")
)

// LLMError represents an LLM-specific error
type LLMError struct {
	Provider   string
	StatusCode int
	Message    string
	Err        error
}

func (e *LLMError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s error (HTTP %d): %s", e.Provider, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Provider, e.Message)
}

func (e *LLMError) Unwrap() error {
	return e.Err
}

// NewLLMError creates a new LLMError
func NewLLMError(provider string, statusCode int, message string, err error) *LLMError {
	return &LLMError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Err:        err,
	}
}
