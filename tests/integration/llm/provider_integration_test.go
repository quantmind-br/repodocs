package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
)

// TestProviderFromConfig_Integration tests creating providers from configuration
func TestProviderFromConfig_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{
			name:     "openai_provider",
			provider: "openai",
			model:    "gpt-4",
		},
		{
			name:     "anthropic_provider",
			provider: "anthropic",
			model:    "claude-3-sonnet",
		},
		{
			name:     "google_provider",
			provider: "google",
			model:    "gemini-pro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.LLMConfig{
				Provider: tt.provider,
				APIKey:   "test-key",
				BaseURL:  "http://localhost:8080",
				Model:    tt.model,
			}

			provider, err := llm.NewProviderFromConfig(cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.provider, provider.Name())
			assert.NoError(t, provider.Close())
		})
	}
}

// TestCircuitBreakerWithProvider_Integration tests circuit breaker with real provider calls
func TestCircuitBreakerWithProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Fail first 3 requests, succeed on 4th
		if requestCount <= 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [{"type": "text", "text": "Success"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet",
	})
	require.NoError(t, err)
	defer provider.Close()

	// Create circuit breaker with low threshold
	cb := llm.NewCircuitBreaker(llm.CircuitBreakerConfig{
		FailureThreshold:         2,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             100 * time.Millisecond,
	})

	// First few requests should fail and open circuit
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Logf("Circuit opened after %d failures", i)
			break
		}
		_, err := provider.Complete(ctx, &domain.LLMRequest{
			Messages: []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
		})
		if err != nil {
			cb.RecordFailure()
		}
	}

	// Circuit should be open now
	assert.Equal(t, llm.StateOpen, cb.State())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should transition to half-open on next Allow
	if cb.Allow() {
		assert.Equal(t, llm.StateHalfOpen, cb.State())
	}
}

// TestRetryWithProvider_Integration tests retry logic with provider
func TestRetryWithProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		// Fail first 2 attempts with rate limit, succeed on 3rd
		if attemptCount <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [{"type": "text", "text": "Success after retries"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet",
	})
	require.NoError(t, err)
	defer provider.Close()

	config := llm.RetryConfig{
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2.0,
		JitterFactor:    0,
	}
	retrier := llm.NewRetrier(config, nil)

	var response *domain.LLMResponse
	err = retrier.Execute(context.Background(), func() error {
		var err error
		response, err = provider.Complete(context.Background(), &domain.LLMRequest{
			Messages: []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
		})
		return err
	})

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "Success after retries", response.Content)
	assert.Equal(t, 3, attemptCount) // Failed twice, succeeded on 3rd
}

// TestRateLimiterWithProvider_Integration tests rate limiting with provider
func TestRateLimiterWithProvider_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [{"type": "text", "text": "Response"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet",
	})
	require.NoError(t, err)
	defer provider.Close()

	// Create rate limiter: 600 requests per minute = 10 per second
	// Burst of 5 means first 5 requests go through immediately
	limiter := llm.NewTokenBucket(600, 5)
	ctx := context.Background()

	// First 5 requests should succeed immediately (burst)
	start := time.Now()
	for i := 0; i < 5; i++ {
		err := limiter.Wait(ctx)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 100*time.Millisecond, "Burst requests should be fast")

	// 6th request should wait for refill
	start = time.Now()
	err = limiter.Wait(ctx)
	require.NoError(t, err)
	elapsed = time.Since(start)
	assert.Greater(t, elapsed, 50*time.Millisecond, "Should wait for token refill")
}

// TestProviderErrorHandling_Integration tests error handling across provider types
func TestProviderErrorHandling_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name            string
		provider        string
		statusCode      int
		errorResponse   string
		expectRetryable bool
		expectedErrType error
	}{
		{
			name:            "anthropic_rate_limit",
			provider:        "anthropic",
			statusCode:      429,
			errorResponse:   `{"error": {"type": "rate_limit_error", "message": "Rate limit"}}`,
			expectRetryable: true,
			expectedErrType: domain.ErrLLMRateLimited,
		},
		{
			name:            "openai_rate_limit",
			provider:        "openai",
			statusCode:      429,
			errorResponse:   `{"error": {"message": "Rate limit"}}`,
			expectRetryable: true,
			expectedErrType: domain.ErrLLMRateLimited,
		},
		{
			name:            "google_rate_limit",
			provider:        "google",
			statusCode:      429,
			errorResponse:   `{"error": {"code": 429, "message": "Rate limit"}}`,
			expectRetryable: true,
			expectedErrType: domain.ErrLLMRateLimited,
		},
		{
			name:            "anthropic_auth_error",
			provider:        "anthropic",
			statusCode:      401,
			errorResponse:   `{"error": {"type": "authentication_error", "message": "Invalid key"}}`,
			expectRetryable: false,
			expectedErrType: domain.ErrLLMAuthFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.errorResponse))
			}))
			defer server.Close()

			provider, err := llm.NewProvider(llm.ProviderConfig{
				Provider: tt.provider,
				APIKey:   "test-key",
				BaseURL:  server.URL,
				Model:    "test-model",
			})
			require.NoError(t, err)
			defer provider.Close()

			_, err = provider.Complete(context.Background(), &domain.LLMRequest{
				Messages: []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
			})

			require.Error(t, err)

			// Check if error is retryable
			isRetryable := llm.IsRetryableError(err)
			assert.Equal(t, tt.expectRetryable, isRetryable)

			// Check error type
			var llmErr *domain.LLMError
			require.ErrorAs(t, err, &llmErr)
			if tt.expectedErrType != nil {
				assert.ErrorIs(t, llmErr, tt.expectedErrType)
			}
		})
	}
}

// TestProviderTimeout_Integration tests request timeout handling
func TestProviderTimeout_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add delay longer than client timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Model:    "claude-3-sonnet",
		Timeout:  50 * time.Millisecond, // Short timeout
	})
	require.NoError(t, err)
	defer provider.Close()

	ctx := context.Background()
	start := time.Now()
	_, err = provider.Complete(ctx, &domain.LLMRequest{
		Messages: []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
	})
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Less(t, elapsed, 150*time.Millisecond, "Should timeout quickly")
}
