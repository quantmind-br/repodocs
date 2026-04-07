package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// CONF-02 verification: `grep "lmstudio" internal/tui/forms.go` confirms the TUI
// dropdown already includes LM Studio as a provider option (added in Phase 1).
// No code changes needed for CONF-02.

func TestNewLMStudioProvider(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL:     "http://localhost:1234/v1",
		Model:       "test-model",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	provider, err := NewLMStudioProvider(cfg, &http.Client{Timeout: 30 * time.Second})
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "lmstudio", provider.Name())
}

func TestLMStudioProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Empty(t, r.Header.Get("Authorization"))

		var reqBody map[string]interface{}
		err := decodeJSON(r.Body, &reqBody)
		require.NoError(t, err)
		assert.Equal(t, "lmstudio-model", reqBody["model"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "test-1",
			"model": "lmstudio-model",
			"choices": [{"message": {"role": "assistant", "content": "Test response"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "lmstudio-model",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, "lmstudio-model", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestLMStudioProvider_Complete_WithAPIKey(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "test-1",
			"model": "m",
			"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
		APIKey:  "my-secret-key",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer my-secret-key", receivedAuth)
}

func TestLMStudioProvider_Complete_WithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "test-1",
			"model": "m",
			"choices": [{"message": {"role": "assistant", "content": "ok"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
		}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
}

func TestLMStudioProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "internal server error", "type": "server_error"}}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Equal(t, http.StatusInternalServerError, llmErr.StatusCode)
}

func TestLMStudioProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "rate limit exceeded"}}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Equal(t, http.StatusTooManyRequests, llmErr.StatusCode)
	assert.ErrorIs(t, llmErr, domain.ErrLLMRateLimited)
}

func TestLMStudioProvider_Complete_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
		APIKey:  "bad-key",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Equal(t, http.StatusUnauthorized, llmErr.StatusCode)
	assert.ErrorIs(t, llmErr, domain.ErrLLMAuthFailed)
}

func TestLMStudioProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": "test", "model": "m", "choices": [], "usage": {}}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Contains(t, llmErr.Message, "no choices")
}

func TestLMStudioProvider_Complete_ConnectionRefused(t *testing.T) {
	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: "http://localhost:19999",
		Model:   "m",
	}, &http.Client{Timeout: 1 * time.Second})
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Contains(t, llmErr.Message, "request failed")
}

func TestLMStudioProvider_Complete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = provider.Complete(ctx, &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
}

func TestLMStudioProvider_Complete_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestLMStudioProvider_Complete_NoModelLoaded503(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`no model is currently loaded`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)

	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "lmstudio", llmErr.Provider)
	assert.Contains(t, llmErr.Message, "no model is loaded")
}

func TestLMStudioProvider_Close(t *testing.T) {
	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: "http://localhost:1234/v1",
		Model:   "m",
	}, &http.Client{})
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestLMStudioProvider_Complete_WithSystemMessage(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "test-1",
			"model": "m",
			"choices": [{"message": {"role": "assistant", "content": "I am helpful!"}, "finish_reason": "stop"}],
			"usage": {"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25}
		}`))
	}))
	defer server.Close()

	provider, err := NewLMStudioProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "m",
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
	assert.Equal(t, "user", messages[1].(map[string]interface{})["role"])
}
