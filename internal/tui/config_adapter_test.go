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
			RateLimit: config.RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 120,
				BurstSize:         20,
				MaxRetries:        5,
				InitialDelay:      2 * time.Second,
				MaxDelay:          120 * time.Second,
				Multiplier:        2.5,
				CircuitBreaker: config.CircuitBreakerConfig{
					Enabled:                  true,
					FailureThreshold:         10,
					SuccessThresholdHalfOpen: 2,
					ResetTimeout:             60 * time.Second,
				},
			},
		},
		Exclude: []string{"*.pdf", "*.zip", ".*/admin.*"},
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

	// Exclude patterns (joined with \n)
	assert.Equal(t, "*.pdf\n*.zip\n.*/admin.*", values.ExcludePatterns)

	// Rate limit
	assert.True(t, values.RateLimitEnabled)
	assert.Equal(t, "120", values.RateLimitRequestsPerMinute)
	assert.Equal(t, "20", values.RateLimitBurstSize)
	assert.Equal(t, "5", values.RateLimitMaxRetries)
	assert.Equal(t, "2s", values.RateLimitInitialDelay)
	assert.Equal(t, "2m0s", values.RateLimitMaxDelay)
	assert.Equal(t, "2.50", values.RateLimitMultiplier)

	// Circuit breaker
	assert.True(t, values.CircuitBreakerEnabled)
	assert.Equal(t, "10", values.CircuitBreakerFailureThreshold)
	assert.Equal(t, "2", values.CircuitBreakerSuccessThreshold)
	assert.Equal(t, "1m0s", values.CircuitBreakerResetTimeout)

	assert.Equal(t, []string{"*.pdf", "*.zip", ".*/admin.*"}, values.Exclude)
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

		ExcludePatterns: "pattern1\npattern2\n  pattern3  \n\npattern4",

		RateLimitEnabled:           true,
		RateLimitRequestsPerMinute: "100",
		RateLimitBurstSize:         "15",
		RateLimitMaxRetries:        "4",
		RateLimitInitialDelay:      "500ms",
		RateLimitMaxDelay:          "30s",
		RateLimitMultiplier:        "1.5",

		CircuitBreakerEnabled:          true,
		CircuitBreakerFailureThreshold: "8",
		CircuitBreakerSuccessThreshold: "3",
		CircuitBreakerResetTimeout:     "45s",

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

	// Exclude patterns - should be split, trimmed, empty lines removed
	assert.Equal(t, []string{"pattern1", "pattern2", "pattern3", "pattern4"}, cfg.Exclude)

	// Rate limit
	assert.True(t, cfg.LLM.RateLimit.Enabled)
	assert.Equal(t, 100, cfg.LLM.RateLimit.RequestsPerMinute)
	assert.Equal(t, 15, cfg.LLM.RateLimit.BurstSize)
	assert.Equal(t, 4, cfg.LLM.RateLimit.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, cfg.LLM.RateLimit.InitialDelay)
	assert.Equal(t, 30*time.Second, cfg.LLM.RateLimit.MaxDelay)
	assert.Equal(t, 1.5, cfg.LLM.RateLimit.Multiplier)

	// Circuit breaker
	assert.True(t, cfg.LLM.RateLimit.CircuitBreaker.Enabled)
	assert.Equal(t, 8, cfg.LLM.RateLimit.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 3, cfg.LLM.RateLimit.CircuitBreaker.SuccessThresholdHalfOpen)
	assert.Equal(t, 45*time.Second, cfg.LLM.RateLimit.CircuitBreaker.ResetTimeout)
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

func TestToConfig_EmptyExcludePatterns(t *testing.T) {
	values := &ConfigValues{
		ExcludePatterns: "",
	}

	cfg, err := values.ToConfig()
	require.NoError(t, err)

	// Empty string should result in empty slice (not nil)
	assert.Empty(t, cfg.Exclude)
}

func TestToConfig_DefaultRateLimitAndCircuitBreaker(t *testing.T) {
	values := &ConfigValues{
		// Empty strings for all new fields
		RateLimitRequestsPerMinute:     "",
		RateLimitBurstSize:             "",
		RateLimitMaxRetries:            "",
		RateLimitInitialDelay:          "",
		RateLimitMaxDelay:              "",
		RateLimitMultiplier:            "",
		CircuitBreakerFailureThreshold: "",
		CircuitBreakerSuccessThreshold: "",
		CircuitBreakerResetTimeout:     "",
	}

	cfg, err := values.ToConfig()
	require.NoError(t, err)

	// Should use defaults
	assert.Equal(t, config.DefaultRateLimitRequestsPerMinute, cfg.LLM.RateLimit.RequestsPerMinute)
	assert.Equal(t, config.DefaultRateLimitBurstSize, cfg.LLM.RateLimit.BurstSize)
	assert.Equal(t, config.DefaultRateLimitMaxRetries, cfg.LLM.RateLimit.MaxRetries)
	assert.Equal(t, config.DefaultRateLimitInitialDelay, cfg.LLM.RateLimit.InitialDelay)
	assert.Equal(t, config.DefaultRateLimitMaxDelay, cfg.LLM.RateLimit.MaxDelay)
	assert.Equal(t, config.DefaultRateLimitMultiplier, cfg.LLM.RateLimit.Multiplier)
	assert.Equal(t, config.DefaultCircuitBreakerFailureThreshold, cfg.LLM.RateLimit.CircuitBreaker.FailureThreshold)
	assert.Equal(t, config.DefaultCircuitBreakerSuccessThresholdHalfOpen, cfg.LLM.RateLimit.CircuitBreaker.SuccessThresholdHalfOpen)
	assert.Equal(t, config.DefaultCircuitBreakerResetTimeout, cfg.LLM.RateLimit.CircuitBreaker.ResetTimeout)
}
