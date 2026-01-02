package llm_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/llm"
)

func TestNewProviderFromConfig_NotConfigured(t *testing.T) {
	cfg := &config.LLMConfig{}

	_, err := llm.NewProviderFromConfig(cfg)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMNotConfigured)
}

func TestNewProviderFromConfig_MissingAPIKey(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
	}

	_, err := llm.NewProviderFromConfig(cfg)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMMissingAPIKey)
}

func TestNewProviderFromConfig_MissingBaseURL(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		APIKey:   "test-key",
	}

	_, err := llm.NewProviderFromConfig(cfg)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMMissingBaseURL)
}

func TestNewProviderFromConfig_MissingModel(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
	}

	_, err := llm.NewProviderFromConfig(cfg)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMMissingModel)
}

func TestNewProviderFromConfig_InvalidProvider(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "invalid",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "test-model",
	}

	_, err := llm.NewProviderFromConfig(cfg)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrLLMInvalidProvider)
}

func TestNewProviderFromConfig_OpenAI(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "gpt-4",
	}

	provider, err := llm.NewProviderFromConfig(cfg)

	require.NoError(t, err)
	assert.Equal(t, "openai", provider.Name())
}

func TestNewProviderFromConfig_Anthropic(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "claude-3-sonnet",
	}

	provider, err := llm.NewProviderFromConfig(cfg)

	require.NoError(t, err)
	assert.Equal(t, "anthropic", provider.Name())
}

func TestNewProviderFromConfig_Google(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "google",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "gemini-pro",
	}

	provider, err := llm.NewProviderFromConfig(cfg)

	require.NoError(t, err)
	assert.Equal(t, "google", provider.Name())
}

func TestNewProvider_WithCustomHTTPClient(t *testing.T) {
	cfg := llm.ProviderConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "gpt-4",
	}

	provider, err := llm.NewProvider(cfg)

	require.NoError(t, err)
	assert.Equal(t, "openai", provider.Name())
}

func TestNewProvider_BaseURLTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		provider string
	}{
		{
			name:     "openai_with_trailing_slash",
			baseURL:  "http://localhost/",
			provider: "openai",
		},
		{
			name:     "anthropic_with_trailing_slash",
			baseURL:  "http://localhost/",
			provider: "anthropic",
		},
		{
			name:     "google_with_trailing_slash",
			baseURL:  "http://localhost/",
			provider: "google",
		},
		{
			name:     "openai_without_trailing_slash",
			baseURL:  "http://localhost",
			provider: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Provider: tt.provider,
				APIKey:   "test-key",
				BaseURL:  tt.baseURL,
				Model:    "test-model",
			})

			require.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestNewProvider_DefaultTimeout(t *testing.T) {
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "http://localhost",
		Model:    "gpt-4",
		Timeout:  0, // Should default to 60s
	})

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_CustomTimeout(t *testing.T) {
	customHTTPClient := &http.Client{}
	provider, err := llm.NewProvider(llm.ProviderConfig{
		Provider:   "openai",
		APIKey:     "test-key",
		BaseURL:    "http://localhost",
		Model:      "gpt-4",
		HTTPClient: customHTTPClient,
	})

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_AnthropicDefaults(t *testing.T) {
	tests := []struct {
		name       string
		maxTokens  int
		expectDefault bool
	}{
		{
			name:       "zero_max_tokens_uses_default",
			maxTokens:  0,
			expectDefault: true,
		},
		{
			name:       "custom_max_tokens",
			maxTokens:  8192,
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Provider:  "anthropic",
				APIKey:    "test-key",
				BaseURL:   "http://localhost",
				Model:     "claude-3-sonnet",
				MaxTokens: tt.maxTokens,
			})

			require.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestNewProvider_AllProviders(t *testing.T) {
	providers := []string{"openai", "anthropic", "google"}

	for _, providerName := range providers {
		t.Run(providerName, func(t *testing.T) {
			provider, err := llm.NewProvider(llm.ProviderConfig{
				Provider: providerName,
				APIKey:   "test-key",
				BaseURL:  "http://localhost",
				Model:    "test-model",
			})

			require.NoError(t, err)
			assert.Equal(t, providerName, provider.Name())
		})
	}
}

func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
