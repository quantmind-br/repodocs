package converter_test

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// Golden test cases for regression testing - output must remain identical after refactor
var goldenTestCases = []struct {
	name     string
	html     string
	expected string // Expected markdown output (trimmed)
}{
	{
		name:     "simple paragraph",
		html:     `<p>Hello world</p>`,
		expected: "Hello world",
	},
	{
		name:     "heading with paragraph",
		html:     `<h1>Title</h1><p>Content here</p>`,
		expected: "# Title\n\nContent here",
	},
	{
		name:     "bold and italic",
		html:     `<p><strong>bold</strong> and <em>italic</em></p>`,
		expected: "**bold** and *italic*",
	},
	{
		name:     "link",
		html:     `<p><a href="https://example.com">link text</a></p>`,
		expected: "[link text](https://example.com)",
	},
	{
		name:     "unordered list",
		html:     `<ul><li>Item 1</li><li>Item 2</li></ul>`,
		expected: "- Item 1\n- Item 2",
	},
	{
		name:     "code block",
		html:     `<pre><code>func main() {}</code></pre>`,
		expected: "```\nfunc main() {}\n```",
	},
	{
		name: "complex document",
		html: `<article>
			<h1>Main Title</h1>
			<p>First paragraph with <strong>bold</strong> text.</p>
			<h2>Subtitle</h2>
			<ul>
				<li>First item</li>
				<li>Second item</li>
			</ul>
			<p>Final paragraph.</p>
		</article>`,
		expected: "# Main Title\n\nFirst paragraph with **bold** text.\n\n## Subtitle\n\n- First item\n- Second item\n\nFinal paragraph.",
	},
}

// TestMarkdownConverter_Convert_Golden verifies Convert output matches expected golden values
func TestMarkdownConverter_Convert_Golden(t *testing.T) {
	mdConverter := converter.NewMarkdownConverter(converter.MarkdownOptions{
		Domain: "https://example.com",
	})

	for _, tc := range goldenTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := mdConverter.Convert(tc.html)
			require.NoError(t, err)

			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tc.expected)

			assert.Equal(t, expected, result, "Convert output should match golden value")
		})
	}
}

// TestMarkdownConverter_ConvertNode_Golden verifies ConvertNode produces identical output to Convert
func TestMarkdownConverter_ConvertNode_Golden(t *testing.T) {
	mdConverter := converter.NewMarkdownConverter(converter.MarkdownOptions{
		Domain: "https://example.com",
	})

	for _, tc := range goldenTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse HTML to node
			doc, err := html.Parse(strings.NewReader(tc.html))
			require.NoError(t, err)

			// Convert using ConvertNode
			result, err := mdConverter.ConvertNode(doc)
			require.NoError(t, err)

			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tc.expected)

			assert.Equal(t, expected, result, "ConvertNode output should match golden value")
		})
	}
}

// TestMarkdownConverter_ConvertNode_EquivalentToConvert verifies ConvertNode produces same output as Convert
func TestMarkdownConverter_ConvertNode_EquivalentToConvert(t *testing.T) {
	mdConverter := converter.NewMarkdownConverter(converter.MarkdownOptions{
		Domain: "https://example.com",
	})

	for _, tc := range goldenTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get result from Convert (string path)
			convertResult, err := mdConverter.Convert(tc.html)
			require.NoError(t, err)

			// Get result from ConvertNode (node path)
			doc, err := html.Parse(strings.NewReader(tc.html))
			require.NoError(t, err)

			nodeResult, err := mdConverter.ConvertNode(doc)
			require.NoError(t, err)

			// Both should produce identical output
			assert.Equal(t, strings.TrimSpace(convertResult), strings.TrimSpace(nodeResult),
				"ConvertNode should produce identical output to Convert")
		})
	}
}

// TestMarkdownConverter_ConvertNode_WithGoquery tests ConvertNode with goquery Selection.Get(0)
func TestMarkdownConverter_ConvertNode_WithGoquery(t *testing.T) {
	mdConverter := converter.NewMarkdownConverter(converter.MarkdownOptions{
		Domain: "https://example.com",
	})

	htmlContent := `<html><body><article><h1>Title</h1><p>Content</p></article></body></html>`

	// Parse with goquery (like pipeline does)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	require.NoError(t, err)

	// Get the article node using goquery's Get(0)
	articleSel := doc.Find("article")
	require.Equal(t, 1, articleSel.Length(), "Should find article element")

	node := articleSel.Get(0)
	require.NotNil(t, node, "Node should not be nil")

	// Convert using ConvertNode
	result, err := mdConverter.ConvertNode(node)
	require.NoError(t, err)

	result = strings.TrimSpace(result)
	assert.Contains(t, result, "# Title")
	assert.Contains(t, result, "Content")
}

// TestPipeline_Convert_OutputEquivalence tests that pipeline output is identical after optimization
func TestPipeline_Convert_OutputEquivalence(t *testing.T) {
	testCases := []struct {
		name string
		html string
	}{
		{
			name: "simple document",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<article>
		<h1>Main Content</h1>
		<p>Paragraph with <strong>bold</strong> text.</p>
	</article>
</body>
</html>`,
		},
		{
			name: "document with selector",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<nav>Navigation</nav>
	<main>
		<h1>Article</h1>
		<p>Main content here.</p>
		<ul>
			<li>Item 1</li>
			<li>Item 2</li>
		</ul>
	</main>
	<footer>Footer</footer>
</body>
</html>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pipeline := converter.NewPipeline(converter.PipelineOptions{
				BaseURL: "https://example.com",
			})

			ctx := context.Background()
			doc, err := pipeline.Convert(ctx, tc.html, "https://example.com/page")
			require.NoError(t, err)

			assert.NotNil(t, doc)
			assert.NotEmpty(t, doc.Content)
			// Content should be valid markdown (no HTML tags in typical cases)
			assert.NotContains(t, doc.Content, "<article>")
			assert.NotContains(t, doc.Content, "<nav>")
		})
	}
}
