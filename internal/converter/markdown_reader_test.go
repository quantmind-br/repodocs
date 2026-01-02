package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewMarkdownReader tests creating a new markdown reader
func TestNewMarkdownReader(t *testing.T) {
	reader := NewMarkdownReader()
	assert.NotNil(t, reader)
}

// TestMarkdownReader_ParseFrontmatter tests frontmatter parsing
func TestMarkdownReader_ParseFrontmatter(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name           string
		content        string
		expectTitle    string
		expectDesc     string
		expectBodyContains string
	}{
		{
			name: "full frontmatter",
			content: `---
title: Test Title
description: Test Description
tags: [tag1, tag2]
---

# Body Content

This is the body.`,
			expectTitle:    "Test Title",
			expectDesc:     "Test Description",
			expectBodyContains: "Body Content",
		},
		{
			name: "no frontmatter",
			content: `# Direct Title

Direct content.`,
			expectTitle: "Direct Title",
			expectDesc: "",
			expectBodyContains: "Direct content.",
		},
		{
			name: "invalid frontmatter - no closing",
			content: `---
title: Test
# Content`,
			expectTitle: "Content",
			expectBodyContains: "Content",
		},
		{
			name: "frontmatter with summary",
			content: `---
title: Test
summary: This is a summary
---

# Content`,
			expectTitle: "Test",
			expectDesc: "This is a summary",
		},
		{
			name: "empty content",
			content: "",
			expectTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := reader.Read(tt.content, "https://example.com")
			assert.NoError(t, err)

			if tt.expectTitle != "" {
				assert.Equal(t, tt.expectTitle, doc.Title)
			}
			if tt.expectDesc != "" {
				assert.Equal(t, tt.expectDesc, doc.Description)
			}
			if tt.expectBodyContains != "" {
				assert.Contains(t, doc.Content, tt.expectBodyContains)
			}
		})
	}
}

