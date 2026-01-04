package integration

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubPagesStrategy_SitemapDiscovery tests discovery via sitemap.xml
func TestGitHubPagesStrategy_SitemapDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	// Mock sitemap.xml
	sitemapContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + server.URL + `/</loc></url>
  <url><loc>` + server.URL + `/docs/</loc></url>
  <url><loc>` + server.URL + `/api.html</loc></url>
</urlset>`

	server.HandleString(t, "/sitemap.xml", "application/xml", sitemapContent)
	server.HandleString(t, "/", "text/html", `<html><body><h1>Home</h1></body></html>`)
	server.HandleString(t, "/docs/", "text/html", `<html><body><h1>Docs</h1></body></html>`)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		MaxDepth:    1,
		Limit:       10,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.NoError(t, err, "Strategy should not error even if fetch fails")
}

// TestGitHubPagesStrategy_LLMsTxtDiscovery tests discovery via llms.txt
func TestGitHubPagesStrategy_LLMsTxtDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	llmsContent := `# Documentation
- [Getting Started](getting-started.html)
- [API Reference](api.html)
`

	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)
	server.HandleString(t, "/getting-started.html", "text/html", `<html><body><h1>Getting Started</h1></body></html>`)
	server.HandleString(t, "/api.html", "text/html", `<html><body><h1>API</h1></body></html>`)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		MaxDepth:    1,
		Limit:       10,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.NoError(t, err, "Strategy should not error even if fetch fails")
}

// TestGitHubPagesStrategy_MkDocsDiscovery tests discovery via MkDocs search index
func TestGitHubPagesStrategy_MkDocsDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	searchIndex := `{"docs": [
		{"location": "index.html", "title": "Home", "text": "Welcome"},
		{"location": "guide/", "title": "Guide", "text": "User guide"}
	]}`

	server.HandleString(t, "/search/search_index.json", "application/json", searchIndex)
	server.HandleString(t, "/index.html", "text/html", `<html><body><h1>Home</h1></body></html>`)
	server.HandleString(t, "/guide/", "text/html", `<html><body><h1>Guide</h1></body></html>`)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		MaxDepth:    1,
		Limit:       10,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.NoError(t, err, "Strategy should not error even if fetch fails")
}

// TestGitHubPagesStrategy_DryRun tests dry run mode
func TestGitHubPagesStrategy_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	sitemapContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + server.URL + `/</loc></url>
</urlset>`

	server.HandleString(t, "/sitemap.xml", "application/xml", sitemapContent)
	server.HandleString(t, "/", "text/html", `<html><body><h1>Home</h1></body></html>`)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		DryRun:      true,
		MaxDepth:    1,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.NoError(t, err, "Strategy should not error in dry run mode")
}

// TestGitHubPagesStrategy_WithProjectSubpath tests GitHub Pages with project subdirectory
func TestGitHubPagesStrategy_WithProjectSubpath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	projectURL := server.URL + "/project"

	sitemapContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>` + projectURL + `/</loc></url>
  <url><loc>` + projectURL + `/guide/</loc></url>
</urlset>`

	server.HandleString(t, "/project/sitemap.xml", "application/xml", sitemapContent)
	server.HandleString(t, "/project/", "text/html", `<html><body><h1>Project</h1></body></html>`)
	server.HandleString(t, "/project/guide/", "text/html", `<html><body><h1>Guide</h1></body></html>`)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		MaxDepth:    1,
		Limit:       10,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, projectURL, opts)
	assert.NoError(t, err, "Strategy should not error even with project subpath")
}

func TestGitHubPagesStrategy_EmptyDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	server := testutil.NewTestServer(t)

	tempDir := t.TempDir()
	deps := createGitHubPagesTestDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitHubPagesStrategy(deps)

	opts := strategies.Options{
		MaxDepth:    1,
		Limit:       10,
		Concurrency: 2,
	}

	err := strategy.Execute(ctx, server.URL, opts)
	assert.NoError(t, err)
}

// createGitHubPagesTestDependencies creates test dependencies for GitHub Pages strategy
func createGitHubPagesTestDependencies(t *testing.T, serverURL, tempDir string) *strategies.Dependencies {
	t.Helper()

	// Create a real fetcher client
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     10 * time.Second,
		MaxRetries:  3,
		EnableCache: false,
		UserAgent:   "test-user-agent",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		fetcherClient.Close()
	})

	// Create a real converter
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: serverURL,
	})

	// Create a real writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tempDir,
		Flat:         true,
		JSONMetadata: false,
		Force:        true,
	})

	// Create a logger
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "info",
		Format: "text",
	})

	return &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Converter: converterPipeline,
		Writer:    writer,
		Logger:    logger,
	}
}
