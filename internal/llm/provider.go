package llm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Default base URLs for each provider
const (
	DefaultOpenAIBaseURL    = "https://api.openai.com/v1"
	DefaultAnthropicBaseURL = "https://api.anthropic.com/v1"
	DefaultGoogleBaseURL    = "https://generativelanguage.googleapis.com"
	DefaultOllamaBaseURL    = "http://localhost:11434"
)

type ProviderConfig struct {
	Provider    string
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	MaxRetries  int
	HTTPClient  *http.Client
}

// DefaultBaseURL returns the default base URL for a given provider.
// Returns empty string if provider is unknown.
func DefaultBaseURL(provider string) string {
	switch provider {
	case "openai":
		return DefaultOpenAIBaseURL
	case "anthropic":
		return DefaultAnthropicBaseURL
	case "google":
		return DefaultGoogleBaseURL
	case "ollama":
		return DefaultOllamaBaseURL
	default:
		return ""
	}
}

func NewProviderFromConfig(cfg *config.LLMConfig) (domain.LLMProvider, error) {
	if cfg.Provider == "" {
		return nil, domain.ErrLLMNotConfigured
	}
	if cfg.APIKey == "" && cfg.Provider != "ollama" {
		return nil, domain.ErrLLMMissingAPIKey
	}
	if cfg.Model == "" {
		return nil, domain.ErrLLMMissingModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL(cfg.Provider)
		if baseURL == "" {
			return nil, domain.ErrLLMMissingBaseURL
		}
	}

	pcfg := ProviderConfig{
		Provider:    cfg.Provider,
		APIKey:      cfg.APIKey,
		BaseURL:     baseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		MaxRetries:  cfg.MaxRetries,
	}

	return NewProvider(pcfg)
}

func NewProvider(cfg ProviderConfig) (domain.LLMProvider, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(cfg, httpClient)
	case "anthropic":
		return NewAnthropicProvider(cfg, httpClient)
	case "google":
		return NewGoogleProvider(cfg, httpClient)
	case "ollama":
		return NewOllamaProvider(cfg, httpClient)
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrLLMInvalidProvider, cfg.Provider)
	}
}
