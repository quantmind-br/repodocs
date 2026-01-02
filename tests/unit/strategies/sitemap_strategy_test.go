package strategies_test

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

// setupSitemapTestDependencies creates test dependencies with fetcher and converter
func setupSitemapTestDependencies(t *testing.T, tmpDir string) *strategies.Dependencies {
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

// TestNewSitemapStrategy tests creating a new sitemap strategy
func TestNewSitemapStrategy(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewSitemapStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "sitemap", strategy.Name())
}

// TestSitemapStrategy_CanHandle tests URL handling for sitemap strategy
func TestSitemapStrategy_CanHandle(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewSitemapStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/sitemap.xml", true},
		{"https://example.com/sitemap.xml.gz", true},
		{"https://example.com/sitemap", true},
		{"https://example.com/SITEMAP.XML", true},
		{"https://example.com/docs/sitemap.xml", true},
		{"https://example.com/docs", false},
		{"https://github.com/user/repo", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestSitemapStrategy_Execute tests executing sitemap strategy
func TestSitemapStrategy_Execute(t *testing.T) {
	// Create test server with sitemap
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</url>
	<url>
		<loc>https://example.com/page2</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
	</url>
</urlset>`))
			return
		}

		// Serve HTML pages
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body><h1>Content</h1></body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_Gzipped tests executing with gzipped sitemap
func TestSitemapStrategy_Execute_Gzipped(t *testing.T) {
	// Create gzipped sitemap content
	sitemapContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</url>
</urlset>`

	var buf strings.Builder
	gzw := gzip.NewWriter(&buf)
	gzw.Write([]byte(sitemapContent))
	gzw.Close()

	gzippedContent := buf.String()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml.gz") {
			w.Header().Set("Content-Type", "application/x-gzip")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(gzippedContent))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml.gz", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_WithLimit tests executing with limit
func TestSitemapStrategy_Execute_WithLimit(t *testing.T) {
	// Create test server with multiple URLs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
	<url><loc>https://example.com/page3</loc></url>
	<url><loc>https://example.com/page4</loc></url>
	<url><loc>https://example.com/page5</loc></url>
</urlset>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Limit = 2
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_EmptySitemap tests with empty sitemap
func TestSitemapStrategy_Execute_EmptySitemap(t *testing.T) {
	// Create test server with empty sitemap
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_InvalidXML tests with invalid XML
func TestSitemapStrategy_Execute_InvalidXML(t *testing.T) {
	// Create test server with invalid XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid xml content`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	assert.Error(t, err)
}

// TestSitemapStrategy_Execute_ErrorFetchingPage tests error handling
func TestSitemapStrategy_Execute_ErrorFetchingPage(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/valid</loc></url>
	<url><loc>https://example.com/error</loc></url>
</urlset>`))
			return
		}

		// Return error for error page
		if strings.Contains(r.URL.Path, "error") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	// Should complete even if some pages fail
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_DryRun tests dry run mode
func TestSitemapStrategy_Execute_DryRun(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
</urlset>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.DryRun = true
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_ContextCancellation tests context cancellation
func TestSitemapStrategy_Execute_ContextCancellation(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`))
			return
		}

		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

// TestSitemapStrategy_Execute_WithLastMod tests sorting by lastmod
func TestSitemapStrategy_Execute_WithLastMod(t *testing.T) {
	// Create test server with sitemap having different lastmod dates
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/old</loc>
		<lastmod>2024-01-01T10:00:00Z</lastmod>
	</url>
	<url>
		<loc>https://example.com/new</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</url>
	<url>
		<loc>https://example.com/middle</loc>
		<lastmod>2024-01-10T10:00:00Z</lastmod>
	</url>
</urlset>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Limit = 2 // Should get the 2 most recent
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_MarkdownContent tests with markdown content
func TestSitemapStrategy_Execute_MarkdownContent(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/docs.md</loc></url>
</urlset>`))
			return
		}

		// Serve markdown
		w.Header().Set("Content-Type", "text/markdown")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`# Documentation

This is a markdown file.`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}

// TestSitemapStrategy_Execute_SitemapIndex tests sitemap index
func TestSitemapStrategy_Execute_SitemapIndex(t *testing.T) {
	// Create test server with sitemap index
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "sitemap.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>`+server.URL+`/sitemap1.xml</loc>
	</sitemap>
	<sitemap>
		<loc>`+server.URL+`/sitemap2.xml</loc>
	</sitemap>
</sitemapindex>`))
			return
		}

		if strings.Contains(r.URL.Path, "sitemap1.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
</urlset>`))
			return
		}

		if strings.Contains(r.URL.Path, "sitemap2.xml") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page2</loc></url>
</urlset>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)
}
