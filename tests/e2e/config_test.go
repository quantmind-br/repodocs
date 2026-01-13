package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	cmd := exec.Command(cliBinary, "config", "init")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "config init failed: %s", string(output))

	configPath := filepath.Join(homeDir, ".repodocs", "config.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err, "config file not created")

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "output:")
	assert.Contains(t, string(data), "concurrency:")
	assert.Contains(t, string(data), "cache:")
}

func TestConfigInitExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configDir := filepath.Join(homeDir, ".repodocs")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configPath := filepath.Join(configDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("existing: true"), 0644))

	cmd := exec.Command(cliBinary, "config", "init")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	output, err := cmd.CombinedOutput()
	assert.Error(t, err, "config init should fail when file exists")
	assert.Contains(t, string(output), "already exists")
}

func TestConfigShow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configDir := filepath.Join(homeDir, ".repodocs")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	configContent := `output:
  directory: ./custom-docs
  flat: true
concurrency:
  workers: 8
cache:
  enabled: true
`
	configPath := filepath.Join(configDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cmd := exec.Command(cliBinary, "config", "show")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "config show failed: %s", string(output))

	assert.Contains(t, string(output), "directory:")
	assert.Contains(t, string(output), "workers:")
}

func TestConfigPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0755))

	cmd := exec.Command(cliBinary, "config", "path")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "config path failed: %s", string(output))

	result := strings.TrimSpace(string(output))
	assert.Contains(t, result, ".repodocs")
	assert.Contains(t, result, "config.yaml")
}

func TestConfigHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	cmd := exec.Command(cliBinary, "config", "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "config --help failed: %s", string(output))

	assert.Contains(t, string(output), "edit")
	assert.Contains(t, string(output), "show")
	assert.Contains(t, string(output), "init")
	assert.Contains(t, string(output), "path")
}

func TestConfigSubcommandHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	requireBinary(t)

	subcommands := []string{"edit", "show", "init", "path"}

	for _, sub := range subcommands {
		t.Run(sub, func(t *testing.T) {
			cmd := exec.Command(cliBinary, "config", sub, "--help")
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "config %s --help failed: %s", sub, string(output))
			assert.Contains(t, string(output), "Usage:")
		})
	}
}
