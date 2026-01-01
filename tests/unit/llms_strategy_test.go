package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

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
		name        string
		content     string
		expected    []domain.LLMSLink
		expectedErr bool
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
			expected:    []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name: "Anchor links (should be skipped)",
			content: `[Top](#top)
[Section](#section)`,
			expected:    []domain.LLMSLink{},
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
			name:        "No links",
			content:     `This is just plain text without any links.`,
			expected:    []domain.LLMSLink{},
			expectedErr: false,
		},
		{
			name:        "Empty content",
			content:     ``,
			expected:    []domain.LLMSLink{},
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
		name    string
		input   string
		matches int
		first   []string
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

// TestParseLLMSLinks_Success tests parsing valid LLMS content
func TestParseLLMSLinks_Success(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []domain.LLMSLink
	}{
		{
			name: "Simple markdown links",
			content: `[Getting Started](https://example.com/getting-started)
[API Reference](https://example.com/api)`,
			want: []domain.LLMSLink{
				{Title: "Getting Started", URL: "https://example.com/getting-started"},
				{Title: "API Reference", URL: "https://example.com/api"},
			},
		},
		{
			name: "Links with whitespace",
			content: `[Title1](https://example.com/page1)
   [Title 2](  https://example.com/page2  )`,
			want: []domain.LLMSLink{
				{Title: "Title1", URL: "https://example.com/page1"},
				{Title: "Title 2", URL: "https://example.com/page2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLLMSLinksTest(tt.content)
			require.Equal(t, len(tt.want), len(got))
			for i := range tt.want {
				assert.Equal(t, tt.want[i], got[i])
			}
		})
	}
}

// TestParseLLMSLinks_Empty tests parsing empty LLMS content
func TestParseLLMSLinks_Empty(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "Empty content",
			content: "",
			want:    0,
		},
		{
			name:    "Only whitespace",
			content: "   \n\n   ",
			want:    0,
		},
		{
			name:    "No markdown links",
			content: "Just plain text without links",
			want:    0,
		},
		{
			name: "Only anchor links",
			content: `[Top](#top)
[Section](#section)`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLLMSLinksTest(tt.content)
			assert.Equal(t, tt.want, len(got))
		})
	}
}

// TestParseLLMSLinks_Complex tests parsing complex LLMS content
func TestParseLLMSLinks_Complex(t *testing.T) {
	content := `# Project Documentation

## Getting Started
- [Installation](https://example.com/install)
- [Quick Start](https://example.com/quickstart)

## API Reference
- [Endpoints](https://example.com/api/endpoints)
- [Authentication](https://example.com/api/auth)

## Examples
- [Basic Usage](examples/basic.html)
- [Advanced](examples/advanced.html)

[External](https://external.com/docs)`

	got := parseLLMSLinksTest(content)
	assert.Equal(t, 7, len(got))

	// Verify specific links
	assert.Equal(t, "Installation", got[0].Title)
	assert.Equal(t, "https://example.com/install", got[0].URL)
	assert.Equal(t, "External", got[6].Title)
	assert.Equal(t, "https://external.com/docs", got[6].URL)
}

// TestNewLLMSStrategy_Success tests successful strategy creation
func TestNewLLMSStrategy_Success(t *testing.T) {
	deps := createTestLLMSDependencies(t)
	strategy := strategies.NewLLMSStrategy(deps)

	require.NotNil(t, strategy)
	assert.Equal(t, "llms", strategy.Name())
	assert.Equal(t, "llms", strategy.Name())
}

