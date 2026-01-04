package config_test

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig_DefaultValues(t *testing.T) {
	defaults := config.Default()

	// Check output defaults
	assert.Equal(t, "./docs", defaults.Output.Directory)
	assert.False(t, defaults.Output.Flat)
	assert.False(t, defaults.Output.JSONMetadata)
	assert.False(t, defaults.Output.Overwrite)

	// Check concurrency defaults
	assert.Equal(t, config.DefaultWorkers, defaults.Concurrency.Workers)
	assert.Equal(t, config.DefaultMaxDepth, defaults.Concurrency.MaxDepth)
	assert.Equal(t, config.DefaultTimeout, defaults.Concurrency.Timeout)

	// Check cache defaults
	assert.True(t, defaults.Cache.Enabled)
	assert.Equal(t, config.DefaultCacheTTL, defaults.Cache.TTL)

	// Check rendering defaults
	assert.False(t, defaults.Rendering.ForceJS)
	assert.Equal(t, config.DefaultJSTimeout, defaults.Rendering.JSTimeout)
	assert.True(t, defaults.Rendering.ScrollToEnd)

	// Check logging defaults
	assert.Equal(t, "info", defaults.Logging.Level)
	assert.Equal(t, "pretty", defaults.Logging.Format)
}

func TestConfig_Validate_FixesInvalidWorkers(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers: 0, // Invalid
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
}

func TestConfig_Validate_FixesInvalidMaxDepth(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  5,
			MaxDepth: 0, // Invalid
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultMaxDepth, cfg.Concurrency.MaxDepth)
}

func TestConfig_Validate_FixesInvalidTimeout(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  5,
			MaxDepth: 3,
			Timeout:  500 * time.Millisecond, // Less than 1 second
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
}

func TestConfig_Validate_FixesInvalidCacheTTL(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  5,
			MaxDepth: 3,
			Timeout:  30 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL: 30 * time.Second, // Less than 1 minute
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
}

func TestConfig_Validate_FixesInvalidJSTimeout(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  5,
			MaxDepth: 3,
			Timeout:  30 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL: 24 * time.Hour,
		},
		Rendering: config.RenderingConfig{
			JSTimeout: 500 * time.Millisecond, // Less than 1 second
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
}

