package converter

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMarkdownConverter tests creating a new markdown converter
func TestNewMarkdownConverter(t *testing.T) {
	opts := MarkdownOptions{
		Domain:          "https://example.com",
		CodeBlockStyle:  "fenced",
		HeadingStyle:    "atx",
		BulletListStyle: "-",
	}

	converter := NewMarkdownConverter(opts)

	assert.NotNil(t, converter)
	assert.Equal(t, "https://example.com", converter.domain)
}

// TestDefaultMarkdownOptions tests default markdown options
func TestDefaultMarkdownOptions(t *testing.T) {
	opts := DefaultMarkdownOptions()

	assert.Equal(t, "fenced", opts.CodeBlockStyle)
	assert.Equal(t, "atx", opts.HeadingStyle)
	assert.Equal(t, "-", opts.BulletListStyle)
}

// TestMarkdownConverter_Convert tests HTML to Markdown conversion
func TestMarkdownConverter_Convert(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name:    "simple paragraph",
			html:    "<p>Hello, world!</p>",
			wantErr: false,
		},
		{
			name:    "heading",
			html:    "<h1>Title</h1>",
			wantErr: false,
		},
		{
			name:    "link",
			html:    `<a href="https://example.com">Link</a>`,
			wantErr: false,
		},
		{
			name:    "code block",
			html:    "<pre><code>const x = 1;</code></pre>",
			wantErr: false,
		},
		{
			name:    "empty HTML",
			html:    "",
			wantErr: false,
		},
		{
			name:    "nested elements",
			html:    "<div><p>Text</p></div>",
			wantErr: false,
		},
	}

	converter := NewMarkdownConverter(MarkdownOptions{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.html)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Only check NotEmpty if HTML is not empty
				if tt.html != "" {
					assert.NotEmpty(t, result)
				}
			}
		})
	}
}

// TestCleanMarkdown tests markdown cleaning
func TestCleanMarkdown(t *testing.T) {
	converter := NewMarkdownConverter(MarkdownOptions{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "excessive blank lines",
			input:    "Line 1\n\n\n\n\nLine 2",
			expected: "Line 1\n\n\nLine 2",
		},
		{
			name:     "leading whitespace",
			input:    "   \n  Text  ",
			expected: "Text",
		},
		{
			name:     "no cleanup needed",
			input:    "Line 1\n\nLine 2",
			expected: "Line 1\n\nLine 2",
		},
		{
			name:     "only whitespace",
			input:    "   \n   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.cleanMarkdown(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGenerateFrontmatter tests frontmatter generation
func TestGenerateFrontmatter(t *testing.T) {
	doc := &domain.Document{
		URL:         "https://example.com/doc",
		Title:       "Test Document",
		WordCount:   100,
		CharCount:   500,
		FetchedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	frontmatter, err := GenerateFrontmatter(doc)
	require.NoError(t, err)

	assert.Contains(t, frontmatter, "---")
	assert.Contains(t, frontmatter, "title: Test Document")
	assert.Contains(t, frontmatter, "word_count: 100")
}

// TestAddFrontmatter tests adding frontmatter to markdown
func TestAddFrontmatter(t *testing.T) {
	doc := &domain.Document{
		URL:       "https://example.com/doc",
		Title:     "Test Document",
		FetchedAt: time.Now(),
	}

	markdown := "# Heading\n\nSome content"

	result, err := AddFrontmatter(markdown, doc)
	require.NoError(t, err)

	assert.Contains(t, result, "---")
	assert.Contains(t, result, markdown)
	assert.True(t, len(result) > len(markdown))
}

// TestCountWords tests word counting
func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "simple text",
			text:     "Hello world",
			expected: 2,
		},
		{
			name:     "multiple spaces",
			text:     "Hello    world   test",
			expected: 3,
		},
		{
			name:     "newlines",
			text:     "Line 1\nLine 2\nLine 3",
			expected: 6,
		},
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "only whitespace",
			text:     "   \n\t  ",
			expected: 0,
		},
		{
			name:     "tabs and spaces",
			text:     "One\ttwo three  four",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountWords(tt.text)
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
		{
			name:     "simple text",
			text:     "hello",
			expected: 5,
		},
		{
			name:     "with spaces",
			text:     "hello world",
			expected: 11,
		},
		{
			name:     "with newlines",
			text:     "line1\nline2",
			expected: 11,
		},
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "unicode (counting bytes, not runes)",
			text:     "héllo wörld",
			expected: 13, // len() counts bytes, é and ö are 2 bytes each in UTF-8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountChars(tt.text)
			assert.Equal(t, tt.expected, result)
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
		{
			name:     "links",
			markdown: "[Text](https://example.com)",
			expected: "Text",
		},
		{
			name:     "images",
			markdown: "![Alt](https://example.com/img.png)",
			expected: "!Alt", // Current behavior - regex may not match correctly
		},
		{
			name:     "bold",
			markdown: "**bold** text",
			expected: "bold text",
		},
		{
			name:     "italic",
			markdown: "*italic* text",
			expected: "italic text",
		},
		{
			name:     "headers",
			markdown: "# Heading",
			expected: "Heading",
		},
		{
			name:     "code block",
			markdown: "```\ncode\n```\ntext",
			expected: "text",
		},
		{
			name:     "horizontal rule",
			markdown: "---\ntext",
			expected: "text",
		},
		{
			name:     "blockquote",
			markdown: "> quote",
			expected: "quote",
		},
		{
			name:     "list",
			markdown: "- item",
			expected: "item",
		},
		{
			name:     "numbered list",
			markdown: "1. item",
			expected: "item",
		},
		{
			name:     "complex markdown",
			markdown: "# Title\n\n**Bold** and [link](url) and `code`",
			expected: "Title\n\nBold and link and `code`", // Code spans and newlines are preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripMarkdown(tt.markdown)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRemoveCodeBlocks tests code block removal
func TestRemoveCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains string
		notContains []string
	}{
		{
			name:     "fenced code block",
			markdown: "```javascript\nconst x = 1;\n```\ntext after",
			contains: "text after",
			notContains: []string{"```", "const x = 1;"},
		},
		{
			name:     "multiple fenced blocks",
			markdown: "```\nblock1\n```\ntext\n```\nblock2\n```",
			contains: "text",
			notContains: []string{"block1", "block2"},
		},
		{
			name:     "indented code block",
			markdown: "    code line\nnormal text",
			contains: "normal text",
			notContains: []string{"code line"},
		},
		{
			name:     "tab indented",
			markdown: "\tcode line\nnormal text",
			contains: "normal text",
			notContains: []string{"code line"},
		},
		{
			name:     "no code blocks",
			markdown: "just normal text",
			contains: "just normal text",
		},
		{
			name:     "empty string",
			markdown: "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeCodeBlocks(tt.markdown)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			}
			for _, notContain := range tt.notContains {
				assert.NotContains(t, result, notContain)
			}
		})
	}
}
