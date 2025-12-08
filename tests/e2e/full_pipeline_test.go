package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Binary path for CLI tests - built in setup
	cliBinary = "/tmp/repodocs"
)

// TestCrawl_RealWebsite tests the full crawl pipeline via CLI
func TestCrawl_RealWebsite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock website with realistic content
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Test Documentation Site</title>
			<meta charset="utf-8">
		</head>
		<body>
			<header>
				<h1>Documentation Site</h1>
				<nav>
					<a href="/getting-started">Getting Started</a>
					<a href="/api">API Reference</a>
					<a href="/guides">Guides</a>
				</nav>
			</header>
			<main>
				<h2>Welcome</h2>
				<p>This is a test documentation site.</p>
				<pre><code>console.log("Hello World");</code></pre>
			</main>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/getting-started", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Getting Started</title></head>
		<body>
			<h1>Getting Started</h1>
			<h2>Installation</h2>
			<p>Install the tool using:</p>
			<pre><code>npm install -g tool</code></pre>
			<h2>Quick Start</h2>
			<ol>
				<li>Download the tool</li>
				<li>Configure it</li>
				<li>Run it</li>
			</ol>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>API Reference</title></head>
		<body>
			<h1>API Reference</h1>
			<h2>Methods</h2>
			<h3>GET /endpoint</h3>
			<p>Returns data from the endpoint.</p>
			<h3>POST /endpoint</h3>
			<p>Creates new data.</p>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/guides", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Guides</title></head>
		<body>
			<h1>Guides</h1>
			<h2>Advanced Usage</h2>
			<p>Learn advanced features.</p>
			<a href="/guides/advanced">Read more</a>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/guides/advanced", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Advanced Guide</title></head>
		<body>
			<h1>Advanced Usage</h1>
			<p>Detailed advanced guide content.</p>
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

	// Run CLI command
	cmd := exec.Command(cliBinary, server.URL, "-o", tmpDir, "-d", "2", "--force")
	output, err := cmd.CombinedOutput()
	t.Logf("CLI Output: %s", string(output))
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify files were created
	var mdFiles []string
	var jsonFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".md" {
			mdFiles = append(mdFiles, path)
		} else if ext == ".json" {
			jsonFiles = append(jsonFiles, path)
		}
		return nil
	})

	t.Logf("Found %d markdown files and %d json files", len(mdFiles), len(jsonFiles))
	assert.Greater(t, len(mdFiles), 0, "Should have created markdown files")

	// JSON files are optional unless --json-meta is specified
	if len(jsonFiles) > 0 {
		t.Logf("JSON files created: %v", jsonFiles)
	}

	// Verify content of at least one file
	indexPath := filepath.Join(tmpDir, "index.md")
	if _, err := os.Stat(indexPath); err == nil {
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Documentation Site", "Should contain site title")
	}
}

// TestCrawl_GitHubRepo simulates a GitHub repository crawl
func TestCrawl_GitHubRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// Simulate GitHub README using simple paths
	mux.HandleFunc("/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>GitHub Repo</title></head>
		<body>
			<article class="markdown-body">
				<h1>Project Name</h1>
				<p>A great project for testing.</p>
				<h2>Installation</h2>
				<pre><code>go get github.com/owner/repo</code></pre>
				<h2>Usage</h2>
				<pre><code>repo --help</code></pre>
			</article>
		</body>
		</html>
		`))
	})

	mux.HandleFunc("/owner/repo/tree/main", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>File Tree</title></head>
		<body>
			<h1>Repository Structure</h1>
			<ul>
				<li><a href="/owner/repo/blob/main/README.md">README.md</a></li>
				<li><a href="/owner/repo/blob/main/docs/guide.md">docs/guide.md</a></li>
			</ul>
		</body>
		</html>
		`))
	})

	// Also handle the blob paths
	mux.HandleFunc("/owner/repo/blob/main/README.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>README</title></head>
		<body>
			<article class="markdown-body">
				<h1>README</h1>
				<p>Project documentation.</p>
			</article>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-github-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Use a URL that looks like GitHub
	githubURL := server.URL + "/owner/repo"

	// Run CLI with GitHub-like URL
	cmd := exec.Command(cliBinary, githubURL, "-o", tmpDir, "--force")
	output, err := cmd.CombinedOutput()
	t.Logf("CLI Output: %s", string(output))
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify output
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0, "Should have created output files")
}

