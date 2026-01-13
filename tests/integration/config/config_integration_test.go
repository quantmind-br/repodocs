package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/config"
)

func TestConfigSaveAndLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	original := &config.Config{
		Output: config.OutputConfig{
			Directory:    "./output",
			Flat:         true,
			Overwrite:    true,
			JSONMetadata: true,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  10,
			Timeout:  60 * time.Second,
			MaxDepth: 5,
		},
		Cache: config.CacheConfig{
			Enabled:   true,
			TTL:       48 * time.Hour,
			Directory: "/tmp/cache",
		},
		Rendering: config.RenderingConfig{
			ForceJS:     true,
			JSTimeout:   30 * time.Second,
			ScrollToEnd: true,
		},
		Stealth: config.StealthConfig{
			UserAgent:      "TestAgent/1.0",
			RandomDelayMin: 100 * time.Millisecond,
			RandomDelayMax: 500 * time.Millisecond,
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
		LLM: config.LLMConfig{
			Provider:        "openai",
			Model:           "gpt-4",
			MaxTokens:       2000,
			Temperature:     0.5,
			EnhanceMetadata: true,
		},
		Exclude: []string{"*.pdf", "*.zip"},
	}

	err := config.SaveTo(original, configPath)
	require.NoError(t, err)

	_, err = os.Stat(configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "directory: ./output")
	assert.Contains(t, string(data), "workers: 10")
	assert.Contains(t, string(data), "provider: openai")
}

func TestConfigSaveCreatesDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dirs", "config.yaml")

	cfg := config.Default()
	err := config.SaveTo(cfg, nestedPath)
	require.NoError(t, err)

	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}

func TestConfigSavePermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := config.Default()
	err := config.SaveTo(cfg, configPath)
	require.NoError(t, err)

	info, err := os.Stat(configPath)
	require.NoError(t, err)

	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0644), mode)
}

func TestRoundTripWithAllFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	original := &config.Config{
		Output: config.OutputConfig{
			Directory:    "./docs",
			Flat:         false,
			Overwrite:    true,
			JSONMetadata: true,
		},
		Concurrency: config.ConcurrencyConfig{
			Workers:  8,
			Timeout:  45 * time.Second,
			MaxDepth: 6,
		},
		Cache: config.CacheConfig{
			Enabled:   true,
			TTL:       12 * time.Hour,
			Directory: "/custom/cache",
		},
		Rendering: config.RenderingConfig{
			ForceJS:     true,
			JSTimeout:   20 * time.Second,
			ScrollToEnd: false,
		},
		Stealth: config.StealthConfig{
			UserAgent:      "CustomAgent/2.0",
			RandomDelayMin: 200 * time.Millisecond,
			RandomDelayMax: 1 * time.Second,
		},
		Logging: config.LoggingConfig{
			Level:  "warn",
			Format: "pretty",
		},
		LLM: config.LLMConfig{
			Provider:        "anthropic",
			APIKey:          "sk-ant-test",
			BaseURL:         "https://api.anthropic.com",
			Model:           "claude-3-opus",
			MaxTokens:       4096,
			Temperature:     0.7,
			Timeout:         60 * time.Second,
			EnhanceMetadata: true,
		},
		Exclude: []string{"*.log", "*.tmp", "/admin/*"},
	}

	err := config.SaveTo(original, configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "directory: ./docs")
	assert.Contains(t, string(data), "workers: 8")
	assert.Contains(t, string(data), "timeout: 45s")
	assert.Contains(t, string(data), "ttl: 12h0m0s")
	assert.Contains(t, string(data), "provider: anthropic")
	assert.Contains(t, string(data), "model: claude-3-opus")
	assert.Contains(t, string(data), "level: warn")
}

func TestConfigSaveOverwrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg1 := config.Default()
	cfg1.Output.Directory = "./first"
	err := config.SaveTo(cfg1, configPath)
	require.NoError(t, err)

	cfg2 := config.Default()
	cfg2.Output.Directory = "./second"
	err = config.SaveTo(cfg2, configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "directory: ./second")
	assert.NotContains(t, string(data), "directory: ./first")
}

func TestConfigSaveEmptyExclude(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := config.Default()
	cfg.Exclude = nil

	err := config.SaveTo(cfg, configPath)
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "exclude: []")
}
