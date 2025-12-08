package app_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMarkdownOptions(t *testing.T) {
	opts := converter.DefaultMarkdownOptions()

	assert.Equal(t, "fenced", opts.CodeBlockStyle, "Default code block style should be 'fenced'")
	assert.Equal(t, "atx", opts.HeadingStyle, "Default heading style should be 'atx'")
	assert.Equal(t, "-", opts.BulletListStyle, "Default bullet list style should be '-'")
	assert.Empty(t, opts.Domain, "Default domain should be empty")
}

func TestConvertHTMLWithSelector(t *testing.T) {
	tests := []struct {
		name            string
		html            string
		sourceURL       string
		selector        string
		wantTitle       string
		wantContains    []string
		wantNotContains []string
		wantErr         bool
	}{
		{
			name: "basic selector extraction",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Page Title</title></head>
			<body>
				<nav>Navigation content</nav>
				<article class="main-content">
					<h1>Article Title</h1>
					<p>This is the main article content.</p>
				</article>
				<aside>Sidebar content</aside>
			</body>
			</html>`,
			sourceURL:       "https://example.com/article",
			selector:        "article.main-content",
			wantTitle:       "Page Title",
			wantContains:    []string{"Article Title", "main article content"},
			wantNotContains: []string{},
			wantErr:         false,
		},
		{
			name: "selector with ID",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Test Page</title></head>
			<body>
				<div id="sidebar">Sidebar</div>
				<div id="content">
					<h1>Main Content</h1>
					<p>Important information here.</p>
				</div>
			</body>
			</html>`,
			sourceURL:       "https://example.com/test",
			selector:        "#content",
			wantTitle:       "Test Page",
			wantContains:    []string{"Main Content", "Important information"},
			wantNotContains: []string{},
			wantErr:         false,
		},
		{
			name: "nested selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Nested Test</title></head>
			<body>
				<div class="wrapper">
					<div class="inner">
						<section class="docs">
							<h2>Documentation</h2>
							<p>Doc content here.</p>
						</section>
					</div>
				</div>
			</body>
			</html>`,
			sourceURL:    "https://example.com/docs",
			selector:     ".wrapper .inner .docs",
			wantTitle:    "Nested Test",
			wantContains: []string{"Documentation", "Doc content"},
			wantErr:      false,
		},
		{
			name: "selector not found falls back to readability",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Fallback Test</title></head>
			<body>
				<article>
					<h1>Actual Content</h1>
					<p>This should be extracted by readability.</p>
				</article>
			</body>
			</html>`,
			sourceURL:    "https://example.com/fallback",
			selector:     ".nonexistent-class",
			wantTitle:    "Fallback Test",
			wantContains: []string{"Actual Content"},
			wantErr:      false,
		},
		{
			name: "empty selector uses readability",
			html: `<!DOCTYPE html>
			<html>
			<head><title>No Selector</title></head>
			<body>
				<main>
					<h1>Main Heading</h1>
					<p>Paragraph content for testing.</p>
				</main>
			</body>
			</html>`,
			sourceURL:    "https://example.com/no-selector",
			selector:     "",
			wantTitle:    "No Selector",
			wantContains: []string{"Main Heading"},
			wantErr:      false,
		},
		{
			name: "tag selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Tag Selector</title></head>
			<body>
				<header>Header content</header>
				<main>
					<h1>Main Section</h1>
					<p>Content in main tag.</p>
				</main>
				<footer>Footer content</footer>
			</body>
			</html>`,
			sourceURL:    "https://example.com/main",
			selector:     "main",
			wantTitle:    "Tag Selector",
			wantContains: []string{"Main Section", "Content in main tag"},
			wantErr:      false,
		},
		{
			name: "multiple attribute selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Attribute Test</title></head>
			<body>
				<div data-role="sidebar">Sidebar</div>
				<div data-role="content" class="primary">
					<h1>Primary Content</h1>
					<p>Data attribute selected content.</p>
				</div>
			</body>
			</html>`,
			sourceURL:    "https://example.com/attr",
			selector:     "[data-role='content']",
			wantTitle:    "Attribute Test",
			wantContains: []string{"Primary Content"},
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := converter.ConvertHTMLWithSelector(tc.html, tc.sourceURL, tc.selector)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, doc)

			assert.Equal(t, tc.sourceURL, doc.URL)
			assert.Equal(t, tc.wantTitle, doc.Title)

			for _, want := range tc.wantContains {
				assert.Contains(t, doc.Content, want, "Expected content to contain: %s", want)
			}

			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, doc.Content, notWant, "Expected content to NOT contain: %s", notWant)
			}
		})
	}
}

