package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid filename",
			input:    "test-file.md",
			expected: "test-file.md",
		},
		{
			name:     "invalid characters",
			input:    "test:file<>?*.md",
			expected: "test-file-.md",
		},
		{
			name:     "multiple spaces and dashes",
			input:    "test--file  name.md",
			expected: "test-file-name.md",
		},
		{
			name:     "leading and trailing dashes",
			input:    "-test-file-.md",
			expected: "test-file.md",
		},
		{
			name:     "multiple spaces and dashes",
			input:    "test--file  name.md",
			expected: "test-file-name.md",
		},
		{
			name:     "leading and trailing dashes",
			input:    "-test-file-.md",
			expected: "test-file.md",
		},
		{
			name:     "Windows reserved name CON",
			input:    "CON.md",
			expected: "_CON.md",
		},
		{
			name:     "Windows reserved name PRN",
			input:    "PRN.txt",
			expected: "_PRN.txt",
		},
		{
			name:     "very long filename",
			input:    strings.Repeat("a", 250) + ".md",
			expected: strings.Repeat("a", 200-3) + ".md",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "untitled",
		},
		{
			name:     "only invalid characters",
			input:    "<>:\"|?*",
			expected: "untitled",
		},
		{
			name:     "path separators",
			input:    "test/file/name.md",
			expected: "test-file-name.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLToFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple URL",
			url:      "https://example.com/page",
			expected: "page.md",
		},
		{
			name:     "URL with path",
			url:      "https://example.com/docs/api",
			expected: "docs-api.md",
		},
		{
			name:     "URL with HTML extension",
			url:      "https://example.com/page.html",
			expected: "page.md",
		},
		{
			name:     "URL with MDX extension",
			url:      "https://example.com/docs/quickstart.mdx",
			expected: "docs-quickstart.md",
		},
		{
			name:     "root URL",
			url:      "https://example.com/",
			expected: "index.md",
		},
		{
			name:     "URL with query params",
			url:      "https://example.com/page?param=value",
			expected: "page.md",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "not-a-url.md",
		},
		{
			name:     "URL with special chars",
			url:      "https://example.com/page:1",
			expected: "page-1.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLToFilename(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple path",
			url:      "https://example.com/docs/api",
			expected: "docs/api.md",
		},
		{
			name:     "nested path",
			url:      "https://example.com/docs/api/v1/endpoints",
			expected: "docs/api/v1/endpoints.md",
		},
		{
			name:     "root URL",
			url:      "https://example.com/",
			expected: "index.md",
		},
		{
			name:     "URL with HTML extension",
			url:      "https://example.com/docs/page.html",
			expected: "docs/page.md",
		},
		{
			name:     "URL with MDX extension",
			url:      "https://example.com/docs/quickstart.mdx",
			expected: "docs/quickstart.md",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "not-a-url.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLToPath(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseDir  string
		url      string
		flat     bool
		expected string
	}{
		{
			name:     "flat mode",
			baseDir:  "/output",
			url:      "https://example.com/docs/api",
			flat:     true,
			expected: "/output/docs-api.md",
		},
		{
			name:     "nested mode",
			baseDir:  "/output",
			url:      "https://example.com/docs/api",
			flat:     false,
			expected: "/output/docs/api.md",
		},
		{
			name:     "root URL flat",
			baseDir:  "/output",
			url:      "https://example.com/",
			flat:     true,
			expected: "/output/index.md",
		},
		{
			name:     "root URL nested",
			baseDir:  "/output",
			url:      "https://example.com/",
			flat:     false,
			expected: "/output/index.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePath(tt.baseDir, tt.url, tt.flat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePathFromRelative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseDir  string
		relPath  string
		flat     bool
		expected string
	}{
		{
			name:     "flat mode with extension",
			baseDir:  "/output",
			relPath:  "docs/api.md",
			flat:     true,
			expected: "/output/docs-api.md",
		},
		{
			name:     "nested mode",
			baseDir:  "/output",
			relPath:  "docs/api.md",
			flat:     false,
			expected: "/output/docs/api.md",
		},
		{
			name:     "flat mode without extension",
			baseDir:  "/output",
			relPath:  "docs/api",
			flat:     true,
			expected: "/output/docs-api.md",
		},
		{
			name:     "nested mode with subdirs",
			baseDir:  "/output",
			relPath:  "src/docs/api/v1.md",
			flat:     false,
			expected: "/output/src/docs/api/v1.md",
		},
		{
			name:     "flat mode with special chars",
			baseDir:  "/output",
			relPath:  "docs/my file name.md",
			flat:     true,
			expected: "/output/docs-my-file-name.md",
		},
		{
			name:     "flat mode deep path",
			baseDir:  "/output",
			relPath:  "docs/developers/tools/memory.md",
			flat:     true,
			expected: "/output/docs-developers-tools-memory.md",
		},
		{
			name:     "flat mode root file",
			baseDir:  "/output",
			relPath:  "README.md",
			flat:     true,
			expected: "/output/README.md",
		},
		{
			name:     "flat mode with mdx extension",
			baseDir:  "/output",
			relPath:  "docs/guide/intro.mdx",
			flat:     true,
			expected: "/output/docs-guide-intro.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePathFromRelative(tt.baseDir, tt.relPath, tt.flat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mdPath   string
		expected string
	}{
		{
			name:     "standard markdown file",
			mdPath:   "/output/docs/api.md",
			expected: "/output/docs/api.json",
		},
		{
			name:     "root file",
			mdPath:   "/output/index.md",
			expected: "/output/index.json",
		},
		{
			name:     "no extension",
			mdPath:   "/output/docs/api",
			expected: "/output/docs/api.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONPath(tt.mdPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "valid filename",
			filename: "test.md",
			expected: true,
		},
		{
			name:     "invalid characters",
			filename: "test:file.md",
			expected: false,
		},
		{
			name:     "empty string",
			filename: "",
			expected: false,
		},
		{
			name:     "dot",
			filename: ".",
			expected: false,
		},
		{
			name:     "double dot",
			filename: "..",
			expected: false,
		},
		{
			name:     "Windows reserved name",
			filename: "CON",
			expected: false,
		},
		{
			name:     "control character",
			filename: "test\x00file.md",
			expected: false,
		},
		{
			name:     "valid with spaces",
			filename: "test file.md",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureDir(t *testing.T) {
	t.Parallel()

	t.Run("creates directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testPath := filepath.Join(tempDir, "subdir", "file.txt")

		err := EnsureDir(testPath)
		require.NoError(t, err)

		// Check that the directory was created
		info, err := os.Stat(filepath.Dir(testPath))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("existing directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testPath := filepath.Join(tempDir, "file.txt")

		err := EnsureDir(testPath)
		require.NoError(t, err)

		// Should not error if directory already exists
		err = EnsureDir(testPath)
		require.NoError(t, err)
	})
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "home directory with slash",
			input:    "~/test",
			expected: filepath.Join(os.Getenv("HOME"), "test"),
		},
		{
			name:     "home directory only",
			input:    "~",
			expected: os.Getenv("HOME"),
		},
		{
			name:     "regular path",
			input:    "/tmp/test",
			expected: "/tmp/test",
		},
		{
			name:     "relative path",
			input:    "./test",
			expected: "./test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
