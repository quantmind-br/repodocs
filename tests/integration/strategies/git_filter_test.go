package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitStrategy_Execute_PathFiltering tests path filtering functionality
// This test verifies that the Git strategy can:
// 1. Extract from a repository
// 2. Handle non-existent paths gracefully
func TestGitStrategy_Execute_PathFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	outputDir := t.TempDir()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "warn"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	// Use a small, stable public repository
	repoURL := "https://github.com/golang/example"

	t.Run("BasicExtraction", func(t *testing.T) {
		opts := strategies.DefaultOptions()
		opts.Output = filepath.Join(outputDir, "basic-run")

		// Basic extraction - should work even if no .md files are found
		err := strategy.Execute(context.Background(), repoURL, opts)
		require.NoError(t, err)

		// The test completes without error
		// The output directory may or may not be created depending on whether
		// .md files were found in the repository
	})

	t.Run("NonExistentPath", func(t *testing.T) {
		opts := strategies.DefaultOptions()
		opts.Output = filepath.Join(outputDir, "error-run")

		// Using a non-existent filter path to test error handling
		opts.FilterURL = "this-directory-definitely-does-not-exist-12345"

		err := strategy.Execute(context.Background(), repoURL, opts)
		// Should fail with clear error message
		require.Error(t, err)
		assert.Contains(t, err.Error(), "filter path")
	})

	t.Run("DryRunMode", func(t *testing.T) {
		opts := strategies.DefaultOptions()
		opts.Output = filepath.Join(outputDir, "dryrun-run")
		opts.DryRun = true

		// Dry run should not create any files
		err := strategy.Execute(context.Background(), repoURL, opts)
		require.NoError(t, err)

		// Verify no files were created
		_, err = os.Stat(opts.Output)
		assert.True(t, os.IsNotExist(err), "Dry run should not create output directory")
	})
}
