package config

import "time"

// Config represents the application configuration
type Config struct {
	Output      OutputConfig      `mapstructure:"output"`
	Concurrency ConcurrencyConfig `mapstructure:"concurrency"`
	Cache       CacheConfig       `mapstructure:"cache"`
	Rendering   RenderingConfig   `mapstructure:"rendering"`
	Stealth     StealthConfig     `mapstructure:"stealth"`
	Exclude     []string          `mapstructure:"exclude"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

// OutputConfig contains output-related settings
type OutputConfig struct {
	Directory    string `mapstructure:"directory"`
	Flat         bool   `mapstructure:"flat"`
	JSONMetadata bool   `mapstructure:"json_metadata"`
	Overwrite    bool   `mapstructure:"overwrite"`
}

// ConcurrencyConfig contains concurrency settings
type ConcurrencyConfig struct {
	Workers  int           `mapstructure:"workers"`
	Timeout  time.Duration `mapstructure:"timeout"`
	MaxDepth int           `mapstructure:"max_depth"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Enabled   bool          `mapstructure:"enabled"`
	TTL       time.Duration `mapstructure:"ttl"`
	Directory string        `mapstructure:"directory"`
}

// RenderingConfig contains JavaScript rendering settings
type RenderingConfig struct {
	ForceJS     bool          `mapstructure:"force_js"`
	JSTimeout   time.Duration `mapstructure:"js_timeout"`
	ScrollToEnd bool          `mapstructure:"scroll_to_end"`
}

// StealthConfig contains stealth mode settings
type StealthConfig struct {
	UserAgent      string        `mapstructure:"user_agent"`
	RandomDelayMin time.Duration `mapstructure:"random_delay_min"`
	RandomDelayMax time.Duration `mapstructure:"random_delay_max"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Concurrency.Workers < 1 {
		c.Concurrency.Workers = DefaultWorkers
	}
	if c.Concurrency.MaxDepth < 1 {
		c.Concurrency.MaxDepth = DefaultMaxDepth
	}
	if c.Concurrency.Timeout < time.Second {
		c.Concurrency.Timeout = DefaultTimeout
	}
	if c.Cache.TTL < time.Minute {
		c.Cache.TTL = DefaultCacheTTL
	}
	if c.Rendering.JSTimeout < time.Second {
		c.Rendering.JSTimeout = DefaultJSTimeout
	}
	return nil
}
