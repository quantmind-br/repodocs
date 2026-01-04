package utils_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger_DefaultOptions(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{})
	require.NotNil(t, logger)
}

func TestNewLogger_WithCustomOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})
	require.NotNil(t, logger)

	logger.Info().Msg("test message")
	assert.Contains(t, buf.String(), "test message")
}

func TestNewLogger_PrettyFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "pretty",
		Output: &buf,
	})
	require.NotNil(t, logger)

	logger.Info().Msg("pretty test")
	// Pretty format includes ANSI escape codes or human-readable output
	assert.NotEmpty(t, buf.String())
}

func TestNewLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})
	require.NotNil(t, logger)

	logger.Info().Msg("json test")
	// JSON format should contain JSON structure
	output := buf.String()
	assert.Contains(t, output, "{")
	assert.Contains(t, output, "json test")
}

func TestNewLogger_VerboseOverridesLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   "error", // Would normally filter out debug
		Format:  "json",
		Output:  &buf,
		Verbose: true, // Should enable debug level
	})
	require.NotNil(t, logger)

	logger.Debug().Msg("debug message")
	assert.Contains(t, buf.String(), "debug message")
}

func TestNewLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		logLevel  string
		shouldLog bool
	}{
		{"debug level logs debug", "debug", "debug", true},
		{"info level logs info", "info", "info", true},
		{"warn level logs warn", "warn", "warn", true},
		{"error level logs error", "error", "error", true},
		{"info level filters debug", "info", "debug", false},
		{"warn level filters info", "warn", "info", false},
		{"error level filters warn", "error", "warn", false},
		{"invalid level defaults to info", "invalid", "info", true},
		{"empty level defaults to info", "", "info", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := utils.NewLogger(utils.LoggerOptions{
				Level:  tt.level,
				Format: "json",
				Output: &buf,
			})
			require.NotNil(t, logger)

			// Log at the specified log level
			switch tt.logLevel {
			case "debug":
				logger.Debug().Msg("test")
			case "info":
				logger.Info().Msg("test")
			case "warn":
				logger.Warn().Msg("test")
			case "error":
				logger.Error().Msg("test")
			}

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String(), "expected message to be logged")
			} else {
				assert.Empty(t, buf.String(), "expected message to be filtered")
			}
		})
	}
}

func TestNewDefaultLogger(t *testing.T) {
	logger := utils.NewDefaultLogger()
	require.NotNil(t, logger)

	// Default logger should be at info level with pretty format
	// We can verify it was created successfully
}

func TestNewVerboseLogger(t *testing.T) {
	logger := utils.NewVerboseLogger()
	require.NotNil(t, logger)

	// Verbose logger should be at debug level
	// We can verify it was created successfully
}

func TestLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	componentLogger := logger.WithComponent("test-component")
	require.NotNil(t, componentLogger)

	componentLogger.Info().Msg("component message")
	output := buf.String()
	assert.Contains(t, output, "test-component")
	assert.Contains(t, output, "component message")
}

func TestLogger_WithURL(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	urlLogger := logger.WithURL("https://example.com/page")
	require.NotNil(t, urlLogger)

	urlLogger.Info().Msg("url message")
	output := buf.String()
	assert.Contains(t, output, "https://example.com/page")
	assert.Contains(t, output, "url message")
}

func TestLogger_WithStrategy(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	strategyLogger := logger.WithStrategy("crawler")
	require.NotNil(t, strategyLogger)

	strategyLogger.Info().Msg("strategy message")
	output := buf.String()
	assert.Contains(t, output, "crawler")
	assert.Contains(t, output, "strategy message")
}

func TestLogger_ChainedWith(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	chainedLogger := logger.
		WithComponent("comp").
		WithURL("http://test.com").
		WithStrategy("sitemap")
	require.NotNil(t, chainedLogger)

	chainedLogger.Info().Msg("chained message")
	output := buf.String()
	assert.Contains(t, output, "comp")
	assert.Contains(t, output, "http://test.com")
	assert.Contains(t, output, "sitemap")
	assert.Contains(t, output, "chained message")
}

func TestSetGlobalLevel(t *testing.T) {
	// Test that SetGlobalLevel doesn't panic and accepts various levels
	levels := []string{"debug", "info", "warn", "error", "invalid", ""}
	for _, level := range levels {
		t.Run("level_"+level, func(t *testing.T) {
			// Should not panic
			utils.SetGlobalLevel(level)
		})
	}
	// Reset to info for other tests
	utils.SetGlobalLevel("info")
}

func TestLogger_TimestampIncluded(t *testing.T) {
	var buf bytes.Buffer
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	logger.Info().Msg("timestamp test")
	output := buf.String()
	// JSON output should include timestamp field
	assert.True(t, strings.Contains(output, "time") || strings.Contains(output, "timestamp"),
		"expected timestamp in output: %s", output)
}
