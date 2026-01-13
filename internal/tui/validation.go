package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Validation error messages
var (
	ErrRequired      = errors.New("this field is required")
	ErrInvalidNumber = errors.New("must be a valid number")
	ErrPositiveInt   = errors.New("must be a positive integer")
	ErrInvalidRange  = errors.New("value out of valid range")
)

// ValidateRequired ensures a string value is not empty
func ValidateRequired(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrRequired
	}
	return nil
}

// ValidateDuration validates that a string can be parsed as a time.Duration
func ValidateDuration(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil // Empty is valid (will use default)
	}
	_, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format (use: 30s, 5m, 1h): %w", err)
	}
	return nil
}

// ValidatePositiveInt validates that a string represents a positive integer
func ValidatePositiveInt(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil // Empty is valid (will use default)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return ErrInvalidNumber
	}
	if n < 1 {
		return ErrPositiveInt
	}
	return nil
}

// ValidateIntRange validates that a string represents an integer within a range
func ValidateIntRange(min, max int) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return ErrInvalidNumber
		}
		if n < min || n > max {
			return fmt.Errorf("%w: must be between %d and %d", ErrInvalidRange, min, max)
		}
		return nil
	}
}

// ValidateFloat validates that a string represents a valid float64
func ValidateFloat(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	_, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("must be a valid decimal number")
	}
	return nil
}

// ValidateFloatRange validates that a string represents a float within a range
func ValidateFloatRange(min, max float64) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("must be a valid decimal number")
		}
		if n < min || n > max {
			return fmt.Errorf("%w: must be between %.2f and %.2f", ErrInvalidRange, min, max)
		}
		return nil
	}
}

// ValidateLogLevel validates log level values
func ValidateLogLevel(s string) error {
	validLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
		"panic": true,
	}
	if !validLevels[strings.ToLower(s)] {
		return fmt.Errorf("invalid log level: must be one of trace, debug, info, warn, error, fatal, panic")
	}
	return nil
}

// ValidateLogFormat validates log format values
func ValidateLogFormat(s string) error {
	validFormats := map[string]bool{
		"json":   true,
		"pretty": true,
		"text":   true,
	}
	if !validFormats[strings.ToLower(s)] {
		return fmt.Errorf("invalid log format: must be json, pretty, or text")
	}
	return nil
}

// ValidateLLMProvider validates LLM provider values
func ValidateLLMProvider(s string) error {
	if s == "" {
		return nil // Empty is valid (LLM disabled)
	}
	validProviders := map[string]bool{
		"openai":    true,
		"anthropic": true,
		"google":    true,
	}
	if !validProviders[strings.ToLower(s)] {
		return fmt.Errorf("invalid LLM provider: must be openai, anthropic, or google")
	}
	return nil
}
