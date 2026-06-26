package config

import (
	"fmt"
	"net"
	"net/url"
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
	Proxy       ProxyConfig       `mapstructure:"proxy" yaml:"proxy"`
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
	JitterFactor      float64              `mapstructure:"jitter_factor" yaml:"jitter_factor"`
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
	// CDPEndpoint, when set, connects JS rendering to an external CDP browser
	// (e.g. CloakBrowser or Camoufox sidecar) instead of launching local Chrome.
	CDPEndpoint string `mapstructure:"cdp_endpoint" yaml:"cdp_endpoint"`
}

// StealthConfig contains stealth mode settings
type StealthConfig struct {
	UserAgent      string        `mapstructure:"user_agent" yaml:"user_agent"`
	RandomDelayMin time.Duration `mapstructure:"random_delay_min" yaml:"random_delay_min"`
	RandomDelayMax time.Duration `mapstructure:"random_delay_max" yaml:"random_delay_max"`
}

// ProxyConfig contains proxy settings applied to both HTTP fetching and JS
// rendering. A proxy can be configured either as a single fully-qualified URL
// (e.g. "socks5://user:pass@host:1080") or via the structured Type/Host/Port/
// Username/Password fields. When both are present, URL takes precedence.
//
// Supported schemes: http, https, socks5, socks5h. SOCKS5 supports username/
// password authentication for the HTTP fetcher. Note that headless Chrome (used
// for JS rendering) cannot authenticate SOCKS5 proxies — see the renderer docs.
type ProxyConfig struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled"`
	URL      string `mapstructure:"url" yaml:"url"`
	Type     string `mapstructure:"type" yaml:"type"`
	Host     string `mapstructure:"host" yaml:"host"`
	Port     int    `mapstructure:"port" yaml:"port"`
	Username string `mapstructure:"username" yaml:"username"`
	Password string `mapstructure:"password" yaml:"password"`
}

// supportedProxySchemes lists the proxy schemes accepted by both the HTTP
// fetcher (tls-client) and the JS renderer (headless Chrome).
var supportedProxySchemes = map[string]struct{}{
	"http":    {},
	"https":   {},
	"socks5":  {},
	"socks5h": {},
}

// normalizeProxyScheme lower-cases the scheme and applies sensible aliases.
// An empty scheme defaults to socks5, matching this feature's primary use case.
func normalizeProxyScheme(scheme string) string {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	switch scheme {
	case "":
		return "socks5"
	case "socks":
		return "socks5"
	default:
		return scheme
	}
}

// validateProxyScheme returns an error if scheme is not supported.
func validateProxyScheme(scheme string) error {
	if _, ok := supportedProxySchemes[scheme]; !ok {
		return fmt.Errorf("unsupported proxy scheme %q (supported: http, https, socks5, socks5h)", scheme)
	}
	return nil
}

// requireProxyPort enforces an explicit port for SOCKS5 proxies. Unlike http/
// https (where tls-client defaults to 80/443), the SOCKS5 dialer passes the
// host straight to net.Dial and fails with an opaque "missing port in address"
// at request time, so we reject it early. u.Hostname()/u.Port() exclude any
// userinfo, keeping credentials out of the error message.
func requireProxyPort(scheme string, u *url.URL) error {
	if (scheme == "socks5" || scheme == "socks5h") && u.Port() == "" {
		return fmt.Errorf("%s proxy requires an explicit port (host %q)", scheme, u.Hostname())
	}
	return nil
}

