package converter_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownReader_Read(t *testing.T) {
	reader := converter.NewMarkdownReader()

	t.Run("full document with frontmatter", func(t *testing.T) {
		content := `---
title: Test Document
description: A test description
tags:
  - test
  - example
---

# Introduction

This is the first paragraph.

## Section One

Some content with a [link](https://example.com).

### Subsection

- List item 1
- List item 2
`

		doc, err := reader.Read(content, "https://example.com/docs/test.md")
		require.NoError(t, err)

		assert.Equal(t, "Test Document", doc.Title)
		assert.Equal(t, "A test description", doc.Description)
		assert.Equal(t, "https://example.com/docs/test.md", doc.URL)
		assert.NotEmpty(t, doc.ContentHash)
		assert.Greater(t, doc.WordCount, 0)

		assert.Contains(t, doc.Headers, "h1")
		assert.Contains(t, doc.Headers, "h2")
		assert.Contains(t, doc.Headers, "h3")
		assert.Equal(t, []string{"Introduction"}, doc.Headers["h1"])

		assert.Contains(t, doc.Links, "https://example.com")
	})

	t.Run("document without frontmatter", func(t *testing.T) {
		content := `# My Title

This is the description paragraph.

## Section

Content here.
`

		doc, err := reader.Read(content, "https://example.com/doc.md")
		require.NoError(t, err)

		assert.Equal(t, "My Title", doc.Title)
		assert.Equal(t, "This is the description paragraph.", doc.Description)
	})

	t.Run("relative links are resolved", func(t *testing.T) {
		content := `Check out [the guide](./guide.md) and [API docs](/api/reference.md).`

		doc, err := reader.Read(content, "https://example.com/docs/intro.md")
		require.NoError(t, err)

		assert.Contains(t, doc.Links, "https://example.com/docs/guide.md")
		assert.Contains(t, doc.Links, "https://example.com/api/reference.md")
	})

	t.Run("code blocks are ignored for headers and links", func(t *testing.T) {
		content := "# Real Title\n\n```markdown\n# Not a title\n[not a link](http://ignored.com)\n```\n"

		doc, err := reader.Read(content, "https://example.com/test.md")
		require.NoError(t, err)

		assert.Equal(t, "Real Title", doc.Title)
		assert.Len(t, doc.Headers["h1"], 1)
		assert.NotContains(t, doc.Links, "http://ignored.com")
	})

	t.Run("empty content", func(t *testing.T) {
		doc, err := reader.Read("", "https://example.com/empty.md")
		require.NoError(t, err)

		assert.Equal(t, "", doc.Title)
		assert.Equal(t, 0, doc.WordCount)
	})

	t.Run("malformed frontmatter is treated as content", func(t *testing.T) {
		content := `---
invalid: yaml: content: [
---

# Title
`

		doc, err := reader.Read(content, "https://example.com/test.md")
		require.NoError(t, err)

		assert.NotEmpty(t, doc.Content)
	})
}

func TestMarkdownReader_ParseFrontmatter(t *testing.T) {
	reader := converter.NewMarkdownReader()

	tests := []struct {
		name        string
		content     string
		wantTitle   string
		wantHasBody bool
	}{
		{
			name: "standard frontmatter",
			content: `---
title: Hello World
---

Body content`,
			wantTitle:   "Hello World",
			wantHasBody: true,
		},
		{
			name: "no frontmatter",
			content: `# Title

Body content`,
			wantTitle:   "Title",
			wantHasBody: true,
		},
		{
			name: "frontmatter only",
			content: `---
title: Only Frontmatter
---`,
			wantTitle:   "Only Frontmatter",
			wantHasBody: false,
		},
		{
			name: "unclosed frontmatter",
			content: `---
title: Unclosed

Body content`,
			wantTitle:   "",
			wantHasBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := reader.Read(tt.content, "https://example.com/test.md")
			require.NoError(t, err)

			if tt.wantTitle != "" {
				assert.Equal(t, tt.wantTitle, doc.Title)
			}
			if tt.wantHasBody {
				assert.NotEmpty(t, doc.Content)
			}
		})
	}
}

