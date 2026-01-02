package converter

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSanitizer tests creating a new sanitizer
func TestNewSanitizer(t *testing.T) {
	tests := []struct {
		name string
		opts SanitizerOptions
	}{
		{
			name: "full options",
			opts: SanitizerOptions{
				BaseURL:          "https://example.com",
				RemoveNavigation: true,
				RemoveComments:   true,
			},
		},
		{
			name: "minimal options",
			opts: SanitizerOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer(tt.opts)
			assert.NotNil(t, sanitizer)
			assert.Equal(t, tt.opts.BaseURL, sanitizer.baseURL)
			assert.Equal(t, tt.opts.RemoveNavigation, sanitizer.removeNavigation)
			assert.Equal(t, tt.opts.RemoveComments, sanitizer.removeComments)
		})
	}
}

// TestSanitizer_Sanitize tests HTML sanitization
func TestSanitizer_Sanitize(t *testing.T) {
	tests := []struct {
		name        string
		opts        SanitizerOptions
		input       string
		contains    []string
		notContains []string
		wantErr     bool
	}{
		{
			name: "remove script tags",
			opts: SanitizerOptions{},
			input: `<html><body><script>alert('xss')</script><p>Content</p></body></html>`,
			contains:    []string{"Content"},
			notContains: []string{"<script>", "alert"},
		},
		{
			name: "remove style tags",
			opts: SanitizerOptions{},
			input: `<html><head><style>body{color:red;}</style></head><body><p>Content</p></body></html>`,
			contains:    []string{"Content"},
			notContains: []string{"<style>", "color:red"},
		},
		{
			name: "remove navigation elements",
			opts: SanitizerOptions{RemoveNavigation: true},
			input: `<html><body><nav class="navigation">Menu</nav><p>Content</p></body></html>`,
			contains:    []string{"Content"},
			notContains: []string{"<nav", "Menu"},
		},
		{
			name: "remove hidden elements",
			opts: SanitizerOptions{},
			input: `<html><body><div style="display:none">Hidden</div><p>Visible</p></body></html>`,
			contains:    []string{"Visible"},
			notContains: []string{"Hidden", "display:none"},
		},
		{
			name: "normalize URLs",
			opts: SanitizerOptions{BaseURL: "https://example.com/path/"},
			input: `<html><body><a href="/relative">Link</a></body></html>`,
			contains:    []string{"https://example.com/relative"},
		},
		{
			name: "remove empty elements",
			opts: SanitizerOptions{},
			input: `<html><body><p></p><p>Content</p></body></html>`,
			contains:    []string{"Content"},
			notContains: []string{"<p></p>"},
		},
		{
			name: "empty HTML",
			opts: SanitizerOptions{},
			input: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer(tt.opts)
			result, err := sanitizer.Sanitize(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				for _, contain := range tt.contains {
					assert.Contains(t, result, contain)
				}
				for _, notContain := range tt.notContains {
					assert.NotContains(t, result, notContain)
				}
			}
		})
	}
}