func TestConvertHTML(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		sourceURL    string
		wantContains []string
		wantErr      bool
	}{
		{
			name: "basic conversion",
			html: `<html><head><title>Test</title></head>
				<body><h1>Hello World</h1><p>Content here.</p></body></html>`,
			sourceURL:    "https://example.com/test",
			wantContains: []string{"# Hello World", "Content here"},
			wantErr:      false,
		},
		{
			name:         "empty html",
			html:         `<html><body></body></html>`,
			sourceURL:    "https://example.com/empty",
			wantContains: []string{},
			wantErr:      false,
		},
		{
			name: "with links",
			html: `<html><body>
				<a href="https://example.com/other">Link text</a>
			</body></html>`,
			sourceURL:    "https://example.com/page",
			wantContains: []string{"Link text"},
			wantErr:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := converter.ConvertHTML(tc.html, tc.sourceURL)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, doc)

			for _, want := range tc.wantContains {
				assert.Contains(t, doc.Content, want)
			}
		})
	}
}

func TestPipeline_DocumentMetadata(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head>
		<title>Metadata Test Page</title>
		<meta name="description" content="This is a test page description">
	</head>
	<body>
		<article>
			<h1>Main Heading</h1>
			<p>Some content with multiple words for testing.</p>
			<a href="https://example.com/link1">Link 1</a>
			<a href="https://example.com/link2">Link 2</a>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/metadata-test")
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Check metadata fields
	assert.Equal(t, "https://example.com/metadata-test", doc.URL)
	assert.Equal(t, "Metadata Test Page", doc.Title)
	assert.Equal(t, "This is a test page description", doc.Description)

	// Check statistics
	assert.Greater(t, doc.WordCount, 0)
	assert.Greater(t, doc.CharCount, 0)
	assert.NotEmpty(t, doc.ContentHash)

	// Check links extraction
	assert.NotEmpty(t, doc.Links)

	// Check headers extraction
	assert.NotEmpty(t, doc.Headers)
}

func TestPipeline_ConvertHTML_WithCodeBlocks(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Code Test</title></head>
	<body>
		<article>
			<h1>Code Examples</h1>
			<pre><code class="language-go">func main() {
	fmt.Println("Hello")
}</code></pre>
			<p>Inline code: <code>fmt.Println()</code></p>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/code")
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Code blocks should be present
	assert.Contains(t, doc.Content, "```")
	assert.Contains(t, doc.Content, "func main()")
	assert.Contains(t, doc.Content, "`fmt.Println()`")
}

func TestPipeline_HashConsistency(t *testing.T) {
	html := `<html><body><h1>Test</h1><p>Content</p></body></html>`
	url := "https://example.com/hash-test"

	doc1, err := converter.ConvertHTML(html, url)
	require.NoError(t, err)

	doc2, err := converter.ConvertHTML(html, url)
	require.NoError(t, err)

	// Same input should produce same hash
	assert.Equal(t, doc1.ContentHash, doc2.ContentHash)
	assert.NotEmpty(t, doc1.ContentHash)
}

func TestPipeline_DifferentContentDifferentHash(t *testing.T) {
	html1 := `<html><body><h1>Test 1</h1><p>Content 1</p></body></html>`
	html2 := `<html><body><h1>Test 2</h1><p>Content 2</p></body></html>`
	url := "https://example.com/hash-test"

	doc1, err := converter.ConvertHTML(html1, url)
	require.NoError(t, err)

	doc2, err := converter.ConvertHTML(html2, url)
	require.NoError(t, err)

	// Different content should produce different hashes
	assert.NotEqual(t, doc1.ContentHash, doc2.ContentHash)
}

