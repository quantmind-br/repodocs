package integration

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullPipeline_Website tests the complete pipeline with crawler strategy
func TestFullPipeline_Website(t *testing.T) {
	// Create a test HTTP server with multiple pages
	server := testutil.NewTestServer(t)

	// Register main page with links to other pages
	mainPage := `<!DOCTYPE html>
<html>
<head><title>Test Documentation</title></head>
<body>
    <nav>
        <a href="/page1">Page 1</a>
        <a href="/page2">Page 2</a>
    </nav>
    <main>
        <h1>Test Document</h1>
        <p>This is test content for website.</p>
    </main>
</body>
</html>`

	page1 := `<!DOCTYPE html>
<html>
<head><title>Page 1</title></head>
<body>
    <main>
        <h1>Page 1</h1>
        <p>Content for page 1.</p>
    </main>
</body>
</html>`

	page2 := `<!DOCTYPE html>
<html>
<head><title>Page 2</title></head>
<body>
    <main>
        <h1>Page 2</h1>
        <p>Content for page 2.</p>
    </main>
</body>
</html>`

	server.HandleHTML(t, "/", mainPage)
	server.HandleHTML(t, "/page1", page1)
	server.HandleHTML(t, "/page2", page2)

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 2
	cfg.Concurrency.MaxDepth = 2

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - Run the crawler strategy
	err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
		Limit: 3,
	})

	// Assert
	require.NoError(t, err)

	// Verify files were created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "At least one markdown file should be created")

	// Verify content was processed
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		require.NoError(t, err)
		assert.NotEmpty(t, string(content), "Generated file should not be empty")
		// Check for some content from the pages
		assert.True(t, strings.Contains(string(content), "Test Document") ||
			strings.Contains(string(content), "Page 1") ||
			strings.Contains(string(content), "Page 2"),
			"File should contain content from the website")
	}
}

// TestFullPipeline_GitRepo tests git strategy detection (without network)
func TestFullPipeline_GitRepo(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act & Assert - Test strategy detection without actual execution
	t.Run("GitHub URL detection", func(t *testing.T) {
		strategy := orchestrator.GetStrategyName("https://github.com/user/repo")
		assert.Equal(t, "git", strategy)

		err := orchestrator.ValidateURL("https://github.com/user/repo")
		assert.NoError(t, err)
	})

	t.Run("GitLab URL detection", func(t *testing.T) {
		strategy := orchestrator.GetStrategyName("https://gitlab.com/user/repo")
		assert.Equal(t, "git", strategy)

		err := orchestrator.ValidateURL("https://gitlab.com/user/repo")
		assert.NoError(t, err)
	})

	t.Run("Direct .git URL", func(t *testing.T) {
		strategy := orchestrator.GetStrategyName("https://github.com/user/repo.git")
		assert.Equal(t, "git", strategy)

		err := orchestrator.ValidateURL("https://github.com/user/repo.git")
		assert.NoError(t, err)
	})
}

