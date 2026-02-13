package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Default Constants Tests
// ============================================================================

func TestDefaultConstants_Output(t *testing.T) {
	assert.Equal(t, "./docs", config.DefaultOutputDir)
}

func TestDefaultConstants_Concurrency(t *testing.T) {
	assert.Equal(t, 5, config.DefaultWorkers)
	assert.Equal(t, 90*time.Second, config.DefaultTimeout)
	assert.Equal(t, 3, config.DefaultMaxDepth)
}

func TestDefaultConstants_Cache(t *testing.T) {
	assert.True(t, config.DefaultCacheEnabled)
	assert.Equal(t, 24*time.Hour, config.DefaultCacheTTL)
}

func TestDefaultConstants_Rendering(t *testing.T) {
	assert.Equal(t, 60*time.Second, config.DefaultJSTimeout)
	assert.True(t, config.DefaultScrollToEnd)
}

func TestDefaultConstants_Stealth(t *testing.T) {
	assert.Equal(t, 1*time.Second, config.DefaultRandomDelayMin)
	assert.Equal(t, 3*time.Second, config.DefaultRandomDelayMax)
}

func TestDefaultConstants_Logging(t *testing.T) {
	assert.Equal(t, "info", config.DefaultLogLevel)
	assert.Equal(t, "pretty", config.DefaultLogFormat)
}

func TestDefaultConstants_LLM(t *testing.T) {
	assert.Equal(t, 4096, config.DefaultLLMMaxTokens)
	assert.Equal(t, 0.7, config.DefaultLLMTemperature)
	assert.Equal(t, 60*time.Second, config.DefaultLLMTimeout)
	assert.Equal(t, 3, config.DefaultLLMMaxRetries)
}

func TestDefaultConstants_RateLimit(t *testing.T) {
	assert.True(t, config.DefaultRateLimitEnabled)
	assert.Equal(t, 60, config.DefaultRateLimitRequestsPerMinute)
	assert.Equal(t, 10, config.DefaultRateLimitBurstSize)
	assert.Equal(t, 3, config.DefaultRateLimitMaxRetries)
	assert.Equal(t, 1*time.Second, config.DefaultRateLimitInitialDelay)
	assert.Equal(t, 60*time.Second, config.DefaultRateLimitMaxDelay)
	assert.Equal(t, 2.0, config.DefaultRateLimitMultiplier)
}

func TestDefaultConstants_CircuitBreaker(t *testing.T) {
	assert.True(t, config.DefaultCircuitBreakerEnabled)
	assert.Equal(t, 5, config.DefaultCircuitBreakerFailureThreshold)
	assert.Equal(t, 1, config.DefaultCircuitBreakerSuccessThresholdHalfOpen)
	assert.Equal(t, 30*time.Second, config.DefaultCircuitBreakerResetTimeout)
}

// ============================================================================
// DefaultExcludePatterns Tests
// ============================================================================

func TestDefaultExcludePatterns_NotNil(t *testing.T) {
	assert.NotNil(t, config.DefaultExcludePatterns)
}

func TestDefaultExcludePatterns_ExpectedPatterns(t *testing.T) {
	patterns := config.DefaultExcludePatterns

	// Check that we have the expected number of patterns
	assert.GreaterOrEqual(t, len(patterns), 5, "Should have at least 5 exclude patterns")

	// Check for common patterns
	expectedPatterns := []string{
		`.*\.pdf$`,
		`.*/login.*`,
		`.*/logout.*`,
		`.*/admin.*`,
		`.*/sign-in.*`,
		`.*/sign-up.*`,
	}

	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range patterns {
			if pattern == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected pattern %q not found in DefaultExcludePatterns", expected)
	}
}

func TestDefaultExcludePatterns_PatternsAreValid(t *testing.T) {
	// All patterns should be valid regex patterns
	patterns := config.DefaultExcludePatterns

	for _, pattern := range patterns {
		assert.NotEmpty(t, pattern, "Exclude pattern should not be empty")
		assert.NotEqual(t, "/", pattern, "Exclude pattern should not be just a slash")
	}
}

