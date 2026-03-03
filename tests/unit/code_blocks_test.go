package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseHTML(t *testing.T, html string) *goquery.Document {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	require.NoError(t, err)
	return doc
}

// --- PreserveCodeLanguages / RestoreCodeLanguages ---

func TestPreserveAndRestoreCodeLanguages(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantLang string
	}{
		{
			name:     "language- prefix preserved",
			html:     `<pre><code class="language-go">fmt.Println("hi")</code></pre>`,
			wantLang: "go",
		},
		{
			name:     "lang- prefix preserved",
			html:     `<pre><code class="lang-python">print("hi")</code></pre>`,
			wantLang: "python",
		},
		{
			name:     "data-language preserved",
			html:     `<pre><code data-language="rust">fn main() {}</code></pre>`,
			wantLang: "rust",
		},
		{
			name:     "data-lang preserved",
			html:     `<pre><code data-lang="typescript">const x = 1</code></pre>`,
			wantLang: "typescript",
		},
		{
			name:     "bare class preserved",
			html:     `<pre><code class="python">print("hi")</code></pre>`,
			wantLang: "python",
		},
		{
			name:     "hljs combined class preserved",
			html:     `<pre><code class="hljs javascript">var x = 1;</code></pre>`,
			wantLang: "javascript",
		},
		{
			name:     "no language - no attribute set",
			html:     `<pre><code>plain code</code></pre>`,
			wantLang: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseHTML(t, tc.html)

			converter.PreserveCodeLanguages(doc.Selection)

			// Check data-repodocs-lang was set
			code := doc.Find("pre code")
			lang, exists := code.Attr("data-repodocs-lang")
			if tc.wantLang == "" {
				assert.False(t, exists, "should not set data-repodocs-lang for no-language code")
			} else {
				assert.True(t, exists)
				assert.Equal(t, tc.wantLang, lang)
			}

			// Now restore
			converter.RestoreCodeLanguages(doc.Selection)

			if tc.wantLang != "" {
				class, _ := code.Attr("class")
				assert.Contains(t, class, "language-"+tc.wantLang)
				_, hasAttr := code.Attr("data-repodocs-lang")
				assert.False(t, hasAttr, "data-repodocs-lang should be removed after restore")
			}
		})
	}
}

// --- NormalizeCodeLanguages ---

func TestNormalizeCodeLanguages(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantClass string
	}{
		{
			name:      "data-language normalized",
			html:      `<code data-language="rust">code</code>`,
			wantClass: "language-rust",
		},
		{
			name:      "data-lang normalized",
			html:      `<code data-lang="go">code</code>`,
			wantClass: "language-go",
		},
		{
			name:      "bare class normalized",
			html:      `<code class="python">code</code>`,
			wantClass: "python language-python",
		},
		{
			name:      "hljs + language class normalized",
			html:      `<code class="hljs ruby">code</code>`,
			wantClass: "hljs ruby language-ruby",
		},
		{
			name:      "already has language- prefix - skip",
			html:      `<code class="language-go">code</code>`,
			wantClass: "language-go",
		},
		{
			name:      "already has lang- prefix - skip",
			html:      `<code class="lang-python">code</code>`,
			wantClass: "lang-python",
		},
		{
			name:      "unknown bare class - not normalized",
			html:      `<code class="myCustomClass">code</code>`,
			wantClass: "myCustomClass",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseHTML(t, tc.html)
			converter.NormalizeCodeLanguages(doc.Selection)

			code := doc.Find("code")
			class, _ := code.Attr("class")
			assert.Equal(t, tc.wantClass, class)
		})
	}
}

// --- StripLineNumbers ---

func TestStripLineNumbers(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantContain string
		wantExclude string
	}{
		{
			name:        "span ln class removed",
			html:        `<pre><code><span class="ln">1</span>import os<br/><span class="ln">2</span>print("hi")</code></pre>`,
			wantContain: "import os",
			wantExclude: ">1<",
		},
		{
			name:        "span line-number class removed",
			html:        `<pre><code><span class="line-number">1</span>x = 1</code></pre>`,
			wantContain: "x = 1",
			wantExclude: "line-number",
		},
		{
			name:        "span linenumber class removed",
			html:        `<pre><code><span class="linenumber">42</span>code here</code></pre>`,
			wantContain: "code here",
			wantExclude: "linenumber",
		},
		{
			name:        "regular spans preserved",
			html:        `<pre><code><span class="keyword">func</span> main()</code></pre>`,
			wantContain: "keyword",
			wantExclude: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseHTML(t, tc.html)
			converter.StripLineNumbers(doc.Selection)

			html, err := doc.Html()
			require.NoError(t, err)

			if tc.wantContain != "" {
				assert.Contains(t, html, tc.wantContain)
			}
			if tc.wantExclude != "" {
				assert.NotContains(t, html, tc.wantExclude)
			}
		})
	}
}

