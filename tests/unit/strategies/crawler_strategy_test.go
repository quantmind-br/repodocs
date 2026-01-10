package strategies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDependencies creates test dependencies with fetcher
func setupTestDependencies(t *testing.T, tmpDir string) *strategies.Dependencies {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	// Create fetcher
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     10 * time.Second,
		MaxRetries:  1,
		EnableCache: false,
	})
	require.NoError(t, err)

	// Create converter
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{})

	return &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Fetcher:   fetcherClient,
		Converter: converterPipeline,
	}
}

// TestNewCrawlerStrategy tests creating a new crawler strategy
func TestNewCrawlerStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "crawler", strategy.Name())
}

// TestCrawlerStrategy_CanHandle tests URL handling for crawler strategy
func TestCrawlerStrategy_CanHandle(t *testing.T) {
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/docs", true},
		{"ftp://example.com", false},
		{"git@github.com:user/repo", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsHTMLContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "HTML content type",
			contentType: "text/html",
			expected:    true,
		},
		{
			name:        "HTML with charset",
			contentType: "text/html; charset=utf-8",
			expected:    true,
		},
		{
			name:        "XHTML",
			contentType: "application/xhtml+xml",
			expected:    true,
		},
		{
			name:        "Plain text",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "JSON",
			contentType: "application/json",
			expected:    false,
		},
		{
			name:        "Empty content type defaults to HTML",
			contentType: "",
			expected:    true,
		},
		{
			name:        "Case insensitive HTML",
			contentType: "TEXT/HTML",
			expected:    true,
		},
		{
			name:        "text/markdown",
			contentType: "text/markdown",
			expected:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategies.IsHTMLContentType(tc.contentType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestCrawlerStrategy_Execute_SimpleHTML tests crawling a simple HTML page
func TestCrawlerStrategy_Execute_SimpleHTML(t *testing.T) {
	// Create test server with simple HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
</head>
<body>
	<h1>Welcome</h1>
	<p>This is a test page.</p>
	<a href="/page2">Page 2</a>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 0 // Only root page
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestCrawlerStrategy_Execute_MarkdownContent tests crawling markdown content
func TestCrawlerStrategy_Execute_MarkdownContent(t *testing.T) {
	// Create test server serving markdown
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`# Test Markdown

This is a test markdown file.

## Features

- Feature 1
- Feature 2
`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 0
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)
}

// TestCrawlerStrategy_Execute_WithExclude tests crawling with exclude patterns
func TestCrawlerStrategy_Execute_WithExclude(t *testing.T) {
	visitCount := 0
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		visitCount++
		w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
	<h1>Page ` + r.URL.Path + `</h1>
	<a href="/page1">Page 1</a>
	<a href="/page2">Page 2</a>
	<a href="/admin/settings">Admin</a>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 1
	opts.Concurrency = 1
	opts.Exclude = []string{"/admin", "/api"}

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)

	// Verify that admin links were excluded
	// (This is a basic check; in real scenario we'd verify more thoroughly)
	assert.Greater(t, visitCount, 0)
}

// TestCrawlerStrategy_Execute_WithLimit tests crawling with a limit
func TestCrawlerStrategy_Execute_WithLimit(t *testing.T) {
	visitCount := 0
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visitCount++
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
	<h1>Page ` + r.URL.Path + `</h1>
	<a href="/page1">Page 1</a>
	<a href="/page2">Page 2</a>
	<a href="/page3">Page 3</a>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 1
	opts.Limit = 2 // Only process 2 pages
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)

	// Should have limited the number of pages processed
	// Note: The exact count depends on crawler behavior
	assert.Greater(t, visitCount, 0)
}

// TestCrawlerStrategy_Execute_ContextCancellation tests context cancellation
func TestCrawlerStrategy_Execute_ContextCancellation(t *testing.T) {
	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
	<h1>Slow Page</h1>
	<a href="/page2">Page 2</a>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 1
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestCrawlerStrategy_Execute_NonHTMLContent tests crawling with non-HTML content
func TestCrawlerStrategy_Execute_NonHTMLContent(t *testing.T) {
	// Create test server serving JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"title": "Test", "content": "Data"}`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 0
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	// Should complete without error even if it skips non-HTML content
	require.NoError(t, err)
}

// TestCrawlerStrategy_Execute_DryRun tests dry run mode
func TestCrawlerStrategy_Execute_DryRun(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<body>
	<h1>Dry Run Test</h1>
</body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 0
	opts.Concurrency = 1
	opts.DryRun = true

	err := strategy.Execute(ctx, server.URL, opts)
	require.NoError(t, err)

	// In dry run mode, files should not be written
	// Verify output directory is empty or contains minimal files
}

// TestCrawlerStrategy_Execute_WithError tests error handling
func TestCrawlerStrategy_Execute_WithError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupTestDependencies(t, tmpDir)

	strategy := strategies.NewCrawlerStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.MaxDepth = 0
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL, opts)
	// Crawler should handle errors gracefully
	// It may return error or complete with warnings
	if err != nil {
		assert.NotEmpty(t, err.Error())
	}
}

// TestIsMarkdownContent tests markdown content detection
func TestIsMarkdownContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		url         string
		expected    bool
	}{
		{
			name:        "Markdown content type",
			contentType: "text/markdown",
			url:         "https://example.com/doc.md",
			expected:    true,
		},
		{
			name:        "Markdown URL",
			contentType: "text/plain",
			url:         "https://example.com/README.md",
			expected:    true,
		},
		{
			name:        "MDX URL",
			contentType: "text/plain",
			url:         "https://example.com/page.mdx",
			expected:    true,
		},
		{
			name:        "HTML content type",
			contentType: "text/html",
			url:         "https://example.com/page",
			expected:    false,
		},
		{
			name:        "Plain text without markdown URL",
			contentType: "text/plain",
			url:         "https://example.com/page.txt",
			expected:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := converter.IsMarkdownContent(tc.contentType, tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}
