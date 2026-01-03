package integration

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
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

func TestDocsRSStrategy_Execute_Success(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	fixturePath := filepath.Join(wd, "../../../tests/fixtures/docsrs/serde_crate_root.html")
	htmlContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.HandleHTML(t, "/serde/1.0.0/serde/", string(htmlContent))

	tempDir := t.TempDir()
	deps := createTestDocsRSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost("127.0.0.1")

	opts := strategies.DefaultOptions()
	opts.Limit = 1
	opts.MaxDepth = 1

	err = strategy.Execute(ctx, server.URL+"/serde/1.0.0/serde/", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.NotEmpty(t, outputFiles, "No output files were created")

	var foundMarkdown bool
	for _, file := range outputFiles {
		if filepath.Ext(file.Name()) == ".md" {
			foundMarkdown = true
			break
		}
	}
	assert.True(t, foundMarkdown, "No markdown file was created")
}

func TestDocsRSStrategy_Execute_DryRun(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	fixturePath := filepath.Join(wd, "../../../tests/fixtures/docsrs/serde_crate_root.html")
	htmlContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.HandleHTML(t, "/serde/1.0.0/serde/", string(htmlContent))

	tempDir := t.TempDir()
	deps := createTestDocsRSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost("127.0.0.1")

	opts := strategies.DefaultOptions()
	opts.DryRun = true
	opts.Limit = 1
	opts.MaxDepth = 1

	err = strategy.Execute(ctx, server.URL+"/serde/1.0.0/serde/", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, outputFiles, "Files were created in dry run mode")
}

func TestDocsRSStrategy_Execute_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	server.Handle(t, "/serde/1.0.0/serde/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost("127.0.0.1")

	opts := strategies.DefaultOptions()
	opts.Limit = 1
	opts.MaxDepth = 1

	err := strategy.Execute(ctx, server.URL+"/serde/1.0.0/serde/", opts)
	require.NoError(t, err)
}

func TestDocsRSStrategy_Execute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	fixturePath := filepath.Join(wd, "../../../tests/fixtures/docsrs/serde_crate_root.html")
	htmlContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.HandleHTML(t, "/serde/1.0.0/serde/", string(htmlContent))

	tempDir := t.TempDir()
	deps := createTestDocsRSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost("127.0.0.1")

	cancel()

	opts := strategies.DefaultOptions()
	opts.Limit = 1
	opts.MaxDepth = 1

	err = strategy.Execute(ctx, server.URL+"/serde/1.0.0/serde/", opts)
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

func TestDocsRSStrategy_Execute_WithModules(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)

	crateRootPath := filepath.Join(wd, "../../../tests/fixtures/docsrs/serde_crate_root.html")
	crateRootHTML, err := os.ReadFile(crateRootPath)
	require.NoError(t, err)

	modulePath := filepath.Join(wd, "../../../tests/fixtures/docsrs/serde_module.html")
	moduleHTML, err := os.ReadFile(modulePath)
	require.NoError(t, err)

	server.HandleHTML(t, "/serde/1.0.0/serde/", string(crateRootHTML))
	server.HandleHTML(t, "/serde/1.0.0/serde/de/index.html", string(moduleHTML))
	server.HandleHTML(t, "/serde/1.0.0/serde/ser/index.html", string(moduleHTML))

	tempDir := t.TempDir()
	deps := createTestDocsRSDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost("127.0.0.1")

	opts := strategies.DefaultOptions()
	opts.Limit = 5
	opts.MaxDepth = 2

	err = strategy.Execute(ctx, server.URL+"/serde/1.0.0/serde/", opts)
	require.NoError(t, err)
}

func createTestDocsRSDependencies(t *testing.T, baseURL string, outputDir string) *strategies.Dependencies {
	t.Helper()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
		Flat:    true,
	})

	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     10 * time.Second,
		MaxRetries:  1,
		EnableCache: false,
	})
	require.NoError(t, err)

	converterPipeline := converter.NewPipeline(converter.PipelineOptions{
		ContentSelector: "#main-content",
	})

	return &strategies.Dependencies{
		Logger:    logger,
		Writer:    writer,
		Fetcher:   fetcherClient,
		Converter: converterPipeline,
	}
}