func TestMarkdownReader_ExtractHeaders(t *testing.T) {
	reader := converter.NewMarkdownReader()

	content := `# H1 Title

## H2 Section

### H3 Subsection

#### H4 Deep

##### H5 Deeper

###### H6 Deepest

## Another H2
`

	doc, err := reader.Read(content, "https://example.com/test.md")
	require.NoError(t, err)

	assert.Len(t, doc.Headers["h1"], 1)
	assert.Len(t, doc.Headers["h2"], 2)
	assert.Len(t, doc.Headers["h3"], 1)
	assert.Len(t, doc.Headers["h4"], 1)
	assert.Len(t, doc.Headers["h5"], 1)
	assert.Len(t, doc.Headers["h6"], 1)

	assert.Equal(t, "H1 Title", doc.Headers["h1"][0])
	assert.Contains(t, doc.Headers["h2"], "H2 Section")
	assert.Contains(t, doc.Headers["h2"], "Another H2")
}

func TestMarkdownReader_ExtractLinks(t *testing.T) {
	reader := converter.NewMarkdownReader()

	content := `
[External](https://external.com)
[Internal](/docs/page.md)
[Relative](./sibling.md)
[With Title](https://example.com "Example Site")
[Anchor Only](#section)
[Email](mailto:test@example.com)
[Phone](tel:+1234567890)
[JavaScript](javascript:void(0))
`

	doc, err := reader.Read(content, "https://base.com/docs/current.md")
	require.NoError(t, err)

	assert.Contains(t, doc.Links, "https://external.com")
	assert.Contains(t, doc.Links, "https://base.com/docs/page.md")
	assert.Contains(t, doc.Links, "https://base.com/docs/sibling.md")
	assert.Contains(t, doc.Links, "https://example.com")

	for _, link := range doc.Links {
		assert.NotContains(t, link, "#section")
		assert.NotContains(t, link, "mailto:")
		assert.NotContains(t, link, "tel:")
		assert.NotContains(t, link, "javascript:")
	}
}

func TestMarkdownReader_DescriptionTruncation(t *testing.T) {
	reader := converter.NewMarkdownReader()

	longParagraph := "This is a very long paragraph that exceeds three hundred characters. " +
		"It contains a lot of text that should be truncated when used as a description. " +
		"We need to make sure that the truncation works correctly and adds an ellipsis at the end. " +
		"This sentence pushes us well over the limit to ensure truncation happens properly."

	content := "# Title\n\n" + longParagraph

	doc, err := reader.Read(content, "https://example.com/test.md")
	require.NoError(t, err)

	assert.LessOrEqual(t, len(doc.Description), 300)
	assert.True(t, len(doc.Description) > 0)
	if len(longParagraph) > 300 {
		assert.True(t, strings.HasSuffix(doc.Description, "..."))
	}
}

var strings = struct {
	HasSuffix func(s, suffix string) bool
}{
	HasSuffix: func(s, suffix string) bool {
		return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
	},
}

func TestMarkdownReader_TildeCodeBlocks(t *testing.T) {
	reader := converter.NewMarkdownReader()

	content := `# Real Title

~~~python
# This is a comment, not a heading
def hello():
    print("world")
~~~

[Real link](https://real.com)
`

	doc, err := reader.Read(content, "https://example.com/test.md")
	require.NoError(t, err)

	assert.Equal(t, "Real Title", doc.Title)
	assert.Len(t, doc.Headers["h1"], 1)
	assert.Contains(t, doc.Links, "https://real.com")
}

func TestMarkdownReader_ClosingATXHeaders(t *testing.T) {
	reader := converter.NewMarkdownReader()

	content := `# Title with closing hashes #

## Section ##

### Subsection ###
`

	doc, err := reader.Read(content, "https://example.com/test.md")
	require.NoError(t, err)

	assert.Equal(t, "Title with closing hashes", doc.Title)
	assert.Contains(t, doc.Headers["h2"], "Section")
	assert.Contains(t, doc.Headers["h3"], "Subsection")
}
