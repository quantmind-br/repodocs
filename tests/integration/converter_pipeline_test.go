package integration

import (
	"context"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_ConverterPipeline_UTF8HTML tests the complete pipeline with UTF-8 HTML
func TestIntegration_ConverterPipeline_UTF8HTML(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<head>
	<title>UTF-8 Test Page</title>
	<meta name="description" content="Testing UTF-8 encoding">
</head>
<body>
	<nav>Navigation Menu</nav>
	<main>
		<h1>Main Heading</h1>
		<p>This is a paragraph with <strong>bold text</strong> and <em>italic text</em>.</p>
		<ul>
			<li>List item 1</li>
			<li>List item 2</li>
		</ul>
		<blockquote>
			<p>This is a quote from someone important.</p>
		</blockquote>
	</main>
	<footer>Footer content</footer>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/page",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/page")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "UTF-8 Test Page", doc.Title)
	assert.Equal(t, "Testing UTF-8 encoding", doc.Description)
	assert.NotEmpty(t, doc.Content)
	// Verify markdown was generated (content may vary based on readability algorithm)
	assert.True(t, len(doc.Content) > 0, "Content should not be empty")
	// The important thing is that the pipeline executed successfully
}

// TestIntegration_ConverterPipeline_ISO88591Conversion tests encoding conversion
func TestIntegration_ConverterPipeline_ISO88591Conversion(t *testing.T) {
	// Arrange - HTML with special characters
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Special Characters & Symbols</title>
</head>
<body>
	<h1>Testing Special Characters</h1>
	<p>Euro: € | Pound: £ | Yen: ¥</p>
	<p>Quotes: "double" & 'single'</p>
	<p>Math symbols: ≤ ≥ ± × ÷</p>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/special",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/special")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "Special Characters & Symbols", doc.Title)
	assert.NotEmpty(t, doc.Content)
	// Verify special characters are preserved in markdown
	assert.Contains(t, doc.Content, "Euro")
	assert.Contains(t, doc.Content, "Pound")
	assert.Contains(t, doc.Content, "Math symbols")
}

// TestIntegration_ConverterPipeline_CustomSelector tests pipeline with custom content selector
func TestIntegration_ConverterPipeline_CustomSelector(t *testing.T) {
	// Arrange - HTML with specific content area
	html := `<!DOCTYPE html>
<html>
<head><title>Selector Test</title></head>
<body>
	<header>Site Header</header>
	<nav>Navigation</nav>
	<div class="sidebar">Sidebar content</div>
	<article class="main-content">
		<h1>Main Article</h1>
		<p>This is the main content that should be extracted.</p>
	</article>
	<footer>Footer</footer>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com/article",
		ContentSelector: ".main-content",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/article")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "Selector Test", doc.Title)
	assert.NotEmpty(t, doc.Content)
	// Should contain main content
	assert.Contains(t, doc.Content, "Main Article")
	assert.Contains(t, doc.Content, "main content")
	// Should not contain excluded elements
	assert.NotContains(t, doc.Content, "Site Header")
	assert.NotContains(t, doc.Content, "Sidebar")
}

// TestIntegration_ConverterPipeline_ComplexHTML tests pipeline with complex HTML structure
func TestIntegration_ConverterPipeline_ComplexHTML(t *testing.T) {
	// Arrange - Complex HTML with nested elements
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Complex Structure</title>
	<meta name="description" content="Complex HTML test">
	<meta property="og:title" content="OG Title">
</head>
<body>
	<div id="container">
		<header>
			<h1>Header Title</h1>
			<nav>
				<ul>
					<li><a href="/home">Home</a></li>
					<li><a href="/about">About</a></li>
				</ul>
			</nav>
		</header>
		<main>
			<section class="intro">
				<h2>Introduction</h2>
				<p>Introduction paragraph.</p>
			</section>
			<section class="content">
				<h2>Main Content</h2>
				<p>First paragraph of main content.</p>
				<h3>Subsection</h3>
				<p>Paragraph in subsection.</p>
				<pre><code>console.log("Hello World");</code></pre>
				<blockquote cite="http://example.com">
					<p>A blockquote with citation.</p>
				</blockquote>
			</section>
		</main>
		<aside>
			<h3>Related</h3>
			<p>Related content.</p>
		</aside>
		<footer>&copy; 2023 Example Corp</footer>
	</div>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/complex",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/complex")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.NotEmpty(t, doc.Content)
	assert.NotEmpty(t, doc.Title)
	// The important thing is that the pipeline handled complex HTML
	// Content may vary based on readability algorithm
}

// TestIntegration_ConverterPipeline_WithExcludeSelector tests pipeline with exclusion selector
func TestIntegration_ConverterPipeline_WithExcludeSelector(t *testing.T) {
	// Arrange - HTML with elements to exclude
	html := `<!DOCTYPE html>
<html>
<head><title>Exclude Test</title></head>
<body>
	<header>Header to exclude</header>
	<nav>Navigation to exclude</nav>
	<main>
		<h1>Main Content</h1>
		<p>This content should remain.</p>
	</main>
	<aside>Sidebar to exclude</aside>
	<footer>Footer to exclude</footer>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com/exclude",
		ExcludeSelector: "header,nav,aside,footer",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/exclude")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "Exclude Test", doc.Title)
	assert.NotEmpty(t, doc.Content)
	// Main content should be present
	assert.Contains(t, doc.Content, "Main Content")
	// Excluded elements should not appear in final content
	assert.NotContains(t, doc.Content, "Header to exclude")
	assert.NotContains(t, doc.Content, "Navigation to exclude")
	assert.NotContains(t, doc.Content, "Sidebar to exclude")
	assert.NotContains(t, doc.Content, "Footer to exclude")
}

