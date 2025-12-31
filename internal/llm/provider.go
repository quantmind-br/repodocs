package llm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
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

func NewProviderFromConfig(cfg *config.LLMConfig) (domain.LLMProvider, error) {
	if cfg.Provider == "" {
		return nil, domain.ErrLLMNotConfigured
	}
	if cfg.APIKey == "" {
		return nil, domain.ErrLLMMissingAPIKey
	}
	if cfg.BaseURL == "" {
		return nil, domain.ErrLLMMissingBaseURL
	}
	if cfg.Model == "" {
		return nil, domain.ErrLLMMissingModel
	}

	pcfg := ProviderConfig{
		Provider:    cfg.Provider,
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
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
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrLLMInvalidProvider, cfg.Provider)
	}
}
