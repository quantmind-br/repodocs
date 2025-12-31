package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
)

func TestOpenAIProvider_Name(t *testing.T) {
	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "gpt-4",
	}, http.DefaultClient)
	require.NoError(t, err)

	assert.Equal(t, "openai", provider.Name())
}

func TestOpenAIProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "Hello!"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestOpenAIProvider_Complete_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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
	assert.Equal(t, "openai", llmErr.Provider)
	assert.Equal(t, "Invalid API key", llmErr.Message)
}

func TestOpenAIProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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
}

func TestOpenAIProvider_Complete_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"model": "gpt-4",
			"choices": [],
			"usage": {"prompt_tokens": 10, "completion_tokens": 0, "total_tokens": 10}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
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
	assert.Contains(t, llmErr.Message, "no choices")
}

func TestOpenAIProvider_Complete_WithSystemMessage(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"model": "gpt-4",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "I am helpful!"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}
		}`))
	}))
	defer server.Close()

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleSystem, Content: "You are helpful"},
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	messages := receivedBody["messages"].([]interface{})
	assert.Len(t, messages, 2)
	assert.Equal(t, "system", messages[0].(map[string]interface{})["role"])
}

func TestOpenAIProvider_Close(t *testing.T) {
	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "gpt-4",
	}, http.DefaultClient)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}