func TestConfig_Validate_PreservesValidValues(t *testing.T) {
	cfg := &config.Config{
		Output: config.OutputConfig{
			Directory:    "/custom/dir",
			Flat:         true,
			JSONMetadata: true,
			Overwrite:    true,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  10,
			MaxDepth: 5,
			Timeout:  60 * time.Second,
		},
		Cache: config.CacheConfig{
			Enabled:   false,
			TTL:       48 * time.Hour,
			Directory: "/custom/cache",
		},
		Rendering: config.RenderingConfig{
			ForceJS:     true,
			JSTimeout:   120 * time.Second,
			ScrollToEnd: false,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// All values should be preserved
	assert.Equal(t, "/custom/dir", cfg.Output.Directory)
	assert.True(t, cfg.Output.Flat)
	assert.True(t, cfg.Output.JSONMetadata)
	assert.Equal(t, 10, cfg.Concurrency.Workers)
	assert.Equal(t, 5, cfg.Concurrency.MaxDepth)
	assert.Equal(t, 60*time.Second, cfg.Concurrency.Timeout)
	assert.Equal(t, 48*time.Hour, cfg.Cache.TTL)
	assert.Equal(t, 120*time.Second, cfg.Rendering.JSTimeout)
}

func TestOutputConfig_Structure(t *testing.T) {
	cfg := config.OutputConfig{
		Directory:    "./docs",
		Flat:         true,
		JSONMetadata: true,
		Overwrite:    false,
	}

	assert.Equal(t, "./docs", cfg.Directory)
	assert.True(t, cfg.Flat)
	assert.True(t, cfg.JSONMetadata)
	assert.False(t, cfg.Overwrite)
}

func TestConcurrencyConfig_Structure(t *testing.T) {
	cfg := config.ConcurrencyConfig{
		Workers:  8,
		Timeout:  45 * time.Second,
		MaxDepth: 4,
	}

	assert.Equal(t, 8, cfg.Workers)
	assert.Equal(t, 45*time.Second, cfg.Timeout)
	assert.Equal(t, 4, cfg.MaxDepth)
}

func TestCacheConfig_Structure(t *testing.T) {
	cfg := config.CacheConfig{
		Enabled:   true,
		TTL:       12 * time.Hour,
		Directory: "~/.cache/repodocs",
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 12*time.Hour, cfg.TTL)
	assert.Equal(t, "~/.cache/repodocs", cfg.Directory)
}

func TestRenderingConfig_Structure(t *testing.T) {
	cfg := config.RenderingConfig{
		ForceJS:     true,
		JSTimeout:   90 * time.Second,
		ScrollToEnd: true,
	}

	assert.True(t, cfg.ForceJS)
	assert.Equal(t, 90*time.Second, cfg.JSTimeout)
	assert.True(t, cfg.ScrollToEnd)
}

func TestStealthConfig_Structure(t *testing.T) {
	cfg := config.StealthConfig{
		UserAgent:      "Custom User Agent",
		RandomDelayMin: 1 * time.Second,
		RandomDelayMax: 3 * time.Second,
	}

	assert.Equal(t, "Custom User Agent", cfg.UserAgent)
	assert.Equal(t, 1*time.Second, cfg.RandomDelayMin)
	assert.Equal(t, 3*time.Second, cfg.RandomDelayMax)
}

func TestLoggingConfig_Structure(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "json",
	}

	assert.Equal(t, "debug", cfg.Level)
	assert.Equal(t, "json", cfg.Format)
}

// ============================================================================
// LLM Configuration Tests
// ============================================================================

func TestLLMConfig_Structure(t *testing.T) {
	cfg := config.LLMConfig{
		Provider:        "anthropic",
		APIKey:          "test-key",
		BaseURL:         "https://api.example.com",
		Model:           "claude-3",
		MaxTokens:       8192,
		Temperature:     0.5,
		Timeout:         120 * time.Second,
		MaxRetries:      5,
		EnhanceMetadata: true,
	}

	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, "https://api.example.com", cfg.BaseURL)
	assert.Equal(t, "claude-3", cfg.Model)
	assert.Equal(t, 8192, cfg.MaxTokens)
	assert.Equal(t, 0.5, cfg.Temperature)
	assert.Equal(t, 120*time.Second, cfg.Timeout)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.True(t, cfg.EnhanceMetadata)
}

func TestLLMConfig_RateLimitStructure(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 120,
		BurstSize:         20,
		MaxRetries:        5,
		InitialDelay:      2 * time.Second,
		MaxDelay:          120 * time.Second,
		Multiplier:        3.0,
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled:                  true,
			FailureThreshold:         10,
			SuccessThresholdHalfOpen: 2,
			ResetTimeout:             60 * time.Second,
		},
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 120, cfg.RequestsPerMinute)
	assert.Equal(t, 20, cfg.BurstSize)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 2*time.Second, cfg.InitialDelay)
	assert.Equal(t, 120*time.Second, cfg.MaxDelay)
	assert.Equal(t, 3.0, cfg.Multiplier)
	assert.True(t, cfg.CircuitBreaker.Enabled)
	assert.Equal(t, 10, cfg.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 2, cfg.CircuitBreaker.SuccessThresholdHalfOpen)
	assert.Equal(t, 60*time.Second, cfg.CircuitBreaker.ResetTimeout)
}

func TestCircuitBreakerConfig_Structure(t *testing.T) {
	cfg := config.CircuitBreakerConfig{
		Enabled:                  true,
		FailureThreshold:         7,
		SuccessThresholdHalfOpen: 3,
		ResetTimeout:             45 * time.Second,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 7, cfg.FailureThreshold)
	assert.Equal(t, 3, cfg.SuccessThresholdHalfOpen)
	assert.Equal(t, 45*time.Second, cfg.ResetTimeout)
}

// ============================================================================
// Validation Edge Cases
// ============================================================================

