package converter_test

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMarkdownConverter tests creating a new markdown converter
func TestNewMarkdownConverter(t *testing.T) {
	opts := converter.MarkdownOptions{
		Domain: "https://example.com",
	}
	mdConverter := converter.NewMarkdownConverter(opts)
	assert.NotNil(t, mdConverter)
}

// TestMarkdownConverter_Convert tests HTML to Markdown conversion
func TestMarkdownConverter_Convert(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		expectedInMD   []string
		notExpectedInMD []string
	}{
		{
			name:           "headings",
			html:           `<h1>Title 1</h1><h2>Title 2</h2>`,
			expectedInMD:   []string{"# Title 1", "## Title 2"},
			notExpectedInMD: []string{"<h1>", "<h2>"},
		},
		{
			name:           "paragraphs",
			html:           `<p>Paragraph 1</p><p>Paragraph 2</p>`,
			expectedInMD:   []string{"Paragraph 1", "Paragraph 2"},
			notExpectedInMD: []string{"<p>"},
		},
		{
			name:           "bold and italic",
			html:           `<p><strong>bold</strong> and <em>italic</em></p>`,
			expectedInMD:   []string{"**bold**", "*italic*"},
			notExpectedInMD: []string{"<strong>", "<em>"},
		},
		{
			name:           "links",
			html:           `<p><a href="https://example.com">link</a></p>`,
			expectedInMD:   []string{"[link](https://example.com)"},
			notExpectedInMD: []string{"<a href"},
		},
		{
			name:           "lists",
			html:           `<ul><li>Item 1</li><li>Item 2</li></ul>`,
			expectedInMD:   []string{"- Item 1", "- Item 2"},
			notExpectedInMD: []string{"<ul>", "<li>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdConverter := converter.NewMarkdownConverter(converter.MarkdownOptions{
				Domain: "https://example.com",
			})

			markdown, err := mdConverter.Convert(tt.html)
			require.NoError(t, err)
			assert.NotEmpty(t, markdown)

			for _, expected := range tt.expectedInMD {
				assert.Contains(t, markdown, expected)
			}
			for _, notExpected := range tt.notExpectedInMD {
				assert.NotContains(t, markdown, notExpected)
			}
		})
	}
}

// TestStripMarkdown tests markdown stripping
func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{"links", "[text](url)", "text"},
		{"images", "![alt](url)", "alt"},
		{"bold", "**bold**", "bold"},
		{"italic", "*italic*", "italic"},
		{"headers", "# Header", "Header"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.StripMarkdown(tt.markdown)
			assert.Contains(t, result, tt.expected)
		})
	}
}

// TestCountWords tests word counting
func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"simple", "Hello world", 2},
		{"spaces", "one  two   three", 3},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.CountWords(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCountChars tests character counting
func TestCountChars(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"simple", "Hello", 5},
		{"with spaces", "Hello world", 11},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.CountChars(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenerateFrontmatter tests YAML frontmatter generation
func TestGenerateFrontmatter(t *testing.T) {
	doc := &domain.Document{
		URL:       "https://example.com/test",
		Title:     "Test Document",
		FetchedAt: time.Now(),
		WordCount: 100,
	}

	frontmatter, err := converter.GenerateFrontmatter(doc)
	require.NoError(t, err)
	assert.Contains(t, frontmatter, "---")
	assert.Contains(t, frontmatter, "title: Test Document")
	assert.Contains(t, frontmatter, "url: https://example.com/test")
}

// TestAddFrontmatter tests adding frontmatter to markdown
func TestAddFrontmatter(t *testing.T) {
	markdown := "# Main Content\n\nThis is the content."
	doc := &domain.Document{
		URL:       "https://example.com/test",
		Title:     "Test",
		FetchedAt: time.Now(),
	}

	result, err := converter.AddFrontmatter(markdown, doc)
	require.NoError(t, err)
	assert.Contains(t, result, "---")
	assert.Contains(t, result, "title: Test")
	assert.Contains(t, result, "# Main Content")
}

// TestDefaultMarkdownOptions tests default markdown options
func TestDefaultMarkdownOptions(t *testing.T) {
	opts := converter.DefaultMarkdownOptions()
	assert.Equal(t, "fenced", opts.CodeBlockStyle)
	assert.Equal(t, "atx", opts.HeadingStyle)
	assert.Equal(t, "-", opts.BulletListStyle)
}
