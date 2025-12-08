package utils

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger is a wrapper around zerolog.Logger
type Logger struct {
	zerolog.Logger
}

// LoggerOptions contains options for creating a logger
type LoggerOptions struct {
	Level   string
	Format  string // "pretty" or "json"
	Output  io.Writer
	Verbose bool
}

// NewLogger creates a new logger with the given options
func NewLogger(opts LoggerOptions) *Logger {
	var output io.Writer = os.Stderr
	if opts.Output != nil {
		output = opts.Output
	}

	// Set up pretty or JSON output
	if opts.Format == "pretty" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	// Parse log level
	level := parseLogLevel(opts.Level)
	if opts.Verbose {
		level = zerolog.DebugLevel
	}

	// Create logger
	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger}
}

// NewDefaultLogger creates a logger with default settings
func NewDefaultLogger() *Logger {
	return NewLogger(LoggerOptions{
		Level:  "info",
		Format: "pretty",
	})
}

// NewVerboseLogger creates a verbose logger
func NewVerboseLogger() *Logger {
	return NewLogger(LoggerOptions{
		Level:   "debug",
		Format:  "pretty",
		Verbose: true,
	})
}

// parseLogLevel parses a log level string
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithComponent returns a logger with a component field
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("component", component).Logger(),
	}
}

// WithURL returns a logger with a URL field
func (l *Logger) WithURL(url string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("url", url).Logger(),
	}
}

// WithStrategy returns a logger with a strategy field
func (l *Logger) WithStrategy(strategy string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("strategy", strategy).Logger(),
	}
}

// SetGlobalLevel sets the global log level
func SetGlobalLevel(level string) {
	zerolog.SetGlobalLevel(parseLogLevel(level))
}
