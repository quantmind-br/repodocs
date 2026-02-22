package llm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	JitterFactor    float64
}

// DefaultRetryConfig returns sensible retry defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     60 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.1,
	}
}

// Retrier executes operations with retry logic
type Retrier struct {
	config RetryConfig
	logger *utils.Logger
}

// NewRetrier creates a new Retrier
func NewRetrier(config RetryConfig, logger *utils.Logger) *Retrier {
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.InitialInterval <= 0 {
		config.InitialInterval = time.Second
	}
	if config.MaxInterval <= 0 {
		config.MaxInterval = 60 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.JitterFactor < 0 {
		config.JitterFactor = 0
	}

	return &Retrier{
		config: config,
		logger: logger,
	}
}

// Execute runs the operation with retry logic
func (r *Retrier) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 && r.logger != nil {
				r.logger.Info().Int("attempts", attempt+1).Msg("LLM request succeeded after retries")
			}
			return nil
		}

		if !IsRetryableError(lastErr) {
			return lastErr
		}

		if attempt >= r.config.MaxRetries {
			break
		}

		backoff := r.calculateBackoff(attempt)
		if r.logger != nil {
			r.logger.Warn().
				Int("attempt", attempt+1).
				Int("max_retries", r.config.MaxRetries).
				Dur("backoff", backoff).
				Err(lastErr).
				Msg("Retrying LLM request after error")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}

	return fmt.Errorf("%w: %v", domain.ErrLLMMaxRetriesExceeded, lastErr)
}

func (r *Retrier) calculateBackoff(attempt int) time.Duration {
	backoff := float64(r.config.InitialInterval) * math.Pow(r.config.Multiplier, float64(attempt))

	if r.config.JitterFactor > 0 {
		jitter := backoff * r.config.JitterFactor * (rand.Float64()*2 - 1)
		backoff += jitter
	}

	if backoff > float64(r.config.MaxInterval) {
		backoff = float64(r.config.MaxInterval)
	}

	if backoff < 0 {
		backoff = 0
	}

	return time.Duration(backoff)
}

// IsRetryableError checks if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// HTTP client timeouts (e.g. http.Client.Timeout) are retryable.
	// Must check BEFORE context.DeadlineExceeded since *url.Error wraps it.
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return true
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, domain.ErrLLMRateLimited) {
		return true
	}

	var llmErr *domain.LLMError
	if errors.As(err, &llmErr) {
		return ShouldRetryStatusCode(llmErr.StatusCode)
	}

	var fetchErr *domain.FetchError
	if errors.As(err, &fetchErr) {
		return ShouldRetryStatusCode(fetchErr.StatusCode)
	}

	return false
}

// ShouldRetryStatusCode checks if an HTTP status code is retryable
func ShouldRetryStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// ShouldRetry is kept for backward compatibility
func ShouldRetry(statusCode int) bool {
	return ShouldRetryStatusCode(statusCode)
}

// CalculateBackoff is kept for backward compatibility
func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	backoff := float64(cfg.InitialInterval) * math.Pow(cfg.Multiplier, float64(attempt))

	jitter := backoff * 0.1 * (rand.Float64()*2 - 1)
	backoff += jitter

	if backoff > float64(cfg.MaxInterval) {
		backoff = float64(cfg.MaxInterval)
	}

	return time.Duration(backoff)
}
