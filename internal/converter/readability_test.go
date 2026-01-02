package converter

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExtractContent tests creating a new content extractor
func TestNewExtractContent(t *testing.T) {
	tests := []struct {
		name     string
		selector string
	}{
		{"with selector", ".content"},
		{"without selector", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExtractContent(tt.selector)
			assert.NotNil(t, extractor)
			assert.Equal(t, tt.selector, extractor.selector)
		})
	}
}

// TestExtractContent_Extract tests content extraction
func TestExtractContent_Extract(t *testing.T) {
	tests := []struct {
		name           string
		selector       string
		html           string
		sourceURL      string
		wantErr        bool
		shouldContain  string
	}{
		{
			name:          "with matching selector",
			selector:      ".main-content",
			html:          `<html><body><div class="main-content">Main</div><div class="sidebar">Side</div></body></html>`,
			sourceURL:     "https://example.com",
			wantErr:       false,
			shouldContain: "Main",
		},
		{
			name:          "selector not found - fallback to readability",
			selector:      ".nonexistent",
			html:          `<html><body><article><p>Article content</p></article></body></html>`,
			sourceURL:     "https://example.com",
			wantErr:       false,
			shouldContain: "Article content",
		},
		{
			name:          "without selector - use readability",
			selector:      "",
			html:          `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			sourceURL:     "https://example.com",
			wantErr:       false,
			shouldContain: "Content",
		},
		{
			name:      "with title tag",
			selector:  "",
			html:      `<html><head><title>Page Title</title></head><body><p>Content</p></body></html>`,
			sourceURL: "https://example.com",
			wantErr:   false,
		},
		{
			name:      "empty HTML",
			selector:  "",
			html:      "",
			sourceURL: "https://example.com",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExtractContent(tt.selector)
			content, _, err := extractor.Extract(tt.html, tt.sourceURL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Only check NotEmpty if HTML is not empty
				if tt.html != "" && tt.shouldContain == "" {
					assert.NotEmpty(t, content)
				}

				if tt.shouldContain != "" {
					assert.Contains(t, content, tt.shouldContain)
				}
			}
		})
	}
}

// TestExtractTitle tests title extraction
func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "title tag",
			html:     `<html><head><title>Page Title</title></head></html>`,
			expected: "Page Title",
		},
		{
			name:     "h1 tag",
			html:     `<html><body><h1>Main Heading</h1></body></html>`,
			expected: "Main Heading",
		},
		{
			name:     "og:title meta",
			html:     `<html><head><meta property="og:title" content="OG Title"></head></html>`,
			expected: "OG Title",
		},
		{
			name:     "title tag takes precedence",
			html:     `<html><head><title>Title</title><meta property="og:title" content="OG"></head><body><h1>H1</h1></body></html>`,
			expected: "Title",
		},
		{
			name:     "h1 fallback when no title",
			html:     `<html><body><h1>Heading</h1></body></html>`,
			expected: "Heading",
		},
		{
			name:     "no title found",
			html:     `<html><body><p>Content</p></body></html>`,
			expected: "",
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)
			result := extractTitle(doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractDescription tests description extraction
func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "meta description",
			html:     `<html><head><meta name="description" content="Page description"></head></html>`,
			expected: "Page description",
		},
		{
			name:     "og:description",
			html:     `<html><head><meta property="og:description" content="OG description"></head></html>`,
			expected: "OG description",
		},
		{
			name:     "meta description takes precedence",
			html:     `<html><head><meta name="description" content="Meta"><meta property="og:description" content="OG"></head></html>`,
			expected: "Meta",
		},
		{
			name:     "og:description fallback",
			html:     `<html><head><meta property="og:description" content="OG"></head></html>`,
			expected: "OG",
		},
		{
			name:     "no description found",
			html:     `<html><body><p>Content</p></body></html>`,
			expected: "",
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			require.NoError(t, err)
			result := ExtractDescription(doc)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractHeaders tests header extraction
func TestExtractHeaders(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected map[string][]string
	}{
		{
			name: "all header levels",
			html: `<h1>H1</h1><h2>H2</h2><h3>H3</h3><h4>H4</h4><h5>H5</h5><h6>H6</h6>`,
			expected: map[string][]string{
				"h1": {"H1"},
				"h2": {"H2"},
				"h3": {"H3"},
				"h4": {"H4"},
				"h5": {"H5"},
				"h6": {"H6"},
			},
		},
		{
			name: "multiple headers of same level",
			html: `<h2>First</h2><p>Content</p><h2>Second</h2>`,
			expected: map[string][]string{
				"h2": {"First", "Second"},
			},
		},
		{
			name:     "empty headers are skipped",
			html:     `<h2></h2><h2>Valid</h2>`,
			expected: map[string][]string{
				"h2": {"Valid"},
			},
		},
		{
			name:     "no headers",
			html:     `<p>Just paragraphs</p>`,
			expected: map[string][]string{},
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHeaders(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractLinks tests link extraction
func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		baseURL     string
		expected    []string
		notExpected []string
	}{
		{
			name:     "absolute links",
			html:     `<a href="https://example.com/page">Link</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "relative links",
			html:     `<a href="/page">Link</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:        "skip anchors",
			html:        `<a href="#section">Anchor</a>`,
			baseURL:     "https://example.com",
			notExpected: []string{"#section"},
		},
		{
			name:        "skip javascript",
			html:        `<a href="javascript:void(0)">JS</a>`,
			baseURL:     "https://example.com",
			notExpected: []string{"javascript:void(0)"},
		},
		{
			name:        "skip mailto",
			html:        `<a href="mailto:test@example.com">Email</a>`,
			baseURL:     "https://example.com",
			notExpected: []string{"mailto:test@example.com"},
		},
		{
			name:        "skip tel",
			html:        `<a href="tel:+1234567890">Phone</a>`,
			baseURL:     "https://example.com",
			notExpected: []string{"tel:+1234567890"},
		},
		{
			name:     "multiple links",
			html:     `<a href="/page1">Link1</a><a href="/page2">Link2</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:     "no links",
			html:     `<p>Just text</p>`,
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:     "empty HTML",
			html:     "",
			baseURL:  "https://example.com",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractLinks(tt.html, tt.baseURL)

			for _, exp := range tt.expected {
				assert.Contains(t, result, exp)
			}

			for _, notExp := range tt.notExpected {
				assert.NotContains(t, result, notExp)
			}
		})
	}
}

// TestExtractBody tests body extraction fallback
func TestExtractBody(t *testing.T) {
	extractor := NewExtractContent("")

	tests := []struct {
		name     string
		html     string
		contains string
	}{
		{
			name:     "extract body content",
			html:     `<html><head><title>Title</title></head><body><p>Body content</p></body></html>`,
			contains: "Body content",
		},
		{
			name:     "no body tag",
			html:     `<p>Direct content</p>`,
			contains: "Direct content",
		},
		{
			name:     "empty HTML",
			html:     "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _, err := extractor.extractBody(tt.html)
			require.NoError(t, err)

			if tt.contains != "" {
				assert.Contains(t, content, tt.contains)
			}
		})
	}
}