// --- CleanEmptyCodeBlocks ---

func TestCleanEmptyCodeBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty code block removed",
			input: "text before\n\n```\n\n```\n\ntext after",
			want:  "text before\n\n\ntext after",
		},
		{
			name:  "empty code block with language removed",
			input: "before\n\n```go\n\n```\n\nafter",
			want:  "before\n\n\nafter",
		},
		{
			name:  "non-empty code block preserved",
			input: "```go\nfunc main() {}\n```",
			want:  "```go\nfunc main() {}\n```",
		},
		{
			name:  "no code blocks unchanged",
			input: "just regular text\nwith newlines",
			want:  "just regular text\nwith newlines",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.CleanEmptyCodeBlocks(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- Code-aware sanitizer ---

func TestSanitizer_PreservesCodeInFooter(t *testing.T) {
	html := `<div>
		<h1>Title</h1>
		<p>Content</p>
		<footer>
			<pre><code class="language-go">fmt.Println("in footer")</code></pre>
		</footer>
	</div>`

	doc := parseHTML(t, html)
	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	cleanDoc, err := sanitizer.SanitizeDocument(doc)
	require.NoError(t, err)

	result, _ := cleanDoc.Html()
	assert.Contains(t, result, "fmt.Println", "code block inside footer should be preserved")
	assert.NotContains(t, result, "<footer>", "footer tag itself should be removed")
}

func TestSanitizer_PreservesCodeInAside(t *testing.T) {
	html := `<div>
		<h1>Title</h1>
		<aside>
			<p>Note:</p>
			<pre><code class="language-python">print("example")</code></pre>
		</aside>
		<p>Main content</p>
	</div>`

	doc := parseHTML(t, html)
	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	cleanDoc, err := sanitizer.SanitizeDocument(doc)
	require.NoError(t, err)

	result, _ := cleanDoc.Html()
	assert.Contains(t, result, "print(", "code block inside aside should be preserved")
	assert.Contains(t, result, "example", "code content inside aside should be preserved")
}

func TestSanitizer_PreservesCodeInNavClass(t *testing.T) {
	html := `<div>
		<div class="sidebar">
			<pre><code>sidebar code example</code></pre>
		</div>
		<p>Main content</p>
	</div>`

	doc := parseHTML(t, html)
	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	cleanDoc, err := sanitizer.SanitizeDocument(doc)
	require.NoError(t, err)

	result, _ := cleanDoc.Html()
	assert.Contains(t, result, "sidebar code example", "code block inside sidebar should be preserved")
}

func TestSanitizer_RemovesFooterWithoutCode(t *testing.T) {
	html := `<div>
		<p>Content</p>
		<footer><p>Copyright 2024</p></footer>
	</div>`

	doc := parseHTML(t, html)
	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	cleanDoc, err := sanitizer.SanitizeDocument(doc)
	require.NoError(t, err)

	result, _ := cleanDoc.Html()
	assert.NotContains(t, result, "Copyright", "footer without code should still be removed")
}

// --- End-to-end pipeline tests ---

func TestPipeline_CodeLanguagePreservedThroughReadability(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Code Language Test</title></head>
	<body>
		<article>
			<h1>Go Tutorial</h1>
			<p>Here is a Go code example that demonstrates the basics of the language.</p>
			<pre><code class="language-go">package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}</code></pre>
			<p>This example shows a basic Go program with fmt package usage and function declaration.</p>
		</article>
	</body>
	</html>`

	// No selector = Readability path
	doc, err := converter.ConvertHTML(html, "https://example.com/go-tutorial")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "```go", "language annotation should be preserved through Readability")
	assert.Contains(t, doc.Content, `fmt.Println("Hello, World!")`)
}

func TestPipeline_DataLanguageNormalized(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Hugo Site</title></head>
	<body>
		<article>
			<h1>Rust Guide</h1>
			<p>A comprehensive guide to Rust programming language fundamentals and advanced features.</p>
			<pre><code data-language="rust">fn main() {
    println!("Hello from Rust!");
}</code></pre>
			<p>This Rust code demonstrates the basic structure of a Rust program with macro usage.</p>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/rust-guide")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "```rust", "data-language should be normalized to language annotation")
}

func TestPipeline_BareClassNormalized(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Python Examples</title></head>
	<body>
		<article>
			<h1>Python Basics</h1>
			<p>Learn Python programming from scratch with practical examples and real-world usage.</p>
			<pre><code class="python">def hello():
    print("Hello, World!")

hello()</code></pre>
			<p>This Python code demonstrates function definition and calling in Python programming.</p>
		</article>
	</body>
	</html>`

	doc, err := converter.ConvertHTML(html, "https://example.com/python-basics")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "```python", "bare class should be normalized to language annotation")
}