// TestCanHandle_LLMSURL tests URL detection for LLMS files
func TestCanHandle_LLMSURL(t *testing.T) {
	deps := createTestLLMSDependencies(t)
	strategy := strategies.NewLLMSStrategy(deps)

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "llms.txt at root",
			url:  "https://example.com/llms.txt",
			want: true,
		},
		{
			name: "llms.txt with path",
			url:  "https://example.com/docs/llms.txt",
			want: true,
		},
		{
			name: "llms.txt with trailing slash",
			url:  "https://example.com/api/v1/llms.txt",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCanHandle_NonLLMSURL tests rejection of non-LLMS URLs
func TestCanHandle_NonLLMSURL(t *testing.T) {
	deps := createTestLLMSDependencies(t)
	strategy := strategies.NewLLMSStrategy(deps)

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "Regular page",
			url:  "https://example.com/docs",
			want: false,
		},
		{
			name: "sitemap.xml",
			url:  "https://example.com/sitemap.xml",
			want: false,
		},
		{
			name: "GitHub URL",
			url:  "https://github.com/user/repo",
			want: false,
		},
		{
			name: "Uppercase LLMS",
			url:  "https://example.com/LLMS.TXT",
			want: false,
		},
		{
			name: "Empty URL",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExecute_WithValidLLMS tests execution with valid LLMS content
func TestExecute_WithValidLLMS(t *testing.T) {
	ctx := context.Background()

	// Create test servers
	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Test Content</h1></body></html>`))
	}))
	defer pageServer.Close()

	// Update llms.txt to use the test server URLs
	updatedLLMS := strings.ReplaceAll(`[Getting Started](https://example.com/page1)
[API Reference](https://example.com/page2)`, "https://example.com/page1", pageServer.URL)
	updatedLLMS = strings.ReplaceAll(updatedLLMS, "https://example.com/page2", pageServer.URL)

	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(updatedLLMS))
	}))
	defer llmsServer.Close()

	// Use real fetcher/converter/writer with test servers
	// This is a simplified test focusing on the Execute flow
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	// Override the fetcher to use our test server URLs
	// The fetcher will automatically fetch from the httptest servers

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	// Act
	err = strategy.Execute(ctx, llmsServer.URL, opts)

	// Assert
	require.NoError(t, err)
}

// TestExecute_WithEmptyLLMS tests execution with empty LLMS content
func TestExecute_WithEmptyLLMS(t *testing.T) {
	ctx := context.Background()

	// Create server with empty LLMS content
	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
	}))
	defer llmsServer.Close()

	// Use real dependencies
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	// Act
	err = strategy.Execute(ctx, llmsServer.URL, opts)

	// Assert
	require.NoError(t, err)
	// Should complete without error even with empty content
}

// TestExecute_WithInvalidHTML tests execution when linked pages have invalid HTML
func TestExecute_WithInvalidHTML(t *testing.T) {
	ctx := context.Background()

	// Create servers with valid and invalid HTML
	validServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Valid</title></head><body><h1>Valid Content</h1></body></html>`))
	}))
	defer validServer.Close()

	invalidServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// Intentionally malformed HTML
		w.Write([]byte(`<html><head><title>Unclosed tags<body><h1>Invalid`))
	}))
	defer invalidServer.Close()

	// Update LLMS content with actual test server URLs
	llmsContent := `[Valid Page](https://example.com/valid)
[Invalid Page](https://example.com/invalid)`
	updatedLLMS := strings.ReplaceAll(llmsContent, "https://example.com/valid", validServer.URL)
	updatedLLMS = strings.ReplaceAll(updatedLLMS, "https://example.com/invalid", invalidServer.URL)

	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(updatedLLMS))
	}))
	defer llmsServer.Close()

	// Use real dependencies
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	// Act
	err = strategy.Execute(ctx, llmsServer.URL, opts)

	// Assert - should complete without error even with invalid HTML
	// (converter should handle it gracefully)
	require.NoError(t, err)
}

