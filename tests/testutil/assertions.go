package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertDocumentContent asserts document has expected content
func AssertDocumentContent(t *testing.T, doc *domain.Document, expectedURL, expectedTitle, expectedContent string) {
	t.Helper()

	require.NotNil(t, doc)
	assert.Equal(t, expectedURL, doc.URL)
	assert.Equal(t, expectedTitle, doc.Title)
	assert.Equal(t, expectedContent, doc.Content)
}

// AssertDocumentMarkdown asserts document has expected markdown content
func AssertDocumentMarkdown(t *testing.T, doc *domain.Document, expectedMarkdown string) {
	t.Helper()

	require.NotNil(t, doc)
	assert.Equal(t, expectedMarkdown, doc.Content)
}

// AssertDocumentMetadata asserts document has expected metadata
func AssertDocumentMetadata(t *testing.T, doc *domain.Document, expectedDescription string, expectedWordCount int) {
	t.Helper()

	require.NotNil(t, doc)
	assert.Equal(t, expectedDescription, doc.Description)
	assert.Equal(t, expectedWordCount, doc.WordCount)
}

// AssertDocumentHasHeaders asserts document has expected headers map
func AssertDocumentHasHeaders(t *testing.T, doc *domain.Document, expectedHeaders map[string][]string) {
	t.Helper()

	require.NotNil(t, doc)
	assert.Equal(t, expectedHeaders, doc.Headers)
}

// AssertDocumentHasHeaderLevel asserts document has expected headers for a specific level
func AssertDocumentHasHeaderLevel(t *testing.T, doc *domain.Document, level string, expectedValues []string) {
	t.Helper()

	require.NotNil(t, doc)
	headers, ok := doc.Headers[level]
	require.True(t, ok, "Headers should contain level %s", level)
	assert.Equal(t, expectedValues, headers)
}

// AssertDocumentHasLinks asserts document has expected links
func AssertDocumentHasLinks(t *testing.T, doc *domain.Document, expectedLinks []string) {
	t.Helper()

	require.NotNil(t, doc)
	assert.Equal(t, expectedLinks, doc.Links)
}

// AssertFileExists asserts a file exists at the given path
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	assert.True(t, fileExists(path), "File should exist at %s", path)
}

// AssertFileNotExists asserts a file does not exist at the given path
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	assert.False(t, fileExists(path), "File should not exist at %s", path)
}

// AssertFileContains asserts a file contains expected content
func AssertFileContains(t *testing.T, path, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), expectedContent)
}

// AssertFileEquals asserts a file equals expected content
func AssertFileEquals(t *testing.T, path, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

// AssertDirExists asserts a directory exists
func AssertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "Path should be a directory: %s", path)
}

// AssertFilesInDir asserts expected number of files exist in directory
func AssertFilesInDir(t *testing.T, dirPath string, expectedCount int, pattern string) {
	t.Helper()

	if pattern == "" {
		pattern = "*"
	}

	files, err := filepath.Glob(filepath.Join(dirPath, pattern))
	require.NoError(t, err)
	assert.Equal(t, expectedCount, len(files), "Expected %d files in %s, got %d", expectedCount, dirPath, len(files))
}

// AssertMarkdownFileWithFrontmatter asserts a markdown file has proper frontmatter
func AssertMarkdownFileWithFrontmatter(t *testing.T, path, expectedTitle string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "---")
	assert.Contains(t, contentStr, "title:")
	assert.Contains(t, contentStr, expectedTitle)
}

// fileExists is a helper to check if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
