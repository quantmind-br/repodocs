package unit

import (
	"regexp"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMSStrategy_NewLLMSStrategy(t *testing.T) {
	// Arrange
	deps := createTestLLMSDependencies(t)

	// Act
	strategy := strategies.NewLLMSStrategy(deps)

	// Assert
	require.NotNil(t, strategy)
	assert.Equal(t, "llms", strategy.Name())
}

func TestLLMSStrategy_CanHandle(t *testing.T) {
	// Arrange
	deps := createTestLLMSDependencies(t)
	strategy := strategies.NewLLMSStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "llms.txt at root",
			url:      "https://example.com/llms.txt",
			expected: true,
		},
		{
			name:     "llms.txt with trailing slash",
			url:      "https://example.com/docs/llms.txt",
			expected: true,
		},
		{
			name:     "Uppercase extension - should NOT match (case-sensitive)",
			url:      "https://example.com/LLMS.TXT",
			expected: false,
		},
		{
			name:     "Mixed case - should NOT match (case-sensitive)",
			url:      "https://example.com/docs/Llms.Txt",
			expected: false,
		},
		{
			name:     "Regular page",
			url:      "https://example.com/docs",
			expected: false,
		},
		{
			name:     "sitemap.xml",
			url:      "https://example.com/sitemap.xml",
			expected: false,
		},
		{
			name:     "GitHub URL",
			url:      "https://github.com/user/repo",
			expected: false,
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	// Act & Assert
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLLMSStrategy_Name(t *testing.T) {
	// Arrange
	deps := createTestLLMSDependencies(t)
	strategy := strategies.NewLLMSStrategy(deps)

	// Act
	name := strategy.Name()

	// Assert
	assert.Equal(t, "llms", name)
}

func TestParseLLMSLinksTest(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expected     []domain.LLMSLink
		expectedErr  bool
	}{
		{
			name: "Simple markdown links",
			content: `# Documentation

- [Getting Started](https://example.com/getting-started)
- [API Reference](https://example.com/api)
- [Examples](https://example.com/examples)`,
			expected: []domain.LLMSLink{
				{Title: "Getting Started", URL: "https://example.com/getting-started"},
				{Title: "API Reference", URL: "https://example.com/api"},
				{Title: "Examples", URL: "https://example.com/examples"},
			},
			expectedErr: false,
		},
		{
			name: "Links with whitespace - trimmed",
			content: `[Title1](https://example.com/page1)
   [Title 2](  https://example.com/page2  )`,
			expected: []domain.LLMSLink{
				{Title: "Title1", URL: "https://example.com/page1"},
				{Title: "Title 2", URL: "https://example.com/page2"},
			},
			expectedErr: false,
		},
		{
			name: "Empty links - skipped",
			content: `[Empty]()
[Also Empty](   )`,
			expected: []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name: "Anchor links (should be skipped)",
			content: `[Top](#top)
[Section](#section)`,
			expected: []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name: "Mixed anchor and normal links",
			content: `[Top](#top)
[Valid Link](https://example.com/page)
[#hash](#hash)`,
			expected: []domain.LLMSLink{
				{Title: "Valid Link", URL: "https://example.com/page"},
			},
			expectedErr: false,
		},
		{
			name: "Links with special characters",
			content: `[API Docs](https://example.com/api?version=1&format=json)
[Download](https://example.com/files/doc.pdf)`,
			expected: []domain.LLMSLink{
				{Title: "API Docs", URL: "https://example.com/api?version=1&format=json"},
				{Title: "Download", URL: "https://example.com/files/doc.pdf"},
			},
			expectedErr: false,
		},
		{
			name: "Relative URLs",
			content: `[Home](index.html)
[About](about.html)`,
			expected: []domain.LLMSLink{
				{Title: "Home", URL: "index.html"},
				{Title: "About", URL: "about.html"},
			},
			expectedErr: false,
		},
		{
			name: "No links",
			content: `This is just plain text without any links.`,
			expected: []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name: "Empty content",
			content: ``,
			expected: []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name: "Malformed markdown - regex matches partial",
			content: `[Title without closing bracket
[Another title](url)`,
			expected: []domain.LLMSLink{
				{Title: "Title without closing bracket\n[Another title", URL: "url"},
			},
			expectedErr: false,
		},
		{
			name: "Complex markdown with multiple link types",
			content: `# Project Documentation

## Getting Started
- [Installation](https://example.com/install)
- [Quick Start](https://example.com/quickstart)

## API Reference
- [Endpoints](https://example.com/api/endpoints)
- [Authentication](https://example.com/api/auth)

## Examples
- [Basic Usage](examples/basic.html)
- [Advanced](examples/advanced.html)

[External](https://external.com/docs)`,
			expected: []domain.LLMSLink{
				{Title: "Installation", URL: "https://example.com/install"},
				{Title: "Quick Start", URL: "https://example.com/quickstart"},
				{Title: "Endpoints", URL: "https://example.com/api/endpoints"},
				{Title: "Authentication", URL: "https://example.com/api/auth"},
				{Title: "Basic Usage", URL: "examples/basic.html"},
				{Title: "Advanced", URL: "examples/advanced.html"},
				{Title: "External", URL: "https://external.com/docs"},
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := parseLLMSLinksTest(tt.content)

			// Assert
			if !tt.expectedErr {
				require.NoError(t, nil)
				assert.Equal(t, len(tt.expected), len(result), "Number of links mismatch")

				// Check each link
				for i, expectedLink := range tt.expected {
					if i < len(result) {
						assert.Equal(t, expectedLink.Title, result[i].Title, "Title mismatch at index %d", i)
						assert.Equal(t, expectedLink.URL, result[i].URL, "URL mismatch at index %d", i)
					}
				}
			}
		})
	}
}

func TestParseLLMSLinksTest_LinkRegex(t *testing.T) {
	// Test the regex pattern directly
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	tests := []struct {
		name     string
		input    string
		matches  int
		first    []string
	}{
		{
			name:    "Single link",
			input:   "[Title](https://example.com)",
			matches: 1,
			first:   []string{"[Title](https://example.com)", "Title", "https://example.com"},
		},
		{
			name:    "Multiple links",
			input:   "[Link1](url1) [Link2](url2)",
			matches: 2,
			first:   []string{"[Link1](url1)", "Link1", "url1"},
		},
		{
			name:    "No brackets",
			input:   "Just plain text",
			matches: 0,
			first:   nil,
		},
		{
			name:    "Malformed - missing closing paren",
			input:   "[Title](url",
			matches: 0,
			first:   nil,
		},
		{
			name:    "Title with special chars",
			input:   "[API Reference v1.0](https://example.com/api)",
			matches: 1,
			first:   []string{"[API Reference v1.0](https://example.com/api)", "API Reference v1.0", "https://example.com/api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := linkRegex.FindAllStringSubmatch(tt.input, -1)
			assert.Equal(t, tt.matches, len(matches), "Number of matches mismatch")

			if tt.first != nil && len(matches) > 0 {
				assert.Equal(t, tt.first, matches[0], "First match mismatch")
			}
		})
	}
}

func TestLLMSStrategy_Execute(t *testing.T) {
	// This is a complex integration test that requires mocking
	// For now, we'll test the basic structure
	t.Skip("Requires full dependency injection and network mocking")
}

// Test edge cases for parsing
func TestParseLLMSLinksTest_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []domain.LLMSLink
	}{
		{
			name:     "Empty title - trimmed to empty",
			content:  `[](https://example.com)`,
			expected: []domain.LLMSLink{},
		},
		{
			name:     "Title with newlines - regex matches across newlines",
			content:  "[Title\nWith Newlines](https://example.com)",
			expected: []domain.LLMSLink{
				{Title: "Title\nWith Newlines", URL: "https://example.com"},
			},
		},
		{
			name:     "URL with parentheses (malformed)",
			content:  `[Title](https://example.com/page(id))`,
			expected: []domain.LLMSLink{{Title: "Title", URL: "https://example.com/page(id"}},
		},
		{
			name:     "Nested brackets in title - doesn't match (no closing ] before ())",
			content:  `[[Nested]](https://example.com)`,
			expected: []domain.LLMSLink{},
		},
		{
			name:     "Title with brackets - will not match (stops at first ])",
			content:  `[Title [with] brackets](https://example.com)`,
			expected: []domain.LLMSLink{},
		},
		{
			name:     "Multiple whitespace in title - trimmed",
			content:  `[  Title  With  Spaces  ](https://example.com)`,
			expected: []domain.LLMSLink{{Title: "Title  With  Spaces", URL: "https://example.com"}},
		},
		{
			name:     "Link with port",
			content:  `[Local](http://localhost:8080)`,
			expected: []domain.LLMSLink{{Title: "Local", URL: "http://localhost:8080"}},
		},
		{
			name:     "HTTPS and HTTP",
			content:  `[HTTPS](https://secure.com) [HTTP](http://insecure.com)`,
			expected: []domain.LLMSLink{
				{Title: "HTTPS", URL: "https://secure.com"},
				{Title: "HTTP", URL: "http://insecure.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLLMSLinksTest(tt.content)
			assert.Equal(t, len(tt.expected), len(result), "Number of links mismatch")

			for i, expectedLink := range tt.expected {
				if i < len(result) {
					assert.Equal(t, expectedLink.Title, result[i].Title, "Title mismatch at index %d", i)
					assert.Equal(t, expectedLink.URL, result[i].URL, "URL mismatch at index %d", i)
				}
			}
		})
	}
}

// Helper to create test dependencies
func createTestLLMSDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()

	// Create minimal test dependencies
	return &strategies.Dependencies{}
}

// Test helpers that mirror unexported functions from llms.go

// linkRegex matches markdown links: [Title](url) (mirror from llms.go)
var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

// parseLLMSLinks parses markdown links from llms.txt content (mirror from llms.go)
func parseLLMSLinksTest(content string) []domain.LLMSLink {
	var links []domain.LLMSLink

	matches := linkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			title := strings.TrimSpace(match[1])
			url := strings.TrimSpace(match[2])

			// Skip empty URLs or anchors
			if url == "" || strings.HasPrefix(url, "#") {
				continue
			}

			links = append(links, domain.LLMSLink{
				Title: title,
				URL:   url,
			})
		}
	}

	return links
}
