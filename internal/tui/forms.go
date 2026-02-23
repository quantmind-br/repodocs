package tui

import (
	"github.com/charmbracelet/huh"
)

func CreateOutputForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("directory").
				Title("Output Directory").
				Description("Where to save extracted documentation").
				Value(&values.OutputDirectory).
				Placeholder("./docs").
				CharLimit(256),

			huh.NewConfirm().
				Key("flat").
				Title("Flat Structure").
				Description("Save all files in a single directory (no subdirectories)").
				Value(&values.OutputFlat),

			huh.NewConfirm().
				Key("overwrite").
				Title("Overwrite Existing").
				Description("Overwrite existing files without prompting").
				Value(&values.OutputOverwrite),

			huh.NewConfirm().
				Key("json_metadata").
				Title("JSON Metadata").
				Description("Generate .json metadata files alongside markdown").
				Value(&values.JSONMetadata),
		),
	).WithTheme(GetTheme())
}

func CreateConcurrencyForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("workers").
				Title("Workers").
				Description("Number of concurrent workers (1-50)").
				Value(&values.Workers).
				Placeholder("5").
				CharLimit(3).
				Validate(ValidateIntRange(1, 50)),

			huh.NewInput().
				Key("timeout").
				Title("Request Timeout").
				Description("HTTP request timeout (e.g., 30s, 1m)").
				Value(&values.Timeout).
				Placeholder("30s").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewInput().
				Key("max_depth").
				Title("Max Crawl Depth").
				Description("Maximum depth for recursive crawling (1-100)").
				Value(&values.MaxDepth).
				Placeholder("4").
				CharLimit(3).
				Validate(ValidateIntRange(1, 100)),
		),
	).WithTheme(GetTheme())
}

func CreateCacheForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("enabled").
				Title("Enable Cache").
				Description("Cache fetched pages to reduce network requests").
				Value(&values.CacheEnabled),

			huh.NewInput().
				Key("ttl").
				Title("Cache TTL").
				Description("How long to keep cached pages (e.g., 24h, 7d)").
				Value(&values.CacheTTL).
				Placeholder("24h").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewInput().
				Key("directory").
				Title("Cache Directory").
				Description("Directory for cache storage").
				Value(&values.CacheDirectory).
				Placeholder("~/.repodocs/cache").
				CharLimit(256),
		),
	).WithTheme(GetTheme())
}

func CreateRenderingForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("force_js").
				Title("Force JavaScript Rendering").
				Description("Always render pages with headless browser").
				Value(&values.ForceJS),

			huh.NewInput().
				Key("js_timeout").
				Title("JS Timeout").
				Description("Timeout for JavaScript rendering (e.g., 10s, 30s)").
				Value(&values.JSTimeout).
				Placeholder("10s").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewConfirm().
				Key("scroll_to_end").
				Title("Scroll to End").
				Description("Scroll page to load lazy content").
				Value(&values.ScrollToEnd),
		),
	).WithTheme(GetTheme())
}

func CreateStealthForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("user_agent").
				Title("User Agent").
				Description("Custom User-Agent header (leave empty for default)").
				Value(&values.UserAgent).
				Placeholder("Mozilla/5.0...").
				CharLimit(256),

			huh.NewInput().
				Key("delay_min").
				Title("Min Random Delay").
				Description("Minimum delay between requests (e.g., 100ms, 1s)").
				Value(&values.RandomDelayMin).
				Placeholder("100ms").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewInput().
				Key("delay_max").
				Title("Max Random Delay").
				Description("Maximum delay between requests (e.g., 500ms, 2s)").
				Value(&values.RandomDelayMax).
				Placeholder("500ms").
				CharLimit(10).
				Validate(ValidateDuration),
		),
	).WithTheme(GetTheme())
}

func CreateLoggingForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("level").
				Title("Log Level").
				Description("Minimum log level to display").
				Options(
					huh.NewOption("Trace", "trace"),
					huh.NewOption("Debug", "debug"),
					huh.NewOption("Info", "info"),
					huh.NewOption("Warn", "warn"),
					huh.NewOption("Error", "error"),
				).
				Value(&values.LogLevel),

			huh.NewSelect[string]().
				Key("format").
				Title("Log Format").
				Description("Output format for logs").
				Options(
					huh.NewOption("Pretty (human-readable)", "pretty"),
					huh.NewOption("JSON (structured)", "json"),
					huh.NewOption("Text (plain)", "text"),
				).
				Value(&values.LogFormat),
		),
	).WithTheme(GetTheme())
}

func CreateLLMForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("provider").
				Title("LLM Provider").
				Description("AI provider for metadata enrichment").
				Options(
					huh.NewOption("None (disabled)", ""),
					huh.NewOption("OpenAI", "openai"),
					huh.NewOption("Anthropic", "anthropic"),
					huh.NewOption("Google", "google"),
					huh.NewOption("Ollama", "ollama"),
				).
				Value(&values.LLMProvider),

			huh.NewInput().
				Key("api_key").
				Title("API Key").
				Description("API key for the selected provider").
				Value(&values.LLMAPIKey).
				EchoMode(huh.EchoModePassword),

			huh.NewInput().
				Key("base_url").
				Title("Base URL").
				Description("Custom API endpoint (leave empty for default)").
				Value(&values.LLMBaseURL).
				Placeholder("https://api.openai.com/v1").
				CharLimit(256),

			huh.NewInput().
				Key("model").
				Title("Model").
				Description("Model name to use").
				Value(&values.LLMModel).
				Placeholder("gpt-4o-mini").
				CharLimit(64),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("max_tokens").
				Title("Max Tokens").
				Description("Maximum tokens for LLM response").
				Value(&values.LLMMaxTokens).
				Placeholder("1000").
				CharLimit(10).
				Validate(ValidatePositiveInt),

			huh.NewInput().
				Key("temperature").
				Title("Temperature").
				Description("Creativity level (0.0-2.0)").
				Value(&values.LLMTemperature).
				Placeholder("0.7").
				CharLimit(10).
				Validate(ValidateFloatRange(0, 2)),

			huh.NewInput().
				Key("timeout").
				Title("LLM Timeout").
				Description("Timeout for LLM requests").
				Value(&values.LLMTimeout).
				Placeholder("30s").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewConfirm().
				Key("enhance_metadata").
				Title("Enhance Metadata").
				Description("Use LLM to generate summaries and tags").
				Value(&values.LLMEnhanceMetadata),
		),
	).WithTheme(GetTheme())
}

func CreateExcludeForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Key("exclude_patterns").
				Title("Exclude Patterns").
				Description("Regex patterns to exclude (one per line)").
				Value(&values.ExcludePatterns).
				Placeholder(".*\\.pdf$\n.*/login.*"),
		),
	).WithTheme(GetTheme())
}

func CreateRateLimitForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("enabled").
				Title("Enable Rate Limiting").
				Description("Enable rate limiting for LLM API calls").
				Value(&values.RateLimitEnabled),

			huh.NewInput().
				Key("requests_per_minute").
				Title("Requests Per Minute").
				Description("Maximum requests per minute (1-1000)").
				Value(&values.RateLimitRequestsPerMinute).
				Placeholder("60").
				CharLimit(4).
				Validate(ValidateIntRange(1, 1000)),

			huh.NewInput().
				Key("burst_size").
				Title("Burst Size").
				Description("Maximum burst requests (1-100)").
				Value(&values.RateLimitBurstSize).
				Placeholder("10").
				CharLimit(3).
				Validate(ValidateIntRange(1, 100)),

			huh.NewInput().
				Key("max_retries").
				Title("Max Retries").
				Description("Maximum retry attempts (0-10)").
				Value(&values.RateLimitMaxRetries).
				Placeholder("3").
				CharLimit(2).
				Validate(ValidateIntRange(0, 10)),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("initial_delay").
				Title("Initial Delay").
				Description("Initial delay between retries (e.g., 1s)").
				Value(&values.RateLimitInitialDelay).
				Placeholder("1s").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewInput().
				Key("max_delay").
				Title("Max Delay").
				Description("Maximum delay between retries (e.g., 1m)").
				Value(&values.RateLimitMaxDelay).
				Placeholder("1m0s").
				CharLimit(10).
				Validate(ValidateDuration),

			huh.NewInput().
				Key("multiplier").
				Title("Backoff Multiplier").
				Description("Backoff multiplier (1.0-5.0)").
				Value(&values.RateLimitMultiplier).
				Placeholder("2.0").
				CharLimit(10).
				Validate(ValidateFloatRange(1.0, 5.0)),
		),
	).WithTheme(GetTheme())
}

func CreateCircuitBreakerForm(values *ConfigValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key("enabled").
				Title("Enable Circuit Breaker").
				Description("Enable circuit breaker for LLM API calls").
				Value(&values.CircuitBreakerEnabled),

			huh.NewInput().
				Key("failure_threshold").
				Title("Failure Threshold").
				Description("Failures before opening circuit (1-50)").
				Value(&values.CircuitBreakerFailureThreshold).
				Placeholder("5").
				CharLimit(2).
				Validate(ValidateIntRange(1, 50)),

			huh.NewInput().
				Key("success_threshold").
				Title("Success Threshold").
				Description("Successes in half-open to close (1-10)").
				Value(&values.CircuitBreakerSuccessThreshold).
				Placeholder("1").
				CharLimit(2).
				Validate(ValidateIntRange(1, 10)),

			huh.NewInput().
				Key("reset_timeout").
				Title("Reset Timeout").
				Description("Time before half-open state (e.g., 30s)").
				Value(&values.CircuitBreakerResetTimeout).
				Placeholder("30s").
				CharLimit(10).
				Validate(ValidateDuration),
		),
	).WithTheme(GetTheme())
}

func GetFormForCategory(category string, values *ConfigValues) *huh.Form {
	switch category {
	case "output":
		return CreateOutputForm(values)
	case "exclude":
		return CreateExcludeForm(values)
	case "concurrency":
		return CreateConcurrencyForm(values)
	case "cache":
		return CreateCacheForm(values)
	case "rendering":
		return CreateRenderingForm(values)
	case "stealth":
		return CreateStealthForm(values)
	case "logging":
		return CreateLoggingForm(values)
	case "llm", "llm_basic":
		return CreateLLMForm(values)
	case "llm_rate_limit":
		return CreateRateLimitForm(values)
	case "llm_circuit_breaker":
		return CreateCircuitBreakerForm(values)
	default:
		return nil
	}
}
