package llm

import (
	"math"
	"math/rand"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

func ShouldRetry(statusCode int) bool {
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

func CalculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	backoff := float64(cfg.InitialInterval) * math.Pow(cfg.Multiplier, float64(attempt))

	jitter := backoff * 0.1 * (rand.Float64()*2 - 1)
	backoff += jitter

	if backoff > float64(cfg.MaxInterval) {
		backoff = float64(cfg.MaxInterval)
	}

	return time.Duration(backoff)
}
