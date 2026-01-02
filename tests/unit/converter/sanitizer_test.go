package converter_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSanitizer tests creating a new sanitizer
func TestNewSanitizer(t *testing.T) {
	tests := []struct {
		name    string
		opts    converter.SanitizerOptions
		wantNil bool
	}{
		{
			name: "basic sanitizer",
			opts: converter.SanitizerOptions{
				BaseURL: "https://example.com",
			},
			wantNil: false,
		},
		{
			name: "with all options",
			opts: converter.SanitizerOptions{
				BaseURL:          "https://example.com",
				RemoveNavigation: true,
				RemoveComments:   true,
			},
			wantNil: false,
		},
		{
			name:    "empty options",
			opts:    converter.SanitizerOptions{},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(tt.opts)

			if tt.wantNil {
				assert.Nil(t, sanitizer)
			} else {
				assert.NotNil(t, sanitizer)
			}
		})
	}
}

// TestSanitizer_RemoveScripts tests removal of script tags
func TestSanitizer_RemoveScripts(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script>
		console.log("test");
	</script>
</head>
<body>
	<h1>Title</h1>
	<script>alert("remove me");</script>
	<p>Content</p>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "console.log")
	assert.NotContains(t, result, "alert")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

// TestSanitizer_RemoveStyles tests removal of style tags
func TestSanitizer_RemoveStyles(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<style>
		body { color: red; }
	</style>
</head>
<body>
	<h1>Title</h1>
	<style>.more-style { display: none; }</style>
	<p>Content</p>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "color: red")
	assert.NotContains(t, result, "more-style")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

// TestSanitizer_RemoveNavigation tests removal of navigation elements
func TestSanitizer_RemoveNavigation(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<nav class="navigation">
		<ul>
			<li><a href="/">Home</a></li>
		</ul>
	</nav>
	<header class="header">Header</header>
	<main>
		<h1>Main Content</h1>
	</main>
	<footer class="footer">Footer</footer>
	<aside class="sidebar">Sidebar</aside>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL:          "https://example.com",
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "Home")
	assert.NotContains(t, result, "Header")
	assert.NotContains(t, result, "Footer")
	assert.NotContains(t, result, "Sidebar")
	assert.Contains(t, result, "Main Content")
}

// TestSanitizer_RemoveNavigationDisabled tests keeping navigation when disabled
func TestSanitizer_RemoveNavigationDisabled(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<nav class="navigation">
		<ul>
			<li><a href="/">Home</a></li>
		</ul>
	</nav>
	<main>
		<h1>Main Content</h1>
	</main>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL:          "https://example.com",
		RemoveNavigation: false,
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.Contains(t, result, "Home")
	assert.Contains(t, result, "Main Content")
}

// TestSanitizer_RemoveFormElements tests removal of form elements
func TestSanitizer_RemoveFormElements(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Title</h1>
	<form action="/submit">
		<input type="text" name="username">
		<button type="submit">Submit</button>
		<select name="choice">
			<option>Option 1</option>
		</select>
		<textarea name="comment"></textarea>
	</form>
	<p>Content</p>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "form")
	assert.NotContains(t, result, "input")
	assert.NotContains(t, result, "button")
	assert.NotContains(t, result, "select")
	assert.NotContains(t, result, "textarea")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

// TestSanitizer_RemoveIframesAndEmbeds tests removal of iframes and embeds
func TestSanitizer_RemoveIframesAndEmbeds(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Title</h1>
	<iframe src="https://example.com/frame"></iframe>
	<embed src="movie.swf">
	<object data="file.pdf"></object>
	<applet code="MyApplet.class"></applet>
	<p>Content</p>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "iframe")
	assert.NotContains(t, result, "embed")
	assert.NotContains(t, result, "object")
	assert.NotContains(t, result, "applet")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Content")
}

