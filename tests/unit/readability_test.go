package unit

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractContent_WithSelector(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		selector     string
		sourceURL    string
		wantContent  string
		wantTitle    string
		wantContains []string
	}{
		{
			name: "extract by class selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Test Page</title></head>
			<body>
				<div class="nav">Navigation</div>
				<div class="content">
					<h1>Main Content</h1>
					<p>Important text here.</p>
				</div>
				<div class="footer">Footer</div>
			</body>
			</html>`,
			selector:     ".content",
			sourceURL:    "https://example.com/test",
			wantTitle:    "Test Page",
			wantContains: []string{"Main Content", "Important text"},
		},
		{
			name: "extract by ID selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>ID Test</title></head>
			<body>
				<div id="main-article">
					<h1>Article Title</h1>
					<p>Article body content.</p>
				</div>
			</body>
			</html>`,
			selector:     "#main-article",
			sourceURL:    "https://example.com/article",
			wantTitle:    "ID Test",
			wantContains: []string{"Article Title", "Article body"},
		},
		{
			name: "extract by tag selector",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Tag Test</title></head>
			<body>
				<article>
					<h1>Article Heading</h1>
					<p>Article paragraph.</p>
				</article>
			</body>
			</html>`,
			selector:     "article",
			sourceURL:    "https://example.com/tag",
			wantTitle:    "Tag Test",
			wantContains: []string{"Article Heading", "Article paragraph"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := converter.NewExtractContent(tc.selector)
			content, title, err := extractor.Extract(tc.html, tc.sourceURL)

			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, title)

			for _, want := range tc.wantContains {
				assert.Contains(t, content, want)
			}
		})
	}
}

func TestExtractContent_SelectorFallbackToReadability(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Fallback Test</title></head>
	<body>
		<article>
			<h1>Real Content</h1>
			<p>This is the actual content that readability should extract.</p>
			<p>More paragraphs to help readability identify this as main content.</p>
			<p>Even more content to make it clearer this is the main article.</p>
		</article>
	</body>
	</html>`

	// Use a selector that doesn't exist
	extractor := converter.NewExtractContent(".nonexistent-class")
	content, title, err := extractor.Extract(html, "https://example.com/fallback")

	require.NoError(t, err)
	assert.Equal(t, "Fallback Test", title)
	// Readability should have extracted the article content
	assert.Contains(t, content, "Real Content")
}

func TestExtractContent_NoSelector(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>No Selector Test</title></head>
	<body>
		<main>
			<h1>Primary Content</h1>
			<p>This is the main content of the page.</p>
			<p>Additional paragraphs help readability.</p>
			<p>More text content here.</p>
		</main>
	</body>
	</html>`

	// Empty selector should use readability
	extractor := converter.NewExtractContent("")
	content, title, err := extractor.Extract(html, "https://example.com/no-selector")

	require.NoError(t, err)
	assert.Equal(t, "No Selector Test", title)
	assert.NotEmpty(t, content)
}

func TestExtractContent_ExtractBody(t *testing.T) {
	// Test the extractBody fallback by providing HTML that readability fails on
	tests := []struct {
		name      string
		html      string
		wantTitle string
	}{
		{
			name: "minimal html with body",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Body Test</title></head>
			<body>
				<p>Simple content</p>
			</body>
			</html>`,
			wantTitle: "Body Test",
		},
		{
			name: "no body tag",
			html: `<!DOCTYPE html>
			<html>
			<head><title>No Body</title></head>
			</html>`,
			wantTitle: "No Body",
		},
		{
			name: "empty body",
			html: `<!DOCTYPE html>
			<html>
			<head><title>Empty Body</title></head>
			<body></body>
			</html>`,
			wantTitle: "Empty Body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := converter.NewExtractContent("")
			_, title, err := extractor.Extract(tc.html, "https://example.com/body")

			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, title)
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantTitle string
	}{
		{
			name:      "title from title tag",
			html:      `<html><head><title>Title Tag</title></head><body><p>Content</p></body></html>`,
			wantTitle: "Title Tag",
		},
		{
			name:      "title from og:title meta",
			html:      `<html><head><meta property="og:title" content="OG Title"></head><body><p>Content</p></body></html>`,
			wantTitle: "OG Title",
		},
		{
			name:      "prefer title tag over h1",
			html:      `<html><head><title>Title Tag</title></head><body><h1>H1 Title</h1></body></html>`,
			wantTitle: "Title Tag",
		},
		{
			name:      "empty title returns empty string",
			html:      `<html><head></head><body><p>No title anywhere</p></body></html>`,
			wantTitle: "",
		},
		{
			name:      "title with whitespace is trimmed",
			html:      `<html><head><title>  Trimmed Title  </title></head><body></body></html>`,
			wantTitle: "Trimmed Title",
		},
		// Note: h1 fallback depends on goquery document being parsed with the body,
		// which may not work through readability. The title extraction from h1
		// is best-effort and may not work in all cases.
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := converter.NewExtractContent("")
			_, title, err := extractor.Extract(tc.html, "https://example.com/title")

			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, title)
		})
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "from meta description",
			html: `<html><head><meta name="description" content="Page description here"></head><body></body></html>`,
			want: "Page description here",
		},
		{
			name: "from og:description",
			html: `<html><head><meta property="og:description" content="OG description"></head><body></body></html>`,
			want: "OG description",
		},
		{
			name: "prefer meta description over og:description",
			html: `<html><head>
				<meta name="description" content="Meta description">
				<meta property="og:description" content="OG description">
			</head><body></body></html>`,
			want: "Meta description",
		},
		{
			name: "no description returns empty",
			html: `<html><head></head><body></body></html>`,
			want: "",
		},
		{
			name: "description with whitespace is trimmed",
			html: `<html><head><meta name="description" content="  Trimmed  "></head><body></body></html>`,
			want: "Trimmed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := converter.ConvertHTML(tc.html, "https://example.com/desc")
			require.NoError(t, err)
			assert.Equal(t, tc.want, doc.Description)
		})
	}
}

