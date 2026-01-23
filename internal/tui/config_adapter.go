package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
)

// ConfigValues holds form values that map to Config struct.
// Numeric and duration fields are stored as strings for form editing.
type ConfigValues struct {
	OutputDirectory string
	OutputFlat      bool
	OutputOverwrite bool
	JSONMetadata    bool

	Workers  string
	Timeout  string
	MaxDepth string

	CacheEnabled   bool
	CacheTTL       string
	CacheDirectory string

	ForceJS     bool
	JSTimeout   string
	ScrollToEnd bool

	UserAgent      string
	RandomDelayMin string
	RandomDelayMax string

	LogLevel  string
	LogFormat string

	LLMProvider        string
	LLMAPIKey          string
	LLMBaseURL         string
	LLMModel           string
	LLMMaxTokens       string
	LLMTemperature     string
	LLMTimeout         string
	LLMEnhanceMetadata bool

	ExcludePatterns string

	RateLimitEnabled           bool
	RateLimitRequestsPerMinute string
	RateLimitBurstSize         string
	RateLimitMaxRetries        string
	RateLimitInitialDelay      string
	RateLimitMaxDelay          string
	RateLimitMultiplier        string

	CircuitBreakerEnabled          bool
	CircuitBreakerFailureThreshold string
	CircuitBreakerSuccessThreshold string
	CircuitBreakerResetTimeout     string

	Exclude []string
}

// FromConfig converts a Config to ConfigValues for form editing
func FromConfig(cfg *config.Config) *ConfigValues {
	return &ConfigValues{
		OutputDirectory: cfg.Output.Directory,
		OutputFlat:      cfg.Output.Flat,
		OutputOverwrite: cfg.Output.Overwrite,
		JSONMetadata:    cfg.Output.JSONMetadata,

		Workers:  strconv.Itoa(cfg.Concurrency.Workers),
		Timeout:  formatDuration(cfg.Concurrency.Timeout),
		MaxDepth: strconv.Itoa(cfg.Concurrency.MaxDepth),

		CacheEnabled:   cfg.Cache.Enabled,
		CacheTTL:       formatDuration(cfg.Cache.TTL),
		CacheDirectory: cfg.Cache.Directory,

		ForceJS:     cfg.Rendering.ForceJS,
		JSTimeout:   formatDuration(cfg.Rendering.JSTimeout),
		ScrollToEnd: cfg.Rendering.ScrollToEnd,

		UserAgent:      cfg.Stealth.UserAgent,
		RandomDelayMin: formatDuration(cfg.Stealth.RandomDelayMin),
		RandomDelayMax: formatDuration(cfg.Stealth.RandomDelayMax),

		LogLevel:  cfg.Logging.Level,
		LogFormat: cfg.Logging.Format,

		LLMProvider:        cfg.LLM.Provider,
		LLMAPIKey:          cfg.LLM.APIKey,
		LLMBaseURL:         cfg.LLM.BaseURL,
		LLMModel:           cfg.LLM.Model,
		LLMMaxTokens:       strconv.Itoa(cfg.LLM.MaxTokens),
		LLMTemperature:     strconv.FormatFloat(cfg.LLM.Temperature, 'f', 2, 64),
		LLMTimeout:         formatDuration(cfg.LLM.Timeout),
		LLMEnhanceMetadata: cfg.LLM.EnhanceMetadata,

		ExcludePatterns: strings.Join(cfg.Exclude, "\n"),

		RateLimitEnabled:           cfg.LLM.RateLimit.Enabled,
		RateLimitRequestsPerMinute: strconv.Itoa(cfg.LLM.RateLimit.RequestsPerMinute),
		RateLimitBurstSize:         strconv.Itoa(cfg.LLM.RateLimit.BurstSize),
		RateLimitMaxRetries:        strconv.Itoa(cfg.LLM.RateLimit.MaxRetries),
		RateLimitInitialDelay:      formatDuration(cfg.LLM.RateLimit.InitialDelay),
		RateLimitMaxDelay:          formatDuration(cfg.LLM.RateLimit.MaxDelay),
		RateLimitMultiplier:        strconv.FormatFloat(cfg.LLM.RateLimit.Multiplier, 'f', 2, 64),

		CircuitBreakerEnabled:          cfg.LLM.RateLimit.CircuitBreaker.Enabled,
		CircuitBreakerFailureThreshold: strconv.Itoa(cfg.LLM.RateLimit.CircuitBreaker.FailureThreshold),
		CircuitBreakerSuccessThreshold: strconv.Itoa(cfg.LLM.RateLimit.CircuitBreaker.SuccessThresholdHalfOpen),
		CircuitBreakerResetTimeout:     formatDuration(cfg.LLM.RateLimit.CircuitBreaker.ResetTimeout),

		Exclude: cfg.Exclude,
	}
}

