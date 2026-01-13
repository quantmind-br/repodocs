package tui

import (
	"fmt"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
)

// ConfigValues holds form values that map to Config struct.
// Duration fields are stored as strings for form editing.
type ConfigValues struct {
	OutputDirectory string
	OutputFlat      bool
	OutputOverwrite bool
	JSONMetadata    bool

	Workers  int
	Timeout  string
	MaxDepth int

	CacheEnabled   bool
	CacheTTL       string
	CacheDirectory string

	ForceJS     bool
	JSTimeout   string
	ScrollToEnd bool

	UserAgent      string
	RandomDelayMin string
	RandomDelayMax string

	LogLevel  string
	LogFormat string

	LLMProvider        string
	LLMAPIKey          string
	LLMBaseURL         string
	LLMModel           string
	LLMMaxTokens       int
	LLMTemperature     float64
	LLMTimeout         string
	LLMEnhanceMetadata bool

	Exclude []string
}

// FromConfig converts a Config to ConfigValues for form editing
func FromConfig(cfg *config.Config) *ConfigValues {
	return &ConfigValues{
		OutputDirectory: cfg.Output.Directory,
		OutputFlat:      cfg.Output.Flat,
		OutputOverwrite: cfg.Output.Overwrite,
		JSONMetadata:    cfg.Output.JSONMetadata,

		Workers:  cfg.Concurrency.Workers,
		Timeout:  formatDuration(cfg.Concurrency.Timeout),
		MaxDepth: cfg.Concurrency.MaxDepth,

		CacheEnabled:   cfg.Cache.Enabled,
		CacheTTL:       formatDuration(cfg.Cache.TTL),
		CacheDirectory: cfg.Cache.Directory,

		ForceJS:     cfg.Rendering.ForceJS,
		JSTimeout:   formatDuration(cfg.Rendering.JSTimeout),
		ScrollToEnd: cfg.Rendering.ScrollToEnd,

		UserAgent:      cfg.Stealth.UserAgent,
		RandomDelayMin: formatDuration(cfg.Stealth.RandomDelayMin),
		RandomDelayMax: formatDuration(cfg.Stealth.RandomDelayMax),

		LogLevel:  cfg.Logging.Level,
		LogFormat: cfg.Logging.Format,

		LLMProvider:        cfg.LLM.Provider,
		LLMAPIKey:          cfg.LLM.APIKey,
		LLMBaseURL:         cfg.LLM.BaseURL,
		LLMModel:           cfg.LLM.Model,
		LLMMaxTokens:       cfg.LLM.MaxTokens,
		LLMTemperature:     cfg.LLM.Temperature,
		LLMTimeout:         formatDuration(cfg.LLM.Timeout),
		LLMEnhanceMetadata: cfg.LLM.EnhanceMetadata,

		Exclude: cfg.Exclude,
	}
}

// ToConfig converts ConfigValues back to a Config struct
func (v *ConfigValues) ToConfig() (*config.Config, error) {
	timeout, err := parseDurationOrDefault(v.Timeout, config.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	cacheTTL, err := parseDurationOrDefault(v.CacheTTL, config.DefaultCacheTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache_ttl: %w", err)
	}

	jsTimeout, err := parseDurationOrDefault(v.JSTimeout, config.DefaultJSTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid js_timeout: %w", err)
	}

	delayMin, err := parseDurationOrDefault(v.RandomDelayMin, config.DefaultRandomDelayMin)
	if err != nil {
		return nil, fmt.Errorf("invalid random_delay_min: %w", err)
	}

	delayMax, err := parseDurationOrDefault(v.RandomDelayMax, config.DefaultRandomDelayMax)
	if err != nil {
		return nil, fmt.Errorf("invalid random_delay_max: %w", err)
	}

	llmTimeout, err := parseDurationOrDefault(v.LLMTimeout, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid llm_timeout: %w", err)
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Directory:    v.OutputDirectory,
			Flat:         v.OutputFlat,
			Overwrite:    v.OutputOverwrite,
			JSONMetadata: v.JSONMetadata,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  v.Workers,
			Timeout:  timeout,
			MaxDepth: v.MaxDepth,
		},
		Cache: config.CacheConfig{
			Enabled:   v.CacheEnabled,
			TTL:       cacheTTL,
			Directory: v.CacheDirectory,
		},
		Rendering: config.RenderingConfig{
			ForceJS:     v.ForceJS,
			JSTimeout:   jsTimeout,
			ScrollToEnd: v.ScrollToEnd,
		},
		Stealth: config.StealthConfig{
			UserAgent:      v.UserAgent,
			RandomDelayMin: delayMin,
			RandomDelayMax: delayMax,
		},
		Logging: config.LoggingConfig{
			Level:  v.LogLevel,
			Format: v.LogFormat,
		},
		LLM: config.LLMConfig{
			Provider:        v.LLMProvider,
			APIKey:          v.LLMAPIKey,
			BaseURL:         v.LLMBaseURL,
			Model:           v.LLMModel,
			MaxTokens:       v.LLMMaxTokens,
			Temperature:     v.LLMTemperature,
			Timeout:         llmTimeout,
			EnhanceMetadata: v.LLMEnhanceMetadata,
		},
		Exclude: v.Exclude,
	}

	return cfg, nil
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func parseDurationOrDefault(s string, defaultVal time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultVal, nil
	}
	return time.ParseDuration(s)
}