func TestConfig_Validate_AllZeroValues(t *testing.T) {
	cfg := &config.Config{
		// All zero/invalid values
		Concurrency: config.ConcurrencyConfig{
			Workers:  0,
			MaxDepth: 0,
			Timeout:  0,
		},
		Cache: config.CacheConfig{
			TTL: 0,
		},
		Rendering: config.RenderingConfig{
			JSTimeout: 0,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// All should be corrected to defaults
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, config.DefaultMaxDepth, cfg.Concurrency.MaxDepth)
	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
}

func TestConfig_Validate_NegativeValues(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  -5,
			MaxDepth: -1,
			Timeout:  -100 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL: -24 * time.Hour,
		},
		Rendering: config.RenderingConfig{
			JSTimeout: -60 * time.Second,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// Negative values should be corrected to defaults
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, config.DefaultMaxDepth, cfg.Concurrency.MaxDepth)
	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
}

func TestConfig_Validate_BoundaryValues(t *testing.T) {
	tests := []struct {
		name              string
		value             time.Duration
		expectedTimeout   time.Duration
		expectedCacheTTL  time.Duration
		expectedJSTimeout time.Duration
	}{
		{"zero timeout", 0, config.DefaultTimeout, config.DefaultCacheTTL, config.DefaultJSTimeout},
		{"one nanosecond", 1 * time.Nanosecond, config.DefaultTimeout, config.DefaultCacheTTL, config.DefaultJSTimeout},
		{"999 milliseconds", 999 * time.Millisecond, config.DefaultTimeout, config.DefaultCacheTTL, config.DefaultJSTimeout},
		{"exactly one second", 1 * time.Second, 1 * time.Second, config.DefaultCacheTTL, 1 * time.Second},
		{"one second minus one ns", 1*time.Second - 1, config.DefaultTimeout, config.DefaultCacheTTL, config.DefaultJSTimeout},
		{"59 seconds for cache TTL", 59 * time.Second, 59 * time.Second, config.DefaultCacheTTL, 59 * time.Second},
		{"exactly one minute", 1 * time.Minute, 1 * time.Minute, 1 * time.Minute, 1 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Concurrency: config.ConcurrencyConfig{
					Workers:  5,
					MaxDepth: 3,
					Timeout:  tt.value,
				},
				Cache: config.CacheConfig{
					TTL: tt.value,
				},
				Rendering: config.RenderingConfig{
					JSTimeout: tt.value,
				},
			}

			err := cfg.Validate()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTimeout, cfg.Concurrency.Timeout)
			assert.Equal(t, tt.expectedCacheTTL, cfg.Cache.TTL)
			assert.Equal(t, tt.expectedJSTimeout, cfg.Rendering.JSTimeout)
		})
	}
}

func TestConfig_Validate_LargeValidValues(t *testing.T) {
	cfg := &config.Config{
		Concurrency: config.ConcurrencyConfig{
			Workers:  1000,
			MaxDepth: 100,
			Timeout:  3600 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL: 720 * time.Hour, // 30 days
		},
		Rendering: config.RenderingConfig{
			JSTimeout: 600 * time.Second,
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// Large valid values should be preserved
	assert.Equal(t, 1000, cfg.Concurrency.Workers)
	assert.Equal(t, 100, cfg.Concurrency.MaxDepth)
	assert.Equal(t, 3600*time.Second, cfg.Concurrency.Timeout)
	assert.Equal(t, 720*time.Hour, cfg.Cache.TTL)
	assert.Equal(t, 600*time.Second, cfg.Rendering.JSTimeout)
}

func TestConfig_Validate_PartialInvalidConfig(t *testing.T) {
	cfg := &config.Config{
		Output: config.OutputConfig{
			Directory: "/valid/path",
			Flat:      true,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  0,                // Invalid
			MaxDepth: 5,                // Valid
			Timeout:  30 * time.Second, // Valid
		},
		Cache: config.CacheConfig{
			Enabled: true,
			TTL:     0, // Invalid
		},
		Rendering: config.RenderingConfig{
			JSTimeout: 60 * time.Second, // Valid
		},
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// Invalid values corrected, valid values preserved
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, 5, cfg.Concurrency.MaxDepth)
	assert.Equal(t, 30*time.Second, cfg.Concurrency.Timeout)
	assert.Equal(t, "/valid/path", cfg.Output.Directory)
	assert.True(t, cfg.Output.Flat)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Equal(t, 60*time.Second, cfg.Rendering.JSTimeout)
}

// ============================================================================
// Table-Driven Validation Tests
// ============================================================================

func TestConfig_Validate_WorkersTable(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		expected int
	}{
		{"zero workers", 0, config.DefaultWorkers},
		{"negative workers", -10, config.DefaultWorkers},
		{"one worker", 1, 1},
		{"valid workers", 10, 10},
		{"large workers", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Concurrency: config.ConcurrencyConfig{
					Workers:  tt.workers,
					MaxDepth: 3,
					Timeout:  30 * time.Second,
				},
			}

			err := cfg.Validate()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Concurrency.Workers)
		})
	}
}