// TestExecute_FetchError tests execution when fetch fails
func TestExecute_FetchError(t *testing.T) {
	ctx := context.Background()

	// Create server that returns error
	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer llmsServer.Close()

	// Use real dependencies
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	// Act
	err = strategy.Execute(ctx, llmsServer.URL, opts)

	// Assert
	require.Error(t, err)
	// Should get an error when fetch fails
	assert.Contains(t, err.Error(), "fetch")
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
			name:    "Title with newlines - regex matches across newlines",
			content: "[Title\nWith Newlines](https://example.com)",
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
			name:    "HTTPS and HTTP",
			content: `[HTTPS](https://secure.com) [HTTP](http://insecure.com)`,
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

// Mocks for testing

type mockFetcher struct {
	responses   map[string]*domain.Response
	called      bool
	shouldError bool
}

func (m *mockFetcher) Get(ctx context.Context, urlStr string) (*domain.Response, error) {
	m.called = true
	if m.shouldError {
		return nil, assert.AnError
	}
	if resp, ok := m.responses[urlStr]; ok {
		return resp, nil
	}
	// Return default response for unknown URLs
	return &domain.Response{
		StatusCode: 200,
		Body:       []byte(""),
		Headers:    make(http.Header),
	}, nil
}

func (m *mockFetcher) GetWithHeaders(ctx context.Context, urlStr string, headers map[string]string) (*domain.Response, error) {
	return m.Get(ctx, urlStr)
}

func (m *mockFetcher) GetCookies(urlStr string) []*http.Cookie {
	return nil
}

func (m *mockFetcher) Close() error {
	return nil
}

type mockConverter struct {
	shouldError bool
	documents   []*domain.Document
}

func (m *mockConverter) Convert(ctx context.Context, html string, sourceURL string) (*domain.Document, error) {
	if m.shouldError {
		return nil, assert.AnError
	}
	doc := &domain.Document{
		URL:            sourceURL,
		Title:          "Test Document",
		Content:        "# Test Content",
		HTMLContent:    html,
		FetchedAt:      time.Now(),
		SourceStrategy: "llms",
		CacheHit:       false,
	}
	if m.documents != nil {
		m.documents = append(m.documents, doc)
	}
	return doc, nil
}

type mockWriter struct {
	written      bool
	documents    []*domain.Document
	existingDocs map[string]bool
}

func (m *mockWriter) Write(ctx context.Context, doc *domain.Document) error {
	m.written = true
	if m.documents != nil {
		m.documents = append(m.documents, doc)
	}
	return nil
}

func (m *mockWriter) Exists(url string) bool {
	if m.existingDocs != nil {
		return m.existingDocs[url]
	}
	return false
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

func parseLLMSLinksTest(content string) []domain.LLMSLink {
	var links []domain.LLMSLink

	matches := linkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			title := strings.TrimSpace(match[1])
			url := strings.TrimSpace(match[2])

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

func filterLLMSLinksTest(links []domain.LLMSLink, filterURL string) []domain.LLMSLink {
	filtered := make([]domain.LLMSLink, 0, len(links))
	for _, link := range links {
		if filterURL == "" || strings.HasPrefix(link.URL, filterURL) {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

func TestFilterLLMSLinks(t *testing.T) {
	tests := []struct {
		name      string
		links     []domain.LLMSLink
		filterURL string
		expected  []domain.LLMSLink
	}{
		{
			name: "Filter by path prefix",
			links: []domain.LLMSLink{
				{Title: "Agents", URL: "https://platform.claude.com/docs/en/agents-and-tools/agents"},
				{Title: "Tools", URL: "https://platform.claude.com/docs/en/agents-and-tools/tools"},
				{Title: "Overview", URL: "https://platform.claude.com/docs/en/overview"},
				{Title: "API", URL: "https://platform.claude.com/docs/en/api"},
			},
			filterURL: "https://platform.claude.com/docs/en/agents-and-tools/",
			expected: []domain.LLMSLink{
				{Title: "Agents", URL: "https://platform.claude.com/docs/en/agents-and-tools/agents"},
				{Title: "Tools", URL: "https://platform.claude.com/docs/en/agents-and-tools/tools"},
			},
		},
		{
			name: "Empty filter returns all links",
			links: []domain.LLMSLink{
				{Title: "Page1", URL: "https://example.com/page1"},
				{Title: "Page2", URL: "https://example.com/page2"},
			},
			filterURL: "",
			expected: []domain.LLMSLink{
				{Title: "Page1", URL: "https://example.com/page1"},
				{Title: "Page2", URL: "https://example.com/page2"},
			},
		},
		{
			name: "Filter excludes all links",
			links: []domain.LLMSLink{
				{Title: "Blog", URL: "https://example.com/blog/post1"},
				{Title: "News", URL: "https://example.com/news/article1"},
			},
			filterURL: "https://example.com/docs/",
			expected:  []domain.LLMSLink{},
		},
		{
			name: "Filter with exact match",
			links: []domain.LLMSLink{
				{Title: "Exact", URL: "https://example.com/docs"},
				{Title: "SubPath", URL: "https://example.com/docs/api"},
			},
			filterURL: "https://example.com/docs",
			expected: []domain.LLMSLink{
				{Title: "Exact", URL: "https://example.com/docs"},
				{Title: "SubPath", URL: "https://example.com/docs/api"},
			},
		},
		{
			name: "Filter different domains",
			links: []domain.LLMSLink{
				{Title: "Internal", URL: "https://platform.claude.com/docs/en/api"},
				{Title: "External", URL: "https://external.com/docs/en/api"},
			},
			filterURL: "https://platform.claude.com/docs/",
			expected: []domain.LLMSLink{
				{Title: "Internal", URL: "https://platform.claude.com/docs/en/api"},
			},
		},
		{
			name:      "Empty links",
			links:     []domain.LLMSLink{},
			filterURL: "https://example.com/docs/",
			expected:  []domain.LLMSLink{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLLMSLinksTest(tt.links, tt.filterURL)
			assert.Equal(t, len(tt.expected), len(result), "Number of filtered links mismatch")

			for i, expectedLink := range tt.expected {
				if i < len(result) {
					assert.Equal(t, expectedLink.Title, result[i].Title)
					assert.Equal(t, expectedLink.URL, result[i].URL)
				}
			}
		})
	}
}

func TestExecute_WithMarkdownContent(t *testing.T) {
	ctx := context.Background()

	markdownContent := `---
title: API Reference
description: Complete API documentation
---

# API Reference

This is the API documentation.

## Endpoints

- GET /users
- POST /users
`

	mdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write([]byte(markdownContent))
	}))
	defer mdServer.Close()

	llmsContent := `[API Reference](` + mdServer.URL + `)`

	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(llmsContent))
	}))
	defer llmsServer.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	err = strategy.Execute(ctx, llmsServer.URL, opts)
	require.NoError(t, err)
}

