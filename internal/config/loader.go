package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from file, environment, and defaults
// Uses the global viper instance to access CLI flag bindings
func Load() (*Config, error) {
	// Use global viper instance to get CLI flag bindings
	v := viper.GetViper()

	// Set defaults
	setDefaults(v)

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(ConfigDir())
	v.AddConfigPath(".")

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	// Environment variables (REPODOCS_*)
	v.SetEnvPrefix("REPODOCS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Validate and apply defaults for invalid values
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadWithViper loads configuration and returns the viper instance
// This is useful for merging CLI flags later
func LoadWithViper() (*Config, *viper.Viper, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(ConfigDir())
	v.AddConfigPath(".")

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, nil, err
		}
	}

	// Environment variables (REPODOCS_*)
	v.SetEnvPrefix("REPODOCS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, nil, err
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	return &cfg, v, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	// Output defaults
	v.SetDefault("output.directory", DefaultOutputDir)
	v.SetDefault("output.flat", false)
	v.SetDefault("output.json_metadata", false)
	v.SetDefault("output.overwrite", false)

	// Concurrency defaults
	v.SetDefault("concurrency.workers", DefaultWorkers)
	v.SetDefault("concurrency.timeout", DefaultTimeout)
	v.SetDefault("concurrency.max_depth", DefaultMaxDepth)

	// Cache defaults
	v.SetDefault("cache.enabled", DefaultCacheEnabled)
	v.SetDefault("cache.ttl", DefaultCacheTTL)
	v.SetDefault("cache.directory", CacheDir())

	// Rendering defaults
	v.SetDefault("rendering.force_js", false)
	v.SetDefault("rendering.js_timeout", DefaultJSTimeout)
	v.SetDefault("rendering.scroll_to_end", DefaultScrollToEnd)

	// Stealth defaults
	v.SetDefault("stealth.user_agent", "")
	v.SetDefault("stealth.random_delay_min", DefaultRandomDelayMin)
	v.SetDefault("stealth.random_delay_max", DefaultRandomDelayMax)

	// Exclude defaults
	v.SetDefault("exclude", DefaultExcludePatterns)

	// Logging defaults
	v.SetDefault("logging.level", DefaultLogLevel)
	v.SetDefault("logging.format", DefaultLogFormat)
}

// EnsureConfigDir creates the config directory if it doesn't exist
func EnsureConfigDir() error {
	dir := ConfigDir()
	return os.MkdirAll(dir, 0755)
}

// EnsureCacheDir creates the cache directory if it doesn't exist
func EnsureCacheDir() error {
	dir := CacheDir()
	return os.MkdirAll(dir, 0755)
}
