package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPlainTextReader(t *testing.T) {
	reader := NewPlainTextReader()
	assert.NotNil(t, reader)
}

func TestPlainTextReader_Read(t *testing.T) {
	reader := NewPlainTextReader()

	tests := []struct {
		name      string
		content   string
		url       string
		wantTitle string
		wantDesc  string
		wantLinks int
	}{
		{
			name: "simple plain text",
			content: `Welcome to the documentation.

This is the first paragraph of content.

More content here.`,
			url:       "https://example.com/docs.txt",
			wantTitle: "Welcome to the documentation.",
			wantDesc:  "Welcome to the documentation.",
			wantLinks: 0,
		},
		{
			name: "plain text with markdown links",
			content: `Documentation Index

[Getting Started](https://example.com/start)
[API Reference](https://example.com/api)`,
			url:       "https://example.com/llms.txt",
			wantTitle: "Documentation Index",
			wantLinks: 2,
		},
		{
			name: "llms.txt format with header",
			content: `# OpenAI Documentation

[Introduction](/docs/introduction)
[API Reference](/docs/api-reference)
[Codex](/docs/codex)`,
			url:       "https://developers.openai.com/llms.txt",
			wantTitle: "OpenAI Documentation",
			wantLinks: 3,
		},
		{
			name:      "empty content uses filename",
			content:   "",
			url:       "https://example.com/empty.txt",
			wantTitle: "empty",
			wantDesc:  "",
			wantLinks: 0,
		},
		{
			name:      "only whitespace uses filename",
			content:   "   \n\n   ",
			url:       "https://example.com/blank.txt",
			wantTitle: "blank",
			wantDesc:  "",
			wantLinks: 0,
		},
		{
			name:      "unicode content",
			content:   "日本語ドキュメント\n\nこれはテストです。",
			url:       "https://example.com/japanese.txt",
			wantTitle: "日本語ドキュメント",
			wantLinks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := reader.Read(tt.content, tt.url)
			require.NoError(t, err)
			assert.NotNil(t, doc)

			assert.Equal(t, tt.wantTitle, doc.Title)
			assert.Len(t, doc.Links, tt.wantLinks)
			assert.NotEmpty(t, doc.ContentHash)
			assert.Equal(t, tt.url, doc.URL)
		})
	}
}

func TestPlainTextReader_ExtractTitle(t *testing.T) {
	reader := NewPlainTextReader()

	tests := []struct {
		name    string
		content string
		url     string
		want    string
	}{
		{
			name:    "first line as title",
			content: "First Line Title\n\nContent here.",
			url:     "https://example.com/doc.txt",
			want:    "First Line Title",
		},
		{
			name:    "markdown header",
			content: "# Markdown Title\n\nContent",
			url:     "https://example.com/doc.txt",
			want:    "Markdown Title",
		},
		{
			name:    "empty content uses filename",
			content: "",
			url:     "https://example.com/my-document.txt",
			want:    "my-document",
		},
		{
			name:    "long title truncated",
			content: strings.Repeat("A", 150),
			url:     "https://example.com/doc.txt",
			want:    strings.Repeat("A", 97) + "...",
		},
		{
			name:    "leading whitespace trimmed",
			content: "   Title with spaces   ",
			url:     "https://example.com/doc.txt",
			want:    "Title with spaces",
		},
		{
			name:    "skip empty lines",
			content: "\n\n\nActual Title\n\nContent",
			url:     "https://example.com/doc.txt",
			want:    "Actual Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.extractTitle(tt.content, tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlainTextReader_ExtractDescription(t *testing.T) {
	reader := NewPlainTextReader()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "first paragraph single line",
			content: "Title\n\nThis is the description paragraph.\n\nMore content.",
			want:    "Title",
		},
		{
			name:    "multi-line paragraph",
			content: "First line\nSecond line\nThird line\n\nNext paragraph.",
			want:    "First line Second line Third line",
		},
		{
			name:    "skip header then get paragraph",
			content: "# Title\n\nThis is description.",
			want:    "This is description.",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "only header",
			content: "# Only Header",
			want:    "",
		},
		{
			name:    "truncate long description",
			content: strings.Repeat("X", 350),
			want:    strings.Repeat("X", 297) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.extractDescription(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlainTextReader_ExtractLinks(t *testing.T) {
	reader := NewPlainTextReader()

	tests := []struct {
		name    string
		content string
		baseURL string
		want    []string
	}{
		{
			name:    "absolute URLs",
			content: "[Link](https://example.com/page)",
			baseURL: "https://base.com/",
			want:    []string{"https://example.com/page"},
		},
		{
			name:    "relative URLs resolved",
			content: "[Link](/docs/page)",
			baseURL: "https://example.com/index.txt",
			want:    []string{"https://example.com/docs/page"},
		},
		{
			name:    "multiple links",
			content: "[A](https://a.com) [B](https://b.com)",
			baseURL: "https://example.com/",
			want:    []string{"https://a.com", "https://b.com"},
		},
		{
			name:    "skip anchor links",
			content: "[Section](#section)",
			baseURL: "https://example.com/",
			want:    nil,
		},
		{
			name:    "skip mailto",
			content: "[Email](mailto:test@example.com)",
			baseURL: "https://example.com/",
			want:    nil,
		},
		{
			name:    "skip tel",
			content: "[Phone](tel:+1234567890)",
			baseURL: "https://example.com/",
			want:    nil,
		},
		{
			name:    "deduplicate links",
			content: "[A](https://x.com) [B](https://x.com)",
			baseURL: "https://example.com/",
			want:    []string{"https://x.com"},
		},
		{
			name:    "no links",
			content: "Plain text without any links",
			baseURL: "https://example.com/",
			want:    nil,
		},
		{
			name:    "links with titles",
			content: `[Link](https://example.com "Title text")`,
			baseURL: "https://example.com/",
			want:    []string{"https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.extractLinks(tt.content, tt.baseURL)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlainTextReader_CalculateHash(t *testing.T) {
	reader := NewPlainTextReader()

	hash1 := reader.calculateHash("content")
	hash2 := reader.calculateHash("content")
	hash3 := reader.calculateHash("different")

	assert.Equal(t, hash1, hash2, "same content should produce same hash")
	assert.NotEqual(t, hash1, hash3, "different content should produce different hash")
	assert.Len(t, hash1, 64, "SHA256 hex should be 64 characters")
}
