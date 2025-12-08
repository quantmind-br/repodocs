package unit

import (
	"bytes"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewDefaultLogger(t *testing.T) {
	logger := utils.NewDefaultLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
}

func TestNewVerboseLogger(t *testing.T) {
	logger := utils.NewVerboseLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, zerolog.DebugLevel, logger.GetLevel())
}

func TestLogger_WithComponent(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "info"})
	componentLogger := logger.WithComponent("test")
	assert.NotNil(t, componentLogger)
}

func TestLogger_WithURL(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "info"})
	urlLogger := logger.WithURL("https://example.com")
	assert.NotNil(t, urlLogger)
}

func TestLogger_WithStrategy(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "info"})
	strategyLogger := logger.WithStrategy("git")
	assert.NotNil(t, strategyLogger)
}

func TestSetGlobalLevel(t *testing.T) {
	// Save original level
	originalLevel := zerolog.GlobalLevel()

	// Test setting different levels
	utils.SetGlobalLevel("debug")
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	utils.SetGlobalLevel("info")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

	utils.SetGlobalLevel("warn")
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())

	utils.SetGlobalLevel("error")
	assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())

	// Restore original level
	zerolog.SetGlobalLevel(originalLevel)
}

func TestNewLogger(t *testing.T) {
	t.Run("with custom output", func(t *testing.T) {
		buf := &bytes.Buffer{}
		logger := utils.NewLogger(utils.LoggerOptions{
			Level:  "info",
			Format: "pretty",
			Output: buf,
		})
		assert.NotNil(t, logger)
		assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
	})

	t.Run("with json format", func(t *testing.T) {
		logger := utils.NewLogger(utils.LoggerOptions{
			Level:  "info",
			Format: "json",
		})
		assert.NotNil(t, logger)
		assert.Equal(t, zerolog.InfoLevel, logger.GetLevel())
	})

	t.Run("with different log levels", func(t *testing.T) {
		tests := []struct {
			level       string
			expected    zerolog.Level
		}{
			{"debug", zerolog.DebugLevel},
			{"info", zerolog.InfoLevel},
			{"warn", zerolog.WarnLevel},
			{"error", zerolog.ErrorLevel},
			{"invalid", zerolog.InfoLevel}, // default case
		}

		for _, tt := range tests {
			t.Run("level "+tt.level, func(t *testing.T) {
				logger := utils.NewLogger(utils.LoggerOptions{Level: tt.level})
				assert.Equal(t, tt.expected, logger.GetLevel())
			})
		}
	})
}