// TestCrawl_PkgGoDev simulates a pkg.go.dev package page
func TestCrawl_PkgGoDev(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// Simulate pkg.go.dev page using simple paths
	mux.HandleFunc("/pkg/go.uber.org/zap", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>pkg.go.dev - go.uber.org/zap</title></head>
		<body>
			<header>
				<h1>go.uber.org/zap</h1>
				<p>Fast, structured, leveled logging in Go.</p>
			</header>
			<main>
				<section>
					<h2>Overview</h2>
					<p>High-performance logging library.</p>
					<pre><code>import "go.uber.org/zap"</code></pre>
				</section>
				<section>
					<h2>Example</h2>
					<pre><code>logger, _ := zap.NewProduction()
					defer logger.Sync()
					logger.Info("Hello", zap.String("url", "example.com"))</code></pre>
				</section>
			</main>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-pkggo-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Use a URL that looks like pkg.go.dev
	pkgURL := server.URL + "/pkg/go.uber.org/zap"

	// Run CLI with pkg.go.dev-like URL
	cmd := exec.Command(cliBinary, pkgURL, "-o", tmpDir, "--force")
	output, err := cmd.CombinedOutput()
	t.Logf("CLI Output: %s", string(output))
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify output
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Greater(t, len(files), 0, "Should have created output files")
}

// TestCrawl_Sitemap tests crawling via sitemap
func TestCrawl_Sitemap(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// Create server first to get URL
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create sitemap using server URL
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
		<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
			<url><loc>` + server.URL + `/page1</loc></url>
			<url><loc>` + server.URL + `/page2</loc></url>
			<url><loc>` + server.URL + `/page3</loc></url>
		</urlset>`))
	})

	// Create pages referenced in sitemap
	for i := 1; i <= 3; i++ {
		path := fmt.Sprintf("/page%d", i)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head><title>Page %d</title></head>
			<body>
				<h1>Page %d</h1>
				<p>Content of page %d.</p>
			</body>
			</html>`, i, i, i)))
		})
	}

	tmpDir, err := os.MkdirTemp("", "repodocs-sitemap-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Run CLI with sitemap URL
	cmd := exec.Command(cliBinary, server.URL+"/sitemap.xml", "-o", tmpDir, "--force")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify output
	var mdCount int
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			mdCount++
		}
		return nil
	})

	assert.Greater(t, mdCount, 0, "Should have created markdown files from sitemap")
}

// TestOutput_ValidMarkdown validates that output is valid markdown
func TestOutput_ValidMarkdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Test Page</title></head>
		<body>
			<h1>Main Title</h1>
			<h2>Subtitle</h2>
			<p>Paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
			<pre><code>code block</code></pre>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>
			<blockquote>
				<p>This is a quote.</p>
			</blockquote>
			<table>
				<tr><th>Header 1</th><th>Header 2</th></tr>
				<tr><td>Cell 1</td><td>Cell 2</td></tr>
			</table>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-markdown-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Run CLI
	cmd := exec.Command(cliBinary, server.URL, "-o", tmpDir, "--force")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Validate markdown files
	var markdownFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			markdownFiles = append(markdownFiles, path)
		}
		return nil
	})

	assert.Greater(t, len(markdownFiles), 0, "Should have markdown files")

	// Validate each markdown file
	for _, mdFile := range markdownFiles {
		content, err := os.ReadFile(mdFile)
		require.NoError(t, err)

		mdContent := string(content)

		// Check for basic markdown elements
		assert.Contains(t, mdContent, "#", "Should contain heading markers")
		assert.NotContains(t, mdContent, "<html>", "Should not contain raw HTML tags")
		assert.NotContains(t, mdContent, "<body>", "Should not contain raw HTML tags")

		// Check for proper markdown formatting
		hasValidMarkdown := strings.Contains(mdContent, "#") || // headers
			strings.Contains(mdContent, "*") || // italics/bold
			strings.Contains(mdContent, "`") // code blocks
		assert.True(t, hasValidMarkdown, "File %s should contain valid markdown elements", mdFile)
	}
}

