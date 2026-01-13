package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/config"
)

func TestFromConfig(t *testing.T) {
	cfg := &config.Config{
		Output: config.OutputConfig{
			Directory:    "./output",
			Flat:         true,
			Overwrite:    true,
			JSONMetadata: true,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  10,
			Timeout:  60 * time.Second,
			MaxDepth: 5,
		},
		Cache: config.CacheConfig{
			Enabled:   true,
			TTL:       48 * time.Hour,
			Directory: "/tmp/cache",
		},
		Rendering: config.RenderingConfig{
			ForceJS:     true,
			JSTimeout:   30 * time.Second,
			ScrollToEnd: true,
		},
		Stealth: config.StealthConfig{
			UserAgent:      "TestAgent",
			RandomDelayMin: 100 * time.Millisecond,
			RandomDelayMax: 500 * time.Millisecond,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		LLM: config.LLMConfig{
			Provider:        "openai",
			APIKey:          "sk-test",
			BaseURL:         "https://api.openai.com/v1",
			Model:           "gpt-4",
			MaxTokens:       2000,
			Temperature:     0.5,
			Timeout:         45 * time.Second,
			EnhanceMetadata: true,
		},
		Exclude: []string{"*.pdf", "*.zip"},
	}

	values := FromConfig(cfg)

	assert.Equal(t, "./output", values.OutputDirectory)
	assert.True(t, values.OutputFlat)
	assert.True(t, values.OutputOverwrite)
	assert.True(t, values.JSONMetadata)

	assert.Equal(t, "10", values.Workers)
	assert.Equal(t, "1m0s", values.Timeout)
	assert.Equal(t, "5", values.MaxDepth)

	assert.True(t, values.CacheEnabled)
	assert.Equal(t, "48h0m0s", values.CacheTTL)
	assert.Equal(t, "/tmp/cache", values.CacheDirectory)

	assert.True(t, values.ForceJS)
	assert.Equal(t, "30s", values.JSTimeout)
	assert.True(t, values.ScrollToEnd)

	assert.Equal(t, "TestAgent", values.UserAgent)
	assert.Equal(t, "100ms", values.RandomDelayMin)
	assert.Equal(t, "500ms", values.RandomDelayMax)

	assert.Equal(t, "debug", values.LogLevel)
	assert.Equal(t, "json", values.LogFormat)

	assert.Equal(t, "openai", values.LLMProvider)
	assert.Equal(t, "sk-test", values.LLMAPIKey)
	assert.Equal(t, "https://api.openai.com/v1", values.LLMBaseURL)
	assert.Equal(t, "gpt-4", values.LLMModel)
	assert.Equal(t, "2000", values.LLMMaxTokens)
	assert.Equal(t, "0.50", values.LLMTemperature)
	assert.Equal(t, "45s", values.LLMTimeout)
	assert.True(t, values.LLMEnhanceMetadata)

	assert.Equal(t, []string{"*.pdf", "*.zip"}, values.Exclude)
}

func TestToConfig(t *testing.T) {
	values := &ConfigValues{
		OutputDirectory: "./docs",
		OutputFlat:      false,
		OutputOverwrite: false,
		JSONMetadata:    true,

		Workers:  "5",
		Timeout:  "30s",
		MaxDepth: "3",

		CacheEnabled:   true,
		CacheTTL:       "24h",
		CacheDirectory: "~/.repodocs/cache",

		ForceJS:     false,
		JSTimeout:   "10s",
		ScrollToEnd: true,

		UserAgent:      "MyAgent",
		RandomDelayMin: "100ms",
		RandomDelayMax: "500ms",

		LogLevel:  "info",
		LogFormat: "pretty",

		LLMProvider:        "anthropic",
		LLMAPIKey:          "sk-ant-test",
		LLMBaseURL:         "",
		LLMModel:           "claude-3-opus",
		LLMMaxTokens:       "4096",
		LLMTemperature:     "0.7",
		LLMTimeout:         "60s",
		LLMEnhanceMetadata: true,

		Exclude: []string{"*.log"},
	}

	cfg, err := values.ToConfig()
	require.NoError(t, err)

	assert.Equal(t, "./docs", cfg.Output.Directory)
	assert.False(t, cfg.Output.Flat)
	assert.False(t, cfg.Output.Overwrite)
	assert.True(t, cfg.Output.JSONMetadata)

	assert.Equal(t, 5, cfg.Concurrency.Workers)
	assert.Equal(t, 30*time.Second, cfg.Concurrency.Timeout)
	assert.Equal(t, 3, cfg.Concurrency.MaxDepth)

	assert.True(t, cfg.Cache.Enabled)
	assert.Equal(t, 24*time.Hour, cfg.Cache.TTL)
	assert.Equal(t, "~/.repodocs/cache", cfg.Cache.Directory)

	assert.False(t, cfg.Rendering.ForceJS)
	assert.Equal(t, 10*time.Second, cfg.Rendering.JSTimeout)
	assert.True(t, cfg.Rendering.ScrollToEnd)

	assert.Equal(t, "MyAgent", cfg.Stealth.UserAgent)
	assert.Equal(t, 100*time.Millisecond, cfg.Stealth.RandomDelayMin)
	assert.Equal(t, 500*time.Millisecond, cfg.Stealth.RandomDelayMax)

	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "pretty", cfg.Logging.Format)

	assert.Equal(t, "anthropic", cfg.LLM.Provider)
	assert.Equal(t, "sk-ant-test", cfg.LLM.APIKey)
	assert.Equal(t, "claude-3-opus", cfg.LLM.Model)
	assert.Equal(t, 4096, cfg.LLM.MaxTokens)
	assert.Equal(t, 0.7, cfg.LLM.Temperature)
	assert.Equal(t, 60*time.Second, cfg.LLM.Timeout)
	assert.True(t, cfg.LLM.EnhanceMetadata)

	assert.Equal(t, []string{"*.log"}, cfg.Exclude)
}

