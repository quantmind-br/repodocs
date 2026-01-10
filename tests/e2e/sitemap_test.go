package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_SitemapParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock server with sitemap and pages
	mux := http.NewServeMux()

	// Serve sitemap
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// Use absolute URLs as required by sitemap spec
		baseURL := "http://" + r.Host
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<url>
				<loc>` + baseURL + `/docs/intro</loc>
				<lastmod>2024-01-15</lastmod>
			</url>
			<url>
				<loc>` + baseURL + `/docs/getting-started</loc>
				<lastmod>2024-01-14</lastmod>
			</url>
			<url>
				<loc>` + baseURL + `/docs/api</loc>
				<lastmod>2024-01-13</lastmod>
			</url>
		</urlset>`))
	})

	// Serve pages
	mux.HandleFunc("/docs/intro", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Introduction</title></head>
		<body>
			<h1>Introduction</h1>
			<p>Welcome to our documentation.</p>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/docs/getting-started", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Getting Started</title></head>
		<body>
			<h1>Getting Started</h1>
			<p>Follow these steps to get started.</p>
			<ol>
				<li>Install the package</li>
				<li>Configure settings</li>
				<li>Start using</li>
			</ol>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/docs/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>API Reference</title></head>
		<body>
			<h1>API Reference</h1>
			<p>Complete API documentation.</p>
			<h2>Endpoints</h2>
			<ul>
				<li>GET /api/users</li>
				<li>POST /api/users</li>
			</ul>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-sitemap-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create dependencies
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:    10 * time.Second,
		MaxRetries: 1,
	})
	require.NoError(t, err)
	defer fetcherClient.Close()

	cacheClient, err := cache.NewBadgerCache(cache.Options{
		InMemory: true,
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	logger := utils.NewLogger(utils.LoggerOptions{
		Level: "error",
	})

	pipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: server.URL,
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Flat:    false,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheClient,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	// Create sitemap strategy
	sitemapStrategy := strategies.NewSitemapStrategy(deps)

	// Execute sitemap parsing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = sitemapStrategy.Execute(ctx, server.URL+"/sitemap.xml", strategies.Options{
		Concurrency: 2,
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: false,
			Force:  true,
		},
	})
	require.NoError(t, err)

	// Verify files were created
	var mdFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})

	// Should have created 3 files (intro, getting-started, api)
	assert.Equal(t, 3, len(mdFiles), "Should have created 3 markdown files")
}

func TestE2E_SitemapIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock server with sitemap index
	mux := http.NewServeMux()

	// Serve sitemap index
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		// Use absolute URLs in the sitemap index
		baseURL := "http://" + r.Host
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<sitemap>
				<loc>` + baseURL + `/sitemap-docs.xml</loc>
				<lastmod>2024-01-15</lastmod>
			</sitemap>
		</sitemapindex>`))
	})

	// Serve sub-sitemap
	mux.HandleFunc("/sitemap-docs.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		baseURL := "http://" + r.Host
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<url>
				<loc>` + baseURL + `/docs/page1</loc>
			</url>
			<url>
				<loc>` + baseURL + `/docs/page2</loc>
			</url>
		</urlset>`))
	})

	// Serve pages
	mux.HandleFunc("/docs/page1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Page 1</h1><p>Content 1</p></body></html>`))
	})

	mux.HandleFunc("/docs/page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Page 2</h1><p>Content 2</p></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-sitemap-index-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fetcherClient, _ := fetcher.NewClient(fetcher.ClientOptions{Timeout: 10 * time.Second})
	defer fetcherClient.Close()

	cacheClient, _ := cache.NewBadgerCache(cache.Options{InMemory: true})
	defer cacheClient.Close()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	pipeline := converter.NewPipeline(converter.PipelineOptions{BaseURL: server.URL})
	writer := output.NewWriter(output.WriterOptions{BaseDir: tmpDir, Force: true})

	deps := &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheClient,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	sitemapStrategy := strategies.NewSitemapStrategy(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = sitemapStrategy.Execute(ctx, server.URL+"/sitemap.xml", strategies.Options{
		Concurrency: 2,
		CommonOptions: domain.CommonOptions{
			Limit: 10,
			Force: true,
		},
	})
	require.NoError(t, err)

	// Count created files
	var fileCount int
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			fileCount++
		}
		return nil
	})

	// Should have created 2 files from the sub-sitemap
	assert.Equal(t, 2, fileCount, "Should have created 2 markdown files from sitemap index")
}

func TestE2E_SitemapWithLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// Serve sitemap with many URLs
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		baseURL := "http://" + r.Host
		xml := `<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
		for i := 1; i <= 20; i++ {
			xml += `<url><loc>` + baseURL + `/page/` + strconv.Itoa(i) + `</loc></url>`
		}
		xml += `</urlset>`
		w.Write([]byte(xml))
	})

	// Serve all pages
	mux.HandleFunc("/page/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Page</h1><p>Content</p></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-sitemap-limit-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fetcherClient, _ := fetcher.NewClient(fetcher.ClientOptions{Timeout: 10 * time.Second})
	defer fetcherClient.Close()

	cacheClient, _ := cache.NewBadgerCache(cache.Options{InMemory: true})
	defer cacheClient.Close()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	pipeline := converter.NewPipeline(converter.PipelineOptions{BaseURL: server.URL})
	writer := output.NewWriter(output.WriterOptions{BaseDir: tmpDir, Force: true})

	deps := &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheClient,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	sitemapStrategy := strategies.NewSitemapStrategy(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Limit to 5 pages
	err = sitemapStrategy.Execute(ctx, server.URL+"/sitemap.xml", strategies.Options{
		Concurrency: 2,
		CommonOptions: domain.CommonOptions{
			Limit: 5,
			Force: true,
		},
	})
	require.NoError(t, err)

	// Count files
	var fileCount int
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			fileCount++
		}
		return nil
	})

	// Should respect the limit
	assert.LessOrEqual(t, fileCount, 5, "Should respect the limit of 5 pages")
}
