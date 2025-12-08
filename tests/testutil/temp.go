package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TempDir creates a temporary directory for testing
func TempDir(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "repodocs-test-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// TempSubDir creates a temporary subdirectory within a base directory
func TempSubDir(t *testing.T, baseDir string) string {
	t.Helper()

	subDir, err := os.MkdirTemp(baseDir, "sub-*")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(subDir)
	})

	return subDir
}

// CreateTempFile creates a temporary file with content
func CreateTempFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp(dir, filename)
	require.NoError(t, err)
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	return tmpFile.Name()
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(t *testing.T, path string) string {
	t.Helper()

	err := os.MkdirAll(path, 0755)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(path)
	})

	return path
}

// TempOutputDir creates a temporary directory structure for output testing
func TempOutputDir(t *testing.T) (baseDir, docsDir string) {
	t.Helper()

	baseDir = TempDir(t)
	docsDir = filepath.Join(baseDir, "docs")

	err := os.MkdirAll(docsDir, 0755)
	require.NoError(t, err)

	return baseDir, docsDir
}
