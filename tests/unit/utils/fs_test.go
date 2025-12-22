package utils_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath_TildePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := utils.ExpandPath("~/docs")
	expected := filepath.Join(home, "docs")
	assert.Equal(t, expected, result)
}

func TestExpandPath_JustTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := utils.ExpandPath("~")
	assert.Equal(t, home, result)
}

func TestExpandPath_AbsolutePath(t *testing.T) {
	path := "/usr/local/bin"
	result := utils.ExpandPath(path)
	assert.Equal(t, path, result)
}

func TestExpandPath_RelativePath(t *testing.T) {
	path := "./docs"
	result := utils.ExpandPath(path)
	assert.Equal(t, path, result)
}

func TestExpandPath_NestedTildePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := utils.ExpandPath("~/path/to/deep/dir")
	expected := filepath.Join(home, "path/to/deep/dir")
	assert.Equal(t, expected, result)
}

func TestExpandPath_TildeInMiddle(t *testing.T) {
	// Tilde in the middle should NOT be expanded
	path := "/path/to/~user/docs"
	result := utils.ExpandPath(path)
	assert.Equal(t, path, result)
}

func TestSanitizeFilename_InvalidCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"removes less than", "file<name", "file-name"},
		{"removes greater than", "file>name", "file-name"},
		{"removes colon", "file:name", "file-name"},
		{"removes quote", "file\"name", "file-name"},
		{"removes pipe", "file|name", "file-name"},
		{"removes question", "file?name", "file-name"},
		{"removes asterisk", "file*name", "file-name"},
		{"removes backslash", "file\\name", "file-name"},
		{"removes forward slash", "file/name", "file-name"},
		{"removes multiple invalid", "<>:\"|?*", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeFilename(tt.input)
			if tt.expected == "" {
				assert.Equal(t, "untitled", result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeFilename_WindowsReserved(t *testing.T) {
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM9", "LPT1", "LPT9"}
	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			result := utils.SanitizeFilename(name)
			assert.True(t, strings.HasPrefix(result, "_"))
		})
	}
}

func TestSanitizeFilename_MultipleSpaces(t *testing.T) {
	result := utils.SanitizeFilename("file   name")
	assert.Equal(t, "file-name", result)
}

func TestSanitizeFilename_LeadingTrailingDashes(t *testing.T) {
	result := utils.SanitizeFilename("--filename--")
	assert.Equal(t, "filename", result)
}

func TestSanitizeFilename_Empty(t *testing.T) {
	result := utils.SanitizeFilename("")
	assert.Equal(t, "untitled", result)
}

func TestSanitizeFilename_LongName(t *testing.T) {
	longName := strings.Repeat("a", 300)
	result := utils.SanitizeFilename(longName)
	assert.LessOrEqual(t, len(result), utils.MaxFilenameLength)
}

func TestURLToFilename(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"simple path", "https://example.com/docs/page", "docs-page.md"},
		{"root url", "https://example.com/", "index.md"},
		{"html extension", "https://example.com/page.html", "page.md"},
		{"php extension", "https://example.com/page.php", "page.md"},
		{"nested path", "https://example.com/a/b/c/d", "a-b-c-d.md"},
		{"invalid url fallback", "not a url", "not-a-url.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.URLToFilename(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLToPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"simple path", "https://example.com/docs/page", filepath.Join("docs", "page.md")},
		{"root url", "https://example.com/", "index.md"},
		{"html extension", "https://example.com/docs/page.html", filepath.Join("docs", "page.md")},
		{"nested path", "https://example.com/a/b/c", filepath.Join("a", "b", "c.md")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.URLToPath(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePath(t *testing.T) {
	baseDir := "/output"

	t.Run("flat mode", func(t *testing.T) {
		result := utils.GeneratePath(baseDir, "https://example.com/docs/page", true)
		expected := filepath.Join(baseDir, "docs-page.md")
		assert.Equal(t, expected, result)
	})

	t.Run("nested mode", func(t *testing.T) {
		result := utils.GeneratePath(baseDir, "https://example.com/docs/page", false)
		expected := filepath.Join(baseDir, "docs", "page.md")
		assert.Equal(t, expected, result)
	})
}

func TestGeneratePathFromRelative(t *testing.T) {
	baseDir := "/output"

	t.Run("flat mode", func(t *testing.T) {
		result := utils.GeneratePathFromRelative(baseDir, "docs/README.md", true)
		expected := filepath.Join(baseDir, "README.md")
		assert.Equal(t, expected, result)
	})

	t.Run("nested mode", func(t *testing.T) {
		result := utils.GeneratePathFromRelative(baseDir, "docs/guide/intro", false)
		expected := filepath.Join(baseDir, "docs", "guide", "intro.md")
		assert.Equal(t, expected, result)
	})

	t.Run("preserves md extension", func(t *testing.T) {
		result := utils.GeneratePathFromRelative(baseDir, "README.md", false)
		expected := filepath.Join(baseDir, "README.md")
		assert.Equal(t, expected, result)
	})
}

func TestJSONPath(t *testing.T) {
	tests := []struct {
		name     string
		mdPath   string
		expected string
	}{
		{"simple", "file.md", "file.json"},
		{"with path", "/output/docs/page.md", "/output/docs/page.json"},
		{"nested", "a/b/c/doc.md", "a/b/c/doc.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.JSONPath(tt.mdPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		valid    bool
	}{
		{"valid simple", "filename.txt", true},
		{"valid with dash", "file-name.txt", true},
		{"valid with underscore", "file_name.txt", true},
		{"empty", "", false},
		{"dot only", ".", false},
		{"double dot", "..", false},
		{"contains colon", "file:name", false},
		{"contains question", "file?name", false},
		{"windows reserved CON", "CON", false},
		{"windows reserved NUL", "NUL.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsValidFilename(tt.filename)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-ensure-dir-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "subdir1", "subdir2", "file.txt")
		err = utils.EnsureDir(targetPath)
		require.NoError(t, err)

		expectedDir := filepath.Join(tmpDir, "subdir1", "subdir2")
		info, err := os.Stat(expectedDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("existing directory ok", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-ensure-dir-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "file.txt")
		err = utils.EnsureDir(targetPath)
		require.NoError(t, err)
	})
}
