package integration

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

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

// TestGitStrategy_RealPublicRepo tests processing actual public repository
func TestGitStrategy_RealPublicRepo(t *testing.T) {
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

	tests := []struct {
		name    string
		repoURL string
		wantErr bool
	}{
		{
			name:    "GitHub public repo",
			repoURL: "https://github.com/golang/example",
			wantErr: false,
		},
		{
			name:    "Small repo with markdown",
			repoURL: "https://github.com/owner/repo", // This will fail, testing error handling
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			opts := strategies.DefaultOptions()
			opts.Output = filepath.Join(outputDir, tt.name)

			err := strategy.Execute(ctx, tt.repoURL, opts)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGitStrategy_WithSubdirectory tests processing specific subdirectory
func TestGitStrategy_WithSubdirectory(t *testing.T) {
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

	repoURL := "https://github.com/golang/example"
	opts := strategies.DefaultOptions()
	opts.Output = filepath.Join(outputDir, "subdir-test")
	opts.FilterURL = "hello" // Test specific subdirectory that doesn't have docs

	err := strategy.Execute(context.Background(), repoURL, opts)
	// When a filter path is specified and no files are found, it returns an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no documentation files found under path: hello")
}

// TestGitStrategy_InvalidRepo tests handling invalid repositories
func TestGitStrategy_InvalidRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	outputDir := t.TempDir()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	tests := []struct {
		name    string
		repoURL string
	}{
		{
			name:    "Invalid URL",
			repoURL: "not-a-valid-url",
		},
		{
			name:    "Non-existent repo",
			repoURL: "https://github.com/this-repo-definitely-does-not-exist-12345/repo",
		},
		{
			name:    "Private repo without auth",
			repoURL: "https://github.com/private-org/private-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			opts := strategies.DefaultOptions()
			opts.Output = filepath.Join(outputDir, tt.name)

			err := strategy.Execute(ctx, tt.repoURL, opts)
			assert.Error(t, err, "Should return error for invalid repository")
		})
	}
}

// TestGitStrategy_Concurrency tests concurrent processing
func TestGitStrategy_Concurrency(t *testing.T) {
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

	repoURL := "https://github.com/golang/example"

	// Run multiple extractions concurrently
	done := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(index int) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			opts := strategies.DefaultOptions()
			opts.Output = filepath.Join(outputDir, "concurrent", strconv.Itoa(index))

			err := strategy.Execute(ctx, repoURL, opts)
			done <- err
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		err := <-done
		assert.NoError(t, err, "Concurrent extraction should succeed")
	}
}
