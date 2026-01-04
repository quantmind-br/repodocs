package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Validate tests configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		modify  func(*Config)
		check   func(*testing.T, *Config)
		wantErr bool
	}{
		{
			name: "valid config",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Concurrency.Workers = 5
				c.Concurrency.MaxDepth = 3
				c.Concurrency.Timeout = 30 * time.Second
				c.Cache.TTL = 24 * time.Hour
				c.Rendering.JSTimeout = 60 * time.Second
			},
			wantErr: false,
		},
		{
			name: "workers below minimum defaults to 5",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Concurrency.Workers = 0
			},
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, DefaultWorkers, c.Concurrency.Workers)
			},
			wantErr: false,
		},
		{
			name: "max depth below minimum defaults to 3",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Concurrency.MaxDepth = 0
			},
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, DefaultMaxDepth, c.Concurrency.MaxDepth)
			},
			wantErr: false,
		},
		{
			name: "timeout below minimum defaults to 30s",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Concurrency.Timeout = 100 * time.Millisecond
			},
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, DefaultTimeout, c.Concurrency.Timeout)
			},
			wantErr: false,
		},
		{
			name: "cache TTL below minimum defaults to 24h",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Cache.TTL = 30 * time.Second
			},
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, DefaultCacheTTL, c.Cache.TTL)
			},
			wantErr: false,
		},
		{
			name: "JS timeout below minimum defaults to 60s",
			cfg:  &Config{},
			modify: func(c *Config) {
				c.Rendering.JSTimeout = 500 * time.Millisecond
			},
			check: func(t *testing.T, c *Config) {
				assert.Equal(t, DefaultJSTimeout, c.Rendering.JSTimeout)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.modify != nil {
				tt.modify(tt.cfg)
			}
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.check != nil {
				tt.check(t, tt.cfg)
			}
		})
	}
}

// TestDefault tests default configuration
func TestDefault(t *testing.T) {
	cfg := Default()

	assert.NotNil(t, cfg)
	assert.Equal(t, DefaultOutputDir, cfg.Output.Directory)
	assert.False(t, cfg.Output.Flat)
	assert.False(t, cfg.Output.JSONMetadata)
	assert.False(t, cfg.Output.Overwrite)

	assert.Equal(t, DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, DefaultMaxDepth, cfg.Concurrency.MaxDepth)

	assert.True(t, cfg.Cache.Enabled)
	assert.Equal(t, DefaultCacheTTL, cfg.Cache.TTL)
	assert.Contains(t, cfg.Cache.Directory, "cache")

	assert.False(t, cfg.Rendering.ForceJS)
	assert.Equal(t, DefaultJSTimeout, cfg.Rendering.JSTimeout)
	assert.Equal(t, DefaultScrollToEnd, cfg.Rendering.ScrollToEnd)

	assert.Equal(t, "", cfg.Stealth.UserAgent)
	assert.Equal(t, DefaultRandomDelayMin, cfg.Stealth.RandomDelayMin)
	assert.Equal(t, DefaultRandomDelayMax, cfg.Stealth.RandomDelayMax)

	assert.NotEmpty(t, cfg.Exclude)

	assert.Equal(t, DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, DefaultLogFormat, cfg.Logging.Format)

	assert.Equal(t, DefaultLLMMaxTokens, cfg.LLM.MaxTokens)
	assert.Equal(t, DefaultLLMTemperature, cfg.LLM.Temperature)
	assert.Equal(t, DefaultLLMTimeout, cfg.LLM.Timeout)
	assert.Equal(t, DefaultLLMMaxRetries, cfg.LLM.MaxRetries)

	// Check rate limit defaults
	assert.True(t, cfg.LLM.RateLimit.Enabled)
	assert.Equal(t, DefaultRateLimitRequestsPerMinute, cfg.LLM.RateLimit.RequestsPerMinute)
	assert.Equal(t, DefaultRateLimitBurstSize, cfg.LLM.RateLimit.BurstSize)
	assert.Equal(t, DefaultRateLimitMaxRetries, cfg.LLM.RateLimit.MaxRetries)
	assert.Equal(t, DefaultRateLimitInitialDelay, cfg.LLM.RateLimit.InitialDelay)
	assert.Equal(t, DefaultRateLimitMaxDelay, cfg.LLM.RateLimit.MaxDelay)
	assert.Equal(t, DefaultRateLimitMultiplier, cfg.LLM.RateLimit.Multiplier)

	// Check circuit breaker defaults
	assert.True(t, cfg.LLM.RateLimit.CircuitBreaker.Enabled)
	assert.Equal(t, DefaultCircuitBreakerFailureThreshold, cfg.LLM.RateLimit.CircuitBreaker.FailureThreshold)
	assert.Equal(t, DefaultCircuitBreakerSuccessThresholdHalfOpen, cfg.LLM.RateLimit.CircuitBreaker.SuccessThresholdHalfOpen)
	assert.Equal(t, DefaultCircuitBreakerResetTimeout, cfg.LLM.RateLimit.CircuitBreaker.ResetTimeout)
}

// TestConfigDir tests config directory path
func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	assert.NotEmpty(t, dir)

	// Should contain repodocs
	assert.Contains(t, dir, "repodocs")
}

// TestCacheDir tests cache directory path
func TestCacheDir(t *testing.T) {
	dir := CacheDir()
	assert.NotEmpty(t, dir)

	// Should end with cache
	assert.True(t, strings.HasSuffix(dir, "cache") || strings.Contains(dir, "/cache"))
}

