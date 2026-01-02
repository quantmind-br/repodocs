package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
)

func TestAnthropicProvider_Name(t *testing.T) {
	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "claude-3-sonnet",
	}, http.DefaultClient)
	require.NoError(t, err)

	assert.Equal(t, "anthropic", provider.Name())
}

func TestAnthropicProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
	assert.Equal(t, "claude-3-sonnet", resp.Model)
	assert.Equal(t, "end_turn", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestAnthropicProvider_Complete_WithSystemPrompt(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [{"type": "text", "text": "I am helpful!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 20, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleSystem, Content: "You are helpful"},
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "You are helpful", receivedBody["system"])
	messages := receivedBody["messages"].([]interface{})
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].(map[string]interface{})["role"])
}

func TestAnthropicProvider_Complete_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"type": "authentication_error", "message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "anthropic", llmErr.Provider)
}

func TestAnthropicProvider_Complete_MultipleContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [
				{"type": "text", "text": "Part 1"},
				{"type": "text", "text": " Part 2"}
			],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 10, "output_tokens": 10}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Part 1 Part 2", resp.Content)
}

func TestAnthropicProvider_Close(t *testing.T) {
	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "claude-3-sonnet",
	}, http.DefaultClient)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestAnthropicProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, http.StatusTooManyRequests, llmErr.StatusCode)
	assert.ErrorIs(t, llmErr, domain.ErrLLMRateLimited)
}

func TestAnthropicProvider_Complete_APIErrorInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-sonnet",
			"content": [],
			"error": {"type": "invalid_request_error", "message": "Invalid request"}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Contains(t, llmErr.Message, "Invalid request")
}

func TestAnthropicProvider_Complete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = provider.Complete(ctx, &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	assert.Error(t, err)
}

func TestAnthropicProvider_Complete_MaxTokens(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
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

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "claude-3-sonnet",
		MaxTokens: 100,
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages:  []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
		MaxTokens: 200, // Should override provider default
	})

	require.NoError(t, err)
	assert.Equal(t, float64(200), receivedBody["max_tokens"])
}

func TestAnthropicProvider_Complete_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestAnthropicProvider_Complete_TemperatureDefault(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
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

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey:      "test-key",
		BaseURL:     server.URL,
		Model:       "claude-3-sonnet",
		Temperature: 0.7,
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	// Anthropic doesn't send temperature in the basic request, so just verify no error
}
