package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Config represents the application configuration
type Config struct {
	Output      OutputConfig      `mapstructure:"output" yaml:"output"`
	Concurrency ConcurrencyConfig `mapstructure:"concurrency" yaml:"concurrency"`
	Cache       CacheConfig       `mapstructure:"cache" yaml:"cache"`
	Rendering   RenderingConfig   `mapstructure:"rendering" yaml:"rendering"`
	Stealth     StealthConfig     `mapstructure:"stealth" yaml:"stealth"`
	Exclude     []string          `mapstructure:"exclude" yaml:"exclude"`
	Logging     LoggingConfig     `mapstructure:"logging" yaml:"logging"`
	LLM         LLMConfig         `mapstructure:"llm" yaml:"llm"`
	Git         GitConfig         `mapstructure:"git" yaml:"git"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider        string          `mapstructure:"provider" yaml:"provider"`
	APIKey          string          `mapstructure:"api_key" yaml:"api_key"`
	BaseURL         string          `mapstructure:"base_url" yaml:"base_url"`
	Model           string          `mapstructure:"model" yaml:"model"`
	MaxTokens       int             `mapstructure:"max_tokens" yaml:"max_tokens"`
	Temperature     float64         `mapstructure:"temperature" yaml:"temperature"`
	Timeout         time.Duration   `mapstructure:"timeout" yaml:"timeout"`
	MaxRetries      int             `mapstructure:"max_retries" yaml:"max_retries"` // Deprecated: use RateLimit.MaxRetries
	EnhanceMetadata bool            `mapstructure:"enhance_metadata" yaml:"enhance_metadata"`
	RateLimit       RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
}

// RateLimitConfig contains rate limiting settings for LLM requests
type RateLimitConfig struct {
	Enabled           bool                 `mapstructure:"enabled" yaml:"enabled"`
	RequestsPerMinute int                  `mapstructure:"requests_per_minute" yaml:"requests_per_minute"`
	BurstSize         int                  `mapstructure:"burst_size" yaml:"burst_size"`
	MaxRetries        int                  `mapstructure:"max_retries" yaml:"max_retries"`
	InitialDelay      time.Duration        `mapstructure:"initial_delay" yaml:"initial_delay"`
	MaxDelay          time.Duration        `mapstructure:"max_delay" yaml:"max_delay"`
	Multiplier        float64              `mapstructure:"multiplier" yaml:"multiplier"`
	CircuitBreaker    CircuitBreakerConfig `mapstructure:"circuit_breaker" yaml:"circuit_breaker"`
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled                  bool          `mapstructure:"enabled" yaml:"enabled"`
	FailureThreshold         int           `mapstructure:"failure_threshold" yaml:"failure_threshold"`
	SuccessThresholdHalfOpen int           `mapstructure:"success_threshold_half_open" yaml:"success_threshold_half_open"`
	ResetTimeout             time.Duration `mapstructure:"reset_timeout" yaml:"reset_timeout"`
}

// OutputConfig contains output-related settings
type OutputConfig struct {
	Directory    string `mapstructure:"directory" yaml:"directory"`
	Flat         bool   `mapstructure:"flat" yaml:"flat"`
	JSONMetadata bool   `mapstructure:"json_metadata" yaml:"json_metadata"`
	Overwrite    bool   `mapstructure:"overwrite" yaml:"overwrite"`
}

// ConcurrencyConfig contains concurrency settings
type ConcurrencyConfig struct {
	Workers  int           `mapstructure:"workers" yaml:"workers"`
	Timeout  time.Duration `mapstructure:"timeout" yaml:"timeout"`
	MaxDepth int           `mapstructure:"max_depth" yaml:"max_depth"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Enabled   bool          `mapstructure:"enabled" yaml:"enabled"`
	TTL       time.Duration `mapstructure:"ttl" yaml:"ttl"`
	Directory string        `mapstructure:"directory" yaml:"directory"`
}

// RenderingConfig contains JavaScript rendering settings
type RenderingConfig struct {
	ForceJS     bool          `mapstructure:"force_js" yaml:"force_js"`
	JSTimeout   time.Duration `mapstructure:"js_timeout" yaml:"js_timeout"`
	ScrollToEnd bool          `mapstructure:"scroll_to_end" yaml:"scroll_to_end"`
}

// StealthConfig contains stealth mode settings
type StealthConfig struct {
	UserAgent      string        `mapstructure:"user_agent" yaml:"user_agent"`
	RandomDelayMin time.Duration `mapstructure:"random_delay_min" yaml:"random_delay_min"`
	RandomDelayMax time.Duration `mapstructure:"random_delay_max" yaml:"random_delay_max"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level"`
	Format string `mapstructure:"format" yaml:"format"`
}

// GitConfig contains git strategy settings
type GitConfig struct {
	MaxFileSize string `mapstructure:"max_file_size" yaml:"max_file_size"`
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
	if c.Git.MaxFileSize == "" {
		c.Git.MaxFileSize = DefaultGitMaxFileSize
	} else {
		if _, err := ParseSize(c.Git.MaxFileSize); err != nil {
			return fmt.Errorf("invalid git.max_file_size: %w", err)
		}
	}
	return nil
}

func ParseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var multiplier int64 = 1
	if strings.HasSuffix(s, "GB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	}

	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("no numeric value in size string")
	}

	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %w", err)
	}

	if n < 0 {
		return 0, fmt.Errorf("negative size not allowed")
	}

	return n * multiplier, nil
}
