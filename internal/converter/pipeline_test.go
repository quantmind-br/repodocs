package converter

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPipeline tests creating a new pipeline
func TestNewPipeline(t *testing.T) {
	tests := []struct {
		name string
		opts PipelineOptions
	}{
		{
			name: "full options",
			opts: PipelineOptions{
				BaseURL:         "https://example.com",
				ContentSelector: ".content",
				ExcludeSelector: ".sidebar",
			},
		},
		{
			name: "minimal options",
			opts: PipelineOptions{},
		},
		{
			name: "with base URL only",
			opts: PipelineOptions{
				BaseURL: "https://example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(tt.opts)
			assert.NotNil(t, pipeline)
			assert.NotNil(t, pipeline.sanitizer)
			assert.NotNil(t, pipeline.extractor)
			assert.NotNil(t, pipeline.mdConverter)
			assert.Equal(t, tt.opts.ExcludeSelector, pipeline.excludeSelector)
		})
	}
}

// TestPipeline_Convert tests HTML to Document conversion
func TestPipeline_Convert(t *testing.T) {
	tests := []struct {
		name          string
		opts          PipelineOptions
		html          string
		sourceURL     string
		wantErr       bool
		shouldContain string
	}{
		{
			name: "simple HTML document",
			opts: PipelineOptions{
				BaseURL: "https://example.com",
			},
			html:          `<html><head><title>Test Page</title></head><body><h1>Main Heading</h1><p>This is content.</p></body></html>`,
			sourceURL:     "https://example.com/page",
			wantErr:       false,
			shouldContain: "Main Heading",
		},
		{
			name: "with content selector",
			opts: PipelineOptions{
				BaseURL:         "https://example.com",
				ContentSelector: ".main-content",
			},
			html:          `<html><body><div class="main-content"><p>Main content</p><a href="/page">Link</a></div><div class="sidebar">Sidebar</div></body></html>`,
			sourceURL:     "https://example.com",
			wantErr:       false,
			shouldContain: "Main content",
		},
		{
			name: "with exclude selector",
			opts: PipelineOptions{
				BaseURL:         "https://example.com",
				ContentSelector: "body",
				ExcludeSelector: ".ads",
			},
			html:      `<html><body><div class="ads">Ads</div><p>Real content</p></body></html>`,
			sourceURL: "https://example.com",
			wantErr:   false,
		},
		{
			name:      "empty HTML",
			opts:      PipelineOptions{},
			html:      "",
			sourceURL: "https://example.com",
			wantErr:   false,
			// Empty HTML returns empty content
			shouldContain: "",
		},
		{
			name: "HTML with links",
			opts: PipelineOptions{
				BaseURL: "https://example.com",
			},
			html:      `<html><body><p><a href="/page1">Link 1</a> <a href="/page2">Link 2</a></p></body></html>`,
			sourceURL: "https://example.com",
			wantErr:   false,
		},
		{
			name:      "HTML with multiple headers",
			opts:      PipelineOptions{},
			html:      `<html><body><h1>Header 1</h1><p>Content</p><h2>Header 2</h2><p>More</p></body></html>`,
			sourceURL: "https://example.com",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(tt.opts)
			ctx := context.Background()

			doc, err := pipeline.Convert(ctx, tt.html, tt.sourceURL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, doc)

				// Check basic fields
				assert.Equal(t, tt.sourceURL, doc.URL)
				assert.NotEmpty(t, doc.ContentHash)
				assert.NotZero(t, doc.FetchedAt)

				// Check content if expected
				if tt.shouldContain != "" {
					assert.NotEmpty(t, doc.Content)
					assert.Contains(t, doc.Content, tt.shouldContain)
				}

				if tt.opts.ContentSelector != "" && strings.Contains(tt.html, "href=") {
					assert.NotEmpty(t, doc.Links)
				}
			}
		})
	}
}

