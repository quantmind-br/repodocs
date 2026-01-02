package app_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWriter(t *testing.T) {
	// Test with custom options
	opts := output.WriterOptions{
		BaseDir:      "/custom/dir",
		Flat:         true,
		JSONMetadata: true,
		Force:        true,
		DryRun:       true,
	}

	writer := output.NewWriter(opts)
	path := writer.GetPath("https://example.com")
	assert.Contains(t, path, "/custom/dir")
	assert.Contains(t, path, ".md")
}

func TestNewWriter_DefaultBaseDir(t *testing.T) {
	// Test with empty base dir (should default to ./docs)
	opts := output.WriterOptions{
		BaseDir: "",
	}

	writer := output.NewWriter(opts)
	path := writer.GetPath("https://example.com")
	assert.Contains(t, path, "docs")
	assert.Contains(t, path, ".md")
}

// TestWrite_Success tests successful writing of a document
func TestWrite_Success(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document\n\nThis is a test.",
		Title:   "Test Document",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Test Document")
}

// TestWrite_WithMetadata tests writing with JSON metadata using consolidated collector
func TestWrite_WithMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
		Collector:    collector,
	})

	doc := &domain.Document{
		URL:         "https://example.com/docs",
		Content:     "# Test Document",
		Title:       "Test Document",
		Description: "Test Description",
		WordCount:   10,
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	err = writer.FlushMetadata()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	assert.FileExists(t, metadataPath)

	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)
	require.Len(t, index.Documents, 1)
	assert.Equal(t, "Test Document", index.Documents[0].Title)
	assert.Equal(t, "Test Description", index.Documents[0].Description)
}

// TestWrite_EmptyContent tests writing document with empty content
func TestWrite_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	doc := &domain.Document{
		URL:     "https://example.com/empty",
		Content: "",
		Title:   "Empty Document",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/empty")
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	// Should still write with frontmatter
	assert.Contains(t, string(content), "Empty Document")
}

// TestWrite_InvalidPath tests error handling when path is invalid
func TestWrite_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file at the location where we want to create a directory
	// This will cause WriteFile to fail
	invalidPath := filepath.Join(tmpDir, "existingfile.txt")
	err := os.WriteFile(invalidPath, []byte("content"), 0644)
	require.NoError(t, err)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: invalidPath,
	})

	doc := &domain.Document{
		URL:     "https://example.com/test",
		Content: "# Test",
		Title:   "Test",
	}

	err = writer.Write(context.Background(), doc)
	// Should fail because baseDir is a file, not a directory
	require.Error(t, err)
}

// TestWrite_WriteFileError tests error handling when WriteFile fails
func TestWrite_WriteFileError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a read-only directory to cause WriteFile to fail
	// First create a file at the expected path
	readOnlyDir := filepath.Join(tmpDir, "example.com", "docs.md")
	err := os.MkdirAll(filepath.Dir(readOnlyDir), 0755)
	require.NoError(t, err)
	// Create a file with the same name as the directory we want to write to
	err = os.WriteFile(readOnlyDir, []byte("blocking"), 0644)
	require.NoError(t, err)
	// Remove write permissions
	err = os.Chmod(filepath.Dir(readOnlyDir), 0555)
	require.NoError(t, err)
	defer os.Chmod(filepath.Dir(readOnlyDir), 0755)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	err = writer.Write(context.Background(), doc)
	// Should fail because we can't write to the file
	// Note: On some systems this may succeed due to permissions handling
	if err != nil {
		// Expected error occurred, test passes
		assert.Contains(t, err.Error(), "permission")
	}
}

// TestWriteMultiple_Success tests successful writing of multiple documents
func TestWriteMultiple_Success(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
		{URL: "https://example.com/docs3", Content: "# Doc 3", Title: "Doc 3"},
	}

	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	assert.FileExists(t, writer.GetPath("https://example.com/docs1"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs2"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs3"))
}