// TestMetadata_ValidJSON validates JSON metadata files
func TestMetadata_ValidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2024 07:28:00 GMT")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Test Page</title>
			<meta name="description" content="Test description">
		</head>
		<body>
			<h1>Test Page</h1>
			<p>Test content.</p>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-metadata-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Run CLI with JSON metadata flag
	cmd := exec.Command(cliBinary, server.URL, "-o", tmpDir, "--json-meta", "--force")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Find and validate JSON metadata files
	var jsonFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" {
			jsonFiles = append(jsonFiles, path)
		}
		return nil
	})

	assert.Greater(t, len(jsonFiles), 0, "Should have created JSON metadata files")

	// Validate each JSON file
	for _, jsonFile := range jsonFiles {
		content, err := os.ReadFile(jsonFile)
		require.NoError(t, err)

		// Validate JSON is well-formed
		var metadata map[string]interface{}
		err = json.Unmarshal(content, &metadata)
		require.NoError(t, err, "JSON file should be valid: %s", jsonFile)

		// Check for expected fields
		assert.Contains(t, metadata, "url", "JSON should contain URL")
		assert.Contains(t, metadata, "title", "JSON should contain title")
		assert.Contains(t, metadata, "fetched_at", "JSON should contain fetched_at")

		// Verify fetched_at is a string
		fetchedAt, ok := metadata["fetched_at"].(string)
		assert.True(t, ok, "fetched_at should be a string")
		assert.NotEmpty(t, fetchedAt, "fetched_at should not be empty")

		// Verify URL matches expected format
		url, ok := metadata["url"].(string)
		assert.True(t, ok, "URL should be a string")
		assert.Contains(t, url, "http", "URL should be valid")
	}
}

// TestCache_PersistsBetweenRuns tests that cache persists between runs
func TestCache_PersistsBetweenRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Cache Test</title></head>
		<body>
			<h1>Cache Test Page</h1>
			<p>This content should be cached.</p>
		</body>
		</html>
		`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir1, err := os.MkdirTemp("", "repodocs-cache-run1-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "repodocs-cache-run2-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir2)

	// First run - should fetch and cache
	cmd1 := exec.Command(cliBinary, server.URL, "-o", tmpDir1, "--force")
	output1, err := cmd1.CombinedOutput()
	require.NoError(t, err, "First run failed: %s", string(output1))

	// Second run - should use cache (disable no-cache to ensure cache is used)
	cmd2 := exec.Command(cliBinary, server.URL, "-o", tmpDir2)
	output2, err := cmd2.CombinedOutput()
	require.NoError(t, err, "Second run failed: %s", string(output2))

	// Both runs should succeed
	// Note: We can't easily verify cache was actually used without inspecting
	// internal cache state, but we verify the runs complete successfully
	assert.Contains(t, string(output1), "Documentation extraction completed", "First run should succeed")
	assert.Contains(t, string(output2), "Documentation extraction completed", "Second run should succeed")

	// Verify outputs were created in both runs
	files1, _ := os.ReadDir(tmpDir1)
	files2, _ := os.ReadDir(tmpDir2)

	assert.Greater(t, len(files1), 0, "First run should create files")
	assert.Greater(t, len(files2), 0, "Second run should create files")
}

// TestConfig_Overrides tests CLI flag overrides
func TestConfig_Overrides(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Config Test</title></head>
		<body>
			<h1>Config Test Page</h1>
			<a href="/page1">Page 1</a>
			<a href="/page2">Page 2</a>
			<a href="/page3">Page 3</a>
			<a href="/admin/page">Admin Page</a>
		</body>
		</html>
		`))
	})

	for i := 1; i <= 3; i++ {
		path := fmt.Sprintf("/page%d", i)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`<html><body><h1>Page %d</h1></body></html>`, i)))
		})
	}

	mux.HandleFunc("/admin/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><h1>Admin Page</h1></body></html>`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-config-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test depth override
	cmd := exec.Command(cliBinary, server.URL, "-o", tmpDir, "-d", "1", "--exclude", ".*/admin.*", "--force")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify configuration was applied
	var mdFiles []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})

	// Should have created files despite depth limit
	assert.Greater(t, len(mdFiles), 0, "Should create files with custom config")

	// Verify exclude pattern worked - admin page should not be present
	var foundAdmin bool
	for _, file := range mdFiles {
		if strings.Contains(strings.ToLower(file), "admin") {
			foundAdmin = true
			break
		}
	}
	assert.False(t, foundAdmin, "Admin pages should be excluded")
}

