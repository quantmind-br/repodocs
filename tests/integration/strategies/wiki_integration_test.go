package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiStrategy_CanHandle(t *testing.T) {
	deps := createTestWikiDependencies(t, t.TempDir())
	strategy := strategies.NewWikiStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"wiki URL", "https://github.com/owner/repo/wiki", true},
		{"wiki URL with page", "https://github.com/owner/repo/wiki/Config", true},
		{"wiki clone URL", "https://github.com/owner/repo.wiki.git", true},
		{"regular repo", "https://github.com/owner/repo", false},
		{"other URL", "https://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWikiStrategy_Execute_InvalidURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	deps := createTestWikiDependencies(t, tempDir)
	strategy := strategies.NewWikiStrategy(deps)

	err := strategy.Execute(ctx, "https://not-a-valid-wiki-url.com", strategies.DefaultOptions())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid wiki URL")
}

func TestWikiStrategy_Execute_NonExistentWiki(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tempDir := t.TempDir()
	deps := createTestWikiDependencies(t, tempDir)
	strategy := strategies.NewWikiStrategy(deps)

	err := strategy.Execute(ctx, "https://github.com/nonexistent-owner-xyz/nonexistent-repo-xyz/wiki", strategies.Options{
		Output: tempDir,
		DryRun: false,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "clone wiki")
}

func TestWikiStrategy_Execute_RealWiki(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("WIKI_INTEGRATION_TEST") == "" {
		t.Skip("Skipping real wiki test. Set WIKI_INTEGRATION_TEST=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tempDir := t.TempDir()
	deps := createTestWikiDependencies(t, tempDir)
	strategy := strategies.NewWikiStrategy(deps)

	err := strategy.Execute(ctx, "https://github.com/Alexays/Waybar/wiki", strategies.Options{
		Output: tempDir,
		Limit:  5,
		DryRun: false,
	})

	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(tempDir, "*.md"))
	require.NoError(t, err)
	assert.Greater(t, len(files), 0, "Expected at least one markdown file")

	indexPath := filepath.Join(tempDir, "index.md")
	_, err = os.Stat(indexPath)
	assert.NoError(t, err, "Expected index.md (Home.md) to exist")
}

func TestWikiStrategy_Execute_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if os.Getenv("WIKI_INTEGRATION_TEST") == "" {
		t.Skip("Skipping real wiki test. Set WIKI_INTEGRATION_TEST=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tempDir := t.TempDir()
	deps := createTestWikiDependencies(t, tempDir)
	strategy := strategies.NewWikiStrategy(deps)

	err := strategy.Execute(ctx, "https://github.com/Alexays/Waybar/wiki", strategies.Options{
		Output: tempDir,
		Limit:  3,
		DryRun: true,
	})

	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(tempDir, "*.md"))
	require.NoError(t, err)
	assert.Equal(t, 0, len(files), "Dry run should not create files")
}

func TestWikiStrategy_Execute_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tempDir := t.TempDir()
	deps := createTestWikiDependencies(t, tempDir)
	strategy := strategies.NewWikiStrategy(deps)

	err := strategy.Execute(ctx, "https://github.com/Alexays/Waybar/wiki", strategies.Options{
		Output: tempDir,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func createTestWikiDependencies(t *testing.T, outputDir string) *strategies.Dependencies {
	t.Helper()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      outputDir,
		Flat:         true,
		JSONMetadata: false,
		Force:        true,
		DryRun:       false,
	})

	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   "error",
		Format:  "pretty",
		Verbose: false,
	})

	return &strategies.Dependencies{
		Fetcher:   nil,
		Renderer:  nil,
		Cache:     nil,
		Converter: nil,
		Writer:    writer,
		Logger:    logger,
	}
}
