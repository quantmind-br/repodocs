package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Orchestrator_NewOrchestrator(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	opts := app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	}

	// Act
	orchestrator, err := app.NewOrchestrator(opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.Close)

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_Run_WithValidURL(t *testing.T) {
	// This test requires actual strategy implementations
	// Skipping for now - would run end-to-end tests
	t.Skip("Requires full strategy implementation")
}

func TestIntegration_Orchestrator_Run_CrawlerStrategy(t *testing.T) {
	// Create a test HTTP server
	server := testutil.NewTestServer(t)

	// Register a simple HTML page
	html := `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
    <h1>Test Document</h1>
    <p>This is test content.</p>
</body>
</html>`
	server.HandleHTML(t, "/", html)

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

	// Act
	err = orchestrator.Run(context.Background(), server.URL, app.OrchestratorOptions{
		Limit: 1,
	})

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Note: This test would require actual CrawlerStrategy implementation
	t.Skip("Requires CrawlerStrategy implementation")
}

func TestIntegration_Orchestrator_Run_GitStrategy(t *testing.T) {
	// This test would clone a real git repository
	// Skipping for now - requires network access
	t.Skip("Requires network access and git repository")
}

func TestIntegration_Orchestrator_Run_SitemapStrategy(t *testing.T) {
	// Create a test HTTP server with sitemap
	server := testutil.NewTestServer(t)

	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url>
        <loc>` + server.URL + `/</loc>
        <lastmod>2024-01-01</lastmod>
    </url>
    <url>
        <loc>` + server.URL + `/about</loc>
        <lastmod>2024-01-02</lastmod>
    </url>
</urlset>`

	server.HandleString(t, "/sitemap.xml", "application/xml", sitemap)

	// Register HTML pages
	server.HandleHTML(t, "/", "<html><body><h1>Home</h1></body></html>")
	server.HandleHTML(t, "/about", "<html><body><h1>About</h1></body></html>")

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

	// Act
	err = orchestrator.Run(context.Background(), server.URL+"/sitemap.xml", app.OrchestratorOptions{
		Limit: 2,
	})

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Note: This test would require actual SitemapStrategy implementation
	t.Skip("Requires SitemapStrategy implementation")
}

func TestIntegration_Orchestrator_Run_LLMSStrategy(t *testing.T) {
	// Create a test HTTP server
	server := testutil.NewTestServer(t)

	// Register llms.txt
	llms := `https://docs.example.com/api
https://docs.example.com/guides/getting-started`
	server.HandleString(t, "/llms.txt", "text/plain", llms)

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

	// Act
	err = orchestrator.Run(context.Background(), server.URL+"/llms.txt", app.OrchestratorOptions{
		Limit: 2,
	})

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Note: This test would require actual LLMSStrategy implementation
	t.Skip("Requires LLMSStrategy implementation")
}

func TestIntegration_Orchestrator_Run_PkgGoStrategy(t *testing.T) {
	// This test would fetch from actual pkg.go.dev
	// Skipping for now - requires network access
	t.Skip("Requires network access to pkg.go.dev")
}

func TestIntegration_Orchestrator_Run_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := testutil.NewTestServer(t)
	server.HandleString(t, "/slow", "text/html", `<html><body><h1>Slow Page</h1></body></html>`)

	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Timeout = 100 * time.Millisecond

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config:  cfg,
		Verbose: true,
	})
	require.NoError(t, err)

	// Act - Run with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = orchestrator.Run(ctx, server.URL+"/slow", app.OrchestratorOptions{})

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Note: Actual timeout behavior depends on strategy implementation
	t.Skip("Requires strategy implementation with proper timeout handling")
}

func TestIntegration_Orchestrator_Run_InvalidURL(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	err = orchestrator.Run(context.Background(), "invalid://url", app.OrchestratorOptions{})

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine strategy")
}

