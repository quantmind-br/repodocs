package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

func TestNewOllamaProvider(t *testing.T) {
	cfg := ProviderConfig{
		BaseURL:     "http://localhost:11434",
		Model:       "llama2",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	provider, err := NewOllamaProvider(cfg, &http.Client{Timeout: 30 * time.Second})
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "ollama", provider.Name())
}

func TestOllamaProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Empty(t, r.Header.Get("Authorization"))

		var reqBody map[string]interface{}
		err := decodeJSON(r.Body, &reqBody)
		require.NoError(t, err)
		assert.Equal(t, "llama2", reqBody["model"])
		assert.Equal(t, false, reqBody["stream"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"created_at": "2023-12-12T14:13:43.416799Z",
			"message": {"role": "assistant", "content": "Test response"},
			"done": true,
			"total_duration": 5191566416,
			"load_duration": 2154458,
			"prompt_eval_count": 26,
			"prompt_eval_duration": 130079000,
			"eval_count": 259,
			"eval_duration": 4232710000
		}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, "llama2", resp.Model)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 26, resp.Usage.PromptTokens)
	assert.Equal(t, 259, resp.Usage.CompletionTokens)
	assert.Equal(t, 285, resp.Usage.TotalTokens)
}

func TestOllamaProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "model 'nonexistent' not found"}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "nonexistent",
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
	assert.Equal(t, "ollama", llmErr.Provider)
	assert.Equal(t, http.StatusInternalServerError, llmErr.StatusCode)
	assert.Contains(t, llmErr.Message, "not found")
}

func TestOllamaProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limit exceeded"}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
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
	assert.Equal(t, "ollama", llmErr.Provider)
	assert.Equal(t, http.StatusTooManyRequests, llmErr.StatusCode)
	assert.ErrorIs(t, llmErr, domain.ErrLLMRateLimited)
}

func TestOllamaProvider_Complete_EmptyMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"created_at": "2023-12-12T14:13:43.416799Z",
			"message": {"role": "assistant", "content": ""},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 0
		}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "", resp.Content)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 0, resp.Usage.CompletionTokens)
}

func TestOllamaProvider_Close(t *testing.T) {
	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama2",
	}, &http.Client{})
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestOllamaProvider_Complete_WithContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
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

func TestOllamaProvider_Complete_WithSystemMessage(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"created_at": "2023-12-12T14:13:43.416799Z",
			"message": {"role": "assistant", "content": "I am helpful!"},
			"done": true,
			"prompt_eval_count": 20,
			"eval_count": 5
		}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
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

func TestOllamaProvider_Complete_StreamFalse(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"created_at": "2023-12-12T14:13:43.416799Z",
			"message": {"role": "assistant", "content": "Response"},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 5
		}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, false, receivedBody["stream"])
}

func TestOllamaProvider_Complete_Options(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "llama2",
			"created_at": "2023-12-12T14:13:43.416799Z",
			"message": {"role": "assistant", "content": "Response"},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 5
		}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL:     server.URL,
		Model:       "llama2",
		MaxTokens:   500,
		Temperature: 0.8,
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	options := receivedBody["options"].(map[string]interface{})
	assert.Equal(t, float64(0.8), options["temperature"])
	assert.Equal(t, float64(500), options["num_predict"])
}

func TestOllamaProvider_Complete_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
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

func TestOllamaProvider_Complete_ErrorInSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error": "unexpected error in response"}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "ollama", llmErr.Provider)
	assert.Contains(t, llmErr.Message, "unexpected error")
}

func TestOllamaProvider_Complete_HTTPErrorWithoutJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.Equal(t, "ollama", llmErr.Provider)
	assert.Equal(t, http.StatusInternalServerError, llmErr.StatusCode)
}

func TestOllamaProvider_Complete_RateLimitViaHandleHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.Error(t, err)
	var llmErr *domain.LLMError
	require.ErrorAs(t, err, &llmErr)
	assert.ErrorIs(t, llmErr.Err, domain.ErrLLMRateLimited)
}

func TestOllamaProvider_Complete_NotDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"model":             "llama2",
			"created_at":        "2023-12-12T14:13:43.416799Z",
			"message":           map[string]string{"role": "assistant", "content": "Partial response"},
			"done":              false,
			"prompt_eval_count": 10,
			"eval_count":        100,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOllamaProvider(ProviderConfig{
		BaseURL: server.URL,
		Model:   "llama2",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "length", resp.FinishReason)
	assert.Equal(t, "Partial response", resp.Content)
}