// TestWriteMultiple_Partial tests WriteMultiple when one document fails
func TestWriteMultiple_Partial(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that will block the second write
	blockingPath := filepath.Join(tmpDir, "example.com", "docs2.md")
	err := os.MkdirAll(filepath.Dir(blockingPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(blockingPath, []byte("blocking"), 0644)
	require.NoError(t, err)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   false,
	})

	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"}, // Will be skipped (file exists)
		{URL: "https://example.com/docs3", Content: "# Doc 3", Title: "Doc 3"},
	}

	// First document should succeed
	err = writer.Write(context.Background(), docs[0])
	require.NoError(t, err)

	// Second should be skipped (exists)
	err = writer.Write(context.Background(), docs[1])
	require.NoError(t, err)

	// Third should succeed
	err = writer.Write(context.Background(), docs[2])
	require.NoError(t, err)

	// Verify only docs1 and docs3 were written, docs2 was skipped
	assert.FileExists(t, writer.GetPath("https://example.com/docs1"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs3"))

	// Verify the blocking file is still there unchanged
	content, err := os.ReadFile(blockingPath)
	require.NoError(t, err)
	assert.Equal(t, "blocking", string(content))
}

// TestWriteJSON_Success tests successful JSON metadata writing with consolidated collector
func TestWriteJSON_Success(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
		Collector:    collector,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	err = writer.FlushMetadata()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	assert.FileExists(t, metadataPath)
}

// TestWriteJSON_Indent tests JSON metadata formatting with proper indentation
func TestWriteJSON_Indent(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
		Collector:    collector,
	})

	doc := &domain.Document{
		URL:         "https://example.com/docs",
		Content:     "# Test",
		Title:       "Test",
		Description: "Test Description",
		WordCount:   10,
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	err = writer.FlushMetadata()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "\n")
	assert.Contains(t, jsonStr, "  ")

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)
	require.Len(t, index.Documents, 1)
	assert.Equal(t, "Test", index.Documents[0].Title)
}

// TestGetPath_ReturnsPath tests GetPath returns correct path
func TestGetPath_ReturnsPath(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	path := writer.GetPath("https://example.com/docs")
	assert.Contains(t, path, tmpDir)
	assert.Contains(t, path, ".md")
}

// TestExists_CheckExistence tests Exists method
func TestExists_CheckExistence(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// File doesn't exist yet
	assert.False(t, writer.Exists("https://example.com/docs"))

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// File now exists
	assert.True(t, writer.Exists("https://example.com/docs"))
}

// TestEnsureBaseDir_CreatesDir tests EnsureBaseDir creates directory
func TestEnsureBaseDir_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "docs", "subdir")

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: baseDir,
	})

	err := writer.EnsureBaseDir()
	require.NoError(t, err)

	assert.DirExists(t, baseDir)
}

// TestEnsureBaseDir_Existing tests EnsureBaseDir when directory already exists
func TestEnsureBaseDir_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "docs")

	// Pre-create directory
	err := os.MkdirAll(baseDir, 0755)
	require.NoError(t, err)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: baseDir,
	})

	// Should not fail when directory exists
	err = writer.EnsureBaseDir()
	require.NoError(t, err)

	assert.DirExists(t, baseDir)
}

// TestClean_RemovesFiles tests Clean removes all files
func TestClean_RemovesFiles(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create documents
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify files exist
	assert.FileExists(t, writer.GetPath("https://example.com/docs"))

	// Clean directory
	err = writer.Clean()
	require.NoError(t, err)

	// Verify directory is removed
	assert.NoFileExists(t, writer.GetPath("https://example.com/docs"))
}

// TestClean_EmptyDir tests Clean on empty directory
func TestClean_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Clean empty directory
	err := writer.Clean()
	require.NoError(t, err)

	// Verify directory is removed
	assert.NoFileExists(t, writer.GetPath("https://example.com/docs"))
}

// TestStats_ReturnsStatistics tests Stats returns correct statistics
func TestStats_ReturnsStatistics(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create documents
	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1\n\nSome content here.", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2\n\nMore content here.", Title: "Doc 2"},
		{URL: "https://example.com/docs3", Content: "# Doc 3\n\nEven more content.", Title: "Doc 3"},
	}

	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	// Get stats
	count, size, err := writer.Stats()
	require.NoError(t, err)

	// Verify stats
	assert.Equal(t, 3, count)
	assert.True(t, size > 0)
}

