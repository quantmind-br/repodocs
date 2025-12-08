package integration

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitStrategy_Execute_FetchError tests error handling when repository can't be fetched
func TestGitStrategy_Execute_FetchError(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	// Setup server to return 404 for repository
	server.Handle404(t, "/nonexistent/repo")

	// Create temporary directory for output
	tempDir := t.TempDir()

	// Create strategy with mocked dependencies
	deps := createTestGitDependencies(t, server.URL, tempDir)
	strategy := strategies.NewGitStrategy(deps)

	// Act
	err := strategy.Execute(ctx, server.URL+"/nonexistent/repo", strategies.DefaultOptions())

	// Assert
	require.Error(t, err)
	// The error should be related to the fetch failure
	assert.Contains(t, err.Error(), "404")
}

// Helper function to create test dependencies for Git strategy integration tests
func createTestGitDependencies(t *testing.T, baseURL, outputDir string) *strategies.Dependencies {
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
		Converter: nil,
		Writer:    writer,
		Logger:    logger,
	}
}