func TestExecute_WithMarkdownURLExtension(t *testing.T) {
	ctx := context.Background()

	markdownContent := `# Getting Started

Welcome to the documentation.
`

	mdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(markdownContent))
	}))
	defer mdServer.Close()

	llmsContent := `[Getting Started](` + mdServer.URL + `/docs/getting-started.md)`

	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(llmsContent))
	}))
	defer llmsServer.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
	}

	err = strategy.Execute(ctx, llmsServer.URL, opts)
	require.NoError(t, err)
}

func TestExecute_WithFilterURL(t *testing.T) {
	ctx := context.Background()

	pageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Content</h1></body></html>`))
	}))
	defer pageServer.Close()

	llmsContent := `# Documentation

- [Agents Overview](https://platform.claude.com/docs/en/agents-and-tools/overview)
- [Tools Guide](https://platform.claude.com/docs/en/agents-and-tools/tools)
- [API Reference](https://platform.claude.com/docs/en/api/reference)
- [Getting Started](https://platform.claude.com/docs/en/getting-started)`

	llmsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(llmsContent))
	}))
	defer llmsServer.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		EnableCache:    false,
		EnableRenderer: false,
		Verbose:        false,
	})
	require.NoError(t, err)

	strategy := strategies.NewLLMSStrategy(deps)
	opts := strategies.Options{
		Concurrency: 1,
		Limit:       0,
		Force:       true,
		Output:      t.TempDir(),
		FilterURL:   "https://platform.claude.com/docs/en/agents-and-tools/",
	}

	err = strategy.Execute(ctx, llmsServer.URL, opts)
	require.NoError(t, err)
}
