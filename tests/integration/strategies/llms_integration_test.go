package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLLMSStrategy_Execute_DryRun tests that dry run mode doesn't create files
func TestLLMSStrategy_Execute_DryRun(t *testing.T) {
	// This test verifies the dry run functionality works correctly
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	// Simple llms.txt with one link
	llmsContent := `# Documentation
[Getting Started](https://example.com/getting-started)`

	// Setup mock server
	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)

	// Create temporary directory for output
	tempDir := t.TempDir()

	// Create strategy with mocked dependencies
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	// Act - execute with dry run enabled
	opts := strategies.Options{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	}
	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)

	// Assert
	require.NoError(t, err)

	// Verify that no files were created in dry run mode
	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, outputFiles, "No files should be created in dry run mode")
}

// TestLLMSStrategy_Execute_EmptyLLMS tests handling of empty llms.txt
func TestLLMSStrategy_Execute_EmptyLLMS(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	// Empty llms.txt
	llmsContent := ""

	// Setup mock server
	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)

	// Create temporary directory for output
	tempDir := t.TempDir()

	// Create strategy with mocked dependencies
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	// Act
	err := strategy.Execute(ctx, server.URL+"/llms.txt", strategies.DefaultOptions())

	// Assert
	require.NoError(t, err)

	// Verify that no files were created (no links to process)
	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, outputFiles, "No files should be created when llms.txt has no links")
}

// TestLLMSStrategy_Execute_FetchError tests error handling when llms.txt can't be fetched
func TestLLMSStrategy_Execute_FetchError(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	// Setup server to return 404 for llms.txt
	server.Handle404(t, "/llms.txt")

	// Create temporary directory for output
	tempDir := t.TempDir()

	// Create strategy with mocked dependencies
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	// Act
	err := strategy.Execute(ctx, server.URL+"/llms.txt", strategies.DefaultOptions())

	// Assert
	require.Error(t, err)
	// The error should be related to the fetch failure
	assert.Contains(t, err.Error(), "404")
}

// TestLLMSStrategy_Execute_WithFixture tests the complete flow with the sample fixture
// Note: This test is skipped by default because tls-client doesn't work well with httptest.
// For full integration testing, use end-to-end tests with real URLs.
func TestLLMSStrategy_Execute_WithFixture(t *testing.T) {
	t.Skip("Skipping: tls-client library doesn't support httptest.Server. Use e2e tests for full integration testing.")

	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	// This test would test the full flow if we could use real HTTP clients
	// Since tls-client doesn't work with httptest, we skip this test
	// For production testing, use: go test -run TestLLMSStrategy_Execute_WithFixture ./tests/integration/... -v
}

func TestLLMSStrategy_Execute_PlainTextFile(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	llmsContent := "[Full Docs](" + server.URL + "/llms-full.txt)"

	plainTextContent := `# Full Documentation

This is the full documentation content in plain text format.
It should be processed by PlainTextReader, not the HTML converter.

[Link to API](https://example.com/api)
[Link to Guide](https://example.com/guide)`

	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)
	server.HandleString(t, "/llms-full.txt", "text/plain", plainTextContent)

	tempDir := t.TempDir()
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	opts := strategies.DefaultOptions()
	opts.Output = tempDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, outputFiles, "Expected output files to be created")
}

func TestLLMSStrategy_Execute_MixedContentTypes(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	llmsContent := "[Plain Text](" + server.URL + "/docs.txt)\n" +
		"[Markdown](" + server.URL + "/readme.md)\n" +
		"[HTML Page](" + server.URL + "/page.html)"

	plainTextContent := "# Plain Text Document\n\nThis is a plain text file."
	markdownContent := "# Markdown Document\n\nThis is a **markdown** file."
	htmlContent := `<!DOCTYPE html><html><head><title>HTML</title></head><body><h1>HTML Document</h1></body></html>`

	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)
	server.HandleString(t, "/docs.txt", "text/plain", plainTextContent)
	server.HandleString(t, "/readme.md", "text/markdown", markdownContent)
	server.HandleString(t, "/page.html", "text/html", htmlContent)

	tempDir := t.TempDir()
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	opts := strategies.DefaultOptions()
	opts.Output = tempDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, outputFiles, "Expected output files to be created")
}

func TestLLMSStrategy_Execute_PlainTextContentTypeDetection(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	llmsContent := "[Content](" + server.URL + "/content)"

	plainTextContent := `Plain Text Content

This is plain text served with text/plain content type
but without a .txt extension in the URL.`

	server.HandleString(t, "/llms.txt", "text/plain", llmsContent)
	server.HandleString(t, "/content", "text/plain; charset=utf-8", plainTextContent)

	tempDir := t.TempDir()
	deps := createTestLLMSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewLLMSStrategy(deps)

	opts := strategies.DefaultOptions()
	opts.Output = tempDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, outputFiles, "Expected output files to be created for plain text content")
}

func createTestLLMSDependencies(t *testing.T, baseURL, outputDir string) *strategies.Dependencies {
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
		BaseURL: baseURL,
	})

	// Create a real writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      outputDir,
		Flat:         false,
		JSONMetadata: false,
		Force:        true,
		DryRun:       false,
	})

	// Create a logger
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   "error",
		Format:  "pretty",
		Verbose: false,
	})

	return &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Renderer:  nil,
		Cache:     nil,
		Converter: converterPipeline,
		Writer:    writer,
		Logger:    logger,
	}
}