func TestPipeline_WordAndCharCount(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Count Test</title></head>
	<body>
		<article>
			<p>One two three four five.</p>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/count")
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Should have word count > 0
	assert.Greater(t, doc.WordCount, 0)
	// Should have char count > word count
	assert.Greater(t, doc.CharCount, doc.WordCount)
}

func TestMarkdownConverter_CleanMarkdown(t *testing.T) {
	// Test that excessive blank lines are removed
	opts := converter.DefaultMarkdownOptions()
	mc := converter.NewMarkdownConverter(opts)

	// HTML that might produce excessive blank lines
	html := `<html><body>
		<h1>Title</h1>


		<p>Content</p>


		<p>More content</p>
	</body></html>`

	result, err := mc.Convert(html)
	require.NoError(t, err)

	// Should not have 4+ consecutive newlines
	assert.NotContains(t, result, "\n\n\n\n")
}

func TestGenerateFrontmatter(t *testing.T) {
	// Create a mock document using ConvertHTML
	html := `<!DOCTYPE html>
	<html>
	<head>
		<title>Frontmatter Test</title>
		<meta name="description" content="Test description">
	</head>
	<body>
		<article>
			<h1>Content Title</h1>
			<p>Some content here.</p>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/frontmatter")
	require.NoError(t, err)

	frontmatter, err := converter.GenerateFrontmatter(doc)
	require.NoError(t, err)

	// Frontmatter should be valid YAML wrapped in ---
	assert.True(t, len(frontmatter) > 0)
	assert.Contains(t, frontmatter, "---")
	assert.Contains(t, frontmatter, "url:")
	assert.Contains(t, frontmatter, "title:")
}

func TestAddFrontmatter(t *testing.T) {
	html := `<html><head><title>Test</title></head><body><h1>Hello</h1></body></html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/test")
	require.NoError(t, err)

	result, err := converter.AddFrontmatter(doc.Content, doc)
	require.NoError(t, err)

	// Result should start with frontmatter
	assert.True(t, len(result) > len(doc.Content))
	assert.Contains(t, result, "---")
	assert.Contains(t, result, doc.Content)
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"multiple words", "hello world test", 3},
		{"with punctuation", "hello, world! test.", 3},
		{"multiple spaces", "hello    world", 2},
		{"newlines", "hello\nworld\ntest", 3},
		{"tabs", "hello\tworld", 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.CountWords(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"simple string", "hello", 5},
		{"with spaces", "hello world", 11},
		// Note: CountChars uses len() which counts bytes, not runes
		// "Hello, 世界" has 7 ASCII bytes + 6 bytes for 2 Chinese chars = 13 bytes
		{"unicode counts bytes", "Hello, 世界", 13},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.CountChars(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "remove bold",
			input:    "This is **bold** text",
			contains: []string{"bold", "text"},
			excludes: []string{"**"},
		},
		{
			name:     "remove italic",
			input:    "This is *italic* text",
			contains: []string{"italic", "text"},
		},
		{
			name:     "remove headers",
			input:    "# Header\n\nContent",
			contains: []string{"Header", "Content"},
			excludes: []string{"# "},
		},
		{
			name:     "remove links keep text",
			input:    "Click [here](https://example.com) for more",
			contains: []string{"here", "for more"},
			excludes: []string{"https://example.com"},
		},
		{
			name:     "remove code blocks",
			input:    "Text\n```go\nfunc main() {}\n```\nMore text",
			contains: []string{"Text", "More text"},
			excludes: []string{"```"},
		},
		{
			name:     "remove list markers",
			input:    "- Item 1\n- Item 2",
			contains: []string{"Item 1", "Item 2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.StripMarkdown(tc.input)
			for _, want := range tc.contains {
				assert.Contains(t, got, want)
			}
			for _, notWant := range tc.excludes {
				assert.NotContains(t, got, notWant)
			}
		})
	}
}
