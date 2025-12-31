package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// RateLimitedProviderConfig holds configuration for the wrapper
type RateLimitedProviderConfig struct {
	RequestsPerMinute        int
	BurstSize                int
	MaxRetries               int
	InitialDelay             time.Duration
	MaxDelay                 time.Duration
	Multiplier               float64
	JitterFactor             float64
	CircuitBreakerEnabled    bool
	FailureThreshold         int
	SuccessThresholdHalfOpen int
	ResetTimeout             time.Duration
}

// DefaultRateLimitedProviderConfig returns sensible defaults
func DefaultRateLimitedProviderConfig() RateLimitedProviderConfig {
	return RateLimitedProviderConfig{
		RequestsPerMinute:        60,
		BurstSize:                10,
		MaxRetries:               3,
		InitialDelay:             time.Second,
		MaxDelay:                 60 * time.Second,
		Multiplier:               2.0,
		JitterFactor:             0.1,
		CircuitBreakerEnabled:    true,
		FailureThreshold:         5,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             30 * time.Second,
	}
}

// RateLimitedProvider wraps an LLMProvider with rate limiting, retry, and circuit breaker
type RateLimitedProvider struct {
	provider       domain.LLMProvider
	rateLimiter    RateLimiter
	retrier        *Retrier
	circuitBreaker CircuitBreaker
	logger         *utils.Logger
}

// NewRateLimitedProvider creates a new rate-limited provider wrapper
func NewRateLimitedProvider(
	provider domain.LLMProvider,
	config RateLimitedProviderConfig,
	logger *utils.Logger,
) *RateLimitedProvider {
	var rateLimiter RateLimiter
	if config.RequestsPerMinute > 0 {
		rateLimiter = NewTokenBucket(config.RequestsPerMinute, config.BurstSize)
	} else {
		rateLimiter = &NoOpRateLimiter{}
	}

	retrier := NewRetrier(RetryConfig{
		MaxRetries:      config.MaxRetries,
		InitialInterval: config.InitialDelay,
		MaxInterval:     config.MaxDelay,
		Multiplier:      config.Multiplier,
		JitterFactor:    config.JitterFactor,
	}, logger)

	var circuitBreaker CircuitBreaker
	if config.CircuitBreakerEnabled {
		circuitBreaker = NewCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold:         config.FailureThreshold,
			SuccessThresholdHalfOpen: config.SuccessThresholdHalfOpen,
			ResetTimeout:             config.ResetTimeout,
		})
	} else {
		circuitBreaker = &NoOpCircuitBreaker{}
	}

	return &RateLimitedProvider{
		provider:       provider,
		rateLimiter:    rateLimiter,
		retrier:        retrier,
		circuitBreaker: circuitBreaker,
		logger:         logger,
	}
}

// Name returns the wrapped provider's name
func (p *RateLimitedProvider) Name() string {
	return p.provider.Name()
}

// Complete executes the request with rate limiting, retry, and circuit breaker
func (p *RateLimitedProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	if p.logger != nil {
		p.logger.Debug().
			Float64("tokens_available", p.rateLimiter.Available()).
			Msg("Waiting for rate limit token")
	}

	if err := p.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	if !p.circuitBreaker.Allow() {
		if p.logger != nil {
			p.logger.Warn().
				Str("state", p.circuitBreaker.State().String()).
				Msg("Circuit breaker is open, rejecting request")
		}
		return nil, domain.ErrLLMCircuitOpen
	}

	var response *domain.LLMResponse
	err := p.retrier.Execute(ctx, func() error {
		var err error
		response, err = p.provider.Complete(ctx, req)
		return err
	})

	if err != nil {
		p.circuitBreaker.RecordFailure()
		if p.logger != nil {
			p.logger.Error().
				Err(err).
				Str("circuit_state", p.circuitBreaker.State().String()).
				Msg("LLM request failed")
		}
		return nil, err
	}

	p.circuitBreaker.RecordSuccess()
	return response, nil
}

// Close closes the wrapped provider
func (p *RateLimitedProvider) Close() error {
	return p.provider.Close()
}
