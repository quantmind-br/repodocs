package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// LoadFixture loads a test fixture file and returns its contents.
// The path is relative to the tests/fixtures directory.
// Usage:
//   data := helpers.LoadFixture(t, "git/sample-repo.tar.gz")
func LoadFixture(t *testing.T, path string) []byte {
	t.Helper()
	fixturePath := filepath.Join("tests", "fixtures", path)
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "Failed to load fixture: %s", fixturePath)
	return data
}

// LoadFixtureString loads a test fixture file and returns its contents as a string.
// The path is relative to the tests/fixtures directory.
// Usage:
//   content := helpers.LoadFixtureString(t, "pkggo/sample_page.html")
func LoadFixtureString(t *testing.T, path string) string {
	t.Helper()
	return string(LoadFixture(t, path))
}

// TempDir creates a temporary directory for testing.
// The directory will be automatically removed when the test completes.
// Usage:
//   dir := helpers.TempDir(t)
//   defer os.RemoveAll(dir)
func TempDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "repodocs-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

// TempFile creates a temporary file for testing with the given content.
// The file will be automatically removed when the test completes.
// Usage:
//   file := helpers.TempFile(t, "test content", "test.txt")
//   defer os.Remove(file.Name())
func TempFile(t *testing.T, content, pattern string) *os.File {
	t.Helper()
	tmpDir := TempDir(t)
	tmpFile, err := os.CreateTemp(tmpDir, pattern)
	require.NoError(t, err, "Failed to create temp file")

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err, "Failed to write temp file")

	err = tmpFile.Close()
	require.NoError(t, err, "Failed to close temp file")

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile
}

// CreateTestTarGz creates a test tar.gz file in memory with the given files.
// Returns the bytes of the tar.gz file.
// Usage:
//   files := []testFile{
//       {Name: "README.md", Content: "# Test"},
//       {Name: "docs/guide.md", Content: "# Guide"},
//   }
//   archiveData := helpers.CreateTestTarGz(t, files)
func CreateTestTarGz(t *testing.T, files []struct {
	Name    string
	Content string
}) []byte {
	t.Helper()
	// This is a simplified version. For production use, you'd want to use
	// the actual tar and gzip packages to create the archive.
	// For now, we'll use a mock implementation.
	require.Fail(t, "CreateTestTarGz not implemented - use createTestTarGz from git_archive_test.go")
	return nil
}