// TestMarkdownReader_ExtractTitle tests title extraction from markdown
func TestMarkdownReader_ExtractTitle(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name        string
		frontmatter *Frontmatter
		body        string
		expected    string
	}{
		{
			name:        "title from frontmatter",
			frontmatter: &Frontmatter{Title: "FM Title"},
			body:        "# H1 Title",
			expected:    "FM Title",
		},
		{
			name:        "title from h1",
			frontmatter: nil,
			body:        "# H1 Title",
			expected:    "H1 Title",
		},
		{
			name:        "no title",
			frontmatter: nil,
			body:        "Just content",
			expected:    "",
		},
		{
			name:        "h1 with trailing hashes",
			frontmatter: nil,
			body:        "# Title ##",
			expected:    "Title",
		},
		{
			name:        "title in code block",
			frontmatter: nil,
			body:        "```\n# Not a title\n```\n# Real Title",
			expected:    "Real Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.extractTitle(tt.frontmatter, tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMarkdownReader_ExtractDescription tests description extraction
func TestMarkdownReader_ExtractDescription(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name        string
		frontmatter *Frontmatter
		body        string
		expected    string
	}{
		{
			name:        "description from frontmatter",
			frontmatter: &Frontmatter{Description: "FM Description"},
			body:        "Content here",
			expected:    "FM Description",
		},
		{
			name:        "summary from frontmatter",
			frontmatter: &Frontmatter{Summary: "FM Summary"},
			body:        "Content here",
			expected:    "FM Summary",
		},
		{
			name:        "description takes precedence",
			frontmatter: &Frontmatter{Description: "Desc", Summary: "Sum"},
			body:        "Content",
			expected:    "Desc",
		},
		{
			name:        "extract from first paragraph",
			frontmatter: nil,
			body:        "This is the first paragraph.\n\nSecond paragraph.",
			expected:    "This is the first paragraph.",
		},
		{
			name:        "skip headers",
			frontmatter: nil,
			body:        "# Header\n\nParagraph content.",
			expected:    "Paragraph content.",
		},
		{
			name:        "skip lists",
			frontmatter: nil,
			body:        "- Item 1\n- Item 2\n\nParagraph after list.",
			expected:    "Paragraph after list.",
		},
		{
			name:        "skip code blocks",
			frontmatter: nil,
			body:        "```\ncode\n```\n\nReal paragraph.",
			expected:    "Real paragraph.",
		},
		{
			name:        "truncate long description",
			frontmatter: nil,
			body:        string(make([]byte, 350)), // 350 chars
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.extractDescription(tt.frontmatter, tt.body)
			if tt.expected == "" && len(tt.body) > 300 {
				// For truncation test, just check it's truncated
				assert.LessOrEqual(t, len(result), 300)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestMarkdownReader_ExtractHeaders tests header extraction from markdown
func TestMarkdownReader_ExtractHeaders(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name     string
		body     string
		expected map[string][]string
	}{
		{
			name: "all header levels",
			body: "# H1\n\n## H2\n\n### H3\n\n#### H4\n\n##### H5\n\n###### H6",
			expected: map[string][]string{
				"h1": {"H1"},
				"h2": {"H2"},
				"h3": {"H3"},
				"h4": {"H4"},
				"h5": {"H5"},
				"h6": {"H6"},
			},
		},
		{
			name: "multiple headers same level",
			body: "# First\n\n# Second\n\n# Third",
			expected: map[string][]string{
				"h1": {"First", "Second", "Third"},
			},
		},
		{
			name: "skip headers in code blocks",
			body: "```\n# Not a header\n```\n\n# Real header",
			expected: map[string][]string{
				"h1": {"Real header"},
			},
		},
		{
			name: "headers with trailing hashes",
			body: "# Title ###",
			expected: map[string][]string{
				"h1": {"Title"},
			},
		},
		{
			name:     "no headers",
			body:     "Just paragraph content.",
			expected: map[string][]string{},
		},
		{
			name:     "empty body",
			body:     "",
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.extractHeaders(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMarkdownReader_ExtractLinks tests link extraction from markdown
func TestMarkdownReader_ExtractLinks(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name     string
		body     string
		baseURL  string
		expected []string
	}{
		{
			name:    "absolute links",
			body:    "[Link](https://example.com/page)",
			baseURL: "https://base.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:    "relative links",
			body:    "[Link](/page)",
			baseURL: "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "skip anchors",
			body:     "[Link](#section)",
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:     "skip javascript",
			body:     "[Link](javascript:void(0))",
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:     "skip mailto",
			body:     "[Email](mailto:test@example.com)",
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:    "multiple links",
			body:    "[One](/page1) [Two](/page2)",
			baseURL: "https://example.com",
			expected: []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:     "skip links in code blocks",
			body:     "```\n[Not a link](/page)\n```\n\n[Real link](/real)",
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/real"},
		},
		{
			name:     "deduplicate links",
			body:     "[Link](/page) [Again](/page)",
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "links with titles",
			body:     `[Link](/page "Title")`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "no links",
			body:     "Just text",
			baseURL:  "https://example.com",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.extractLinks(tt.body, tt.baseURL)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// TestMarkdownReader_CalculateHash tests hash calculation
func TestMarkdownReader_CalculateHash(t *testing.T) {
	reader := NewMarkdownReader()

	tests := []struct {
		name     string
		content  string
		sameAs   string
	}{
		{
			name:    "same content produces same hash",
			content: "Test content",
			sameAs:  "Test content",
		},
		{
			name:    "different content produces different hash",
			content: "Content A",
			sameAs:  "Content B",
		},
		{
			name:    "case sensitive",
			content: "Content",
			sameAs:  "content",
		},
		{
			name:    "whitespace matters",
			content: "Content",
			sameAs:  "Content ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := reader.calculateHash(tt.content)
			hash2 := reader.calculateHash(tt.sameAs)

			if tt.content == tt.sameAs {
				assert.Equal(t, hash1, hash2)
			} else {
				assert.NotEqual(t, hash1, hash2)
			}
		})
	}
}

// TestMarkdownReader_Read tests full document reading
func TestMarkdownReader_Read(t *testing.T) {
	reader := NewMarkdownReader()

	content := `---
title: Test Document
description: Test description
---

# Main Heading

This is a paragraph.

## Sub Heading

Another paragraph.

[Link](https://example.com)
`

	doc, err := reader.Read(content, "https://example.com/doc")
	assert.NoError(t, err)
	assert.NotNil(t, doc)

	assert.Equal(t, "Test Document", doc.Title)
	assert.Equal(t, "Test description", doc.Description)
	assert.Contains(t, doc.Content, "Main Heading")
	assert.NotEmpty(t, doc.Headers)
	assert.NotEmpty(t, doc.Links)
	assert.Greater(t, doc.WordCount, 0)
	assert.Greater(t, doc.CharCount, 0)
	assert.NotEmpty(t, doc.ContentHash)
}
