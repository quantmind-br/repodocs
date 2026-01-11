package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

func TestGoogleProvider_Name(t *testing.T) {
	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "gemini-pro",
	}, http.DefaultClient)
	require.NoError(t, err)

	assert.Equal(t, "google", provider.Name())
}

func TestGoogleProvider_Complete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1beta/models/gemini-pro:generateContent")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("x-goog-api-key"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [{"text": "Hello!"}]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, server.Client())
	require.NoError(t, err)

	resp, err := provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
	assert.Equal(t, "gemini-pro", resp.Model)
	assert.Equal(t, "STOP", resp.FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
	assert.Equal(t, 15, resp.Usage.TotalTokens)
}

func TestGoogleProvider_Complete_WithSystemInstruction(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [{"text": "I am helpful!"}]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 20,
				"candidatesTokenCount": 5,
				"totalTokenCount": 25
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleSystem, Content: "You are helpful"},
			{Role: domain.RoleUser, Content: "Hi"},
		},
	})

	require.NoError(t, err)
	assert.NotNil(t, receivedBody["systemInstruction"])
	contents := receivedBody["contents"].([]interface{})
	assert.Len(t, contents, 1)
	assert.Equal(t, "user", contents[0].(map[string]interface{})["role"])
}

func TestGoogleProvider_Complete_RoleConversion(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [{"text": "Response"}]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 30,
				"candidatesTokenCount": 5,
				"totalTokenCount": 35
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages: []domain.LLMMessage{
			{Role: domain.RoleUser, Content: "Hi"},
			{Role: domain.RoleAssistant, Content: "Hello"},
			{Role: domain.RoleUser, Content: "How are you?"},
		},
	})

	require.NoError(t, err)
	contents := receivedBody["contents"].([]interface{})
	assert.Len(t, contents, 3)
	assert.Equal(t, "user", contents[0].(map[string]interface{})["role"])
	assert.Equal(t, "model", contents[1].(map[string]interface{})["role"])
	assert.Equal(t, "user", contents[2].(map[string]interface{})["role"])
}

func TestGoogleProvider_Complete_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": 403, "message": "API key not valid", "status": "PERMISSION_DENIED"}}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
	assert.Equal(t, "google", llmErr.Provider)
}

func TestGoogleProvider_Complete_NoCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 0,
				"totalTokenCount": 10
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
	assert.Contains(t, llmErr.Message, "no candidates")
}

func TestGoogleProvider_Close(t *testing.T) {
	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost",
		Model:   "gemini-pro",
	}, http.DefaultClient)
	require.NoError(t, err)

	err = provider.Close()
	assert.NoError(t, err)
}

func TestGoogleProvider_Complete_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"code": 429, "message": "Rate limit exceeded", "status": "RESOURCE_EXHAUSTED"}}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGoogleProvider_Complete_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGoogleProvider_Complete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGoogleProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [],
			"error": {"code": 400, "message": "Invalid request", "status": "INVALID_ARGUMENT"}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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

func TestGoogleProvider_Complete_WithTemperature(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [{"text": "Response"}]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}`))
	}))
	defer server.Close()

	temp := 0.7
	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages:    []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
		Temperature: &temp,
	})

	require.NoError(t, err)
	genConfig := receivedBody["generationConfig"].(map[string]interface{})
	assert.Equal(t, 0.7, genConfig["temperature"])
}

func TestGoogleProvider_Complete_WithMaxTokens(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = decodeJSON(r.Body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [{"text": "Response"}]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 10,
				"candidatesTokenCount": 5,
				"totalTokenCount": 15
			}
		}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gemini-pro",
		MaxTokens: 100,
	}, server.Client())
	require.NoError(t, err)

	_, err = provider.Complete(context.Background(), &domain.LLMRequest{
		Messages:  []domain.LLMMessage{{Role: domain.RoleUser, Content: "Hi"}},
		MaxTokens: 200,
	})

	require.NoError(t, err)
	genConfig := receivedBody["generationConfig"].(map[string]interface{})
	assert.Equal(t, float64(200), genConfig["maxOutputTokens"])
}

// TestGoogleProvider_Complete_HTTPErrorWithoutJSONError tests handleHTTPError path
func TestGoogleProvider_Complete_HTTPErrorWithoutJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		// Return valid JSON but without error field, triggering handleHTTPError
		_, _ = w.Write([]byte(`{"candidates": []}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
	assert.Equal(t, "google", llmErr.Provider)
	assert.Equal(t, http.StatusUnauthorized, llmErr.StatusCode)
	assert.ErrorIs(t, llmErr.Err, domain.ErrLLMAuthFailed)
}

// TestGoogleProvider_Complete_RateLimitViaHandleHTTPError tests rate limit via handleHTTPError
func TestGoogleProvider_Complete_RateLimitViaHandleHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		// Return valid JSON but without error field
		_, _ = w.Write([]byte(`{"candidates": []}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
	assert.ErrorIs(t, llmErr.Err, domain.ErrLLMRateLimited)
}

// TestGoogleProvider_Complete_GenericHTTPError tests handleHTTPError with unknown status
func TestGoogleProvider_Complete_GenericHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		// Return valid JSON but without error field
		_, _ = w.Write([]byte(`{"candidates": []}`))
	}))
	defer server.Close()

	provider, err := NewGoogleProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemini-pro",
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
	assert.Equal(t, http.StatusInternalServerError, llmErr.StatusCode)
}