// TestSanitizer_RemoveHiddenElements tests removal of hidden elements
func TestSanitizer_RemoveHiddenElements(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Title</h1>
	<div style="display:none">Hidden content</div>
	<p style="display: none;">Also hidden</p>
	<p hidden>Hidden via attribute</p>
	<p>Visible content</p>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "Hidden content")
	assert.NotContains(t, result, "Also hidden")
	assert.NotContains(t, result, "Hidden via attribute")
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Visible content")
}

// TestSanitizer_RemoveEmptyElements tests removal of empty elements
func TestSanitizer_RemoveEmptyElements(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Title</h1>
	<p></p>
	<div>   </div>
	<span></span>
	<p>Real content here.</p>
	<section></section>
	<article>   </article>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "Real content here")
	// Empty elements should be removed (no excessive blank lines in output)
}

// TestSanitizer_NormalizeURLs tests URL normalization
func TestSanitizer_NormalizeURLs(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		html          string
		expectedLinks []string
	}{
		{
			name:    "relative URLs",
			baseURL: "https://example.com/path/to/page",
			html: `<!DOCTYPE html>
<html>
<body>
	<a href="/absolute">Absolute Path</a>
	<a href="relative">Relative Path</a>
	<a href="../parent">Parent Path</a>
</body>
</html>`,
			expectedLinks: []string{
				"https://example.com/absolute",
				"https://example.com/path/to/relative",
				"https://example.com/path/parent",
			},
		},
		{
			name:    "absolute URLs unchanged",
			baseURL: "https://example.com",
			html: `<!DOCTYPE html>
<html>
<body>
	<a href="https://other.com/page">External</a>
	<a href="http://example.com/other">HTTP</a>
</body>
</html>`,
			expectedLinks: []string{
				"https://other.com/page",
				"http://example.com/other",
			},
		},
		{
			name:    "special URLs preserved",
			baseURL: "https://example.com",
			html: `<!DOCTYPE html>
<html>
<body>
	<a href="#anchor">Anchor</a>
	<a href="javascript:void(0)">JavaScript</a>
	<a href="mailto:test@example.com">Mailto</a>
	<a href="data:text/plain,data">Data</a>
</body>
</html>`,
			expectedLinks: []string{
				"#anchor",
				"javascript:void(0)",
				"mailto:test@example.com",
				"data:text/plain,data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: tt.baseURL,
			})

			result, err := sanitizer.Sanitize(tt.html)
			require.NoError(t, err)

			// Check that expected links are present
			for _, link := range tt.expectedLinks {
				assert.Contains(t, result, link)
			}
		})
	}
}

// TestSanitizer_NormalizeImages tests image URL normalization
func TestSanitizer_NormalizeImages(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<img src="/absolute.png" alt="Absolute">
	<img src="relative.jpg" alt="Relative">
	<img src="https://other.com/image.png" alt="External">
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com/path/to/page",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.Contains(t, result, "https://example.com/absolute.png")
	assert.Contains(t, result, "https://example.com/path/to/relative.jpg")
	assert.Contains(t, result, "https://other.com/image.png")
}

// TestSanitizer_SrcsetNormalization tests srcset attribute normalization
func TestSanitizer_SrcsetNormalization(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<img srcset="/image1.jpg 1x, /image2.jpg 2x" alt="Test">
	<img srcset="image3.jpg 100w, image4.jpg 200w" alt="Test2">
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com/path/",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	// Check that srcset URLs are normalized
	assert.Contains(t, result, "https://example.com/image1.jpg")
	assert.Contains(t, result, "https://example.com/image2.jpg")
	assert.Contains(t, result, "https://example.com/path/image3.jpg")
}

// TestSanitizer_EmptyHTML tests handling empty HTML
func TestSanitizer_EmptyHTML(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name:    "empty string",
			html:    "",
			wantErr: false,
		},
		{
			name:    "only whitespace",
			html:    "   \n\t  ",
			wantErr: false,
		},
		{
			name:    "minimal html",
			html:    "<html></html>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: "https://example.com",
			})

			result, err := sanitizer.Sanitize(tt.html)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestSanitizer_ComplexHTML tests sanitization of complex HTML
