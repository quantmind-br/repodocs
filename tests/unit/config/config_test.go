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