// TestResolveURL tests URL resolution
func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		ref      string
		expected string
	}{
		{
			name:     "absolute URL",
			base:     "https://example.com/",
			ref:      "https://other.com/page",
			expected: "https://other.com/page",
		},
		{
			name:     "relative URL",
			base:     "https://example.com/path/",
			ref:      "page.html",
			expected: "https://example.com/path/page.html",
		},
		{
			name:     "root relative",
			base:     "https://example.com/path/",
			ref:      "/root",
			expected: "https://example.com/root",
		},
		{
			name:     "empty string",
			base:     "https://example.com/",
			ref:      "",
			expected: "",
		},
		{
			name:     "fragment only",
			base:     "https://example.com/",
			ref:      "#section",
			expected: "#section",
		},
		{
			name:     "javascript link",
			base:     "https://example.com/",
			ref:      "javascript:void(0)",
			expected: "javascript:void(0)",
		},
		{
			name:     "mailto link",
			base:     "https://example.com/",
			ref:      "mailto:test@example.com",
			expected: "mailto:test@example.com",
		},
		{
			name:     "data URL",
			base:     "https://example.com/",
			ref:      "data:text/plain,hello",
			expected: "data:text/plain,hello",
		},
		{
			name:     "parent directory",
			base:     "https://example.com/path/page/",
			ref:      "../other",
			expected: "https://example.com/path/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := url.Parse(tt.base)
			require.NoError(t, err)
			result := resolveURL(base, tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeSrcset tests srcset normalization
func TestNormalizeSrcset(t *testing.T) {
	base, err := url.Parse("https://example.com/path/")
	require.NoError(t, err)

	tests := []struct {
		name     string
		srcset   string
		expected string
	}{
		{
			name:     "single URL",
			srcset:   "image.jpg",
			expected: "https://example.com/path/image.jpg",
		},
		{
			name:     "multiple URLs with descriptors",
			srcset:   "small.jpg 400w, medium.jpg 800w",
			expected: "https://example.com/path/small.jpg 400w, https://example.com/path/medium.jpg 800w",
		},
		{
			name:     "absolute URLs",
			srcset:   "https://other.com/image.jpg 400w",
			expected: "https://other.com/image.jpg 400w",
		},
		{
			name:     "root relative",
			srcset:   "/image.jpg 400w",
			expected: "https://example.com/image.jpg 400w",
		},
		{
			name:     "empty srcset",
			srcset:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSrcset(base, tt.srcset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizer_RemoveNavigationByClass tests class-based removal
func TestSanitizer_RemoveNavigationByClass(t *testing.T) {
	tests := []struct {
		name         string
		className    string
		input        string
		shouldRemain bool
	}{
		{"sidebar class", "sidebar", `<div class="sidebar">Menu</div><p>Content</p>`, false},
		{"navigation class", "navigation", `<nav class="navigation">Menu</nav><p>Content</p>`, false},
		{"footer class", "footer", `<div class="footer">Footer</div><p>Content</p>`, false},
		{"content class", "content", `<div class="content">Main</div><p>Content</p>`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer(SanitizerOptions{RemoveNavigation: true})
			result, err := sanitizer.Sanitize(tt.input)
			require.NoError(t, err)

			if tt.shouldRemain {
				assert.Contains(t, result, tt.className)
			} else {
				assert.NotContains(t, result, tt.className)
			}
		})
	}
}

// TestSanitizer_RemoveNavigationByID tests ID-based removal
func TestSanitizer_RemoveNavigationByID(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		input        string
		shouldRemain bool
	}{
		{"sidebar ID", "sidebar", `<div id="sidebar">Menu</div><p>Content</p>`, false},
		{"navigation ID", "navigation", `<nav id="navigation">Menu</nav><p>Content</p>`, false},
		{"content ID", "main", `<div id="main">Content</div>`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer(SanitizerOptions{RemoveNavigation: true})
			result, err := sanitizer.Sanitize(tt.input)
			require.NoError(t, err)

			if tt.shouldRemain {
				assert.Contains(t, result, tt.id)
			} else {
				assert.NotContains(t, result, tt.id)
			}
		})
	}
}

// TestSanitizer_RemoveTags tests tag removal
func TestSanitizer_RemoveTags(t *testing.T) {
	sanitizer := NewSanitizer(SanitizerOptions{})

	tags := []string{
		"script", "style", "noscript", "iframe", "object", "embed",
		"applet", "form", "input", "button", "select", "textarea",
	}

	for _, tag := range tags {
		t.Run("remove_"+tag, func(t *testing.T) {
			input := `<html><body><` + tag + `>Remove me</` + tag + `><p>Keep</p></body></html>`
			result, err := sanitizer.Sanitize(input)
			require.NoError(t, err)
			assert.NotContains(t, result, `<`+tag+`>`)
			assert.Contains(t, result, "Keep")
		})
	}
}
