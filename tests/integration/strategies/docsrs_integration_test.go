package integration

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func getServerHost(serverURL string) string {
	u, _ := url.Parse(serverURL)
	return u.Host
}

func TestDocsRSStrategy_Execute_WithJSON(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	fixturePath := filepath.Join(wd, "../../../tests/testdata/docsrs/minimal_crate.json")
	jsonContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.Handle(t, "/crate/example/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonContent)
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	opts := strategies.DefaultOptions()
	opts.Limit = 10

	err = strategy.Execute(ctx, server.URL+"/crate/example/1.0.0", opts)
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
	fixturePath := filepath.Join(wd, "../../../tests/testdata/docsrs/minimal_crate.json")
	jsonContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.Handle(t, "/crate/example/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonContent)
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	opts := strategies.DefaultOptions()
	opts.DryRun = true
	opts.Limit = 10

	err = strategy.Execute(ctx, server.URL+"/crate/example/1.0.0", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, outputFiles, "Files were created in dry run mode")
}

func TestDocsRSStrategy_Execute_JSONFetchError(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	server.Handle(t, "/crate/example/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	opts := strategies.DefaultOptions()
	opts.Limit = 10

	err := strategy.Execute(ctx, server.URL+"/crate/example/1.0.0", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch rustdoc JSON")
}

func TestDocsRSStrategy_Execute_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	server.Handle(t, "/crate/example/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	opts := strategies.DefaultOptions()
	opts.Limit = 10

	err := strategy.Execute(ctx, server.URL+"/crate/example/1.0.0", opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch rustdoc JSON")
}

func TestDocsRSStrategy_Execute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := testutil.NewTestServer(t)

	wd, err := os.Getwd()
	require.NoError(t, err)
	fixturePath := filepath.Join(wd, "../../../tests/testdata/docsrs/minimal_crate.json")
	jsonContent, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	server.Handle(t, "/crate/example/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonContent)
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	cancel()

	opts := strategies.DefaultOptions()
	opts.Limit = 10

	err = strategy.Execute(ctx, server.URL+"/crate/example/1.0.0", opts)
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}
}

func TestDocsRSStrategy_Execute_EmptyIndex(t *testing.T) {
	ctx := context.Background()
	server := testutil.NewTestServer(t)

	emptyJSON := `{
		"root": "0",
		"crate_version": "1.0.0",
		"format_version": 57,
		"includes_private": false,
		"index": {},
		"paths": {},
		"external_crates": {}
	}`

	server.Handle(t, "/crate/empty/1.0.0/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(emptyJSON))
	})

	tempDir := t.TempDir()
	deps := createTestDocsRSJSONDependencies(t, server.URL, tempDir)
	strategy := strategies.NewDocsRSStrategy(deps)
	strategy.SetBaseHost(getServerHost(server.URL))

	opts := strategies.DefaultOptions()
	opts.Limit = 10

	err := strategy.Execute(ctx, server.URL+"/crate/empty/1.0.0", opts)
	require.NoError(t, err)

	outputFiles, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, outputFiles, "No files should be created for empty index")
}

func createTestDocsRSJSONDependencies(t *testing.T, baseURL string, outputDir string) *strategies.Dependencies {
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

	return &strategies.Dependencies{
		Logger:  logger,
		Writer:  writer,
		Fetcher: fetcherClient,
	}
}