// ToConfig converts ConfigValues back to a Config struct
func (v *ConfigValues) ToConfig() (*config.Config, error) {
	workers, err := parseIntOrDefault(v.Workers, config.DefaultWorkers)
	if err != nil {
		return nil, fmt.Errorf("invalid workers: %w", err)
	}

	maxDepth, err := parseIntOrDefault(v.MaxDepth, config.DefaultMaxDepth)
	if err != nil {
		return nil, fmt.Errorf("invalid max_depth: %w", err)
	}

	timeout, err := parseDurationOrDefault(v.Timeout, config.DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %w", err)
	}

	cacheTTL, err := parseDurationOrDefault(v.CacheTTL, config.DefaultCacheTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache_ttl: %w", err)
	}

	jsTimeout, err := parseDurationOrDefault(v.JSTimeout, config.DefaultJSTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid js_timeout: %w", err)
	}

	delayMin, err := parseDurationOrDefault(v.RandomDelayMin, config.DefaultRandomDelayMin)
	if err != nil {
		return nil, fmt.Errorf("invalid random_delay_min: %w", err)
	}

	delayMax, err := parseDurationOrDefault(v.RandomDelayMax, config.DefaultRandomDelayMax)
	if err != nil {
		return nil, fmt.Errorf("invalid random_delay_max: %w", err)
	}

	llmMaxTokens, err := parseIntOrDefault(v.LLMMaxTokens, config.DefaultLLMMaxTokens)
	if err != nil {
		return nil, fmt.Errorf("invalid llm_max_tokens: %w", err)
	}

	llmTemperature, err := parseFloatOrDefault(v.LLMTemperature, config.DefaultLLMTemperature)
	if err != nil {
		return nil, fmt.Errorf("invalid llm_temperature: %w", err)
	}

	llmTimeout, err := parseDurationOrDefault(v.LLMTimeout, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("invalid llm_timeout: %w", err)
	}

	rateLimitRequestsPerMinute, err := parseIntOrDefault(v.RateLimitRequestsPerMinute, config.DefaultRateLimitRequestsPerMinute)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_requests_per_minute: %w", err)
	}

	rateLimitBurstSize, err := parseIntOrDefault(v.RateLimitBurstSize, config.DefaultRateLimitBurstSize)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_burst_size: %w", err)
	}

	rateLimitMaxRetries, err := parseIntOrDefault(v.RateLimitMaxRetries, config.DefaultRateLimitMaxRetries)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_max_retries: %w", err)
	}

	rateLimitInitialDelay, err := parseDurationOrDefault(v.RateLimitInitialDelay, config.DefaultRateLimitInitialDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_initial_delay: %w", err)
	}

	rateLimitMaxDelay, err := parseDurationOrDefault(v.RateLimitMaxDelay, config.DefaultRateLimitMaxDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_max_delay: %w", err)
	}

	rateLimitMultiplier, err := parseFloatOrDefault(v.RateLimitMultiplier, config.DefaultRateLimitMultiplier)
	if err != nil {
		return nil, fmt.Errorf("invalid rate_limit_multiplier: %w", err)
	}

	circuitBreakerFailureThreshold, err := parseIntOrDefault(v.CircuitBreakerFailureThreshold, config.DefaultCircuitBreakerFailureThreshold)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit_breaker_failure_threshold: %w", err)
	}

	circuitBreakerSuccessThreshold, err := parseIntOrDefault(v.CircuitBreakerSuccessThreshold, config.DefaultCircuitBreakerSuccessThresholdHalfOpen)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit_breaker_success_threshold: %w", err)
	}

	circuitBreakerResetTimeout, err := parseDurationOrDefault(v.CircuitBreakerResetTimeout, config.DefaultCircuitBreakerResetTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid circuit_breaker_reset_timeout: %w", err)
	}

	excludePatterns := strings.Split(v.ExcludePatterns, "\n")
	var excludeList []string
	for _, pattern := range excludePatterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed != "" {
			excludeList = append(excludeList, trimmed)
		}
	}

	cfg := &config.Config{
		Output: config.OutputConfig{
			Directory:    v.OutputDirectory,
			Flat:         v.OutputFlat,
			Overwrite:    v.OutputOverwrite,
			JSONMetadata: v.JSONMetadata,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  workers,
			Timeout:  timeout,
			MaxDepth: maxDepth,
		},
		Cache: config.CacheConfig{
			Enabled:   v.CacheEnabled,
			TTL:       cacheTTL,
			Directory: v.CacheDirectory,
		},
		Rendering: config.RenderingConfig{
			ForceJS:     v.ForceJS,
			JSTimeout:   jsTimeout,
			ScrollToEnd: v.ScrollToEnd,
		},
		Stealth: config.StealthConfig{
			UserAgent:      v.UserAgent,
			RandomDelayMin: delayMin,
			RandomDelayMax: delayMax,
		},
		Logging: config.LoggingConfig{
			Level:  v.LogLevel,
			Format: v.LogFormat,
		},
		LLM: config.LLMConfig{
			Provider:        v.LLMProvider,
			APIKey:          v.LLMAPIKey,
			BaseURL:         v.LLMBaseURL,
			Model:           v.LLMModel,
			MaxTokens:       llmMaxTokens,
			Temperature:     llmTemperature,
			Timeout:         llmTimeout,
			EnhanceMetadata: v.LLMEnhanceMetadata,
			RateLimit: config.RateLimitConfig{
				Enabled:           v.RateLimitEnabled,
				RequestsPerMinute: rateLimitRequestsPerMinute,
				BurstSize:         rateLimitBurstSize,
				MaxRetries:        rateLimitMaxRetries,
				InitialDelay:      rateLimitInitialDelay,
				MaxDelay:          rateLimitMaxDelay,
				Multiplier:        rateLimitMultiplier,
				CircuitBreaker: config.CircuitBreakerConfig{
					Enabled:                  v.CircuitBreakerEnabled,
					FailureThreshold:         circuitBreakerFailureThreshold,
					SuccessThresholdHalfOpen: circuitBreakerSuccessThreshold,
					ResetTimeout:             circuitBreakerResetTimeout,
				},
			},
		},
		Exclude: excludeList,
	}

	return cfg, nil
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func parseDurationOrDefault(s string, defaultVal time.Duration) (time.Duration, error) {
	if s == "" {
		return defaultVal, nil
	}
	return time.ParseDuration(s)
}

func parseIntOrDefault(s string, defaultVal int) (int, error) {
	if s == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(s)
}

func parseFloatOrDefault(s string, defaultVal float64) (float64, error) {
	if s == "" {
		return defaultVal, nil
	}
	return strconv.ParseFloat(s, 64)
}
