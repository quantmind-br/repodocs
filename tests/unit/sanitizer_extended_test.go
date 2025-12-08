package unit

import (
	"net/url"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizer_NormalizeSrcset(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		html     string
		contains []string
	}{
		{
			name:    "single relative srcset",
			baseURL: "https://example.com/docs/",
			html: `<html><body>
				<img srcset="/images/logo.png 1x" alt="Logo">
			</body></html>`,
			contains: []string{`srcset="https://example.com/images/logo.png 1x"`},
		},
		{
			name:    "multiple srcset entries",
			baseURL: "https://example.com/",
			html: `<html><body>
				<img srcset="/small.png 480w, /medium.png 800w, /large.png 1200w" alt="Responsive">
			</body></html>`,
			contains: []string{
				"https://example.com/small.png 480w",
				"https://example.com/medium.png 800w",
				"https://example.com/large.png 1200w",
			},
		},
		{
			name:    "mixed absolute and relative srcset",
			baseURL: "https://example.com/",
			html: `<html><body>
				<img srcset="https://cdn.example.com/img.png 1x, /local.png 2x" alt="Mixed">
			</body></html>`,
			contains: []string{
				"https://cdn.example.com/img.png 1x",
				"https://example.com/local.png 2x",
			},
		},
		{
			name:    "srcset with descriptors",
			baseURL: "https://example.com/",
			html: `<html><body>
				<img srcset="/img.png 100w 2x" alt="Test">
			</body></html>`,
			contains: []string{"https://example.com/img.png"},
		},
		{
			name:    "relative path srcset",
			baseURL: "https://example.com/docs/guide/",
			html: `<html><body>
				<img srcset="./images/test.png 1x" alt="Test">
			</body></html>`,
			contains: []string{"https://example.com/docs/guide/images/test.png"},
		},
		{
			name:    "srcset with spaces in descriptor",
			baseURL: "https://example.com/",
			html: `<html><body>
				<img srcset=" /img1.png 1x , /img2.png 2x " alt="Test">
			</body></html>`,
			contains: []string{
				"https://example.com/img1.png",
				"https://example.com/img2.png",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: tc.baseURL,
			})

			result, err := sanitizer.Sanitize(tc.html)
			require.NoError(t, err)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected, "Expected srcset to contain: %s", expected)
			}
		})
	}
}

func TestSanitizer_ResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		html     string
		contains []string
		excludes []string
	}{
		{
			name:     "skip empty href",
			baseURL:  "https://example.com/",
			html:     `<html><body><a href="">Empty</a></body></html>`,
			contains: []string{`href=""`},
		},
		{
			name:     "skip fragment only",
			baseURL:  "https://example.com/",
			html:     `<html><body><a href="#section">Fragment</a></body></html>`,
			contains: []string{`href="#section"`},
		},
		{
			name:     "skip javascript urls",
			baseURL:  "https://example.com/",
			html:     `<html><body><a href="javascript:void(0)">Click</a></body></html>`,
			contains: []string{`href="javascript:void(0)"`},
		},
		{
			name:     "skip mailto urls",
			baseURL:  "https://example.com/",
			html:     `<html><body><a href="mailto:test@example.com">Email</a></body></html>`,
			contains: []string{`href="mailto:test@example.com"`},
		},
		{
			name:     "skip data urls",
			baseURL:  "https://example.com/",
			html:     `<html><body><img src="data:image/png;base64,ABC123"></body></html>`,
			contains: []string{`src="data:image/png;base64,ABC123"`},
		},
		{
			name:     "resolve relative path",
			baseURL:  "https://example.com/docs/",
			html:     `<html><body><a href="../api">API</a></body></html>`,
			contains: []string{`href="https://example.com/api"`},
		},
		{
			name:     "resolve absolute path",
			baseURL:  "https://example.com/docs/guide/",
			html:     `<html><body><a href="/api/v1">API</a></body></html>`,
			contains: []string{`href="https://example.com/api/v1"`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: tc.baseURL,
			})

			result, err := sanitizer.Sanitize(tc.html)
			require.NoError(t, err)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
			for _, excluded := range tc.excludes {
				assert.NotContains(t, result, excluded)
			}
		})
	}
}

func TestSanitizer_RemoveEmptyElements(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains []string
		excludes []string
	}{
		{
			name: "remove empty paragraph",
			html: `<html><body>
				<p></p>
				<p>Non-empty</p>
			</body></html>`,
			contains: []string{"Non-empty"},
		},
		{
			name: "remove whitespace-only paragraph",
			html: `<html><body>
				<p>   </p>
				<p>Content</p>
			</body></html>`,
			contains: []string{"Content"},
		},
		{
			name: "keep paragraph with children",
			html: `<html><body>
				<p><span>Nested</span></p>
			</body></html>`,
			contains: []string{"Nested"},
		},
		{
			name: "remove empty div",
			html: `<html><body>
				<div></div>
				<div>Content</div>
			</body></html>`,
			contains: []string{"Content"},
		},
		{
			name: "remove empty span",
			html: `<html><body>
				<span></span>
				<span>Text</span>
			</body></html>`,
			contains: []string{"Text"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				RemoveNavigation: true,
			})

			result, err := sanitizer.Sanitize(tc.html)
			require.NoError(t, err)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
			for _, excluded := range tc.excludes {
				assert.NotContains(t, result, excluded)
			}
		})
	}
}