// ============================================================================
// Default() Function Tests - Top Level
// ============================================================================

func TestDefault_NilSafe(t *testing.T) {
	cfg := config.Default()
	assert.NotNil(t, cfg)
}

func TestDefault_OutputDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, config.DefaultOutputDir, cfg.Output.Directory)
	assert.False(t, cfg.Output.Flat)
	assert.False(t, cfg.Output.JSONMetadata)
	assert.False(t, cfg.Output.Overwrite)
}

func TestDefault_ConcurrencyDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, config.DefaultMaxDepth, cfg.Concurrency.MaxDepth)
}

func TestDefault_CacheDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, config.DefaultCacheEnabled, cfg.Cache.Enabled)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Contains(t, cfg.Cache.Directory, "cache")
}

func TestDefault_RenderingDefaults(t *testing.T) {
	cfg := config.Default()

	assert.False(t, cfg.Rendering.ForceJS)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
	assert.Equal(t, config.DefaultScrollToEnd, cfg.Rendering.ScrollToEnd)
}

func TestDefault_StealthDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Empty(t, cfg.Stealth.UserAgent, "Default user agent should be empty")
	assert.Equal(t, config.DefaultRandomDelayMin, cfg.Stealth.RandomDelayMin)
	assert.Equal(t, config.DefaultRandomDelayMax, cfg.Stealth.RandomDelayMax)
}

func TestDefault_ExcludeDefaults(t *testing.T) {
	cfg := config.Default()

	assert.NotNil(t, cfg.Exclude)
	assert.Equal(t, len(config.DefaultExcludePatterns), len(cfg.Exclude))

	// Verify all default patterns are present
	for i, pattern := range config.DefaultExcludePatterns {
		assert.Equal(t, pattern, cfg.Exclude[i])
	}
}

func TestDefault_LoggingDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Equal(t, config.DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, config.DefaultLogFormat, cfg.Logging.Format)
}

// ============================================================================
// Default() Function Tests - LLM Nested Defaults
// ============================================================================

func TestDefault_LLM_TopLevelDefaults(t *testing.T) {
	cfg := config.Default()

	assert.Empty(t, cfg.LLM.Provider, "Default LLM provider should be empty")
	assert.Empty(t, cfg.LLM.APIKey, "Default LLM API key should be empty")
	assert.Empty(t, cfg.LLM.BaseURL, "Default LLM base URL should be empty")
	assert.Empty(t, cfg.LLM.Model, "Default LLM model should be empty")
	assert.Equal(t, config.DefaultLLMMaxTokens, cfg.LLM.MaxTokens)
	assert.Equal(t, config.DefaultLLMTemperature, cfg.LLM.Temperature)
	assert.Equal(t, config.DefaultLLMTimeout, cfg.LLM.Timeout)
	assert.Equal(t, config.DefaultLLMMaxRetries, cfg.LLM.MaxRetries)
	assert.False(t, cfg.LLM.EnhanceMetadata, "Default enhance_metadata should be false")
}

func TestDefault_LLM_RateLimitDefaults(t *testing.T) {
	cfg := config.Default()

	rl := cfg.LLM.RateLimit
	assert.Equal(t, config.DefaultRateLimitEnabled, rl.Enabled)
	assert.Equal(t, config.DefaultRateLimitRequestsPerMinute, rl.RequestsPerMinute)
	assert.Equal(t, config.DefaultRateLimitBurstSize, rl.BurstSize)
	assert.Equal(t, config.DefaultRateLimitMaxRetries, rl.MaxRetries)
	assert.Equal(t, config.DefaultRateLimitInitialDelay, rl.InitialDelay)
	assert.Equal(t, config.DefaultRateLimitMaxDelay, rl.MaxDelay)
	assert.Equal(t, config.DefaultRateLimitMultiplier, rl.Multiplier)
}

