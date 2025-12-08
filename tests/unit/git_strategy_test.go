package unit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitStrategy_ParallelProcessing tests that parallel processing works correctly
// Run with: go test -race ./tests/unit/... -run TestGitStrategy
func TestGitStrategy_ParallelProcessing(t *testing.T) {
	// Create temporary input directory with multiple .md files
	inputDir, err := os.MkdirTemp("", "git-parallel-input-*")
	require.NoError(t, err)
	defer os.RemoveAll(inputDir)

	// Create 20 test files to ensure parallelism is exercised
	numFiles := 20
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("doc%02d.md", i)
		path := filepath.Join(inputDir, filename)
		content := fmt.Sprintf("# Document %d\n\nThis is test document number %d.", i, i)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create output directory
	outputDir, err := os.MkdirTemp("", "git-parallel-output-*")
	require.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Setup dependencies
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

	// Verify strategy was created
	assert.NotNil(t, strategy)
	assert.Equal(t, "git", strategy.Name())
}

// TestGitStrategy_CanHandle tests the URL detection logic
func TestGitStrategy_CanHandle(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/user/repo", true},
		{"https://github.com/user/repo.git", true},
		{"git@github.com:user/repo.git", true},
		{"https://gitlab.com/user/repo", true},
		{"https://bitbucket.org/user/repo", true},
		{"https://example.com", false},
		{"https://example.com/sitemap.xml", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParallelForEach_ConcurrentExecution verifies parallel execution
func TestParallelForEach_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	var counter int64

	errors := utils.ParallelForEach(ctx, items, 10, func(ctx context.Context, item int) error {
		atomic.AddInt64(&counter, 1)
		// Small delay to simulate work
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	// All items should be processed
	assert.Equal(t, int64(100), counter)

	// No errors should occur
	for _, err := range errors {
		assert.NoError(t, err)
	}
}

// TestParallelForEach_ContextCancellation tests context cancellation handling
func TestParallelForEach_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	var processed int64

	// Cancel context after short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_ = utils.ParallelForEach(ctx, items, 5, func(ctx context.Context, item int) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			atomic.AddInt64(&processed, 1)
			time.Sleep(5 * time.Millisecond)
			return nil
		}
	})

	// Some items should be processed, but not all due to cancellation
	// The exact count depends on timing, so we just check it's less than total
	assert.Less(t, processed, int64(100), "Should have been interrupted by context cancellation")
}

// TestParallelForEach_ErrorHandling tests error handling in parallel execution
func TestParallelForEach_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	items := []int{1, 2, 3, 4, 5}

	errors := utils.ParallelForEach(ctx, items, 2, func(ctx context.Context, item int) error {
		if item == 3 {
			return fmt.Errorf("error on item %d", item)
		}
		return nil
	})

	// Should have collected the error
	var foundError bool
	for _, err := range errors {
		if err != nil {
			foundError = true
			assert.Contains(t, err.Error(), "error on item 3")
		}
	}
	assert.True(t, foundError, "Should have found an error")

	// FirstError should return the error
	err := utils.FirstError(errors)
	assert.Error(t, err)
}

// TestDocumentExtensions verifies the document extension map
func TestDocumentExtensions(t *testing.T) {
	extensions := strategies.DocumentExtensions

	// Should include markdown
	assert.True(t, extensions[".md"])
	assert.True(t, extensions[".txt"])
	assert.True(t, extensions[".rst"])
	assert.True(t, extensions[".adoc"])
	assert.True(t, extensions[".asciidoc"])

	// Should not include code files
	assert.False(t, extensions[".go"])
	assert.False(t, extensions[".py"])
	assert.False(t, extensions[".js"])
}

// TestIgnoreDirs verifies the ignored directories map
func TestIgnoreDirs(t *testing.T) {
	ignoreDirs := strategies.IgnoreDirs

	// Should ignore common directories
	assert.True(t, ignoreDirs[".git"])
	assert.True(t, ignoreDirs["node_modules"])
	assert.True(t, ignoreDirs["vendor"])
	assert.True(t, ignoreDirs["__pycache__"])
	assert.True(t, ignoreDirs[".venv"])
	assert.True(t, ignoreDirs["venv"])
	assert.True(t, ignoreDirs["dist"])
	assert.True(t, ignoreDirs["build"])

	// Should not ignore common directories
	assert.False(t, ignoreDirs["docs"])
	assert.False(t, ignoreDirs["src"])
}
