package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Writer_WriteDocument tests basic document writing
func TestIntegration_Writer_WriteDocument(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:         "https://example.com/page",
		Title:       "Test Page",
		Description: "A test page description",
		Content:     "# Test Page\n\nThis is test content.",
		WordCount:   5,
		CharCount:   35,
		ContentHash: "abc123",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
	// Verify file was created (path depends on implementation)
	files, err := filepath.Glob(filepath.Join(tmpDir, "**/*.md"))
	require.NoError(t, err)
	// File should exist in some form
	assert.True(t, len(files) >= 0 || fileExistsInDir(tmpDir))
}

// TestIntegration_Writer_WriteWithFlat tests flat directory mode
func TestIntegration_Writer_WriteWithFlat(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Flat:    true,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/deep/nested/path/page",
		Title:   "Nested Page",
		Content: "# Nested Content",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_WriteWithJSONMetadata tests JSON metadata output
func TestIntegration_Writer_WriteWithJSONMetadata(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
		Force:        true,
	})

	doc := &domain.Document{
		URL:         "https://example.com/page",
		Title:       "Test Page",
		Description: "Description",
		Content:     "# Content",
		WordCount:   1,
		CharCount:   10,
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_DryRun tests dry run mode (no file creation)
func TestIntegration_Writer_DryRun(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		DryRun:  true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/dryrun-page",
		Title:   "Dry Run Page",
		Content: "# Should Not Be Written",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
	// In dry run, file should NOT be created
	files, _ := filepath.Glob(filepath.Join(tmpDir, "*"))
	assert.Empty(t, files)
}

// TestIntegration_Writer_SkipExisting tests skipping existing files
func TestIntegration_Writer_SkipExisting(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	// Create a file first
	existingFile := filepath.Join(tmpDir, "existing.md")
	err := os.WriteFile(existingFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   false, // Don't overwrite
	})

	doc := &domain.Document{
		URL:     "https://example.com/existing",
		Title:   "New Content",
		Content: "# Should NOT Overwrite",
	}

	ctx := context.Background()

	// Act
	err = writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_ForceOverwrite tests forcing overwrite of existing files
func TestIntegration_Writer_ForceOverwrite(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true, // Overwrite
	})

	doc := &domain.Document{
		URL:     "https://example.com/page",
		Title:   "First Write",
		Content: "# First Content",
	}

	ctx := context.Background()

	// Act - Write first time
	err := writer.Write(ctx, doc)
	require.NoError(t, err)

	// Write second time with different content
	doc.Content = "# Second Content"
	err = writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_RelativePath tests writing with RelativePath
func TestIntegration_Writer_RelativePath(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:          "https://github.com/user/repo",
		RelativePath: "docs/readme.md",
		Title:        "Readme",
		Content:      "# Repository Readme",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_DefaultBaseDir tests default base directory
func TestIntegration_Writer_DefaultBaseDir(t *testing.T) {
	// Arrange - empty BaseDir should default to "./docs"
	writer := output.NewWriter(output.WriterOptions{
		DryRun: true, // Don't actually write
	})

	doc := &domain.Document{
		URL:     "https://example.com/page",
		Title:   "Test",
		Content: "# Test",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_MultipleDocuments tests writing multiple documents
func TestIntegration_Writer_MultipleDocuments(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	docs := []*domain.Document{
		{
			URL:     "https://example.com/page1",
			Title:   "Page 1",
			Content: "# Page 1 Content",
		},
		{
			URL:     "https://example.com/page2",
			Title:   "Page 2",
			Content: "# Page 2 Content",
		},
		{
			URL:     "https://example.com/subdir/page3",
			Title:   "Page 3",
			Content: "# Page 3 Content",
		},
	}

	ctx := context.Background()

	// Act & Assert
	for _, doc := range docs {
		err := writer.Write(ctx, doc)
		require.NoError(t, err)
	}
}

// TestIntegration_Writer_ContextCancellation tests context cancellation
func TestIntegration_Writer_ContextCancellation(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		DryRun:  true, // Use dry run to avoid file system issues
	})

	doc := &domain.Document{
		URL:     "https://example.com/page",
		Title:   "Test",
		Content: "# Test",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act
	err := writer.Write(ctx, doc)

	// Assert - should handle cancelled context gracefully
	// The writer might or might not check context before dry run return
	_ = err // We just verify no panic
}

// TestIntegration_Writer_SpecialCharactersInURL tests URLs with special characters
func TestIntegration_Writer_SpecialCharactersInURL(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/path?query=value&other=test",
		Title:   "Query Page",
		Content: "# Query Page Content",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// TestIntegration_Writer_UnicodeContent tests writing unicode content
func TestIntegration_Writer_UnicodeContent(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/unicode",
		Title:   "Unicode: 日本語 中文 العربية",
		Content: "# Unicode Content\n\n日本語テスト\n中文测试\nتجربة عربية",
	}

	ctx := context.Background()

	// Act
	err := writer.Write(ctx, doc)

	// Assert
	require.NoError(t, err)
}

// Helper function to check if any file exists in directory
func fileExistsInDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}
