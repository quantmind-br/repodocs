package tui

import (
	"strconv"

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
				Placeholder("./docs"),

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
	workersStr := strconv.Itoa(values.Workers)
	maxDepthStr := strconv.Itoa(values.MaxDepth)

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("workers").
				Title("Workers").
				Description("Number of concurrent workers (1-50)").
				Value(&workersStr).
				Placeholder("5").
				Validate(ValidateIntRange(1, 50)),

			huh.NewInput().
				Key("timeout").
				Title("Request Timeout").
				Description("HTTP request timeout (e.g., 30s, 1m)").
				Value(&values.Timeout).
				Placeholder("30s").
				Validate(ValidateDuration),

			huh.NewInput().
				Key("max_depth").
				Title("Max Crawl Depth").
				Description("Maximum depth for recursive crawling (1-100)").
				Value(&maxDepthStr).
				Placeholder("4").
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
				Validate(ValidateDuration),

			huh.NewInput().
				Key("directory").
				Title("Cache Directory").
				Description("Directory for cache storage").
				Value(&values.CacheDirectory).
				Placeholder("~/.repodocs/cache"),
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
				Placeholder("Mozilla/5.0..."),

			huh.NewInput().
				Key("delay_min").
				Title("Min Random Delay").
				Description("Minimum delay between requests (e.g., 100ms, 1s)").
				Value(&values.RandomDelayMin).
				Placeholder("100ms").
				Validate(ValidateDuration),

			huh.NewInput().
				Key("delay_max").
				Title("Max Random Delay").
				Description("Maximum delay between requests (e.g., 500ms, 2s)").
				Value(&values.RandomDelayMax).
				Placeholder("500ms").
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
	maxTokensStr := strconv.Itoa(values.LLMMaxTokens)
	tempStr := strconv.FormatFloat(values.LLMTemperature, 'f', 2, 64)

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
				Placeholder("https://api.openai.com/v1"),

			huh.NewInput().
				Key("model").
				Title("Model").
				Description("Model name to use").
				Value(&values.LLMModel).
				Placeholder("gpt-4o-mini"),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("max_tokens").
				Title("Max Tokens").
				Description("Maximum tokens for LLM response").
				Value(&maxTokensStr).
				Placeholder("1000").
				Validate(ValidatePositiveInt),

			huh.NewInput().
				Key("temperature").
				Title("Temperature").
				Description("Creativity level (0.0-2.0)").
				Value(&tempStr).
				Placeholder("0.7").
				Validate(ValidateFloatRange(0, 2)),

			huh.NewInput().
				Key("timeout").
				Title("LLM Timeout").
				Description("Timeout for LLM requests").
				Value(&values.LLMTimeout).
				Placeholder("30s").
				Validate(ValidateDuration),

			huh.NewConfirm().
				Key("enhance_metadata").
				Title("Enhance Metadata").
				Description("Use LLM to generate summaries and tags").
				Value(&values.LLMEnhanceMetadata),
		),
	).WithTheme(GetTheme())
}

func GetFormForCategory(category string, values *ConfigValues) *huh.Form {
	switch category {
	case "output":
		return CreateOutputForm(values)
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
	case "llm":
		return CreateLLMForm(values)
	default:
		return nil
	}
}
