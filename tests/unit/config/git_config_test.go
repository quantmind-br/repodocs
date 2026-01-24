package config_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ParseSize Tests
// ============================================================================

func TestParseSize_ValidFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes only", "1024", 1024},
		{"kilobytes lowercase", "1kb", 1024},
		{"kilobytes uppercase", "1KB", 1024},
		{"megabytes lowercase", "10mb", 10 * 1024 * 1024},
		{"megabytes uppercase", "10MB", 10 * 1024 * 1024},
		{"megabytes mixed case", "10Mb", 10 * 1024 * 1024},
		{"gigabytes lowercase", "1gb", 1024 * 1024 * 1024},
		{"gigabytes uppercase", "1GB", 1024 * 1024 * 1024},
		{"with spaces", "  5MB  ", 5 * 1024 * 1024},
		{"2MB", "2MB", 2 * 1024 * 1024},
		{"100KB", "100KB", 100 * 1024},
		{"zero", "0", 0},
		{"zero MB", "0MB", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.ParseSize(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSize_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only suffix", "MB"},
		{"invalid suffix", "10TB"},
		{"negative number", "-10MB"},
		{"float number", "1.5MB"},
		{"letters only", "abc"},
		{"special chars", "10$MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.ParseSize(tt.input)
			assert.Error(t, err)
		})
	}
}

// ============================================================================
// GitConfig Tests
// ============================================================================

func TestGitConfig_DefaultMaxFileSize(t *testing.T) {
	cfg := config.Default()

	// Default should be 10MB
	assert.Equal(t, "10MB", cfg.Git.MaxFileSize)
}

func TestGitConfig_ParseDefaultMaxFileSize(t *testing.T) {
	cfg := config.Default()

	size, err := config.ParseSize(cfg.Git.MaxFileSize)
	require.NoError(t, err)
	assert.Equal(t, int64(10*1024*1024), size)
}

func TestGitConfig_CustomMaxFileSize(t *testing.T) {
	cfg := &config.Config{
		Git: config.GitConfig{
			MaxFileSize: "2MB",
		},
	}

	size, err := config.ParseSize(cfg.Git.MaxFileSize)
	require.NoError(t, err)
	assert.Equal(t, int64(2*1024*1024), size)
}

// ============================================================================
// Config.Validate Git Tests
// ============================================================================

func TestConfig_Validate_ValidGitMaxFileSize(t *testing.T) {
	cfg := config.Default()
	cfg.Git.MaxFileSize = "5MB"

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "5MB", cfg.Git.MaxFileSize)
}

func TestConfig_Validate_InvalidGitMaxFileSize(t *testing.T) {
	cfg := config.Default()
	cfg.Git.MaxFileSize = "invalid"

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git.max_file_size")
}

func TestConfig_Validate_EmptyGitMaxFileSize(t *testing.T) {
	cfg := config.Default()
	cfg.Git.MaxFileSize = ""

	err := cfg.Validate()
	assert.NoError(t, err)
	// Should default to 10MB
	assert.Equal(t, config.DefaultGitMaxFileSize, cfg.Git.MaxFileSize)
}

func TestConfig_Validate_ZeroGitMaxFileSize(t *testing.T) {
	cfg := config.Default()
	cfg.Git.MaxFileSize = "0"

	err := cfg.Validate()
	assert.NoError(t, err)
	// Zero is valid (disables size limit)
	assert.Equal(t, "0", cfg.Git.MaxFileSize)
}

// ============================================================================
// GitConfig Structure Tests
// ============================================================================

func TestGitConfig_Structure(t *testing.T) {
	cfg := config.GitConfig{
		MaxFileSize: "25MB",
	}

	assert.Equal(t, "25MB", cfg.MaxFileSize)
}
