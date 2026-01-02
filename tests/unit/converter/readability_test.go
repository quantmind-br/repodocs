package converter_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExtractContent tests creating a new content extractor
func TestNewExtractContent(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		wantNil  bool
	}{
		{
			name:     "with selector",
			selector: "article",
			wantNil:  false,
		},
		{
			name:     "empty selector",
			selector: "",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := converter.NewExtractContent(tt.selector)

			if tt.wantNil {
				assert.Nil(t, extractor)
			} else {
				assert.NotNil(t, extractor)
			}
		})
	}
}

// TestExtractContent_WithSelector tests extraction with CSS selector
func TestExtractContent_WithSelector(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<div class="sidebar">Sidebar</div>
	<article class="main-content">
		<h1>Article Title</h1>
		<p>Article content here.</p>
	</article>
	<footer>Footer</footer>
</body>
</html>`

	extractor := converter.NewExtractContent("article.main-content")

	content, title, err := extractor.Extract(html, "https://example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Equal(t, "Test Page", title)
	assert.Contains(t, content, "Article Title")
	assert.Contains(t, content, "Article content here")
	assert.NotContains(t, content, "Sidebar")
	assert.NotContains(t, content, "Footer")
}

// TestExtractContent_WithMultipleMatches tests extraction when selector matches multiple elements
func TestExtractContent_WithMultipleMatches(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<article class="post">
		<h2>Post 1</h2>
		<p>Content 1</p>
	</article>
	<article class="post">
		<h2>Post 2</h2>
		<p>Content 2</p>
	</article>
</body>
</html>`

	extractor := converter.NewExtractContent("article.post")

	content, title, err := extractor.Extract(html, "https://example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, content, "Post 1")
	assert.Contains(t, content, "Content 1")
	assert.Contains(t, content, "Post 2")
	assert.Contains(t, content, "Content 2")
}

// TestExtractContent_WithNoMatches tests extraction when selector matches nothing
func TestExtractContent_WithNoMatches(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<div class="content">
		<h1>Title</h1>
		<p>Content here.</p>
	</div>
</body>
</html>`

	extractor := converter.NewExtractContent("article.nonexistent")

	content, title, err := extractor.Extract(html, "https://example.com")

	require.NoError(t, err)
	// Should fall back to Readability algorithm
	assert.NotEmpty(t, content)
	assert.NotEmpty(t, title)
}

// TestExtractContent_WithoutSelector tests extraction using Readability algorithm
func TestExtractContent_WithoutSelector(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Readability Test</title>
</head>
<body>
	<div class="header">Header</div>
	<div class="sidebar">Sidebar</div>
	<article>
		<h1>Main Article</h1>
		<p>This is the main article content that should be extracted.</p>
		<p>More article content.</p>
	</article>
	<div class="footer">Footer</div>
</body>
</html>`

	extractor := converter.NewExtractContent("")

	content, title, err := extractor.Extract(html, "https://example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Equal(t, "Readability Test", title)
	// Readability should extract the main article content
	assert.Contains(t, content, "Main Article")
}

// TestExtractContent_TitleExtraction tests various title formats
func TestExtractContent_TitleExtraction(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		expectedTitle string
	}{
		{
			name: "title tag",
			html: `<!DOCTYPE html>
<html>
<head><title>Title from Tag</title></head>
<body><h1>Header</h1></body>
</html>`,
			expectedTitle: "Title from Tag",
		},
		{
			name: "h1 fallback",
			html: `<!DOCTYPE html>
<html>
<head></head>
<body><h1>H1 Title</h1><p>Content</p></body>
</html>`,
			expectedTitle: "H1 Title",
		},
		{
			name: "og:title",
			html: `<!DOCTYPE html>
<html>
<head><meta property="og:title" content="OG Title"></head>
<body><p>Content</p></body>
</html>`,
			expectedTitle: "OG Title",
		},
		{
			name: "no title",
			html: `<!DOCTYPE html>
<html>
<head></head>
<body><p>Just content</p></body>
</html>`,
			expectedTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := converter.NewExtractContent("")

			_, title, err := extractor.Extract(tt.html, "https://example.com")

			require.NoError(t, err)
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}

// TestExtractContent_BodyFallback tests fallback to body extraction
func TestExtractContent_BodyFallback(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Body Test</title>
</head>
<body>
	<h1>Body Content</h1>
	<p>This is in the body.</p>
</body>
</html>`

	extractor := converter.NewExtractContent("")

	content, title, err := extractor.Extract(html, "https://example.com")

	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Equal(t, "Body Test", title)
	assert.Contains(t, content, "Body Content")
}

// TestExtractHeaders tests header extraction
func TestExtractHeaders(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Header 1</h1>
	<p>Content 1</p>
	<h2>Header 2.1</h2>
	<p>Content 2.1</p>
	<h2>Header 2.2</h2>
	<p>Content 2.2</p>
	<h3>Header 3.1</h3>
	<p>Content 3.1</p>
	<h4>Header 4.1</h4>
	<p>Content 4.1</p>
	<h5>Header 5.1</h5>
	<p>Content 5.1</p>
	<h6>Header 6.1</h6>
	<p>Content 6.1</p>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	doc, err := pipeline.Convert(nil, html, "https://example.com")

	require.NoError(t, err)
	require.NotNil(t, doc.Headers)

	assert.Contains(t, doc.Headers, "h1")
	assert.Contains(t, doc.Headers, "h2")
	assert.Contains(t, doc.Headers, "h3")
	assert.Contains(t, doc.Headers, "h4")
	assert.Contains(t, doc.Headers, "h5")
	assert.Contains(t, doc.Headers, "h6")

	assert.Equal(t, []string{"Header 1"}, doc.Headers["h1"])
	assert.Equal(t, []string{"Header 2.1", "Header 2.2"}, doc.Headers["h2"])
	assert.Equal(t, []string{"Header 3.1"}, doc.Headers["h3"])
	assert.Equal(t, []string{"Header 4.1"}, doc.Headers["h4"])
	assert.Equal(t, []string{"Header 5.1"}, doc.Headers["h5"])
	assert.Equal(t, []string{"Header 6.1"}, doc.Headers["h6"])
}

// TestExtractLinks tests link extraction
func TestExtractLinks(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Links Test</h1>
	<a href="/page1">Page 1</a>
	<a href="https://external.com/page2">External Page</a>
	<a href="#anchor">Anchor Link</a>
	<a href="javascript:void(0)">JS Link</a>
	<a href="mailto:test@example.com">Email</a>
	<a href="relative/path">Relative</a>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/path/to/page",
	})

	doc, err := pipeline.Convert(nil, html, "https://example.com/path/to/page")

	require.NoError(t, err)

	// Check that absolute and relative links are extracted
	assert.Contains(t, doc.Links, "https://example.com/page1")
	assert.Contains(t, doc.Links, "https://external.com/page2")
	assert.Contains(t, doc.Links, "https://example.com/path/to/relative/path")

	// Check that special URLs are not extracted
	assert.NotContains(t, doc.Links, "#anchor")
	assert.NotContains(t, doc.Links, "javascript:void(0)")
	assert.NotContains(t, doc.Links, "mailto:test@example.com")
}
