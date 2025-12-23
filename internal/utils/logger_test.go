package utils

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {

	t.Run("default logger", func(t *testing.T) {
		logger := NewDefaultLogger()
		require.NotNil(t, logger)
	})

	t.Run("verbose logger", func(t *testing.T) {
		logger := NewVerboseLogger()
		require.NotNil(t, logger)
	})

	t.Run("custom output", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerOptions{
			Level:  "info",
			Format: "json",
			Output: &buf,
		})
		require.NotNil(t, logger)
		logger.Info().Msg("test")
		assert.Contains(t, buf.String(), "test")
	})

	t.Run("pretty format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerOptions{
			Level:  "info",
			Format: "pretty",
			Output: &buf,
		})
		require.NotNil(t, logger)
		logger.Info().Msg("test")
		assert.Contains(t, buf.String(), "test")
	})

	t.Run("verbose option", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerOptions{
			Level:   "info",
			Format:  "json",
			Output:  &buf,
			Verbose: true,
		})
		require.NotNil(t, logger)
		// Verbose should enable debug level
		logger.Debug().Msg("debug test")
		assert.Contains(t, buf.String(), "debug test")
	})
}

func TestLoggerWithComponent(t *testing.T) {

	var buf bytes.Buffer
	logger := NewLogger(LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	componentLogger := logger.WithComponent("fetcher")
	require.NotNil(t, componentLogger)

	componentLogger.Info().Msg("test message")
	output := buf.String()
	assert.Contains(t, output, "fetcher")
	assert.Contains(t, output, "test message")
}

func TestLoggerWithURL(t *testing.T) {

	var buf bytes.Buffer
	logger := NewLogger(LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	urlLogger := logger.WithURL("https://example.com")
	require.NotNil(t, urlLogger)

	urlLogger.Info().Msg("test message")
	output := buf.String()
	assert.Contains(t, output, "https://example.com")
	assert.Contains(t, output, "test message")
}

func TestLoggerWithStrategy(t *testing.T) {

	var buf bytes.Buffer
	logger := NewLogger(LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	strategyLogger := logger.WithStrategy("sitemap")
	require.NotNil(t, strategyLogger)

	strategyLogger.Info().Msg("test message")
	output := buf.String()
	assert.Contains(t, output, "sitemap")
	assert.Contains(t, output, "test message")
}

func TestLoggerLevels(t *testing.T) {

	tests := []struct {
		name      string
		level     string
		logFunc   func(*Logger)
		shouldLog bool
	}{
		{
			name:      "debug level logs debug",
			level:     "debug",
			logFunc:   func(l *Logger) { l.Debug().Msg("debug") },
			shouldLog: true,
		},
		{
			name:      "info level doesn't log debug",
			level:     "info",
			logFunc:   func(l *Logger) { l.Debug().Msg("debug") },
			shouldLog: false,
		},
		{
			name:      "info level logs info",
			level:     "info",
			logFunc:   func(l *Logger) { l.Info().Msg("info") },
			shouldLog: true,
		},
		{
			name:      "warn level logs warn",
			level:     "warn",
			logFunc:   func(l *Logger) { l.Warn().Msg("warn") },
			shouldLog: true,
		},
		{
			name:      "error level logs error",
			level:     "error",
			logFunc:   func(l *Logger) { l.Error().Msg("error") },
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(LoggerOptions{
				Level:  tt.level,
				Format: "json",
				Output: &buf,
			})

			tt.logFunc(logger)
			output := buf.String()

			if tt.shouldLog {
				assert.NotEmpty(t, output)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestSetGlobalLevel(t *testing.T) {

	// This test just ensures the function doesn't panic
	// The actual behavior depends on zerolog's global state
	assert.NotPanics(t, func() {
		SetGlobalLevel("debug")
		SetGlobalLevel("info")
		SetGlobalLevel("warn")
		SetGlobalLevel("error")
	})

	// Reset to info level to avoid affecting other tests
	SetGlobalLevel("info")
}

func TestParseLogLevel(t *testing.T) {

	// Test through NewLogger since parseLogLevel is not exported
	tests := []struct {
		level    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown", "info"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(LoggerOptions{
				Level:  tt.level,
				Format: "json",
				Output: &buf,
			})
			require.NotNil(t, logger)
			// Just verify it was created successfully
		})
	}
}

func TestLoggerChaining(t *testing.T) {

	var buf bytes.Buffer
	logger := NewLogger(LoggerOptions{
		Level:  "info",
		Format: "json",
		Output: &buf,
	})

	// Test chaining multiple With methods
	chained := logger.WithComponent("test").WithURL("https://example.com").WithStrategy("sitemap")
	require.NotNil(t, chained)

	chained.Info().Msg("chained test")
	output := buf.String()

	assert.Contains(t, output, "test")
	assert.Contains(t, output, "https://example.com")
	assert.Contains(t, output, "sitemap")
	assert.Contains(t, output, "chained test")
}

func TestLoggerOutputDefault(t *testing.T) {

	// Test that default output is stderr (we can't easily verify this,
	// but we can verify it doesn't panic)
	logger := NewLogger(LoggerOptions{
		Level:  "info",
		Format: "json",
	})
	require.NotNil(t, logger)
}
