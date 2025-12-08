package config

import (
	"os"
	"path/filepath"
	"time"
)

// Default values
const (
	// Output defaults
	DefaultOutputDir = "./docs"

	// Concurrency defaults
	DefaultWorkers  = 5
	DefaultTimeout  = 30 * time.Second
	DefaultMaxDepth = 3

	// Cache defaults
	DefaultCacheEnabled = true
	DefaultCacheTTL     = 24 * time.Hour

	// Rendering defaults
	DefaultJSTimeout   = 60 * time.Second
	DefaultScrollToEnd = true

	// Stealth defaults
	DefaultRandomDelayMin = 1 * time.Second
	DefaultRandomDelayMax = 3 * time.Second

	// Logging defaults
	DefaultLogLevel  = "info"
	DefaultLogFormat = "pretty"
)

// Default exclude patterns
var DefaultExcludePatterns = []string{
	`.*\.pdf$`,
	`.*/login.*`,
	`.*/logout.*`,
	`.*/admin.*`,
	`.*/sign-in.*`,
	`.*/sign-up.*`,
}

// ConfigDir returns the config directory path
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".repodocs"
	}
	return filepath.Join(home, ".repodocs")
}

// CacheDir returns the cache directory path
func CacheDir() string {
	return filepath.Join(ConfigDir(), "cache")
}

// ConfigFilePath returns the config file path
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Output: OutputConfig{
			Directory:    DefaultOutputDir,
			Flat:         false,
			JSONMetadata: false,
			Overwrite:    false,
		},
		Concurrency: ConcurrencyConfig{
			Workers:  DefaultWorkers,
			Timeout:  DefaultTimeout,
			MaxDepth: DefaultMaxDepth,
		},
		Cache: CacheConfig{
			Enabled:   DefaultCacheEnabled,
			TTL:       DefaultCacheTTL,
			Directory: CacheDir(),
		},
		Rendering: RenderingConfig{
			ForceJS:     false,
			JSTimeout:   DefaultJSTimeout,
			ScrollToEnd: DefaultScrollToEnd,
		},
		Stealth: StealthConfig{
			UserAgent:      "",
			RandomDelayMin: DefaultRandomDelayMin,
			RandomDelayMax: DefaultRandomDelayMax,
		},
		Exclude: DefaultExcludePatterns,
		Logging: LoggingConfig{
			Level:  DefaultLogLevel,
			Format: DefaultLogFormat,
		},
	}
}
