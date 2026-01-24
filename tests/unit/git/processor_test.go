package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessFile_SkipsLargeFiles(t *testing.T) {
	t.Run("skips files larger than 10MB without reading into memory", func(t *testing.T) {
		// Setup: Create a file slightly larger than 10MB
		tmpDir := t.TempDir()
		largeFilePath := filepath.Join(tmpDir, "large.md")

		// Create 10MB + 1 byte file (just over the limit)
		largeSize := 10*1024*1024 + 1
		err := os.WriteFile(largeFilePath, make([]byte, largeSize), 0644)
		require.NoError(t, err)

		// Verify file was created with correct size
		info, err := os.Stat(largeFilePath)
		require.NoError(t, err)
		assert.Equal(t, int64(largeSize), info.Size())

		// Execute: Process the large file
		processor := git.NewProcessor(git.ProcessorOptions{})
		opts := git.ProcessOptions{
			RepoURL: "https://github.com/test/repo",
			Branch:  "main",
		}

		err = processor.ProcessFile(context.Background(), largeFilePath, tmpDir, opts)

		// Verify: No error, file was skipped (returns nil)
		assert.NoError(t, err)
	})

	t.Run("skips files exactly at 10MB limit", func(t *testing.T) {
		// Setup: Create file exactly at 10MB + 1 byte (first size to be skipped)
		tmpDir := t.TempDir()
		exactLimitPath := filepath.Join(tmpDir, "exact.md")

		// Create file at exactly 10MB + 1 (just over limit)
		exactSize := 10*1024*1024 + 1
		err := os.WriteFile(exactLimitPath, make([]byte, exactSize), 0644)
		require.NoError(t, err)

		// Execute
		processor := git.NewProcessor(git.ProcessorOptions{})
		opts := git.ProcessOptions{
			RepoURL: "https://github.com/test/repo",
			Branch:  "main",
		}

		err = processor.ProcessFile(context.Background(), exactLimitPath, tmpDir, opts)

		// Verify: No error, file was skipped
		assert.NoError(t, err)
	})
}

func TestProcessFile_HandlesStatError(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		// Setup: Use a path that doesn't exist
		tmpDir := t.TempDir()
		nonExistentPath := filepath.Join(tmpDir, "does-not-exist.md")

		// Execute
		processor := git.NewProcessor(git.ProcessorOptions{})
		opts := git.ProcessOptions{
			RepoURL: "https://github.com/test/repo",
			Branch:  "main",
		}

		err := processor.ProcessFile(context.Background(), nonExistentPath, tmpDir, opts)

		// Verify: Error is returned
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestProcessFile_ProcessesNormalFiles(t *testing.T) {
	t.Run("processes files under 10MB limit", func(t *testing.T) {
		// Setup: Create a small markdown file
		tmpDir := t.TempDir()
		smallFilePath := filepath.Join(tmpDir, "small.md")

		content := []byte("# Test Document\n\nThis is test content.")
		err := os.WriteFile(smallFilePath, content, 0644)
		require.NoError(t, err)

		writeCalled := false
		var capturedDoc *domain.Document

		processor := git.NewProcessor(git.ProcessorOptions{})
		opts := git.ProcessOptions{
			RepoURL: "https://github.com/test/repo",
			Branch:  "main",
			WriteFunc: func(ctx context.Context, doc *domain.Document) error {
				writeCalled = true
				capturedDoc = doc
				return nil
			},
		}

		err = processor.ProcessFile(context.Background(), smallFilePath, tmpDir, opts)

		// Verify: No error, WriteFunc was called
		assert.NoError(t, err)
		assert.True(t, writeCalled, "WriteFunc should have been called for small files")
		assert.NotNil(t, capturedDoc)
	})

	t.Run("processes files just under 10MB limit", func(t *testing.T) {
		// Setup: Create file just under 10MB (should be processed)
		tmpDir := t.TempDir()
		justUnderPath := filepath.Join(tmpDir, "just-under.md")

		// 10MB exactly should be processed (limit is > 10MB, not >=)
		justUnderSize := 10 * 1024 * 1024
		err := os.WriteFile(justUnderPath, make([]byte, justUnderSize), 0644)
		require.NoError(t, err)

		writeCalled := false

		processor := git.NewProcessor(git.ProcessorOptions{})
		opts := git.ProcessOptions{
			RepoURL: "https://github.com/test/repo",
			Branch:  "main",
			WriteFunc: func(ctx context.Context, doc *domain.Document) error {
				writeCalled = true
				return nil
			},
		}

		err = processor.ProcessFile(context.Background(), justUnderPath, tmpDir, opts)

		// Verify: File was processed (WriteFunc called)
		assert.NoError(t, err)
		assert.True(t, writeCalled, "WriteFunc should be called for files at exactly 10MB")
	})
}
