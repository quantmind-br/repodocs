package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/manifest"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
)

func TestManifest_Integration_MultipleWebSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server1 := testutil.NewTestServer(t)
	server1.HandleHTML(t, "/", `<!DOCTYPE html>
<html>
<head><title>Site 1 - Documentation</title></head>
<body>
<article class="main">
    <h1>Welcome to Site 1</h1>
    <p>This is the main content of site 1.</p>
</article>
</body>
</html>`)

	server2 := testutil.NewTestServer(t)
	server2.HandleHTML(t, "/", `<!DOCTYPE html>
<html>
<head><title>Site 2 - API Reference</title></head>
<body>
<main>
    <h1>API Reference</h1>
    <p>Documentation for the API.</p>
</main>
</body>
</html>`)

	outputDir := testutil.TempDir(t)

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: server1.URL, Strategy: "crawler", ContentSelector: "article.main"},
			{URL: server2.URL, Strategy: "crawler", ContentSelector: "main"},
		},
		Options: manifest.Options{
			ContinueOnError: false,
			Output:          outputDir,
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.RunManifest(context.Background(), manifestCfg, app.OrchestratorOptions{Config: cfg})

	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(outputDir, "*.md"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "Expected at least 1 markdown file")
}

func TestManifest_Integration_SingleSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := testutil.NewTestServer(t)
	server.HandleHTML(t, "/", `<html><head><title>Working</title></head><body><p>Content</p></body></html>`)

	outputDir := testutil.TempDir(t)

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: server.URL},
		},
		Options: manifest.Options{
			ContinueOnError: false,
			Output:          outputDir,
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.RunManifest(context.Background(), manifestCfg, app.OrchestratorOptions{Config: cfg})

	require.NoError(t, err)
}

func TestManifest_Integration_ContinueOnError_True(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	workingServer := testutil.NewTestServer(t)
	workingServer.HandleHTML(t, "/", `<html><head><title>Working</title></head><body><p>Content</p></body></html>`)

	outputDir := testutil.TempDir(t)

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: workingServer.URL},
			{URL: workingServer.URL + "/?page=2"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          outputDir,
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.RunManifest(context.Background(), manifestCfg, app.OrchestratorOptions{Config: cfg})

	require.NoError(t, err)
}

func TestManifest_Integration_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := testutil.NewTestServer(t)
	server.HandleHTML(t, "/", `<html><head><title>Test</title></head><body><p>Content</p></body></html>`)

	outputDir := testutil.TempDir(t)

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: server.URL},
			{URL: server.URL + "/?page=2"},
		},
		Options: manifest.Options{
			Output: outputDir,
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = orchestrator.RunManifest(ctx, manifestCfg, app.OrchestratorOptions{Config: cfg})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestManifest_Integration_LoadAndRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server := testutil.NewTestServer(t)
	server.HandleHTML(t, "/", `<html><head><title>Test</title></head><body><p>Hello World</p></body></html>`)

	outputDir := testutil.TempDir(t)
	manifestDir := testutil.TempDir(t)

	manifestContent := `sources:
  - url: ` + server.URL + `
    strategy: crawler
options:
  output: ` + outputDir + `
`
	manifestPath := filepath.Join(manifestDir, "test-manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifestContent), 0644))

	loader := manifest.NewLoader()
	manifestCfg, err := loader.Load(manifestPath)
	require.NoError(t, err)

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.RunManifest(context.Background(), manifestCfg, app.OrchestratorOptions{Config: cfg})

	require.NoError(t, err)
}

func TestManifest_Integration_EmptySources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	outputDir := testutil.TempDir(t)

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{},
		Options: manifest.Options{
			Output: outputDir,
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = outputDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.RunManifest(context.Background(), manifestCfg, app.OrchestratorOptions{Config: cfg})

	require.NoError(t, err)
}