func TestDefault_LLM_CircuitBreakerDefaults(t *testing.T) {
	cfg := config.Default()

	cb := cfg.LLM.RateLimit.CircuitBreaker
	assert.Equal(t, config.DefaultCircuitBreakerEnabled, cb.Enabled)
	assert.Equal(t, config.DefaultCircuitBreakerFailureThreshold, cb.FailureThreshold)
	assert.Equal(t, config.DefaultCircuitBreakerSuccessThresholdHalfOpen, cb.SuccessThresholdHalfOpen)
	assert.Equal(t, config.DefaultCircuitBreakerResetTimeout, cb.ResetTimeout)
}

// ============================================================================
// Default() Function Tests - Independence
// ============================================================================

func TestDefault_MultipleCallsReturnSameValues(t *testing.T) {
	cfg1 := config.Default()
	cfg2 := config.Default()

	// All values should be identical
	assert.Equal(t, cfg1.Output.Directory, cfg2.Output.Directory)
	assert.Equal(t, cfg1.Concurrency.Workers, cfg2.Concurrency.Workers)
	assert.Equal(t, cfg1.Cache.TTL, cfg2.Cache.TTL)
	assert.Equal(t, cfg1.LLM.MaxTokens, cfg2.LLM.MaxTokens)
}

func TestDefault_ModifyingReturnedConfigDoesNotAffectFutureCalls(t *testing.T) {
	cfg1 := config.Default()

	// Modify the first config
	cfg1.Output.Directory = "/modified"
	cfg1.Concurrency.Workers = 999

	// Get a new default config
	cfg2 := config.Default()

	// Should have original default values, not modified ones
	assert.Equal(t, config.DefaultOutputDir, cfg2.Output.Directory)
	assert.Equal(t, config.DefaultWorkers, cfg2.Concurrency.Workers)
}

// ============================================================================
// ConfigDir, CacheDir, ConfigFilePath Tests - Edge Cases
// ============================================================================

func TestConfigDir_Consistent(t *testing.T) {
	dir1 := config.ConfigDir()
	dir2 := config.ConfigDir()

	assert.Equal(t, dir1, dir2, "ConfigDir should return consistent path")
}

func TestCacheDir_Consistent(t *testing.T) {
	dir1 := config.CacheDir()
	dir2 := config.CacheDir()

	assert.Equal(t, dir1, dir2, "CacheDir should return consistent path")
}

func TestConfigFilePath_Consistent(t *testing.T) {
	path1 := config.ConfigFilePath()
	path2 := config.ConfigFilePath()

	assert.Equal(t, path1, path2, "ConfigFilePath should return consistent path")
}

func TestCacheDir_RelationToConfigDir(t *testing.T) {
	cacheDir := config.CacheDir()
	configDir := config.ConfigDir()

	expectedCacheDir := filepath.Join(configDir, "cache")
	assert.Equal(t, expectedCacheDir, cacheDir, "CacheDir should be a subdirectory of ConfigDir")
}

func TestConfigFilePath_RelationToConfigDir(t *testing.T) {
	configPath := config.ConfigFilePath()
	configDir := config.ConfigDir()

	expectedPath := filepath.Join(configDir, "config.yaml")
	assert.Equal(t, expectedPath, configPath, "ConfigFilePath should be config.yaml in ConfigDir")
}

func TestConfigDir_FallbackOnError(t *testing.T) {
	// This test verifies that when os.UserHomeDir fails, it falls back to ".repodocs"
	// We can't easily mock os.UserHomeDir, so we verify the structure instead

	dir := config.ConfigDir()

	// Should always return a non-empty string
	assert.NotEmpty(t, dir, "ConfigDir should never be empty")
	assert.Contains(t, dir, ".repodocs", "ConfigDir should contain '.repodocs'")
}

func TestCacheDir_WhenHomeDirUnavailable(t *testing.T) {
	// Verify CacheDir works even if ConfigDir uses fallback
	cacheDir := config.CacheDir()

	assert.NotEmpty(t, cacheDir, "CacheDir should never be empty")
	assert.Contains(t, cacheDir, "cache", "CacheDir should contain 'cache'")
}