// Resolve builds the effective proxy URL from the configuration, including any
// credentials. It returns ("", nil) when the proxy is disabled. When the proxy
// is enabled but misconfigured (missing host, unsupported scheme, malformed
// URL) it returns a descriptive error.
func (p ProxyConfig) Resolve() (string, error) {
	if !p.Enabled {
		return "", nil
	}

	// Explicit URL form takes precedence.
	if raw := strings.TrimSpace(p.URL); raw != "" {
		u, err := url.Parse(raw)
		if err != nil {
			// Never echo the raw URL or wrap the url.Error: both may embed the password.
			return "", fmt.Errorf("proxy.url could not be parsed")
		}
		scheme := normalizeProxyScheme(u.Scheme)
		if err := validateProxyScheme(scheme); err != nil {
			return "", err
		}
		u.Scheme = scheme
		if u.Host == "" {
			return "", fmt.Errorf("invalid proxy.url: missing host in %q", u.Redacted())
		}
		// Allow credentials to be supplied separately from the URL.
		if u.User == nil && p.Username != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		}
		if err := requireProxyPort(scheme, u); err != nil {
			return "", err
		}
		return u.String(), nil
	}

	// Structured form.
	host := strings.TrimSpace(p.Host)
	if host == "" {
		return "", fmt.Errorf("proxy is enabled but neither proxy.url nor proxy.host is configured")
	}
	scheme := normalizeProxyScheme(p.Type)
	if err := validateProxyScheme(scheme); err != nil {
		return "", err
	}
	if p.Port > 0 {
		host = net.JoinHostPort(host, strconv.Itoa(p.Port))
	}
	u := &url.URL{Scheme: scheme, Host: host}
	if p.Username != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	if err := requireProxyPort(scheme, u); err != nil {
		return "", err
	}
	return u.String(), nil
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

	// Note: proxy configuration is intentionally validated lazily, at its point
	// of use (applyProxyFlag and NewOrchestrator both call Proxy.Resolve and
	// surface a descriptive error). Validating here would let a broken proxy in
	// the config file block every command — even an attempt to override it with
	// the --proxy flag, which runs after config load.

	// Validate rate limit configuration
	rl := &c.LLM.RateLimit
	if rl.Enabled {
		if rl.RequestsPerMinute < 0 {
			return fmt.Errorf("invalid llm.rate_limit.requests_per_minute: must be >= 0, got %d", rl.RequestsPerMinute)
		}
		if rl.BurstSize < 0 {
			return fmt.Errorf("invalid llm.rate_limit.burst_size: must be >= 0, got %d", rl.BurstSize)
		}
		if rl.MaxRetries < 0 {
			return fmt.Errorf("invalid llm.rate_limit.max_retries: must be >= 0, got %d", rl.MaxRetries)
		}
		if rl.InitialDelay < 0 {
			return fmt.Errorf("invalid llm.rate_limit.initial_delay: must be >= 0, got %s", rl.InitialDelay)
		}
		if rl.MaxDelay < 0 {
			return fmt.Errorf("invalid llm.rate_limit.max_delay: must be >= 0, got %s", rl.MaxDelay)
		}
		if rl.Multiplier < 0 {
			return fmt.Errorf("invalid llm.rate_limit.multiplier: must be >= 0, got %f", rl.Multiplier)
		}
		if rl.JitterFactor < 0 || rl.JitterFactor > 1.0 {
			return fmt.Errorf("invalid llm.rate_limit.jitter_factor: must be between 0.0 and 1.0, got %f", rl.JitterFactor)
		}

		// Validate circuit breaker configuration
		cb := &rl.CircuitBreaker
		if cb.Enabled {
			if cb.FailureThreshold < 1 {
				return fmt.Errorf("invalid llm.rate_limit.circuit_breaker.failure_threshold: must be >= 1, got %d", cb.FailureThreshold)
			}
			if cb.SuccessThresholdHalfOpen < 1 {
				return fmt.Errorf("invalid llm.rate_limit.circuit_breaker.success_threshold_half_open: must be >= 1, got %d", cb.SuccessThresholdHalfOpen)
			}
			if cb.ResetTimeout < time.Second {
				return fmt.Errorf("invalid llm.rate_limit.circuit_breaker.reset_timeout: must be >= 1s, got %s", cb.ResetTimeout)
			}
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
