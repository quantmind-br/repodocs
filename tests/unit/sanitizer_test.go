package app_test

import (
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizer_RemoveScripts(t *testing.T) {
	html := `
	<html>
	<body>
		<h1>Title</h1>
		<script>console.log("malicious");</script>
		<p>Content</p>
		<script type="text/javascript">
			document.write("injected");
		</script>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "console.log")
	assert.NotContains(t, result, "malicious")
	assert.NotContains(t, result, "document.write")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

func TestSanitizer_RemoveStyles(t *testing.T) {
	html := `
	<html>
	<head>
		<style>
			body { font-size: 14px; }
		</style>
	</head>
	<body>
		<h1>Title</h1>
		<style>.hidden { display: none; }</style>
		<p>Content</p>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "font-size")
	assert.NotContains(t, result, ".hidden")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

func TestSanitizer_RemoveIframes(t *testing.T) {
	html := `
	<html>
	<body>
		<h1>Title</h1>
		<iframe src="https://ads.example.com"></iframe>
		<p>Content</p>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "iframe")
	assert.NotContains(t, result, "ads.example.com")
	assert.Contains(t, result, "Title")
}

func TestSanitizer_RemoveNoscript(t *testing.T) {
	html := `
	<html>
	<body>
		<h1>Title</h1>
		<noscript>JavaScript is required</noscript>
		<p>Content</p>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "noscript")
	assert.NotContains(t, result, "JavaScript is required")
	assert.Contains(t, result, "Title")
}

func TestSanitizer_RemoveNavigation(t *testing.T) {
	html := `
	<html>
	<body>
		<nav>
			<a href="/">Home</a>
			<a href="/about">About</a>
		</nav>
		<main>
			<h1>Main Content</h1>
			<p>The real content here.</p>
		</main>
		<footer>
			<p>Copyright 2024</p>
		</footer>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "<nav>")
	assert.NotContains(t, result, "<footer>")
	assert.Contains(t, result, "Main Content")
}

func TestSanitizer_NormalizeURLs(t *testing.T) {
	html := `
	<html>
	<body>
		<a href="/docs/api">API Docs</a>
		<a href="./relative.html">Relative</a>
		<img src="/images/logo.png" alt="Logo">
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com/page/",
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.Contains(t, result, `href="https://example.com/docs/api"`)
	assert.Contains(t, result, `src="https://example.com/images/logo.png"`)
}

func TestSanitizer_RemoveHiddenElements(t *testing.T) {
	html := `
	<html>
	<body>
		<h1>Visible</h1>
		<div style="display:none">Hidden 1</div>
		<div style="display: none">Hidden 2</div>
		<div hidden>Hidden 3</div>
		<p>Also Visible</p>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "Hidden 1")
	assert.NotContains(t, result, "Hidden 2")
	assert.NotContains(t, result, "Hidden 3")
	assert.Contains(t, result, "Visible")
	assert.Contains(t, result, "Also Visible")
}

func TestSanitizer_RemoveByClass(t *testing.T) {
	html := `
	<html>
	<body>
		<div class="sidebar">Sidebar content</div>
		<article>
			<h1>Main Article</h1>
			<p>Article content</p>
		</article>
		<div class="advertisement">Ad content</div>
	</body>
	</html>
	`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "Sidebar content")
	assert.NotContains(t, result, "Ad content")
	assert.Contains(t, result, "Main Article")
}

// Tests for filename sanitization in utils package
func TestSanitizeFilename_SpecialChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file<name>.txt", "file-name-.txt"},
		{"file:name.txt", "file-name.txt"},
		{`file"name.txt`, "file-name.txt"},
		{"file|name.txt", "file-name.txt"},
		{"file?name.txt", "file-name.txt"},
		{"file*name.txt", "file-name.txt"},
		{"file\\name.txt", "file-name.txt"},
		{"file/name.txt", "file-name.txt"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := utils.SanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeFilename_WindowsReserved(t *testing.T) {
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "LPT1"}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			result := utils.SanitizeFilename(name)
			// Reserved names should be prefixed with underscore
			assert.True(t, strings.HasPrefix(result, "_"), "Expected prefix _ for: %s, got: %s", name, result)
		})
	}
}

func TestSanitizeFilename_LongNames(t *testing.T) {
	longName := strings.Repeat("a", 300)
	result := utils.SanitizeFilename(longName)
	assert.LessOrEqual(t, len(result), utils.MaxFilenameLength)
}

func TestSanitizeFilename_EmptyName(t *testing.T) {
	result := utils.SanitizeFilename("")
	assert.Equal(t, "untitled", result)
}

func TestSanitizeFilename_MultipleSpaces(t *testing.T) {
	result := utils.SanitizeFilename("file   name   here")
	assert.NotContains(t, result, "  ")
	assert.Equal(t, "file-name-here", result)
}

func TestURLToFilename(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/", "index.md"},
		{"https://example.com/docs/api", "docs-api.md"},
		{"https://example.com/docs/api.html", "docs-api.md"},
		{"https://example.com/docs/getting-started", "docs-getting-started.md"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := utils.URLToFilename(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestURLToPath(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/", "index.md"},
		{"https://example.com/docs/api", "docs/api.md"},
		{"https://example.com/docs/api.html", "docs/api.md"},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := utils.URLToPath(tc.url)
			// Normalize path separators for cross-platform testing
			expected := strings.ReplaceAll(tc.expected, "/", string([]rune{'/'}))
			result = strings.ReplaceAll(result, "\\", "/")
			assert.Equal(t, expected, result)
		})
	}
}

func TestIsValidFilename(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"valid-file.txt", true},
		{"file<name>.txt", false},
		{"CON", false},
		{"PRN.txt", false},
		{"", false},
		{".", false},
		{"..", false},
		{"normal.md", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.IsValidFilename(tc.name)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJSONPath(t *testing.T) {
	assert.Equal(t, "file.json", utils.JSONPath("file.md"))
	assert.Equal(t, "path/to/file.json", utils.JSONPath("path/to/file.md"))
}