func TestToConfig_InvalidDuration(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
	}{
		{name: "invalid_timeout", field: "timeout", value: "invalid"},
		{name: "invalid_cache_ttl", field: "cache_ttl", value: "notvalid"},
		{name: "invalid_js_timeout", field: "js_timeout", value: "xyz"},
		{name: "invalid_delay_min", field: "delay_min", value: "abc"},
		{name: "invalid_delay_max", field: "delay_max", value: "def"},
		{name: "invalid_llm_timeout", field: "llm_timeout", value: "ghi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := &ConfigValues{
				Timeout:        "30s",
				CacheTTL:       "24h",
				JSTimeout:      "10s",
				RandomDelayMin: "100ms",
				RandomDelayMax: "500ms",
				LLMTimeout:     "30s",
			}

			switch tt.field {
			case "timeout":
				values.Timeout = tt.value
			case "cache_ttl":
				values.CacheTTL = tt.value
			case "js_timeout":
				values.JSTimeout = tt.value
			case "delay_min":
				values.RandomDelayMin = tt.value
			case "delay_max":
				values.RandomDelayMax = tt.value
			case "llm_timeout":
				values.LLMTimeout = tt.value
			}

			_, err := values.ToConfig()
			require.Error(t, err)
		})
	}
}

func TestToConfig_DefaultDurations(t *testing.T) {
	values := &ConfigValues{
		Timeout:        "",
		CacheTTL:       "",
		JSTimeout:      "",
		RandomDelayMin: "",
		RandomDelayMax: "",
		LLMTimeout:     "",
	}

	cfg, err := values.ToConfig()
	require.NoError(t, err)

	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
	assert.Equal(t, config.DefaultRandomDelayMin, cfg.Stealth.RandomDelayMin)
	assert.Equal(t, config.DefaultRandomDelayMax, cfg.Stealth.RandomDelayMax)
}
