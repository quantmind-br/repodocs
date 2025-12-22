package testutil

import (
	"io"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/rs/zerolog"
)

// NewTestLogger creates a test logger that writes to testing.T
func NewTestLogger(t *testing.T) *utils.Logger {
	t.Helper()

	// Create a logger that writes to the test output
	zlogger := zerolog.New(io.Discard).With().
		Timestamp().
		Str("test", t.Name()).
		Logger()

	return &utils.Logger{Logger: zlogger}
}

// NewNoOpLogger creates a logger that discards all output
func NewNoOpLogger() *zerolog.Logger {
	logger := zerolog.New(io.Discard)
	return &logger
}

// NewVerboseLogger creates a logger that writes to both test output and discards
func NewVerboseLogger(t *testing.T) *zerolog.Logger {
	t.Helper()

	// For verbose testing, we can add more detailed logging
	// Currently using no-op but can be enhanced to write to t.Log
	logger := zerolog.New(io.Discard).With().
		Timestamp().
		Str("test", t.Name()).
		Logger()

	return &logger
}