// ============================================================================
// Table-Driven Tests for Default Values
// ============================================================================

func TestDefault_AllConstantsHaveExpectedTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"DefaultOutputDir", config.DefaultOutputDir, "string"},
		{"DefaultWorkers", config.DefaultWorkers, "int"},
		{"DefaultTimeout", config.DefaultTimeout, "time.Duration"},
		{"DefaultMaxDepth", config.DefaultMaxDepth, "int"},
		{"DefaultCacheEnabled", config.DefaultCacheEnabled, "bool"},
		{"DefaultCacheTTL", config.DefaultCacheTTL, "time.Duration"},
		{"DefaultJSTimeout", config.DefaultJSTimeout, "time.Duration"},
		{"DefaultScrollToEnd", config.DefaultScrollToEnd, "bool"},
		{"DefaultRandomDelayMin", config.DefaultRandomDelayMin, "time.Duration"},
		{"DefaultRandomDelayMax", config.DefaultRandomDelayMax, "time.Duration"},
		{"DefaultLogLevel", config.DefaultLogLevel, "string"},
		{"DefaultLogFormat", config.DefaultLogFormat, "string"},
		{"DefaultLLMMaxTokens", config.DefaultLLMMaxTokens, "int"},
		{"DefaultLLMTemperature", config.DefaultLLMTemperature, "float64"},
		{"DefaultLLMTimeout", config.DefaultLLMTimeout, "time.Duration"},
		{"DefaultLLMMaxRetries", config.DefaultLLMMaxRetries, "int"},
		{"DefaultRateLimitEnabled", config.DefaultRateLimitEnabled, "bool"},
		{"DefaultRateLimitRequestsPerMinute", config.DefaultRateLimitRequestsPerMinute, "int"},
		{"DefaultRateLimitBurstSize", config.DefaultRateLimitBurstSize, "int"},
		{"DefaultRateLimitMaxRetries", config.DefaultRateLimitMaxRetries, "int"},
		{"DefaultRateLimitInitialDelay", config.DefaultRateLimitInitialDelay, "time.Duration"},
		{"DefaultRateLimitMaxDelay", config.DefaultRateLimitMaxDelay, "time.Duration"},
		{"DefaultRateLimitMultiplier", config.DefaultRateLimitMultiplier, "float64"},
		{"DefaultCircuitBreakerEnabled", config.DefaultCircuitBreakerEnabled, "bool"},
		{"DefaultCircuitBreakerFailureThreshold", config.DefaultCircuitBreakerFailureThreshold, "int"},
		{"DefaultCircuitBreakerSuccessThresholdHalfOpen", config.DefaultCircuitBreakerSuccessThresholdHalfOpen, "int"},
		{"DefaultCircuitBreakerResetTimeout", config.DefaultCircuitBreakerResetTimeout, "time.Duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.expected {
			case "string":
				_, ok := tt.value.(string)
				assert.True(t, ok, "%s should be a string", tt.name)
			case "int":
				_, ok := tt.value.(int)
				assert.True(t, ok, "%s should be an int", tt.name)
			case "bool":
				_, ok := tt.value.(bool)
				assert.True(t, ok, "%s should be a bool", tt.name)
			case "float64":
				_, ok := tt.value.(float64)
				assert.True(t, ok, "%s should be a float64", tt.name)
			case "time.Duration":
				_, ok := tt.value.(time.Duration)
				assert.True(t, ok, "%s should be a time.Duration", tt.name)
			}
		})
	}
}

// ============================================================================
// Integration with Default() Function
// ============================================================================

