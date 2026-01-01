package converter_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
)

func TestIsMarkdownContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		url         string
		want        bool
	}{
		{
			name:        "text/markdown content type",
			contentType: "text/markdown; charset=UTF-8",
			url:         "https://example.com/docs/page",
			want:        true,
		},
		{
			name:        "text/x-markdown content type",
			contentType: "text/x-markdown",
			url:         "https://example.com/docs/page",
			want:        true,
		},
		{
			name:        "application/markdown content type",
			contentType: "application/markdown",
			url:         "https://example.com/docs/page",
			want:        true,
		},
		{
			name:        "text/html content type with md URL",
			contentType: "text/html; charset=utf-8",
			url:         "https://example.com/docs/page.md",
			want:        false,
		},
		{
			name:        "URL with .md extension",
			contentType: "",
			url:         "https://example.com/docs/readme.md",
			want:        true,
		},
		{
			name:        "URL with .markdown extension",
			contentType: "",
			url:         "https://example.com/docs/readme.markdown",
			want:        true,
		},
		{
			name:        "URL with .mdown extension",
			contentType: "",
			url:         "https://example.com/docs/readme.mdown",
			want:        true,
		},
		{
			name:        "URL with .md and query string",
			contentType: "",
			url:         "https://example.com/docs/readme.md?v=1",
			want:        true,
		},
		{
			name:        "URL with .md and fragment",
			contentType: "",
			url:         "https://example.com/docs/readme.md#section",
			want:        true,
		},
		{
			name:        "URL with .html extension",
			contentType: "",
			url:         "https://example.com/docs/page.html",
			want:        false,
		},
		{
			name:        "empty content type and no extension",
			contentType: "",
			url:         "https://example.com/docs/page",
			want:        false,
		},
		{
			name:        "case insensitive content type",
			contentType: "TEXT/MARKDOWN",
			url:         "https://example.com/docs/page",
			want:        true,
		},
		{
			name:        "case insensitive URL",
			contentType: "",
			url:         "https://example.com/docs/README.MD",
			want:        true,
		},
		{
			name:        "application/xhtml blocks URL check",
			contentType: "application/xhtml+xml",
			url:         "https://example.com/docs/readme.md",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.IsMarkdownContent(tt.contentType, tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsHTMLContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "text/html",
			contentType: "text/html",
			want:        true,
		},
		{
			name:        "text/html with charset",
			contentType: "text/html; charset=utf-8",
			want:        true,
		},
		{
			name:        "application/xhtml+xml",
			contentType: "application/xhtml+xml",
			want:        true,
		},
		{
			name:        "empty content type defaults to HTML",
			contentType: "",
			want:        true,
		},
		{
			name:        "text/markdown is not HTML",
			contentType: "text/markdown",
			want:        false,
		},
		{
			name:        "text/plain is not HTML",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "application/json is not HTML",
			contentType: "application/json",
			want:        false,
		},
		{
			name:        "case insensitive",
			contentType: "TEXT/HTML",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.IsHTMLContent(tt.contentType)
			assert.Equal(t, tt.want, got)
		})
	}
}