func TestSanitizer_ComplexHTML(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<script>var x = 1;</script>
	<style>body { margin: 0; }</style>
</head>
<body>
	<nav class="navigation">
		<ul><li><a href="/">Home</a></li></ul>
	</nav>
	<div class="advertisement">Ad content</div>
	<main>
		<h1>Main Article</h1>
		<p>Content here.</p>
		<form><input type="text"></form>
		<script>alert('remove');</script>
	</main>
	<footer class="footer">Footer</footer>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL:          "https://example.com",
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "var x = 1")
	assert.NotContains(t, result, "margin: 0")
	assert.NotContains(t, result, "Home")
	assert.NotContains(t, result, "Ad content")
	assert.NotContains(t, result, "Footer")
	assert.NotContains(t, result, "alert")
	assert.NotContains(t, result, "input")
	assert.Contains(t, result, "Main Article")
	assert.Contains(t, result, "Content here")
}

// TestSanitizer_PreserveContent tests that important content is preserved
func TestSanitizer_PreserveContent(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Main Title</h1>
	<h2>Subtitle</h2>
	<p>This is <strong>important</strong> content.</p>
	<ul>
		<li>List item 1</li>
		<li>List item 2</li>
	</ul>
	<code>code block</code>
	<pre><pre>more code</pre></pre>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.Contains(t, result, "Main Title")
	assert.Contains(t, result, "Subtitle")
	assert.Contains(t, result, "important")
	assert.Contains(t, result, "List item 1")
	assert.Contains(t, result, "code block")
}

// TestSanitizer_ClassAndIDRemoval tests removal by class and ID
func TestSanitizer_ClassAndIDRemoval(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<div class="sidebar">Sidebar</div>
	<div class="navigation">Nav</div>
	<div id="footer">Footer</div>
	<div id="header">Header</div>
	<div class="content">Keep this</div>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL:          "https://example.com",
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.NotContains(t, result, "Sidebar")
	assert.NotContains(t, result, "Nav")
	assert.NotContains(t, result, "Footer")
	assert.NotContains(t, result, "Header")
	assert.Contains(t, result, "Keep this")
}

// TestSanitizer_MalformedHTML tests handling malformed HTML
func TestSanitizer_MalformedHTML(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name:    "unclosed tags",
			html:    `<div><p>Content`,
			wantErr: false,
		},
		{
			name:    "invalid nesting",
			html:    `<div><p>Content</div></p>`,
			wantErr: false,
		},
		{
			name:    "completely invalid",
			html:    `<><<<>>>invalid<<<>>>`,
			wantErr: false, // goquery is forgiving
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: "https://example.com",
			})

			result, err := sanitizer.Sanitize(tt.html)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestSanitizer_NoBaseURL tests sanitizer without base URL
func TestSanitizer_NoBaseURL(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<a href="/path">Link</a>
	<img src="image.jpg" alt="Image">
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	// URLs should not be normalized without base URL
	assert.Contains(t, result, `href="/path"`)
	assert.Contains(t, result, `src="image.jpg"`)
}

// TestSanitizer_MixedContentTypes tests sanitizing HTML with various content types
func TestSanitizer_MixedContentTypes(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>
	<h1>Article Title</h1>
	<p>Text with <a href="/link">link</a> and <strong>bold</strong>.</p>
	<table>
		<tr><th>Header</th></tr>
		<tr><td>Data</td></tr>
	</table>
	<blockquote>Quote</blockquote>
	<hr>
	<ul><li>Item</li></ul>
	<ol><li>Numbered</li></ol>
</body>
</html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		BaseURL: "https://example.com",
	})

	result, err := sanitizer.Sanitize(html)

	require.NoError(t, err)
	assert.Contains(t, result, "Article Title")
	assert.Contains(t, result, "link")
	assert.Contains(t, result, "bold")
	assert.Contains(t, result, "Header")
	assert.Contains(t, result, "Data")
	assert.Contains(t, result, "Quote")
	assert.Contains(t, result, "Item")
	assert.Contains(t, result, "Numbered")
}