func TestPipeline_CodeInFooterPreserved(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Footer Code Test</title></head>
	<body>
		<article>
			<h1>Examples</h1>
			<p>Main article content explaining the core concepts and providing detailed examples.</p>
			<footer>
				<pre><code class="language-bash">npm install package</code></pre>
			</footer>
		</article>
	</body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/footer-code")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "npm install", "code block in footer should be preserved")
}

func TestPipeline_LineNumbersStripped(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Line Numbers Test</title></head>
	<body>
		<article>
			<h1>Code With Line Numbers</h1>
			<p>This is a detailed code example with line numbers showing file structure and imports.</p>
			<pre><code class="language-go"><span class="ln">1</span>package main
<span class="ln">2</span>
<span class="ln">3</span>import "fmt"
<span class="ln">4</span>
<span class="ln">5</span>func main() {
<span class="ln">6</span>	fmt.Println("test")
<span class="ln">7</span>}</code></pre>
			<p>The above code shows a complete Go program with main function and package declaration.</p>
		</article>
	</body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/line-numbers")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "package main")
	assert.Contains(t, doc.Content, `fmt.Println("test")`)
	// Line numbers should not be present in the output
	assert.NotContains(t, doc.Content, "1package")
	assert.NotContains(t, doc.Content, "3import")
}

func TestPipeline_EmptyCodeBlocksRemoved(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Empty Code Test</title></head>
	<body>
		<article>
			<h1>Title</h1>
			<p>Some content before the empty block to ensure readability works properly.</p>
			<pre><code></code></pre>
			<p>Some content after the empty block to ensure conversion works properly.</p>
		</article>
	</body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/empty-code")
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Should not have empty code fences
	lines := strings.Split(doc.Content, "\n")
	inEmptyBlock := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") && !inEmptyBlock {
			inEmptyBlock = true
			continue
		}
		if inEmptyBlock && strings.HasPrefix(trimmed, "```") {
			// Check if the block was empty (only whitespace between fences)
			var blockContent strings.Builder
			for j := i - 1; j >= 0; j-- {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
					break
				}
				blockContent.WriteString(lines[j])
			}
			assert.NotEmpty(t, strings.TrimSpace(blockContent.String()), "should not have empty code blocks")
			inEmptyBlock = false
		}
	}
}

func TestPipeline_MultipleCodeBlocksPreserved(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Multiple Code Blocks</title></head>
	<body>
		<article>
			<h1>Multi-Language Examples</h1>
			<p>Here are examples in different programming languages to demonstrate language detection.</p>
			<pre><code class="language-go">fmt.Println("Go")</code></pre>
			<pre><code class="language-python">print("Python")</code></pre>
			<pre><code data-language="rust">println!("Rust")</code></pre>
			<pre><code class="javascript">console.log("JS")</code></pre>
			<p>All the above examples demonstrate basic output in four different languages.</p>
		</article>
	</body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/multi")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "```go")
	assert.Contains(t, doc.Content, "```python")
	assert.Contains(t, doc.Content, "```rust")
	assert.Contains(t, doc.Content, "```javascript")
}

func TestPipeline_SyntaxHighlightedSpansPreserved(t *testing.T) {
	html := `<!DOCTYPE html>
	<html>
	<head><title>Syntax Highlighted</title></head>
	<body>
		<article>
			<h1>Highlighted Code</h1>
			<p>This code example uses syntax highlighting spans for Go programming language.</p>
			<pre><code class="language-go"><span class="hljs-keyword">func</span> <span class="hljs-title">main</span>() {
	fmt.Println(<span class="hljs-string">"hello"</span>)
}</code></pre>
			<p>The highlighting shows keywords, function names, and string literals distinctly.</p>
		</article>
	</body>
	</html>`

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: "article",
	})

	doc, err := pipeline.Convert(context.Background(), html, "https://example.com/highlighted")
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Contains(t, doc.Content, "```go")
	assert.Contains(t, doc.Content, "func main()")
	assert.Contains(t, doc.Content, `fmt.Println("hello")`)
}
