package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsMarkdownContent tests markdown content detection
func TestIsMarkdownContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		url         string
		expected    bool
	}{
		{
			name:        "markdown content type",
			contentType: "text/markdown",
			url:         "https://example.com/file",
			expected:    true,
		},
		{
			name:        "x-markdown content type",
			contentType: "text/x-markdown",
			url:         "https://example.com/file",
			expected:    true,
		},
		{
			name:        "application markdown",
			contentType: "application/markdown",
			url:         "https://example.com/file",
			expected:    true,
		},
		{
			name:        "uppercase markdown content type",
			contentType: "TEXT/MARKDOWN",
			url:         "https://example.com/file",
			expected:    true,
		},
		{
			name:        ".md extension",
			contentType: "text/plain",
			url:         "https://example.com/doc.md",
			expected:    true,
		},
		{
			name:        ".markdown extension",
			contentType: "text/plain",
			url:         "https://example.com/doc.markdown",
			expected:    true,
		},
		{
			name:        ".mdown extension",
			contentType: "text/plain",
			url:         "https://example.com/doc.mdown",
			expected:    true,
		},
		{
			name:        ".md with query params",
			contentType: "text/plain",
			url:         "https://example.com/doc.md?version=1",
			expected:    true,
		},
		{
			name:        ".md with fragment",
			contentType: "text/plain",
			url:         "https://example.com/doc.md#section",
			expected:    true,
		},
		{
			name:        "html content type",
			contentType: "text/html",
			url:         "https://example.com/doc.html",
			expected:    false,
		},
		{
			name:        "plain text without md extension",
			contentType: "text/plain",
			url:         "https://example.com/doc.txt",
			expected:    false,
		},
		{
			name:        "empty content type and url",
			contentType: "",
			url:         "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMarkdownContent(tt.contentType, tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsHTMLContent tests HTML content detection
func TestIsHTMLContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "text/html",
			contentType: "text/html",
			expected:    true,
		},
		{
			name:        "application/xhtml",
			contentType: "application/xhtml+xml",
			expected:    true,
		},
		{
			name:        "uppercase html",
			contentType: "TEXT/HTML",
			expected:    true,
		},
		{
			name:        "empty content type",
			contentType: "",
			expected:    true,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "application/json",
			contentType: "application/json",
			expected:    false,
		},
		{
			name:        "image/png",
			contentType: "image/png",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHTMLContent(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