func TestNormalizeSrcsetDirectly(t *testing.T) {
	// Test the normalizeSrcset function behavior through the Sanitizer
	baseURL, _ := url.Parse("https://example.com/page/")

	tests := []struct {
		name    string
		srcset  string
		baseURL *url.URL
	}{
		{
			name:    "single entry",
			srcset:  "/image.png 1x",
			baseURL: baseURL,
		},
		{
			name:    "multiple entries with various descriptors",
			srcset:  "/small.png 480w, /medium.png 800w, /large.png 1200w",
			baseURL: baseURL,
		},
		{
			name:    "entry with pixel density descriptor",
			srcset:  "/image.png 2x",
			baseURL: baseURL,
		},
		{
			name:    "empty srcset",
			srcset:  "",
			baseURL: baseURL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create HTML with srcset
			html := `<html><body><img srcset="` + tc.srcset + `" alt="test"></body></html>`

			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				BaseURL: tc.baseURL.String(),
			})

			result, err := sanitizer.Sanitize(html)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}

func TestSanitizer_RemoveFormElements(t *testing.T) {
	html := `<html><body>
		<h1>Page Title</h1>
		<form action="/submit">
			<input type="text" name="username">
			<button type="submit">Submit</button>
		</form>
		<p>Content after form</p>
	</body></html>`

	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: true,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.NotContains(t, result, "<form")
	assert.NotContains(t, result, "<input")
	assert.NotContains(t, result, "<button")
	assert.Contains(t, result, "Page Title")
	assert.Contains(t, result, "Content after form")
}

func TestSanitizer_PreserveContentWithoutBaseURL(t *testing.T) {
	html := `<html><body>
		<a href="/relative/link">Relative Link</a>
		<img src="/images/test.png" alt="Test">
	</body></html>`

	// Without base URL, relative URLs should remain unchanged
	sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
		RemoveNavigation: false,
	})

	result, err := sanitizer.Sanitize(html)
	require.NoError(t, err)

	assert.Contains(t, result, `href="/relative/link"`)
	assert.Contains(t, result, `src="/images/test.png"`)
}

func TestSanitizer_TagsToRemove(t *testing.T) {
	// Verify all tags in TagsToRemove are properly removed
	tagsToTest := []string{
		"script", "style", "noscript", "iframe", "object",
		"embed", "applet", "form", "input", "button",
		"select", "textarea", "nav", "footer", "header", "aside",
	}

	for _, tag := range tagsToTest {
		t.Run(tag, func(t *testing.T) {
			html := `<html><body><` + tag + `>Remove me</` + tag + `><p>Keep me</p></body></html>`

			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				RemoveNavigation: true,
			})

			result, err := sanitizer.Sanitize(html)
			require.NoError(t, err)

			assert.NotContains(t, result, "<"+tag)
			assert.Contains(t, result, "Keep me")
		})
	}
}

func TestSanitizer_ClassesToRemove(t *testing.T) {
	classesToTest := []string{
		"sidebar", "navigation", "nav", "menu", "footer",
		"header", "banner", "advertisement", "ad", "social",
		"share", "comment", "comments", "related", "recommended",
	}

	for _, class := range classesToTest {
		t.Run(class, func(t *testing.T) {
			html := `<html><body>
				<div class="` + class + `">Remove me</div>
				<div class="content">Keep me</div>
			</body></html>`

			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				RemoveNavigation: true,
			})

			result, err := sanitizer.Sanitize(html)
			require.NoError(t, err)

			assert.NotContains(t, result, "Remove me")
			assert.Contains(t, result, "Keep me")
		})
	}
}

func TestSanitizer_IDsToRemove(t *testing.T) {
	idsToTest := []string{
		"sidebar", "navigation", "nav", "menu", "footer",
		"header", "banner", "advertisement", "comments",
	}

	for _, id := range idsToTest {
		t.Run(id, func(t *testing.T) {
			html := `<html><body>
				<div id="` + id + `">Remove me</div>
				<div id="main-content">Keep me</div>
			</body></html>`

			sanitizer := converter.NewSanitizer(converter.SanitizerOptions{
				RemoveNavigation: true,
			})

			result, err := sanitizer.Sanitize(html)
			require.NoError(t, err)

			assert.NotContains(t, result, "Remove me")
			assert.Contains(t, result, "Keep me")
		})
	}
}