func TestConfig_Validate_MaxDepthTable(t *testing.T) {
	tests := []struct {
		name     string
		maxDepth int
		expected int
	}{
		{"zero depth", 0, config.DefaultMaxDepth},
		{"negative depth", -5, config.DefaultMaxDepth},
		{"depth of one", 1, 1},
		{"valid depth", 5, 5},
		{"large depth", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Concurrency: config.ConcurrencyConfig{
					Workers:  5,
					MaxDepth: tt.maxDepth,
					Timeout:  30 * time.Second,
				},
			}

			err := cfg.Validate()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cfg.Concurrency.MaxDepth)
		})
	}
}

func TestConfig_Validate_TimeoutsTable(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		cacheTTL        time.Duration
		jsTimeout       time.Duration
		expectTimeout   time.Duration
		expectCacheTTL  time.Duration
		expectJSTimeout time.Duration
	}{
		{
			name:            "all below minimum",
			timeout:         500 * time.Millisecond,
			cacheTTL:        30 * time.Second,
			jsTimeout:       500 * time.Millisecond,
			expectTimeout:   config.DefaultTimeout,
			expectCacheTTL:  config.DefaultCacheTTL,
			expectJSTimeout: config.DefaultJSTimeout,
		},
		{
			name:            "all at minimum",
			timeout:         1 * time.Second,
			cacheTTL:        1 * time.Minute,
			jsTimeout:       1 * time.Second,
			expectTimeout:   1 * time.Second,
			expectCacheTTL:  1 * time.Minute,
			expectJSTimeout: 1 * time.Second,
		},
		{
			name:            "all above minimum",
			timeout:         120 * time.Second,
			cacheTTL:        48 * time.Hour,
			jsTimeout:       90 * time.Second,
			expectTimeout:   120 * time.Second,
			expectCacheTTL:  48 * time.Hour,
			expectJSTimeout: 90 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Concurrency: config.ConcurrencyConfig{
					Workers:  5,
					MaxDepth: 3,
					Timeout:  tt.timeout,
				},
				Cache: config.CacheConfig{
					TTL: tt.cacheTTL,
				},
				Rendering: config.RenderingConfig{
					JSTimeout: tt.jsTimeout,
				},
			}

			err := cfg.Validate()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectTimeout, cfg.Concurrency.Timeout)
			assert.Equal(t, tt.expectCacheTTL, cfg.Cache.TTL)
			assert.Equal(t, tt.expectJSTimeout, cfg.Rendering.JSTimeout)
		})
	}
}

// ============================================================================
// Config Structure Validation
// ============================================================================

func TestConfig_AllFieldsPresent(t *testing.T) {
	cfg := config.Default()

	// Verify all top-level fields are present and non-nil
	assert.NotNil(t, cfg.Output)
	assert.NotNil(t, cfg.Concurrency)
	assert.NotNil(t, cfg.Cache)
	assert.NotNil(t, cfg.Rendering)
	assert.NotNil(t, cfg.Stealth)
	assert.NotNil(t, cfg.Exclude)
	assert.NotNil(t, cfg.Logging)
	assert.NotNil(t, cfg.LLM)
}

func TestConfig_LLMNestedFieldsPresent(t *testing.T) {
	cfg := config.Default()

	// Verify LLM nested fields are present
	assert.NotNil(t, cfg.LLM.RateLimit)
	assert.NotNil(t, cfg.LLM.RateLimit.CircuitBreaker)
}

func TestConfig_ExcludeIsCopy(t *testing.T) {
	cfg1 := config.Default()
	cfg2 := config.Default()

	// Modify exclude in cfg1
	cfg1.Exclude = append(cfg1.Exclude, ".*\\.new$")

	// cfg2 should not be affected
	assert.NotEqual(t, len(cfg1.Exclude), len(cfg2.Exclude))
	assert.Equal(t, len(config.DefaultExcludePatterns), len(cfg2.Exclude))
}
