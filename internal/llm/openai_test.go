package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewOpenAIProvider tests creating an OpenAI provider
func TestNewOpenAIProvider(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1/",
		Model:       "gpt-4",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	provider, err := NewOpenAIProvider(cfg, &http.Client{Timeout: 30 * time.Second})
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "openai", provider.Name())
}

// TestOpenAIProvider_Complete_Success tests successful completion
func TestOpenAIProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Send response
		response := openAIResponse{
			ID:    "test-id",
			Model: "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: "Test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, server.Client())
	require.NoError(t, err)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Complete(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

// TestOpenAIProvider_Complete_APIError tests API error response
func TestOpenAIProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openAIResponse{
			Error: &struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			}{
				Message: "Invalid API key",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, server.Client())
	require.NoError(t, err)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Complete(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	assert.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "openai", llmErr.Provider)
	assert.Equal(t, http.StatusUnauthorized, llmErr.StatusCode)
	assert.Contains(t, llmErr.Message, "Invalid API key")
}

// TestOpenAIProvider_Complete_RateLimit tests rate limit error
func TestOpenAIProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		// Return valid JSON but without Error field, so it goes through handleHTTPError
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, server.Client())
	require.NoError(t, err)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Complete(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	assert.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "openai", llmErr.Provider)
	assert.Equal(t, http.StatusTooManyRequests, llmErr.StatusCode)
	assert.ErrorIs(t, err, domain.ErrLLMRateLimited)
}

// TestOpenAIProvider_Complete_EmptyChoices tests empty choices response
func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := openAIResponse{
			ID:    "test-id",
			Model: "gpt-4",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, server.Client())
	require.NoError(t, err)

	ctx := context.Background()
	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	}

	resp, err := provider.Complete(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "no choices")
}

// TestOpenAIProvider_Close tests closing the provider
func TestOpenAIProvider_Close(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, &http.Client{})
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

// TestOpenAIProvider_Complete_WithContextCancellation tests context cancellation
func TestOpenAIProvider_Complete_WithContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIResponse{})
	}))
	defer server.Close()

	cfg := ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}
	provider, err := NewOpenAIProvider(cfg, server.Client())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	}

	_, err = provider.Complete(ctx, req)
	assert.Error(t, err)
}
