package llm

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultRateLimitedProviderConfig tests default config
func TestDefaultRateLimitedProviderConfig(t *testing.T) {
	cfg := DefaultRateLimitedProviderConfig()

	assert.Equal(t, 60, cfg.RequestsPerMinute)
	assert.Equal(t, 10, cfg.BurstSize)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, time.Second, cfg.InitialDelay)
	assert.Equal(t, 60*time.Second, cfg.MaxDelay)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Equal(t, 0.1, cfg.JitterFactor)
	assert.True(t, cfg.CircuitBreakerEnabled)
	assert.Equal(t, 5, cfg.FailureThreshold)
	assert.Equal(t, 1, cfg.SuccessThresholdHalfOpen)
	assert.Equal(t, 30*time.Second, cfg.ResetTimeout)
}

// TestNewRateLimitedProvider tests creating a rate-limited provider
func TestNewRateLimitedProvider(t *testing.T) {
	tests := []struct {
		name  string
		cfg   RateLimitedProviderConfig
		valid bool
	}{
		{
			name: "full config",
			cfg: RateLimitedProviderConfig{
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
			},
			valid: true,
		},
		{
			name: "disabled circuit breaker",
			cfg: RateLimitedProviderConfig{
				RequestsPerMinute:     60,
				BurstSize:             10,
				CircuitBreakerEnabled: false,
			},
			valid: true,
		},
		{
			name: "no rate limiting",
			cfg: RateLimitedProviderConfig{
				RequestsPerMinute: 0,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockLLMProvider{name: "test"}
			p := NewRateLimitedProvider(mockProvider, tt.cfg, nil)

			require.NotNil(t, p)
			assert.Equal(t, "test", p.Name())
		})
	}
}

// TestRateLimitedProvider_Name tests getting the provider name
func TestRateLimitedProvider_Name(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test-provider"}
	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{}, nil)

	assert.Equal(t, "test-provider", p.Name())
}

// TestRateLimitedProvider_Complete_Success tests successful completion
func TestRateLimitedProvider_Complete_Success(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name:      "test",
		response:  &domain.LLMResponse{Content: "test response"},
		err:       nil,
	}

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
	}, logger)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	resp, err := p.Complete(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test response", resp.Content)
}

// TestRateLimitedProvider_Complete_RateLimit tests rate limiting
func TestRateLimitedProvider_Complete_RateLimit(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name:     "test",
		response: &domain.LLMResponse{Content: "test"},
		err:      nil,
	}

	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		RequestsPerMinute: 6, // 1 per 100ms
		BurstSize:         2,  // Allow 2 immediate
	}, nil)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	// First 2 should succeed immediately (burst)
	start := time.Now()
	_, err := p.Complete(ctx, req)
	assert.NoError(t, err)
	_, err = p.Complete(ctx, req)
	assert.NoError(t, err)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 50*time.Millisecond, "Burst requests should be immediate")

	// Third request should be rate limited
	start = time.Now()
	_, err = p.Complete(ctx, req)
	assert.NoError(t, err)
	elapsed = time.Since(start)
	assert.Greater(t, elapsed, 100*time.Millisecond, "Third request should wait for refill")
}

// TestRateLimitedProvider_Complete_CircuitBreaker tests circuit breaker
func TestRateLimitedProvider_Complete_CircuitBreaker(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name: "test",
		err:  &domain.LLMError{StatusCode: 500},
	}

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		CircuitBreakerEnabled: true,
		FailureThreshold:      2,
	}, logger)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	// First two failures should trip the breaker
	_, err := p.Complete(ctx, req)
	assert.Error(t, err)

	_, err = p.Complete(ctx, req)
	assert.Error(t, err)

	// Third request should fail immediately due to open circuit
	_, err = p.Complete(ctx, req)
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMCircuitOpen)
}

// TestRateLimitedProvider_Complete_Retry tests retry logic
func TestRateLimitedProvider_Complete_Retry(t *testing.T) {
	calls := 0
	mockProvider := &mockLLMProvider{
		name: "test",
		fn: func() (*domain.LLMResponse, error) {
			calls++
			if calls < 3 {
				return nil, &domain.LLMError{StatusCode: 429} // Rate limited
			}
			return &domain.LLMResponse{Content: "success"}, nil
		},
	}

	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
	}, nil)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	resp, err := p.Complete(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Content)
	assert.Equal(t, 3, calls)
}

// TestRateLimitedProvider_Complete_ContextCancellation tests context cancellation
func TestRateLimitedProvider_Complete_ContextCancellation(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name: "test",
	}

	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		RequestsPerMinute: 0, // No rate limiting to avoid blocking
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	_, err := p.Complete(ctx, req)
	assert.Error(t, err)
}

// TestRateLimitedProvider_Close tests closing the provider
func TestRateLimitedProvider_Close(t *testing.T) {
	mockProvider := &mockLLMProvider{name: "test"}
	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{}, nil)

	err := p.Close()
	assert.NoError(t, err)
}

// TestRateLimitedProvider_ConcurrentRequests tests concurrent request safety
func TestRateLimitedProvider_ConcurrentRequests(t *testing.T) {
	mockProvider := &mockLLMProvider{
		name:     "test",
		response: &domain.LLMResponse{Content: "test"},
		err:      nil,
	}

	p := NewRateLimitedProvider(mockProvider, RateLimitedProviderConfig{
		RequestsPerMinute: 600, // 10 per second
		BurstSize:         50,
	}, nil)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: "user", Content: "test"}},
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := p.Complete(ctx, req)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// All should complete without deadlock
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Mock LLM provider for testing
type mockLLMProvider struct {
	name     string
	response *domain.LLMResponse
	err      error
	fn       func() (*domain.LLMResponse, error)
	closed   bool
}

func (m *mockLLMProvider) Name() string {
	return m.name
}

func (m *mockLLMProvider) Complete(_ context.Context, _ *domain.LLMRequest) (*domain.LLMResponse, error) {
	if m.fn != nil {
		return m.fn()
	}
	return m.response, m.err
}

func (m *mockLLMProvider) Close() error {
	m.closed = true
	return nil
}
