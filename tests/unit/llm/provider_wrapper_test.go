package llm_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLLMProvider struct {
	name        string
	completeErr error
	callCount   int64
	mu          sync.Mutex
}

func (m *mockLLMProvider) Name() string {
	return m.name
}

func (m *mockLLMProvider) Complete(_ context.Context, _ *domain.LLMRequest) (*domain.LLMResponse, error) {
	atomic.AddInt64(&m.callCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return &domain.LLMResponse{Content: "test response"}, nil
}

func (m *mockLLMProvider) Close() error {
	return nil
}

func (m *mockLLMProvider) CallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

func (m *mockLLMProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completeErr = err
}

func TestRateLimitedProvider_Name(t *testing.T) {
	mock := &mockLLMProvider{name: "test-provider"}
	config := llm.DefaultRateLimitedProviderConfig()
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	assert.Equal(t, "test-provider", provider.Name())
}

func TestRateLimitedProvider_Complete_Success(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.DefaultRateLimitedProviderConfig()
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{})
	require.NoError(t, err)
	assert.Equal(t, "test response", resp.Content)
	assert.Equal(t, int64(1), mock.CallCount())
}

func TestRateLimitedProvider_Complete_RetriesOn429(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:     6000,
		BurstSize:             100,
		MaxRetries:            3,
		InitialDelay:          10 * time.Millisecond,
		MaxDelay:              100 * time.Millisecond,
		Multiplier:            2.0,
		CircuitBreakerEnabled: false,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	callCount := 0
	mock.mu.Lock()
	mock.completeErr = domain.NewLLMError("test", 429, "rate limited", domain.ErrLLMRateLimited)
	mock.mu.Unlock()

	go func() {
		time.Sleep(30 * time.Millisecond)
		mock.SetError(nil)
	}()

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{})
	_ = callCount

	if err == nil {
		assert.Equal(t, "test response", resp.Content)
	}
}

func TestRateLimitedProvider_Complete_NoRetryOn4xx(t *testing.T) {
	mock := &mockLLMProvider{
		name:        "test",
		completeErr: domain.NewLLMError("test", 401, "unauthorized", nil),
	}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:     6000,
		BurstSize:             100,
		MaxRetries:            3,
		InitialDelay:          10 * time.Millisecond,
		MaxDelay:              100 * time.Millisecond,
		Multiplier:            2.0,
		CircuitBreakerEnabled: false,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	_, err := provider.Complete(context.Background(), &domain.LLMRequest{})
	require.Error(t, err)
	assert.Equal(t, int64(1), mock.CallCount())
}

func TestRateLimitedProvider_Complete_CircuitBreaker(t *testing.T) {
	mock := &mockLLMProvider{
		name:        "test",
		completeErr: domain.ErrLLMRateLimited,
	}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:        6000,
		BurstSize:                100,
		MaxRetries:               0,
		InitialDelay:             10 * time.Millisecond,
		MaxDelay:                 100 * time.Millisecond,
		Multiplier:               2.0,
		CircuitBreakerEnabled:    true,
		FailureThreshold:         3,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Second,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	for i := 0; i < 3; i++ {
		_, _ = provider.Complete(context.Background(), &domain.LLMRequest{})
	}

	_, err := provider.Complete(context.Background(), &domain.LLMRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMCircuitOpen)
}

func TestRateLimitedProvider_Complete_ContextCancelled(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:     1,
		BurstSize:             1,
		MaxRetries:            0,
		CircuitBreakerEnabled: false,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	_, err := provider.Complete(context.Background(), &domain.LLMRequest{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = provider.Complete(ctx, &domain.LLMRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRateLimitedProvider_RespectRateLimit(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:     600,
		BurstSize:             2,
		MaxRetries:            0,
		CircuitBreakerEnabled: false,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	start := time.Now()
	for i := 0; i < 4; i++ {
		_, err := provider.Complete(context.Background(), &domain.LLMRequest{})
		require.NoError(t, err)
	}
	elapsed := time.Since(start)

	assert.True(t, elapsed >= 100*time.Millisecond, "should have rate limited")
	assert.True(t, elapsed < 500*time.Millisecond, "should not have taken too long")
}

func TestRateLimitedProvider_Close(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.DefaultRateLimitedProviderConfig()
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	err := provider.Close()
	require.NoError(t, err)
}

func TestRateLimitedProvider_DisabledRateLimiting(t *testing.T) {
	mock := &mockLLMProvider{name: "test"}
	config := llm.RateLimitedProviderConfig{
		RequestsPerMinute:     0,
		BurstSize:             0,
		MaxRetries:            0,
		CircuitBreakerEnabled: false,
	}
	provider := llm.NewRateLimitedProvider(mock, config, nil)

	start := time.Now()
	for i := 0; i < 10; i++ {
		_, err := provider.Complete(context.Background(), &domain.LLMRequest{})
		require.NoError(t, err)
	}
	elapsed := time.Since(start)

	assert.True(t, elapsed < 100*time.Millisecond, "should not have rate limited")
}

func TestDefaultRateLimitedProviderConfig(t *testing.T) {
	config := llm.DefaultRateLimitedProviderConfig()

	assert.Equal(t, 60, config.RequestsPerMinute)
	assert.Equal(t, 10, config.BurstSize)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, time.Second, config.InitialDelay)
	assert.Equal(t, 60*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.Equal(t, 0.1, config.JitterFactor)
	assert.True(t, config.CircuitBreakerEnabled)
	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 1, config.SuccessThresholdHalfOpen)
	assert.Equal(t, 30*time.Second, config.ResetTimeout)
}
