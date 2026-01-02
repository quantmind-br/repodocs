package strategies

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCrawlerStrategy tests creating a new crawler strategy
func TestNewCrawlerStrategy(t *testing.T) {
	deps := &Dependencies{
		Fetcher:   &mockFetcher{},
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		Renderer:  nil, // Renderer is optional
	}

	strategy := NewCrawlerStrategy(deps)

	assert.NotNil(t, strategy)
	assert.NotNil(t, strategy.deps)
	assert.NotNil(t, strategy.fetcher)
	assert.NotNil(t, strategy.converter)
	assert.NotNil(t, strategy.markdownReader)
	assert.NotNil(t, strategy.writer)
	assert.NotNil(t, strategy.logger)
	// renderer may be nil
}

// TestCrawlerStrategy_Name tests the Name method
func TestCrawlerStrategy_Name(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewCrawlerStrategy(deps)

	assert.Equal(t, "crawler", strategy.Name())
}

// TestCrawlerStrategy_CanHandle tests the CanHandle method
func TestCrawlerStrategy_CanHandle(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewCrawlerStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/docs", true},
		{"http://localhost:8080", true},
		{"ftp://example.com", false},
		{"file:///path/to/file", false},
		{"git@github.com:user/repo.git", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCrawlerStrategy_SetFetcher tests the SetFetcher method
func TestCrawlerStrategy_SetFetcher(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewCrawlerStrategy(deps)
	originalFetcher := strategy.fetcher

	// Create a mock fetcher
	mockFetcher := &mockFetcher{}

	strategy.SetFetcher(mockFetcher)

	assert.NotEqual(t, originalFetcher, strategy.fetcher)
	assert.Equal(t, mockFetcher, strategy.fetcher)
}

// TestCrawlerStrategy_Execute_Simple tests basic execution
func TestCrawlerStrategy_Execute_Simple(t *testing.T) {
	// Create test server
	var visited []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = append(visited, r.URL.String())
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Test</title></head><body><h1>Content</h1></body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       1,
		Concurrency: 1,
		MaxDepth:    1,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, visited)
}

// TestCrawlerStrategy_Execute_WithLinks tests crawling with links
func TestCrawlerStrategy_Execute_WithLinks(t *testing.T) {
	var visited []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = append(visited, r.URL.String())
		w.Header().Set("Content-Type", "text/html")

		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<html><body><a href="/page1">Page 1</a><a href="/page2">Page 2</a></body></html>`))
		case "/page1", "/page2":
			w.Write([]byte(`<html><body>Content</body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       3,
		Concurrency: 1,
		MaxDepth:    2,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(visited), 1)
}

// TestCrawlerStrategy_Execute_WithExclude tests exclude patterns
func TestCrawlerStrategy_Execute_WithExclude(t *testing.T) {
	var visited []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = append(visited, r.URL.String())
		w.Header().Set("Content-Type", "text/html")

		if r.URL.Path == "/" {
			w.Write([]byte(`<html><body><a href="/api">API</a><a href="/docs">Docs</a><a href="/admin">Admin</a></body></html>`))
		} else {
			w.Write([]byte(`<html><body>Content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       10,
		Concurrency: 1,
		MaxDepth:    2,
		Exclude:     []string{"/admin", "/api.*"},
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)

	// Admin and /api should not be visited
	for _, v := range visited {
		assert.NotContains(t, v, "/admin")
	}
}

// TestCrawlerStrategy_Execute_WithFilterURL tests base URL filter
func TestCrawlerStrategy_Execute_WithFilterURL(t *testing.T) {
	var visited []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = append(visited, r.URL.String())
		w.Header().Set("Content-Type", "text/html")

		if r.URL.Path == "/" {
			w.Write([]byte(`<html><body><a href="/docs/guide">Guide</a><a href="/blog/post">Blog</a><a href="/api">API</a></body></html>`))
		} else {
			w.Write([]byte(`<html><body>Content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       10,
		Concurrency: 1,
		MaxDepth:    2,
		FilterURL:   server.URL + "/docs",
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)

	// Only /docs should be visited
	hasDocs := false
	hasBlog := false
	hasAPI := false
	for _, v := range visited {
		if strings.Contains(v, "/docs") {
			hasDocs = true
		}
		if strings.Contains(v, "/blog") {
			hasBlog = true
		}
		if strings.Contains(v, "/api") {
			hasAPI = true
		}
	}
	assert.True(t, hasDocs, "Should visit /docs")
	assert.False(t, hasBlog, "Should not visit /blog")
	assert.False(t, hasAPI, "Should not visit /api")
}

// TestCrawlerStrategy_Execute_ContextCancellation tests context cancellation
func TestCrawlerStrategy_Execute_ContextCancellation(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte(`<html><body><a href="/page1">Link</a></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		Limit:       10,
		Concurrency: 1,
		MaxDepth:    2,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "context canceled")
}

// TestCrawlerStrategy_Execute_WithMarkdown tests markdown content handling
func TestCrawlerStrategy_Execute_WithMarkdown(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(`# Markdown Content

This is a **test** markdown file.
`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       1,
		Concurrency: 1,
		MaxDepth:    1,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)
}

// TestIsHTMLContentType tests content type checking
func TestIsHTMLContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"text/html", "text/html", true},
		{"text/html with charset", "text/html; charset=utf-8", true},
		{"application/xhtml", "application/xhtml+xml", true},
		{"uppercase", "TEXT/HTML", true},
		{"empty", "", true},
		{"application/json", "application/json", false},
		{"text/plain", "text/plain", false},
		{"image/png", "image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTMLContentType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCrawlerStrategy_Execute_LimitTests tests page limit
func TestCrawlerStrategy_Execute_LimitTests(t *testing.T) {
	var visited int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited++
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/page1">1</a><a href="/page2">2</a></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       1,
		Concurrency: 1,
		MaxDepth:    2,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)
	// Should only process 1 page (the limit)
}

// TestCrawlerStrategy_Execute_DifferentDomains tests same-domain restriction
func TestCrawlerStrategy_Execute_DifferentDomains(t *testing.T) {
	var visited []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = append(visited, r.URL.String())
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="https://example.com/external">External</a></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		DryRun:         true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := Options{
		Limit:       10,
		Concurrency: 1,
		MaxDepth:    2,
		DryRun:      true,
	}

	err = strategy.Execute(ctx, server.URL+"/", opts)
	assert.NoError(t, err)

	// External domain should not be visited
	for _, v := range visited {
		assert.NotContains(t, v, "example.com")
	}
}

// Mock types for testing

type mockFetcher struct {
	getFunc func(ctx context.Context, url string) (*domain.Response, error)
}

func (m *mockFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url)
	}
	return &domain.Response{
		StatusCode:  http.StatusOK,
		Headers:     http.Header{"Content-Type": []string{"text/html"}},
		Body:        []byte("<html><body>Test</body></html>"),
		ContentType: "text/html",
		URL:         url,
		FromCache:   false,
	}, nil
}

func (m *mockFetcher) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return m.Get(ctx, url)
}

func (m *mockFetcher) GetCookies(url string) []*http.Cookie {
	return nil
}

func (m *mockFetcher) Transport() http.RoundTripper {
	return nil
}

func (m *mockFetcher) Close() error {
	return nil
}

func (m *mockFetcher) SetCache(cache domain.Cache) {}

func (m *mockFetcher) ClearCache() {}