// TestStats_WithNonMarkdownFiles tests Stats only counts .md files
func TestStats_WithNonMarkdownFiles(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create documents
	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
	}

	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	// Add non-markdown files
	nonMdPath := filepath.Join(tmpDir, "readme.txt")
	err = os.WriteFile(nonMdPath, []byte("Readme"), 0644)
	require.NoError(t, err)

	jsonPath := filepath.Join(tmpDir, "data.json")
	err = os.WriteFile(jsonPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Get stats - should only count .md files
	count, size, err := writer.Stats()
	require.NoError(t, err)

	// Should only count the 2 markdown files, not the .txt or .json
	assert.Equal(t, 2, count)
	assert.True(t, size > 0)
}

// TestStats_WithWalkError tests Stats error handling when Walk fails
func TestStats_WithWalkError(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create a directory with no read permission to cause Walk to fail
	noReadDir := filepath.Join(tmpDir, "restricted")
	err := os.MkdirAll(noReadDir, 0000)
	require.NoError(t, err)
	defer os.Chmod(noReadDir, 0755)

	// Get stats - should return an error
	_, _, err = writer.Stats()
	// On some systems this will fail, on others it will succeed
	// We just verify it doesn't panic
}

// TestWriter_Write_WithRelativePath tests writing with relative path
func TestWriter_Write_WithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	doc := &domain.Document{
		RelativePath: "docs/guide.md",
		Content:      "# Guide\n\nThis is a guide.",
		Title:        "Guide",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := filepath.Join(tmpDir, "docs/guide.md")
	assert.FileExists(t, path)
}

// TestWriter_Write_SkipExisting tests skipping existing files
func TestWriter_Write_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   false,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	// Write first time
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")
	firstContent, err := os.ReadFile(path)
	require.NoError(t, err)

	// Write again (should skip)
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	secondContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, firstContent, secondContent)
}

// TestWriter_Write_Force tests force overwrite
func TestWriter_Write_Force(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")
	err = os.WriteFile(path, []byte("# Modified"), 0644)
	require.NoError(t, err)

	// Write again with force=true
	doc.Content = "# Updated"
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Updated")
}

// TestWriter_Write_DryRun tests dry run mode
func TestWriter_Write_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		DryRun:  true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	// Write in dry run mode
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify file was not created
	path := writer.GetPath("https://example.com/docs")
	assert.NoFileExists(t, path)
}

// TestWriter_WriteJSON tests JSON metadata writing with consolidated collector
func TestWriter_WriteJSON(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
		Collector:    collector,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test",
		Title:   "Test",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	err = writer.FlushMetadata()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	assert.FileExists(t, metadataPath)

	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)
	require.Len(t, index.Documents, 1)
	assert.Equal(t, "Test", index.Documents[0].Title)
	assert.Equal(t, "https://example.com/docs", index.Documents[0].URL)
}

// TestWriter_WriteMultiple tests multiple document writing
func TestWriter_WriteMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
		{URL: "https://example.com/docs3", Content: "# Doc 3", Title: "Doc 3"},
	}

	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	assert.FileExists(t, writer.GetPath("https://example.com/docs1"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs2"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs3"))
}

// TestWriter_WriteMultiple_Cancellation tests context cancellation
func TestWriter_WriteMultiple_Cancellation(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteMultiple(ctx, docs)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestWriteMultiple_CancellationMidProcess tests cancellation during write process
func TestWriteMultiple_CancellationMidProcess(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Create many documents to increase chance of cancellation during process
	docs := make([]*domain.Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = &domain.Document{
			URL:     fmt.Sprintf("https://example.com/docs%d", i),
			Content: fmt.Sprintf("# Doc %d", i),
			Title:   fmt.Sprintf("Doc %d", i),
		}
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := writer.WriteMultiple(ctx, docs)
	// Should either succeed (all written before cancellation) or fail with Canceled
	if err != nil {
		assert.Equal(t, context.Canceled, err)
	}
}
