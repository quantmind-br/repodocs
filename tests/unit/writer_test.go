package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
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
	// GetPath returns the full file path, not just the base dir
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
	// GetPath returns the full file path
	path := writer.GetPath("https://example.com")
	// Default base dir is ./docs, and URL converts to index.md in nested structure
	assert.Contains(t, path, "docs")
	assert.Contains(t, path, ".md")
}

func TestWriter_Write_Success(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document\n\nThis is a test.",
		Title:   "Test Document",
	}

	// Write document
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify file was created
	path := writer.GetPath("https://example.com/docs")
	assert.FileExists(t, path)

	// Verify content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Test Document")
}

func TestWriter_Write_WithRelativePath(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create document with relative path
	doc := &domain.Document{
		RelativePath: "docs/guide.md",
		Content:      "# Guide\n\nThis is a guide.",
		Title:        "Guide",
	}

	// Write document
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify file was created with relative path
	path := filepath.Join(tmpDir, "docs/guide.md")
	assert.FileExists(t, path)
}

func TestWriter_Write_SkipExisting(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer with force=false
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   false,
	})

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	// Write document first time
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify file exists
	path := writer.GetPath("https://example.com/docs")
	firstContent, err := os.ReadFile(path)
	require.NoError(t, err)

	// Write again (should skip)
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify content unchanged
	secondContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, firstContent, secondContent)
}

func TestWriter_Write_Force(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer with force=true
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	// Write document
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Modify the file
	path := writer.GetPath("https://example.com/docs")
	err = os.WriteFile(path, []byte("# Modified"), 0644)
	require.NoError(t, err)

	// Write again with force=true (should overwrite)
	doc.Content = "# Updated Document"
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify content was overwritten
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Updated Document")
}

func TestWriter_Write_DryRun(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer with dryRun=true
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		DryRun:  true,
	})

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	// Write document (should not create file)
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Verify file was not created
	path := writer.GetPath("https://example.com/docs")
	assert.NoFileExists(t, path)
}

func TestWriter_WriteJSON(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer with JSON metadata enabled
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      tmpDir,
		JSONMetadata: true,
	})

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	// Write document
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// Get the actual markdown path
	mdPath := writer.GetPath("https://example.com/docs")

	// Verify JSON metadata was created
	jsonPath := utils.JSONPath(mdPath)
	assert.FileExists(t, jsonPath)

	// Verify JSON content
	data, err := os.ReadFile(jsonPath)
	require.NoError(t, err)

	var metadata map[string]interface{}
	err = json.Unmarshal(data, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "Test Document", metadata["title"])
	assert.Equal(t, "https://example.com/docs", metadata["url"])
}

func TestWriter_WriteMultiple(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create multiple documents
	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
		{URL: "https://example.com/docs3", Content: "# Doc 3", Title: "Doc 3"},
	}

	// Write all documents
	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	// Verify all files were created
	assert.FileExists(t, writer.GetPath("https://example.com/docs1"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs2"))
	assert.FileExists(t, writer.GetPath("https://example.com/docs3"))
}

func TestWriter_WriteMultiple_Cancellation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create documents
	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Write documents (should return context cancelled error)
	err := writer.WriteMultiple(ctx, docs)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWriter_Exists(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// File doesn't exist yet
	assert.False(t, writer.Exists("https://example.com/docs"))

	// Create document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
	}

	// Write document
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	// File now exists
	assert.True(t, writer.Exists("https://example.com/docs"))
}

func TestWriter_EnsureBaseDir(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, "docs", "subdir")

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: baseDir,
	})

	// Ensure base directory
	err := writer.EnsureBaseDir()
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, baseDir)
}

func TestWriter_Clean(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create some files
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Test Document",
		Title:   "Test Document",
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

func TestWriter_Stats(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	// Create documents - only .md files will be counted
	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1\n\nSome content here.", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2\n\nMore content here.", Title: "Doc 2"},
		{URL: "https://example.com/docs3", Content: "# Doc 3\n\nEven more content.", Title: "Doc 3"},
	}

	// Write all documents
	err := writer.WriteMultiple(context.Background(), docs)
	require.NoError(t, err)

	// Get stats
	count, size, err := writer.Stats()
	require.NoError(t, err)

	// Verify stats (should count all 3 .md files)
	assert.Equal(t, 3, count)
	assert.True(t, size > 0)
}