// TestCLI_Integration tests overall CLI integration
func TestCLI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	mux := http.NewServeMux()

	// Create a multi-page site
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head><title>Integration Test Site</title></head>
		<body>
			<h1>Integration Test Site</h1>
			<nav>
				<a href="/docs/intro">Introduction</a>
				<a href="/docs/tutorial">Tutorial</a>
				<a href="/docs/api">API</a>
			</nav>
		</body>
		</html>
		`))
	})

	pages := map[string]string{
		"/docs/intro":    "Introduction",
		"/docs/tutorial": "Tutorial",
		"/docs/api":      "API Reference",
	}

	for path, title := range pages {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head><title>%s</title></head>
			<body>
				<h1>%s</h1>
				<p>Content for %s.</p>
			</body>
			</html>`, title, title, title)))
		})
	}

	server := httptest.NewServer(mux)
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "repodocs-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test various CLI flags together
	cmd := exec.Command(
		cliBinary,
		server.URL,
		"-o", tmpDir,
		"-d", "2",
		"-j", "2",
		"-l", "10",
		"--json-meta",
		"--force",
		"--timeout", "30s",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "CLI command failed: %s", string(output))

	// Verify success message in output
	assert.Contains(t, string(output), "Documentation extraction completed", "Should report success")

	// Verify output structure
	var mdFiles []string
	var jsonFiles []string

	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".md" {
			mdFiles = append(mdFiles, path)
		} else if ext == ".json" {
			jsonFiles = append(jsonFiles, path)
		}
		return nil
	})

	assert.Greater(t, len(mdFiles), 0, "Should create markdown files")
	assert.Greater(t, len(jsonFiles), 0, "Should create JSON metadata files")

	// Verify flat output structure if requested
	flatDir, err := os.MkdirTemp("", "repodocs-flat-*")
	require.NoError(t, err)
	defer os.RemoveAll(flatDir)

	cmd2 := exec.Command(cliBinary, server.URL, "-o", flatDir, "--nofolders", "--force")
	_, err = cmd2.CombinedOutput()
	require.NoError(t, err, "Flat output should work")

	// Count files in flat structure
	var flatFileCount int
	filepath.Walk(flatDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			flatFileCount++
		}
		return nil
	})

	assert.Greater(t, flatFileCount, 0, "Flat output should create files")

	// Test dry-run mode
	dryRunDir := filepath.Join(tmpDir, "dryrun")
	cmd3 := exec.Command(cliBinary, server.URL, "-o", dryRunDir, "--dry-run")
	_, err = cmd3.CombinedOutput()
	require.NoError(t, err, "Dry-run should work")

	// Dry-run should not create files
	files, _ := os.ReadDir(dryRunDir)
	assert.Equal(t, 0, len(files), "Dry-run should not create files")
}
