package fetcher

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Retrier handles retry logic with exponential backoff
type Retrier struct {
	maxRetries      int
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
}

// RetrierOptions contains options for creating a Retrier
type RetrierOptions struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultRetrierOptions returns default retrier options
func DefaultRetrierOptions() RetrierOptions {
	return RetrierOptions{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// NewRetrier creates a new Retrier with the given options
func NewRetrier(opts RetrierOptions) *Retrier {
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.InitialInterval <= 0 {
		opts.InitialInterval = 1 * time.Second
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = 30 * time.Second
	}
	if opts.Multiplier <= 0 {
		opts.Multiplier = 2.0
	}

	return &Retrier{
		maxRetries:      opts.MaxRetries,
		initialInterval: opts.InitialInterval,
		maxInterval:     opts.MaxInterval,
		multiplier:      opts.Multiplier,
	}
}

// newBackoff creates a new exponential backoff
func (r *Retrier) newBackoff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = r.initialInterval
	b.MaxInterval = r.maxInterval
	b.Multiplier = r.multiplier
	b.RandomizationFactor = 0.5
	b.Reset()

	return backoff.WithMaxRetries(b, uint64(r.maxRetries))
}

// Retry executes an operation with exponential backoff
func (r *Retrier) Retry(ctx context.Context, operation func() error) error {
	b := r.newBackoff()
	b = backoff.WithContext(b, ctx)

	return backoff.Retry(func() error {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !domain.IsRetryable(err) {
			return backoff.Permanent(err)
		}

		return err
	}, b)
}

// RetryWithValue executes an operation with exponential backoff and returns a value
func RetryWithValue[T any](ctx context.Context, r *Retrier, operation func() (T, error)) (T, error) {
	var result T
	var lastErr error

	b := r.newBackoff()
	b = backoff.WithContext(b, ctx)

	err := backoff.Retry(func() error {
		var err error
		result, err = operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !domain.IsRetryable(err) {
			return backoff.Permanent(err)
		}

		return err
	}, b)

	if err != nil {
		return result, lastErr
	}

	return result, nil
}

// ShouldRetryStatus returns true if the HTTP status code should be retried
func ShouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case 429: // Too Many Requests
		return true
	case 502: // Bad Gateway
		return true
	case 503: // Service Unavailable
		return true
	case 504: // Gateway Timeout
		return true
	}

	// Cloudflare errors (520-530)
	if statusCode >= 520 && statusCode <= 530 {
		return true
	}

	return false
}

// ParseRetryAfter parses the Retry-After header value
func ParseRetryAfter(retryAfter string) time.Duration {
	if retryAfter == "" {
		return 0
	}

	// Try to parse as seconds
	var seconds int
	if _, err := parseRetryAfterInt(retryAfter, &seconds); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as HTTP date (simplified)
	// Full parsing would use time.Parse with HTTP date format
	return 0
}

// parseRetryAfterInt is a helper to parse retry-after as int
func parseRetryAfterInt(s string, result *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return n, nil
}
