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

func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}
