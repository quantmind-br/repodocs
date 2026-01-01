package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ConfigFilePath Tests
// ============================================================================

func TestConfigFilePath_Default(t *testing.T) {
	// ConfigFilePath should return a path ending with config.yaml
	path := config.ConfigFilePath()

	assert.NotEmpty(t, path)
	assert.True(t, strings.HasSuffix(path, "config.yaml"))
	assert.Contains(t, path, ".repodocs")
}

func TestConfigFilePath_IsAbsolute(t *testing.T) {
	// ConfigFilePath should return an absolute path when home directory is available
	path := config.ConfigFilePath()

	// The path should be consistent with ConfigDir
	expectedBase := config.ConfigDir()
	expectedPath := filepath.Join(expectedBase, "config.yaml")

	assert.Equal(t, expectedPath, path)
}

func TestConfigFilePath_ContainsConfigDir(t *testing.T) {
	// ConfigFilePath should be within ConfigDir
	configFilePath := config.ConfigFilePath()
	configDir := config.ConfigDir()

	assert.True(t, strings.HasPrefix(configFilePath, configDir))
}

// ============================================================================
// ConfigDir Tests
// ============================================================================

func TestConfigDir_Default(t *testing.T) {
	// ConfigDir should return a path containing .repodocs
	dir := config.ConfigDir()

	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, ".repodocs")
}

func TestConfigDir_UsesHomeDirectory(t *testing.T) {
	// ConfigDir should use the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	dir := config.ConfigDir()
	expectedDir := filepath.Join(home, ".repodocs")

	assert.Equal(t, expectedDir, dir)
}

// ============================================================================
// CacheDir Tests
// ============================================================================

func TestCacheDir_Default(t *testing.T) {
	// CacheDir should return a path containing cache
	dir := config.CacheDir()

	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "cache")
}

func TestCacheDir_IsSubdirectoryOfConfigDir(t *testing.T) {
	// CacheDir should be a subdirectory of ConfigDir
	cacheDir := config.CacheDir()
	configDir := config.ConfigDir()

	expectedCacheDir := filepath.Join(configDir, "cache")
	assert.Equal(t, expectedCacheDir, cacheDir)
}

// ============================================================================
// Load Tests
// ============================================================================

func TestLoad_WithConfigFile(t *testing.T) {
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

func TestLoad_WithoutConfigFile(t *testing.T) {
	// Use a temporary directory without config file
	tmpDir := t.TempDir()

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Clean environment variables
	os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Note: Load() uses the global viper instance which may have state from previous tests
	// Use LoadWithViper() for isolated testing instead
	cfg, _, err := config.LoadWithViper()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have default values
	assert.Equal(t, "./docs", cfg.Output.Directory)
	assert.Equal(t, config.DefaultWorkers, cfg.Concurrency.Workers)
	assert.True(t, cfg.Cache.Enabled)
}

func TestLoad_WithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
output:
  directory: "/custom/output"
  invalid yaml content here
    - not properly indented
      workers: [broken
`
	require.NoError(t, os.WriteFile(configFile, []byte(invalidContent), 0644))

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Use LoadWithViper for isolated testing (Load uses global viper with state)
	_, _, err = config.LoadWithViper()
	assert.Error(t, err)
}

// ============================================================================
// LoadWithViper Tests
// ============================================================================

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

func TestLoadWithViper_InvalidYAML(t *testing.T) {
	// Clean environment variables
	os.Unsetenv("REPODOCS_OUTPUT_DIRECTORY")
	os.Unsetenv("REPODOCS_CONCURRENCY_WORKERS")

	// Create a temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create invalid YAML content
	invalidContent := `
output:
  directory: "/custom/output"
  invalid yaml content here
    - not properly indented
      workers: [broken
`
	require.NoError(t, os.WriteFile(configFile, []byte(invalidContent), 0644))

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalWd)

	require.NoError(t, os.Chdir(tmpDir))

	// Load config with Viper should fail
	_, _, err = config.LoadWithViper()
	assert.Error(t, err)
}

// ============================================================================
// EnsureConfigDir Tests
// ============================================================================

func TestEnsureConfigDir_Success(t *testing.T) {
	// Ensure config directory
	err := config.EnsureConfigDir()
	require.NoError(t, err)

	// Verify directory exists
	configDir := config.ConfigDir()
	assert.DirExists(t, configDir)
}

func TestEnsureConfigDir_Existing(t *testing.T) {
	// Ensure config directory exists first
	err := config.EnsureConfigDir()
	require.NoError(t, err)

	// Call again - should not fail when directory already exists
	err = config.EnsureConfigDir()
	require.NoError(t, err)

	// Verify directory still exists
	assert.DirExists(t, config.ConfigDir())
}

// ============================================================================
// EnsureCacheDir Tests
// ============================================================================

func TestEnsureCacheDir_Success(t *testing.T) {
	// Ensure cache directory
	err := config.EnsureCacheDir()
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, config.CacheDir())
}

func TestEnsureCacheDir_Existing(t *testing.T) {
	// Ensure cache directory exists first
	err := config.EnsureCacheDir()
	require.NoError(t, err)

	// Call again - should not fail when directory already exists
	err = config.EnsureCacheDir()
	require.NoError(t, err)

	// Verify directory still exists
	assert.DirExists(t, config.CacheDir())
}