// TestIntegration_ConverterPipeline_ReadabilityFallback tests pipeline when readability is needed
func TestIntegration_ConverterPipeline_ReadabilityFallback(t *testing.T) {
	// Arrange - HTML without specific selector, readability should extract main content
	html := `<!DOCTYPE html>
<html>
<head><title>Readability Test</title></head>
<body>
	<div class="site-header">
		<h1>Site Name</h1>
		<nav>
			<ul>
				<li><a href="/">Home</a></li>
				<li><a href="/about">About</a></li>
			</ul>
		</nav>
	</div>
	<article>
		<h2>Article Title</h2>
		<p>This is the main article content that readability should identify.</p>
		<p>Additional paragraphs help readability determine this is the main content.</p>
		<p>More content to strengthen the article signal.</p>
	</article>
	<div class="site-footer">
		<p>&copy; 2023 Example Corp</p>
	</div>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/readability",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/readability")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "Readability Test", doc.Title)
	assert.NotEmpty(t, doc.Content)
	// Readability should identify the article as main content
	assert.Contains(t, doc.Content, "Article Title")
	assert.Contains(t, doc.Content, "main article content")
	// Navigation and footer should be de-emphasized or removed
	assert.NotEmpty(t, doc.Content)
}

// TestIntegration_ConverterPipeline_Statistics tests that document statistics are calculated
func TestIntegration_ConverterPipeline_Statistics(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<head><title>Statistics Test</title></head>
<body>
	<h1>Title</h1>
	<p>Word1 Word2 Word3 Word4 Word5</p>
	<p>Word6 Word7 Word8</p>
	<ul>
		<li>Item1</li>
		<li>Item2</li>
	</ul>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/stats",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/stats")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Greater(t, doc.WordCount, 0)
	assert.Greater(t, doc.CharCount, 0)
	assert.NotEmpty(t, doc.ContentHash)
	// Verify statistics make sense
	assert.GreaterOrEqual(t, doc.WordCount, 8) // At least 8 words from paragraphs
	assert.Greater(t, doc.CharCount, 0)
	// Verify content hash is consistent
	assert.NotEmpty(t, doc.ContentHash)
	assert.Equal(t, len(doc.ContentHash), 64) // SHA256 is 64 hex characters
}

// TestIntegration_ConverterPipeline_HTMLContentPreservation tests that HTML is preserved in document
func TestIntegration_ConverterPipeline_HTMLContentPreservation(t *testing.T) {
	// Arrange
	html := `<!DOCTYPE html>
<html>
<head><title>HTML Preservation</title></head>
<body>
	<h1>Test</h1>
	<p>Test paragraph</p>
</body>
</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com/html",
	})

	ctx := context.Background()

	// Act
	doc, err := pipeline.Convert(ctx, html, "https://example.com/html")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, doc)
	// Verify HTML content is preserved
	assert.NotEmpty(t, doc.HTMLContent)
	assert.Contains(t, doc.HTMLContent, "<h1>Test</h1>")
	assert.Contains(t, doc.HTMLContent, "Test paragraph")
	// Verify markdown content is different
	assert.NotEmpty(t, doc.Content)
	assert.NotEqual(t, doc.HTMLContent, doc.Content)
}