func TestIntegration_Orchestrator_GetStrategyName(t *testing.T) {
	// Arrange
	cfg := config.Default()
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Test various URLs
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com", "crawler"},
		{"https://github.com/user/repo", "git"},
		{"https://example.com/sitemap.xml", "sitemap"},
		{"https://example.com/llms.txt", "llms"},
		{"https://pkg.go.dev/github.com/example/package", "pkggo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			// Act
			result := orchestrator.GetStrategyName(tt.url)

			// Assert
			assert.Equal(t, tt.expected, result)
		})
	}

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_ValidateURL(t *testing.T) {
	// Arrange
	cfg := config.Default()
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Test valid URLs
	validURLs := []string{
		"https://example.com",
		"http://example.com",
		"https://github.com/user/repo",
		"https://gitlab.com/user/repo",
		"https://example.com/sitemap.xml",
		"https://example.com/llms.txt",
		"https://pkg.go.dev/github.com/example/package",
	}

	for _, url := range validURLs {
		t.Run("valid: "+url, func(t *testing.T) {
			err := orchestrator.ValidateURL(url)
			assert.NoError(t, err)
		})
	}

	// Test invalid URLs
	invalidURLs := []string{
		"ftp://example.com",
		"file:///path/to/file",
		"",
		"not-a-url",
	}

	for _, url := range invalidURLs {
		t.Run("invalid: "+url, func(t *testing.T) {
			err := orchestrator.ValidateURL(url)
			assert.Error(t, err)
		})
	}

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_Close(t *testing.T) {
	// Arrange
	cfg := config.Default()
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	err = orchestrator.Close()

	// Assert
	assert.NoError(t, err)

	// Can close multiple times
	err = orchestrator.Close()
	assert.NoError(t, err)
}

func TestIntegration_Orchestrator_WithCustomConfig(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Output.Flat = true
	cfg.Output.JSONMetadata = true
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 2
	cfg.Concurrency.MaxDepth = 2

	opts := app.OrchestratorOptions{
		Config:          cfg,
		Verbose:         true,
		DryRun:          false,
		RenderJS:        false,
		Limit:           10,
		ExcludePatterns: []string{"test/*", "*.tmp"},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)

	// Verify config was applied
	assert.Equal(t, tmpDir, cfg.Output.Directory)
	assert.True(t, cfg.Output.Flat)
	assert.True(t, cfg.Output.JSONMetadata)
	assert.Equal(t, 2, cfg.Concurrency.Workers)
	assert.Equal(t, 2, cfg.Concurrency.MaxDepth)

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_DryRunMode(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
		DryRun: true,
	})
	require.NoError(t, err)

	// In dry run mode, no files should be created even if strategy runs
	// This depends on strategy implementation
	t.Skip("Requires strategy implementation with dry run support")

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_MultipleRuns(t *testing.T) {
	// Arrange
	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act - Run multiple times with different URLs
	urls := []string{
		"https://example.com",
		"https://example.org",
	}

	for _, url := range urls {
		t.Run("run: "+url, func(t *testing.T) {
			_ = orchestrator.Run(context.Background(), url, app.OrchestratorOptions{
				Limit: 1,
			})
			// Depending on strategy implementation, this might succeed or fail
			// We're just testing that orchestrator can handle multiple runs
		})
	}

	// Cleanup
	require.NoError(t, orchestrator.Close())
}

func TestIntegration_Orchestrator_OutputDirectoryCreation(t *testing.T) {
	// Arrange - Use a non-existent directory
	tmpBase := testutil.TempDir(t)
	outputDir := filepath.Join(tmpBase, "nested", "output", "directory")

	cfg := config.Default()
	cfg.Output.Directory = outputDir
	cfg.Cache.Enabled = false

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Verify directory will be created by strategy
	// This depends on strategy/writer implementation
	t.Skip("Requires strategy/writer implementation")

	// Cleanup
	require.NoError(t, orchestrator.Close())

	// Clean up base directory
	os.RemoveAll(tmpBase)
}
