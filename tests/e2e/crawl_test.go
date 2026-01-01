package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_CrawlMockSite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock website with multiple pages
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Home Page</title></head>
		<body>
			<h1>Welcome to Test Site</h1>
			<p>This is the home page.</p>
			<a href="/about">About Us</a>
			<a href="/docs">Documentation</a>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>About Page</title></head>
		<body>
			<h1>About Us</h1>
			<p>We are a test company.</p>
			<a href="/">Home</a>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Documentation</title></head>
		<body>
			<h1>Documentation</h1>
			<p>Read our documentation here.</p>
			<a href="/docs/getting-started">Getting Started</a>
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
			<p>Follow these steps to get started:</p>
			<ol>
				<li>Step one</li>
				<li>Step two</li>
				<li>Step three</li>
			</ol>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-*")
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
		BaseDir:      tmpDir,
		Flat:         false,
		JSONMetadata: true,
		Force:        true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheClient,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	// Create crawler strategy
	crawler := strategies.NewCrawlerStrategy(deps)

	// Execute crawl
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = crawler.Execute(ctx, server.URL, strategies.Options{
		MaxDepth:    2,
		Limit:       10,
		Concurrency: 2,
		DryRun:      false,
		Force:       true,
	})
	require.NoError(t, err)

	// Verify files were created
	var files []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			files = append(files, path)
		}
		return nil
	})

	// Should have created at least the index file
	assert.Greater(t, len(files), 0, "Should have created at least one markdown file")
}

func TestE2E_CrawlWithLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock website with many pages
	mux := http.NewServeMux()

	for i := 1; i <= 20; i++ {
		path := "/page-" + string(rune('0'+i))
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<body>
				<h1>Page Content</h1>
				<p>Content for this page.</p>
			</body>
			</html>
			`))
		})
	}

	// Home page links to all other pages
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html><html><body><h1>Home</h1>`
		for i := 1; i <= 20; i++ {
			html += `<a href="/page-` + string(rune('0'+i)) + `">Page ` + string(rune('0'+i)) + `</a>`
		}
		html += `</body></html>`
		w.Write([]byte(html))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create temporary output directory
	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-limit-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create dependencies (simplified for this test)
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer fetcherClient.Close()

	cacheClient, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer cacheClient.Close()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	pipeline := converter.NewPipeline(converter.PipelineOptions{BaseURL: server.URL})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheClient,
		Converter: pipeline,
		Writer:    writer,
		Logger:    logger,
	}

	crawler := strategies.NewCrawlerStrategy(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Crawl with limit of 5
	err = crawler.Execute(ctx, server.URL, strategies.Options{
		MaxDepth:    2,
		Limit:       5,
		Concurrency: 2,
		Force:       true,
	})
	require.NoError(t, err)

	// Count files
	var fileCount int
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			fileCount++
		}
		return nil
	})

	// Should have at most 5 files due to limit
	assert.LessOrEqual(t, fileCount, 5, "Should respect the limit")
}

func TestE2E_CrawlWithExclude(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<body>
			<h1>Home</h1>
			<a href="/docs">Docs</a>
			<a href="/admin">Admin</a>
			<a href="/login">Login</a>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Documentation</h1></body></html>`))
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Admin Panel</h1></body></html>`))
	})

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Login Page</h1></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-exclude-*")
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

	crawler := strategies.NewCrawlerStrategy(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Crawl with exclusions
	err = crawler.Execute(ctx, server.URL, strategies.Options{
		MaxDepth:    2,
		Concurrency: 1,
		Force:       true,
		Exclude:     []string{".*/admin.*", ".*/login.*"},
	})
	require.NoError(t, err)

	// Check that admin and login were excluded
	var foundAdmin, foundLogin bool
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "admin.md" {
			foundAdmin = true
		}
		if filepath.Base(path) == "login.md" {
			foundLogin = true
		}
		return nil
	})

	assert.False(t, foundAdmin, "Admin page should be excluded")
	assert.False(t, foundLogin, "Login page should be excluded")
}

func TestE2E_CrawlMarkdownContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// HTML home page with links to markdown files
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<body>
			<h1>Documentation</h1>
			<a href="/readme.md">README</a>
			<a href="/api-docs">API Docs</a>
		</body>
		</html>
		`))
	})

	// Markdown file served with .md extension
	mux.HandleFunc("/readme.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		w.Write([]byte(`# Project README

This is the project readme file.

## Features

- Feature one
- Feature two
- Feature three

## Installation

` + "```bash\ngo install ./...\n```" + `
`))
	})

	// Markdown content served with text/markdown content-type
	mux.HandleFunc("/api-docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write([]byte(`# API Documentation

## Endpoints

### GET /api/users

Returns a list of users.

### POST /api/users

Creates a new user.
`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-e2e-markdown-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{Timeout: 10 * time.Second})
	require.NoError(t, err)
	defer fetcherClient.Close()

	cacheClient, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
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

	crawler := strategies.NewCrawlerStrategy(deps)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = crawler.Execute(ctx, server.URL, strategies.Options{
		MaxDepth:    2,
		Concurrency: 1,
		Force:       true,
	})
	require.NoError(t, err)

	// Verify markdown files were created
	var files []string
	var readmeFound, apiDocsFound bool

	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			files = append(files, filepath.Base(path))
			// Check for readme file (may be named readme.md or readme-md.md depending on output logic)
			if filepath.Base(path) == "readme.md" || filepath.Base(path) == "readme-md.md" {
				readmeFound = true
				// Verify content was preserved
				content, _ := os.ReadFile(path)
				assert.Contains(t, string(content), "Project README", "README content should be preserved")
			}
			if filepath.Base(path) == "api-docs.md" {
				apiDocsFound = true
				content, _ := os.ReadFile(path)
				assert.Contains(t, string(content), "API Documentation", "API docs content should be preserved")
			}
		}
		return nil
	})

	assert.Greater(t, len(files), 0, "Should have created markdown files")
	// At least one of the markdown content should be processed
	assert.True(t, readmeFound || apiDocsFound, "Should have processed at least one markdown file")
}