func TestExtractHeaders(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Headers Test</title></head>
	<body>
		<h1>Main Title</h1>
		<h2>Section One</h2>
		<p>Content</p>
		<h2>Section Two</h2>
		<p>More content</p>
		<h3>Subsection</h3>
		<p>Sub content</p>
		<h4>Deep Section</h4>
		<h5>Deeper</h5>
		<h6>Deepest</h6>
	</body>
	</html>`

	headers := converter.ExtractHeaders(html)

	assert.Contains(t, headers["h1"], "Main Title")
	assert.Contains(t, headers["h2"], "Section One")
	assert.Contains(t, headers["h2"], "Section Two")
	assert.Contains(t, headers["h3"], "Subsection")
	assert.Contains(t, headers["h4"], "Deep Section")
	assert.Contains(t, headers["h5"], "Deeper")
	assert.Contains(t, headers["h6"], "Deepest")
}

func TestReadability_ExtractLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		wantLen  int
		contains []string
		excludes []string
	}{
		{
			name: "absolute links",
			html: `<html><body>
				<a href="https://example.com/page1">Link 1</a>
				<a href="https://example.com/page2">Link 2</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  2,
			contains: []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name: "relative links resolved",
			html: `<html><body>
				<a href="/docs/api">API Docs</a>
				<a href="./guide">Guide</a>
			</body></html>`,
			baseURL:  "https://example.com/page/",
			contains: []string{"https://example.com/docs/api", "https://example.com/page/guide"},
		},
		{
			name: "skip anchors",
			html: `<html><body>
				<a href="#section1">Section 1</a>
				<a href="https://example.com/real">Real Link</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  1,
			excludes: []string{"#section1"},
		},
		{
			name: "skip javascript links",
			html: `<html><body>
				<a href="javascript:void(0)">Click</a>
				<a href="https://example.com/real">Real Link</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  1,
			excludes: []string{"javascript:void(0)"},
		},
		{
			name: "skip mailto links",
			html: `<html><body>
				<a href="mailto:test@example.com">Email</a>
				<a href="https://example.com/contact">Contact</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  1,
			excludes: []string{"mailto:test@example.com"},
		},
		{
			name: "skip tel links",
			html: `<html><body>
				<a href="tel:+1234567890">Call</a>
				<a href="https://example.com/contact">Contact</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  1,
			excludes: []string{"tel:+1234567890"},
		},
		{
			name: "empty href ignored",
			html: `<html><body>
				<a href="">Empty</a>
				<a href="https://example.com/real">Real</a>
			</body></html>`,
			baseURL:  "https://example.com/",
			wantLen:  1,
			excludes: []string{""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			links := converter.ExtractLinks(tc.html, tc.baseURL)

			if tc.wantLen > 0 {
				assert.Len(t, links, tc.wantLen)
			}

			for _, want := range tc.contains {
				assert.Contains(t, links, want)
			}

			for _, notWant := range tc.excludes {
				assert.NotContains(t, links, notWant)
			}
		})
	}
}

func TestExtractContent_InvalidURL(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Invalid URL Test</title></head>
	<body>
		<article>
			<h1>Content</h1>
			<p>Some content here.</p>
		</article>
	</body>
	</html>`

	// Invalid URL should not cause error - falls back to default
	extractor := converter.NewExtractContent("")
	content, title, err := extractor.Extract(html, "not-a-valid-url")

	require.NoError(t, err)
	assert.Equal(t, "Invalid URL Test", title)
	assert.NotEmpty(t, content)
}

func TestExtractContent_MalformedHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
	}{
		{
			name: "unclosed tags",
			html: `<html><body><h1>Title<p>Content</body>`,
		},
		{
			name: "missing html tag",
			html: `<body><h1>Title</h1><p>Content</p></body>`,
		},
		{
			name: "only text",
			html: `Just some plain text content`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			extractor := converter.NewExtractContent("")
			content, _, err := extractor.Extract(tc.html, "https://example.com/malformed")

			// Should not return an error for malformed HTML
			require.NoError(t, err)
			assert.NotEmpty(t, content)
		})
	}
}