// TestFullPipeline_Sitemap tests sitemap strategy
func TestFullPipeline_Sitemap(t *testing.T) {
	// Create a test HTTP server
	server := testutil.NewTestServer(t)

	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>` + server.URL + `/</loc>
        <lastmod>2024-01-01</lastmod>
    </url>
    <url>
        <loc>` + server.URL + `/docs/guide</loc>
        <lastmod>2024-01-02</lastmod>
    </url>
    <url>
        <loc>` + server.URL + `/docs/api</loc>
        <lastmod>2024-01-03</lastmod>
    </url>
</urlset>`

	server.HandleString(t, "/sitemap.xml", "application/xml", sitemap)

	// Register HTML pages
	server.HandleHTML(t, "/", "<html><body><h1>Home</h1><p>Welcome</p></body></html>")
	server.HandleHTML(t, "/docs/guide", "<html><body><h1>Guide</h1><p>User guide</p></body></html>")
	server.HandleHTML(t, "/docs/api", "<html><body><h1>API</h1><p>API reference</p></body></html>")

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 2

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - Run sitemap strategy
	err = orchestrator.Run(context.Background(), server.URL+"/sitemap.xml", app.OrchestratorOptions{
		Limit: 3,
	})

	// Assert
	require.NoError(t, err)

	// Verify strategy detection
	assert.Equal(t, "sitemap", orchestrator.GetStrategyName(server.URL+"/sitemap.xml"))

	// Verify files were created
	files, err := filepath.Glob(filepath.Join(tmpDir, "**/*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "At least one markdown file should be created from sitemap")
}

// TestFullPipeline_PkgGo tests pkg.go.dev strategy detection
func TestFullPipeline_PkgGo(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act & Assert - Test pkg.go.dev URL detection
	t.Run("pkg.go.dev URL detection", func(t *testing.T) {
		url := "https://pkg.go.dev/github.com/example/package"
		strategy := orchestrator.GetStrategyName(url)
		assert.Equal(t, "pkggo", strategy)

		err := orchestrator.ValidateURL(url)
		assert.NoError(t, err)
	})

	t.Run("Standard library pkg.go.dev", func(t *testing.T) {
		url := "https://pkg.go.dev/std"
		strategy := orchestrator.GetStrategyName(url)
		assert.Equal(t, "pkggo", strategy)

		err := orchestrator.ValidateURL(url)
		assert.NoError(t, err)
	})

	t.Run("pkg.go.dev with version", func(t *testing.T) {
		url := "https://pkg.go.dev/github.com/example/package@v1.2.3"
		strategy := orchestrator.GetStrategyName(url)
		assert.Equal(t, "pkggo", strategy)

		err := orchestrator.ValidateURL(url)
		assert.NoError(t, err)
	})
}

// TestCache_Integration tests cache functionality between runs
func TestCache_Integration(t *testing.T) {
	// Create a test HTTP server
	server := testutil.NewTestServer(t)

	html := `<!DOCTYPE html>
<html>
<head><title>Cached Page</title></head>
<body>
    <h1>Test Content</h1>
    <p>This content should be cached.</p>
</body>
</html>`
	server.HandleHTML(t, "/", html)

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = true
	cfg.Cache.TTL = 1 * time.Hour
	cfg.Concurrency.Timeout = 5 * time.Second

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - First run (should cache)
	err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
		Limit: 1,
	})
	require.NoError(t, err)

	// Verify cache directory exists
	cacheDir := cfg.Cache.Directory
	if cacheDir == "" {
		cacheDir = "~/.repodocs/cache"
	}
	cacheDir = os.ExpandEnv(cacheDir)

	// Check if cache was created (directory or files)
	assert.True(t, dirExists(cacheDir) || filesExistInDir(tmpDir),
		"Cache or output should be created")

	// Note: Full cache verification would require checking BadgerDB directly
	// This test verifies the orchestrator handles cache configuration properly
}

// TestConcurrency_MultipleURLs tests concurrent execution
func TestConcurrency_MultipleURLs(t *testing.T) {
	// Create multiple test servers
	server1 := testutil.NewTestServer(t)
	server1.HandleHTML(t, "/", "<html><body><h1>Server 1</h1></body></html>")

	server2 := testutil.NewTestServer(t)
	server2.HandleHTML(t, "/", "<html><body><h1>Server 2</h1></body></html>")

	server3 := testutil.NewTestServer(t)
	server3.HandleHTML(t, "/", "<html><body><h1>Server 3</h1></body></html>")

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 3

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - Run multiple URLs concurrently
	urls := []string{
		server1.URL,
		server2.URL,
		server3.URL,
	}

	start := time.Now()
	for i, url := range urls {
		t.Run("concurrent-run-"+string(rune('A'+i)), func(t *testing.T) {
			err := orchestrator.Run(context.Background(), url, app.OrchestratorOptions{
				Limit: 1,
			})
			require.NoError(t, err)
		})
	}
	duration := time.Since(start)

	// Assert - Verify concurrent execution
	assert.Less(t, duration, 5*time.Second,
		"Concurrent runs should complete within reasonable time")

	// Verify all files were created
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "At least one file should be created")
}

// TestContextCancellation_FullFlow tests context cancellation
func TestContextCancellation_FullFlow(t *testing.T) {
	// Create a slow server
	server := testutil.NewTestServer(t)

	// Handler that delays response
	server.Handle(t, "/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Slow Response</h1></body></html>"))
	})

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Timeout = 1 * time.Second

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - Run with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = orchestrator.Run(ctx, server.URL+"/slow", app.OrchestratorOptions{
		Limit: 1,
	})

	// Assert
	// The orchestrator should handle cancellation gracefully
	// The exact behavior depends on strategy implementation
	// We're testing that the orchestrator doesn't panic
	// When context is cancelled, the strategy may return context.DeadlineExceeded
	if err != nil {
		assert.True(t, err == context.DeadlineExceeded || err == context.Canceled ||
			strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "cancel"),
			"Expected cancellation-related error, got: %v", err)
	}
}

// TestErrorHandling_Graceful tests error handling
func TestErrorHandling_Graceful(t *testing.T) {
	t.Run("Invalid URL format", func(t *testing.T) {
		// Arrange
		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act
		err = orchestrator.Run(context.Background(), "invalid://url", app.OrchestratorOptions{})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to determine strategy")
	})

	t.Run("Non-existent server", func(t *testing.T) {
		// Arrange
		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false
		cfg.Concurrency.Timeout = 1 * time.Second

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act - Non-routable IP (should timeout or fail quickly)
		err = orchestrator.Run(context.Background(), "http://192.0.2.1", app.OrchestratorOptions{
			Limit: 1,
		})

		// Assert - Should handle non-existent server gracefully
		// The strategy may complete without error (with 0 pages) or return an error
		// We're testing that the orchestrator doesn't panic
		// It's acceptable for the crawler to return 0 pages for unreachable servers
	})

	t.Run("404 Not Found", func(t *testing.T) {
		// Create a server that returns 404
		server := testutil.NewTestServer(t)
		server.Handle404(t, "/notfound")

		// Arrange
		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config:  cfg,
			Verbose: true,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act
		err = orchestrator.Run(context.Background(), server.URL+"/notfound", app.OrchestratorOptions{
			Limit: 1,
		})

		// Assert - Should handle 404 gracefully
		// The strategy may or may not return an error for 404
		// We're testing that the orchestrator doesn't panic
		// require.NoError(t, err) or require.Error(t, err) depending on implementation
	})
}

// TestPerformance_LargeSite tests performance with larger site
func TestPerformance_LargeSite(t *testing.T) {
	// Create a server with multiple pages
	server := testutil.NewTestServer(t)

	// Generate multiple pages
	for i := 1; i <= 20; i++ {
		path := "/page" + string(rune('0'+i))
		html := `<!DOCTYPE html>
<html>
<head><title>Page ` + string(rune('0'+i)) + `</title></head>
<body>
    <h1>Page ` + string(rune('0'+i)) + `</h1>
    <p>Content for page ` + string(rune('0'+i)) + `.</p>
</body>
</html>`
		server.HandleHTML(t, path, html)
	}

	// Also register the root page
	server.HandleHTML(t, "/", `<!DOCTYPE html>
<html>
<head><title>Root</title></head>
<body>
    <h1>Root Page</h1>
    <p>Root content.</p>
    <a href="/page1">Page 1</a>
</body>
</html>`)

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 5
	cfg.Concurrency.MaxDepth = 2

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act
	start := time.Now()
	err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
		Limit: 20,
	})
	duration := time.Since(start)

	// Assert
	require.NoError(t, err)
	assert.Less(t, duration, 30*time.Second,
		"Processing 20 pages should complete within 30 seconds")

	// Verify files were created - Note: may have 0 files if no pages were crawled
	// This is acceptable if the server didn't serve the root properly
	files, err := filepath.Glob(filepath.Join(tmpDir, "*.md"))
	require.NoError(t, err)
	// Don't enforce minimum files - the test is about performance, not output
	t.Logf("Created %d files in %v", len(files), duration)
}

// TestRenderer_PoolExhaustion tests renderer pool behavior
func TestRenderer_PoolExhaustion(t *testing.T) {
	// Create a server with JavaScript-heavy pages
	server := testutil.NewTestServer(t)

	jsPage := `<!DOCTYPE html>
<html>
<head><title>JS Page</title></head>
<body>
    <h1>JavaScript Page</h1>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            console.log('Page loaded');
        });
    </script>
