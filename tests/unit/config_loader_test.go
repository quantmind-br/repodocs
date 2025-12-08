package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_SuccessWithConfigFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a minimal config file
	configContent := `
output:
  directory: "/custom/output"
concurrency:
  workers: 10
cache:
  enabled: false
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Use the current directory as config path (to pick up our config file)
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Load config
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify config was loaded
	assert.Equal(t, "/custom/output", cfg.Output.Directory)
	assert.Equal(t, 10, cfg.Concurrency.Workers)
	assert.False(t, cfg.Cache.Enabled)
}


func TestLoadWithViper_Success(t *testing.T) {
	// Clean environment variables
	os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create a minimal config file
	configContent := `
output:
  directory: "/custom/output"
concurrency:
  workers: 10
cache:
  enabled: false
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Use the current directory as config path (to pick up our config file)
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Load config with Viper
	cfg, v, err := config.LoadWithViper()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, v)

	// Verify config was loaded
	assert.Equal(t, "/custom/output", cfg.Output.Directory)
	assert.Equal(t, 10, cfg.Concurrency.Workers)
	assert.False(t, cfg.Cache.Enabled)
}

func TestLoadWithViper_FileNotFound(t *testing.T) {
	// Clean environment variables
	os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Use a directory without config file
	tmpDir := t.TempDir()

	// Use the temporary directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Load config with Viper (should succeed with defaults)
	cfg, v, err := config.LoadWithViper()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, v)

	// Should have default values
	assert.Equal(t, "./docs", cfg.Output.Directory)
	assert.True(t, cfg.Cache.Enabled)
}

func TestLoadWithViper_EnvironmentVariables(t *testing.T) {
	// Clean environment variables first
	os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Set environment variables
	os.Setenv("REPODOCS_OUTPUT_DIRECTORY", "/env/output")
	os.Setenv("REPODOCS_CONCURRENCY_WORKERS", "20")
	defer os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	defer os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Use a temporary directory
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Load config with Viper
	cfg, _, err := config.LoadWithViper()
	require.NoError(t, err)

	// Verify environment variables were used
	assert.Equal(t, "/env/output", cfg.Output.Directory)
	assert.Equal(t, 20, cfg.Concurrency.Workers)
}

func TestEnsureConfigDir_Success(t *testing.T) {
	// Ensure config directory
	err := config.EnsureConfigDir()
	require.NoError(t, err)

	// Verify directory was created (note: EnsureConfigDir uses ConfigDir() which is fixed)
	// So we just verify it doesn't error
	assert.DirExists(t, config.CacheDir())
}

func TestEnsureConfigDir_AlreadyExists(t *testing.T) {
	// Ensure config directory (should not fail if already exists)
	err := config.EnsureConfigDir()
	require.NoError(t, err)
}

func TestEnsureCacheDir_Success(t *testing.T) {
	// Ensure cache directory
	err := config.EnsureCacheDir()
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, config.CacheDir())
}

func TestEnsureCacheDir_AlreadyExists(t *testing.T) {
	// Ensure cache directory twice (should not fail)
	err := config.EnsureCacheDir()
	require.NoError(t, err)
}

