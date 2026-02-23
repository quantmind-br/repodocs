package llm

import (
	"net/http"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProviderFromConfig tests creating provider from config
func TestNewProviderFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.LLMConfig
		wantErr error
	}{
		{
			name: "valid openai config",
			cfg: &config.LLMConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  "https://api.openai.com/v1",
				Model:    "gpt-4",
			},
			wantErr: nil,
		},
		{
			name: "valid anthropic config",
			cfg: &config.LLMConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				BaseURL:  "https://api.anthropic.com/v1",
				Model:    "claude-3",
			},
			wantErr: nil,
		},
		{
			name: "valid google config",
			cfg: &config.LLMConfig{
				Provider: "google",
				APIKey:   "test-key",
				BaseURL:  "https://generativelanguage.googleapis.com",
				Model:    "gemini-pro",
			},
			wantErr: nil,
		},
		{
			name: "missing provider",
			cfg: &config.LLMConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.example.com",
				Model:   "test-model",
			},
			wantErr: domain.ErrLLMNotConfigured,
		},
		{
			name: "missing api key",
			cfg: &config.LLMConfig{
				Provider: "openai",
				BaseURL:  "https://api.openai.com/v1",
				Model:    "gpt-4",
			},
			wantErr: domain.ErrLLMMissingAPIKey,
		},
		{
			name: "empty base url uses default for openai",
			cfg: &config.LLMConfig{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: nil,
		},
		{
			name: "empty base url uses default for anthropic",
			cfg: &config.LLMConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3",
			},
			wantErr: nil,
		},
		{
			name: "empty base url uses default for google",
			cfg: &config.LLMConfig{
				Provider: "google",
				APIKey:   "test-key",
				Model:    "gemini-pro",
			},
			wantErr: nil,
		},
		{
			name: "valid ollama config",
			cfg: &config.LLMConfig{
				Provider: "ollama",
				BaseURL:  "http://localhost:11434",
				Model:    "llama2",
			},
			wantErr: nil,
		},
		{
			name: "ollama without api key",
			cfg: &config.LLMConfig{
				Provider: "ollama",
				Model:    "llama2",
			},
			wantErr: nil,
		},
		{
			name: "missing base url for unknown provider",
			cfg: &config.LLMConfig{
				Provider: "unknown",
				APIKey:   "test-key",
				Model:    "model",
			},
			wantErr: domain.ErrLLMMissingBaseURL,
		},
		{
			name: "missing model",
			cfg: &config.LLMConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  "https://api.openai.com/v1",
			},
			wantErr: domain.ErrLLMMissingModel,
		},
		{
			name:    "empty config",
			cfg:     &config.LLMConfig{},
			wantErr: domain.ErrLLMNotConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProviderFromConfig(tt.cfg)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.cfg.Provider, provider.Name())
			}
		})
	}
}

// TestNewProvider tests creating provider directly
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ProviderConfig
		wantErr bool
	}{
		{
			name: "valid openai",
			cfg: ProviderConfig{
				Provider:    "openai",
				APIKey:      "test-key",
				BaseURL:     "https://api.openai.com/v1",
				Model:       "gpt-4",
				MaxTokens:   1000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
				MaxRetries:  3,
			},
			wantErr: false,
		},
		{
			name: "valid anthropic",
			cfg: ProviderConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				BaseURL:  "https://api.anthropic.com/v1",
				Model:    "claude-3",
			},
			wantErr: false,
		},
		{
			name: "valid google",
			cfg: ProviderConfig{
				Provider: "google",
				APIKey:   "test-key",
				BaseURL:  "https://generativelanguage.googleapis.com",
				Model:    "gemini-pro",
			},
			wantErr: false,
		},
		{
			name: "valid ollama",
			cfg: ProviderConfig{
				Provider: "ollama",
				BaseURL:  "http://localhost:11434",
				Model:    "llama2",
			},
			wantErr: false,
		},
		{
			name: "invalid provider",
			cfg: ProviderConfig{
				Provider: "invalid",
				APIKey:   "test-key",
				BaseURL:  "https://api.example.com",
				Model:    "test-model",
			},
			wantErr: true,
		},
		{
			name: "default timeout",
			cfg: ProviderConfig{
				Provider: "openai",
				APIKey:   "test-key",
				BaseURL:  "https://api.openai.com/v1",
				Model:    "gpt-4",
				Timeout:  0, // Should default to 60s
			},
			wantErr: false,
		},
		{
			name: "custom http client",
			cfg: ProviderConfig{
				Provider:   "openai",
				APIKey:     "test-key",
				BaseURL:    "https://api.openai.com/v1",
				Model:      "gpt-4",
				HTTPClient: &http.Client{Timeout: 10 * time.Second},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestProviderConfigDefaults(t *testing.T) {
	cfg := ProviderConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4",
	}

	provider, err := NewProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, "openai", provider.Name())
}

func TestDefaultBaseURL(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"openai", DefaultOpenAIBaseURL},
		{"anthropic", DefaultAnthropicBaseURL},
		{"google", DefaultGoogleBaseURL},
		{"ollama", DefaultOllamaBaseURL},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := DefaultBaseURL(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}