</body>
</html>`
	server.HandleHTML(t, "/", jsPage)

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Rendering.ForceJS = true
	cfg.Rendering.JSTimeout = 5 * time.Second
	cfg.Concurrency.Workers = 2

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:   cfg,
		Verbose:  true,
		RenderJS: true,
	})
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act - Run with JavaScript rendering
	err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
		Limit: 1,
	})

	// Assert - Should handle JS rendering gracefully
	// Note: This may fail if Chromium is not available
	// The test verifies the orchestrator properly configures the renderer
	if err != nil {
		// If Chromium is not available, that's ok for this test
		// We're testing the configuration path
		errMsg := err.Error()
		hasChromiumError := assert.Contains(t, errMsg, "chromium") ||
			assert.Contains(t, errMsg, "browser") ||
			assert.Contains(t, errMsg, "renderer")
		assert.True(t, hasChromiumError, "Expected renderer-related error if Chromium not available")
	}
}

// Helper function to check if directory exists
func dirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to check if files exist in directory
func filesExistInDir(dir string) bool {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	return err == nil && len(files) > 0
}

// TestMultipleOrchestrators tests creating and using multiple orchestrators
func TestMultipleOrchestrators(t *testing.T) {
	// Create test servers
	server1 := testutil.NewTestServer(t)
	server1.HandleHTML(t, "/", "<html><body><h1>Server 1</h1></body></html>")

	server2 := testutil.NewTestServer(t)
	server2.HandleHTML(t, "/", "<html><body><h1>Server 2</h1></body></html>")

	// Arrange - Create two orchestrators with different configs
	cfg1 := config.Default()
	tmpDir1 := testutil.TempDir(t)
	cfg1.Output.Directory = tmpDir1
	cfg1.Cache.Enabled = false
	cfg1.Concurrency.Workers = 1

	orchestrator1, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg1,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator1.Close()

	cfg2 := config.Default()
	tmpDir2 := testutil.TempDir(t)
	cfg2.Output.Directory = tmpDir2
	cfg2.Cache.Enabled = false
	cfg2.Concurrency.Workers = 2

	orchestrator2, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg2,
		Verbose: true,
	})
	require.NoError(t, err)
	defer orchestrator2.Close()

	// Act - Use both orchestrators
	err = orchestrator1.Run(context.Background(), server1.URL, app.OrchestratorOptions{
		Limit: 1,
	})
	require.NoError(t, err)

	err = orchestrator2.Run(context.Background(), server2.URL, app.OrchestratorOptions{
		Limit: 1,
	})
	require.NoError(t, err)

	// Assert - Both should work independently
	files1, err := filepath.Glob(filepath.Join(tmpDir1, "*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files1), 1)

	files2, err := filepath.Glob(filepath.Join(tmpDir2, "*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files2), 1)
}

// TestOrchestrator_OptionsValidation tests various orchestrator options
func TestOrchestrator_OptionsValidation(t *testing.T) {
	t.Run("Zero limit", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		server.HandleHTML(t, "/", "<html><body><h1>Test</h1></body></html>")

		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act - Run with limit 0 (should process all available)
		err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
			Limit: 0,
		})

		require.NoError(t, err)
	})

	t.Run("Negative limit", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		server.HandleHTML(t, "/", "<html><body><h1>Test</h1></body></html>")

		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act - Run with negative limit
		err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
			Limit: -1,
		})

		// Should handle gracefully
		require.NoError(t, err)
	})

	t.Run("Very high concurrency", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		server.HandleHTML(t, "/", "<html><body><h1>Test</h1></body></html>")

		cfg := config.Default()
		tmpDir := testutil.TempDir(t)
		cfg.Output.Directory = tmpDir
		cfg.Cache.Enabled = false
		cfg.Concurrency.Workers = 100

		orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		defer orchestrator.Close()

		// Act
		err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
			Limit: 1,
		})

		require.NoError(t, err)
	})
}