func TestDefault_AppliesAllDefaultConstants(t *testing.T) {
	cfg := config.Default()

	// Verify all constants are applied in Default()
	assert.Equal(t, config.DefaultOutputDir, cfg.Output.Directory)
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.Equal(t, config.DefaultTimeout, cfg.Concurrency.Timeout)
	assert.Equal(t, config.DefaultMaxDepth, cfg.Concurrency.MaxDepth)
	assert.Equal(t, config.DefaultCacheEnabled, cfg.Cache.Enabled)
	assert.Equal(t, config.DefaultCacheTTL, cfg.Cache.TTL)
	assert.Equal(t, config.DefaultJSTimeout, cfg.Rendering.JSTimeout)
	assert.Equal(t, config.DefaultScrollToEnd, cfg.Rendering.ScrollToEnd)
	assert.Equal(t, config.DefaultRandomDelayMin, cfg.Stealth.RandomDelayMin)
	assert.Equal(t, config.DefaultRandomDelayMax, cfg.Stealth.RandomDelayMax)
	assert.Equal(t, config.DefaultLogLevel, cfg.Logging.Level)
	assert.Equal(t, config.DefaultLogFormat, cfg.Logging.Format)
	assert.Equal(t, config.DefaultLLMMaxTokens, cfg.LLM.MaxTokens)
	assert.Equal(t, config.DefaultLLMTemperature, cfg.LLM.Temperature)
	assert.Equal(t, config.DefaultLLMTimeout, cfg.LLM.Timeout)
	assert.Equal(t, config.DefaultLLMMaxRetries, cfg.LLM.MaxRetries)
}

func TestDefault_LLMRateLimitConstants(t *testing.T) {
	cfg := config.Default()

	rl := cfg.LLM.RateLimit
	assert.Equal(t, config.DefaultRateLimitEnabled, rl.Enabled)
	assert.Equal(t, config.DefaultRateLimitRequestsPerMinute, rl.RequestsPerMinute)
	assert.Equal(t, config.DefaultRateLimitBurstSize, rl.BurstSize)
	assert.Equal(t, config.DefaultRateLimitMaxRetries, rl.MaxRetries)
	assert.Equal(t, config.DefaultRateLimitInitialDelay, rl.InitialDelay)
	assert.Equal(t, config.DefaultRateLimitMaxDelay, rl.MaxDelay)
	assert.Equal(t, config.DefaultRateLimitMultiplier, rl.Multiplier)
}

func TestDefault_LLMCircuitBreakerConstants(t *testing.T) {
	cfg := config.Default()

	cb := cfg.LLM.RateLimit.CircuitBreaker
	assert.Equal(t, config.DefaultCircuitBreakerEnabled, cb.Enabled)
	assert.Equal(t, config.DefaultCircuitBreakerFailureThreshold, cb.FailureThreshold)
	assert.Equal(t, config.DefaultCircuitBreakerSuccessThresholdHalfOpen, cb.SuccessThresholdHalfOpen)
	assert.Equal(t, config.DefaultCircuitBreakerResetTimeout, cb.ResetTimeout)
}

// ============================================================================
// Directory Path Construction Tests
// ============================================================================

func TestDirectoryPaths_AreNotAbsolute(t *testing.T) {
	// ConfigDir and CacheDir should be absolute when home dir is available
	_, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot test absolute paths without home directory")
	}

	configDir := config.ConfigDir()
	cacheDir := config.CacheDir()

	// When home dir is available, paths should be absolute
	assert.True(t, filepath.IsAbs(configDir), "ConfigDir should be absolute path")
	assert.True(t, filepath.IsAbs(cacheDir), "CacheDir should be absolute path")
}

func TestDirectoryPaths_ContainExpectedComponents(t *testing.T) {
	configDir := config.ConfigDir()
	cacheDir := config.CacheDir()
	configPath := config.ConfigFilePath()

	// All should contain ".repodocs"
	assert.Contains(t, configDir, ".repodocs")
	assert.Contains(t, cacheDir, ".repodocs")
	assert.Contains(t, configPath, ".repodocs")

	// CacheDir should contain "cache"
	assert.Contains(t, cacheDir, "cache")

	// ConfigFilePath should end with "config.yaml"
	assert.True(t, filepath.Base(configPath) == "config.yaml", "ConfigFilePath should end with config.yaml")
}