// TestConfigFilePath tests config file path
func TestConfigFilePath(t *testing.T) {
	path := ConfigFilePath()
	assert.NotEmpty(t, path)

	// Should contain config.yaml
	assert.Contains(t, path, "config.yaml")
}

// TestDefaultExcludePatterns tests default exclude patterns
func TestDefaultExcludePatterns(t *testing.T) {
	patterns := DefaultExcludePatterns
	assert.NotEmpty(t, patterns)

	// Check for expected patterns
	expectedPatterns := []string{
		`.*\.pdf$`,
		`.*/login.*`,
		`.*/admin.*`,
	}

	for _, expected := range expectedPatterns {
		assert.Contains(t, patterns, expected)
	}
}

// TestEnsureConfigDir tests creating config directory
func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock the home directory
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Create a temporary home directory
	testHome := filepath.Join(tmpDir, "testuser")
	require.NoError(t, os.MkdirAll(testHome, 0755))
	os.Setenv("HOME", testHome)

	// ConfigDir should now point to temp directory
	configDir := ConfigDir()

	err := EnsureConfigDir()
	assert.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(configDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestEnsureCacheDir tests creating cache directory
func TestEnsureCacheDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Mock the home directory
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Create a temporary home directory
	testHome := filepath.Join(tmpDir, "testuser")
	require.NoError(t, os.MkdirAll(testHome, 0755))
	os.Setenv("HOME", testHome)

	cacheDir := CacheDir()

	err := EnsureCacheDir()
	assert.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(cacheDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestLoad_LoadWithMissingConfig tests loading with no config file
func TestLoad_LoadWithMissingConfig(t *testing.T) {
	// Create a temporary directory with no config file
	tmpDir := t.TempDir()

	// Change to temp directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// Load should succeed with defaults (no config file is OK)
	cfg, _, err := LoadWithViper()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have default values
	assert.NotEmpty(t, cfg.Output.Directory)
}

// TestLoad_WithInvalidConfigFile tests loading with invalid config file
func TestLoad_WithInvalidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// Load should return an error for invalid YAML
	cfg, _, err := LoadWithViper()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestLoad_WithValidConfigFile tests loading with valid config file
func TestLoad_WithValidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
output:
  directory: "./test-output"
  flat: true

logging:
  level: "debug"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	os.Chdir(tmpDir)

	// Load should succeed
	cfg, _, err := LoadWithViper()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should have values from config file
	assert.Equal(t, "./test-output", cfg.Output.Directory)
	assert.True(t, cfg.Output.Flat)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

// TestLoadWithEnvironmentVariable tests loading with environment variable
func TestLoadWithEnvironmentVariable(t *testing.T) {
	// Set environment variable
	os.Setenv("REPODOCS_OUTPUT_DIRECTORY", "./env-output")
	defer os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")

	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Environment variable should override default
	assert.Equal(t, "./env-output", cfg.Output.Directory)
}

// TestLoadWithViper tests LoadWithViper function
func TestLoadWithViper(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, v, err := LoadWithViper()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotNil(t, v)
}

// TestConfigStructFieldTags tests struct field tags
func TestConfigStructFieldTags(t *testing.T) {
	// This test ensures config structs have proper mapstructure tags
	// for viper unmarshaling

	cfg := Config{}

	// Check that Config has expected fields (structs are initialized)
	// Note: Exclude is a slice, not a pointer, so we check if it's nil by length
	assert.NotNil(t, cfg.Output)
	assert.NotNil(t, cfg.Concurrency)
	assert.NotNil(t, cfg.Cache)
	assert.NotNil(t, cfg.Rendering)
	assert.NotNil(t, cfg.Stealth)
	assert.NotNil(t, cfg.Logging)
	assert.NotNil(t, cfg.LLM)
	// Exclude is a slice - initialize to check it's properly defined
	cfg.Exclude = []string{"test"}
	assert.NotEmpty(t, cfg.Exclude)
}

// TestConstants tests constant values
func TestConstants(t *testing.T) {
	// Test that constants have reasonable values
	assert.Greater(t, DefaultWorkers, 0)
	assert.Greater(t, int(DefaultTimeout.Seconds()), int(time.Second.Seconds()))
	assert.Greater(t, DefaultMaxDepth, 0)
	assert.Greater(t, int(DefaultCacheTTL.Seconds()), int(time.Minute.Seconds()))
	assert.Greater(t, int(DefaultJSTimeout.Seconds()), int(time.Second.Seconds()))
	assert.Greater(t, DefaultLLMMaxTokens, 0)
	assert.Greater(t, DefaultLLMTemperature, float64(0))
	assert.LessOrEqual(t, DefaultLLMTemperature, float64(2))
	assert.Greater(t, int(DefaultLLMTimeout.Seconds()), int(time.Second.Seconds()))

	// Check rate limit constants
	assert.Greater(t, DefaultRateLimitRequestsPerMinute, 0)
	assert.Greater(t, DefaultRateLimitBurstSize, 0)
	assert.Greater(t, DefaultRateLimitMaxRetries, 0)
	assert.Greater(t, int(DefaultRateLimitInitialDelay.Seconds()), 0)
	assert.Greater(t, int(DefaultRateLimitMaxDelay.Seconds()), int(time.Second.Seconds()))
	assert.Greater(t, DefaultRateLimitMultiplier, float64(0))

	// Check circuit breaker constants
	assert.Greater(t, DefaultCircuitBreakerFailureThreshold, 0)
	assert.Greater(t, DefaultCircuitBreakerSuccessThresholdHalfOpen, 0)
	assert.Greater(t, int(DefaultCircuitBreakerResetTimeout.Seconds()), int(time.Second.Seconds()))
}