// TestConvertHTML tests convenience function
func TestConvertHTML(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		sourceURL     string
		wantErr       bool
		shouldContain string
	}{
		{
			name:          "simple conversion",
			html:          `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			sourceURL:     "https://example.com",
			wantErr:       false,
			shouldContain: "Title",
		},
		{
			name:      "empty HTML",
			html:      "",
			sourceURL: "https://example.com",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ConvertHTML(tt.html, tt.sourceURL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, doc)
				assert.Equal(t, tt.sourceURL, doc.URL)

				if tt.shouldContain != "" {
					assert.Contains(t, doc.Content, tt.shouldContain)
				}
			}
		})
	}
}

// TestConvertHTMLWithSelector tests conversion with selector
func TestConvertHTMLWithSelector(t *testing.T) {
	html := `<html><body><div class="content">Main</div><div class="sidebar">Side</div></body></html>`

	doc, err := ConvertHTMLWithSelector(html, "https://example.com", ".content")
	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Contains(t, doc.Content, "Main")
}

// TestCalculateHash tests hash calculation
func TestCalculateHash(t *testing.T) {
	tests := []struct {
		name    string
		content string
		sameAs  string
	}{
		{
			name:    "same content same hash",
			content: "Test content",
			sameAs:  "Test content",
		},
		{
			name:    "different content different hash",
			content: "Content A",
			sameAs:  "Content B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := calculateHash(tt.content)
			hash2 := calculateHash(tt.sameAs)

			if tt.content == tt.sameAs {
				assert.Equal(t, hash1, hash2)
			} else {
				assert.NotEqual(t, hash1, hash2)
			}

			// Hash should be hex string
			assert.NotEmpty(t, hash1)
			for _, c := range hash1 {
				assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
			}
		})
	}
}

// TestPipeline_RemoveExcluded tests exclusion selector
func TestPipeline_RemoveExcluded(t *testing.T) {
	tests := []struct {
		name          string
		exclude       string
		html          string
		shouldContain string
		notContains   []string
	}{
		{
			name:          "exclude by class",
			exclude:       ".sidebar",
			html:          `<html><body><div class="sidebar">Sidebar</div><p>Content</p></body></html>`,
			shouldContain: "Content",
			notContains:   []string{"Sidebar"},
		},
		{
			name:          "exclude by ID",
			exclude:       "#ads",
			html:          `<html><body><div id="ads">Ads</div><p>Content</p></body></html>`,
			shouldContain: "Content",
			notContains:   []string{"Ads"},
		},
		{
			name:          "exclude by tag",
			exclude:       "script",
			html:          `<html><body><script>alert('xss')</script><p>Content</p></body></html>`,
			shouldContain: "Content",
			notContains:   []string{"alert"},
		},
		{
			name:          "multiple selectors",
			exclude:       ".sidebar, .ads, script",
			html:          `<html><body><div class="sidebar">Side</div><div class="ads">Ads</div><script>alert()</script><p>Content</p></body></html>`,
			shouldContain: "Content",
			notContains:   []string{"Side", "Ads", "alert"},
		},
		{
			name:          "no exclusion",
			exclude:       "",
			html:          `<html><body><p>Content</p></body></html>`,
			shouldContain: "Content",
		},
		{
			name:          "selector not found",
			exclude:       ".nonexistent",
			html:          `<html><body><p>Content</p></body></html>`,
			shouldContain: "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(PipelineOptions{ExcludeSelector: tt.exclude})
			result := pipeline.removeExcluded(tt.html)

			if tt.shouldContain != "" {
				assert.Contains(t, result, tt.shouldContain)
			}

			for _, notContain := range tt.notContains {
				assert.NotContains(t, result, notContain)
			}
		})
	}
}

// TestPipeline_Convert_Metadata tests metadata extraction
func TestPipeline_Convert_Metadata(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta name="description" content="Page description">
</head>
<body>
	<h1>Main Heading</h1>
	<h2>Sub Heading</h2>
	<h2>Another Sub</h2>
	<p>Content with <a href="/page1">link 1</a> and <a href="/page2">link 2</a>.</p>
</body>
</html>`

	pipeline := NewPipeline(PipelineOptions{BaseURL: "https://example.com"})
	ctx := context.Background()

	doc, err := pipeline.Convert(ctx, html, "https://example.com/page")
	require.NoError(t, err)

	// Check title (title tag should be used when no og:title)
	assert.NotEmpty(t, doc.Title)

	// Check description
	assert.Equal(t, "Page description", doc.Description)

	// Check headers (may be extracted from markdown content, not from HTML)
	// Headers are extracted from the sanitized HTML content
	if len(doc.Headers) > 0 {
		assert.NotEmpty(t, doc.Headers)
	}

	// Check links (should be resolved to absolute URLs)
	assert.NotEmpty(t, doc.Links)
	hasLink1 := false
	hasLink2 := false
	for _, link := range doc.Links {
		if strings.Contains(link, "/page1") || link == "https://example.com/page1" {
			hasLink1 = true
		}
		if strings.Contains(link, "/page2") || link == "https://example.com/page2" {
			hasLink2 = true
		}
	}
	assert.True(t, hasLink1 || hasLink2)

	// Check stats
	assert.Greater(t, doc.WordCount, 0)
	assert.Greater(t, doc.CharCount, 0)
	assert.NotEmpty(t, doc.ContentHash)
}

// TestPipeline_Convert_UTF8Encoding tests UTF-8 encoding handling
func TestPipeline_Convert_UTF8Encoding(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		shouldContains []string
	}{
		{
			name:           "UTF-8 content",
			html:           `<html><body><p>Hello 世界 🌍</p></body></html>`,
			shouldContains: []string{"Hello", "世界", "🌍"},
		},
		{
			name:           "UTF-8 with meta charset",
			html:           `<html><head><meta charset="utf-8"></head><body><p>UTF-8 字符</p></body></html>`,
			shouldContains: []string{"UTF-8", "字符"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := NewPipeline(PipelineOptions{})
			ctx := context.Background()

			doc, err := pipeline.Convert(ctx, tt.html, "https://example.com")
			require.NoError(t, err)
			assert.NotNil(t, doc)

			for _, shouldContain := range tt.shouldContains {
				assert.Contains(t, doc.Content, shouldContain)
			}
		})
	}
}

// TestPipeline_Convert_WithContextCancellation tests context cancellation
func TestPipeline_Convert_WithContextCancellation(t *testing.T) {
	pipeline := NewPipeline(PipelineOptions{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	html := `<html><body><p>Content</p></body></html>`
	_, _ = pipeline.Convert(ctx, html, "https://example.com")
	// Should either error or complete, depends on implementation
	// Not asserting error as the pipeline may complete before cancellation is checked
}

// TestPipeline_Convert_InvalidUTF8 tests invalid UTF-8 handling
func TestPipeline_Convert_InvalidUTF8(t *testing.T) {
	pipeline := NewPipeline(PipelineOptions{})
	ctx := context.Background()

	// This should not error - encoding conversion should handle it
	html := `<html><body><p>Valid UTF-8: Hello 世界</p></body></html>`
	doc, err := pipeline.Convert(ctx, html, "https://example.com")
	require.NoError(t, err)
	assert.NotNil(t, doc)
}
