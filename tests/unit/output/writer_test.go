package output_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWriter tests creating a new Writer with various options
func TestNewWriter(t *testing.T) {
	tests := []struct {
		name    string
		opts    output.WriterOptions
		verify  func(t *testing.T, w *output.Writer)
	}{
		{
			name: "with all options",
			opts: output.WriterOptions{
				BaseDir:      "/custom/dir",
				Flat:         true,
				JSONMetadata: true,
				Force:        true,
				DryRun:       true,
				Collector:    nil,
			},
			verify: func(t *testing.T, w *output.Writer) {
				assert.NotNil(t, w)
				path := w.GetPath("https://example.com")
				assert.Contains(t, path, "/custom/dir")
			},
		},
		{
			name: "with default base dir",
			opts: output.WriterOptions{
				BaseDir: "",
			},
			verify: func(t *testing.T, w *output.Writer) {
				assert.NotNil(t, w)
				path := w.GetPath("https://example.com")
				assert.Contains(t, path, "docs")
			},
		},
		{
			name: "with collector",
			opts: output.WriterOptions{
				BaseDir:   "/test/dir",
				Collector: output.NewMetadataCollector(output.CollectorOptions{BaseDir: "/test/dir"}),
			},
			verify: func(t *testing.T, w *output.Writer) {
				assert.NotNil(t, w)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := output.NewWriter(tt.opts)
			tt.verify(t, writer)
		})
	}
}

// TestWriter_Write_Success tests successful document writing
func TestWriter_Write_Success(t *testing.T) {
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
	_, err = os.Stat(path)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Test Document")
}

// TestWriter_Write_WithRelativePath tests writing with relative path (Git sources)
func TestWriter_Write_WithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Flat:    false,
	})

	doc := &domain.Document{
		URL:         "https://github.com/example/repo",
		RelativePath: "docs/guide.md",
		Content:     "# Guide\n\nThis is a guide.",
		Title:       "Guide",
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	expectedPath := filepath.Join(tmpDir, "docs/guide.md")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err)

	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Guide")
}

// TestWriter_Write_WithFrontmatter tests frontmatter is added correctly
func TestWriter_Write_WithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	doc := &domain.Document{
		URL:         "https://example.com/test",
		Content:     "# Main Content",
		Title:       "Test Document",
		Description: "Test description",
		FetchedAt:   time.Now(),
		WordCount:   100,
	}

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/test")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "---")
	assert.Contains(t, contentStr, "title: Test Document")
	assert.Contains(t, contentStr, "url: https://example.com/test")
	assert.Contains(t, contentStr, "# Main Content")
}

// TestWriter_Write_SkipExisting tests that existing files are skipped when Force=false
func TestWriter_Write_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   false,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Original Content",
		Title:   "Test",
	}

	// Write first time
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")
	firstContent, err := os.ReadFile(path)
	require.NoError(t, err)

	// Modify the document and write again (should skip)
	doc.Content = "# Modified Content"
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	secondContent, err := os.ReadFile(path)
	require.NoError(t, err)

	// Content should be unchanged
	assert.Equal(t, firstContent, secondContent)
	assert.Contains(t, string(firstContent), "# Original Content")
}

// TestWriter_Write_ForceOverwrite tests force overwrite
func TestWriter_Write_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Content: "# Original Content",
		Title:   "Test",
	}

	// Write first time
	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")

	// Manually modify the file
	err = os.WriteFile(path, []byte("# Manually Modified"), 0644)
	require.NoError(t, err)

	// Write again with force=true
	doc.Content = "# Updated Content"
	err = writer.Write(context.Background(), doc)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# Updated Content")
	assert.NotContains(t, string(content), "Manually Modified")
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

	err := writer.Write(context.Background(), doc)
	require.NoError(t, err)

	path := writer.GetPath("https://example.com/docs")
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "File should not exist in dry run mode")
}

// TestWriter_Write_WithMetadata tests metadata collection
func TestWriter_Write_WithMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
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

	// Verify document was added to collector
	assert.Equal(t, 1, collector.Count())

	// Flush metadata
	err = writer.FlushMetadata()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	_, err = os.Stat(metadataPath)
	require.NoError(t, err)

	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	assert.Equal(t, 1, index.TotalDocuments)
	assert.Len(t, index.Documents, 1)
	assert.Equal(t, "Test Document", index.Documents[0].Title)
	assert.Equal(t, "Test Description", index.Documents[0].Description)
}

// TestWriter_Write_EmptyContent tests writing with empty content
func TestWriter_Write_EmptyContent(t *testing.T) {
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
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Empty Document")
}

// TestWriter_Write_InvalidPath tests error handling for invalid paths
func TestWriter_Write_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file at the location where we want to create a directory
	invalidPath := filepath.Join(tmpDir, "existingfile.txt")
	err := os.WriteFile(invalidPath, []byte("content"), 0644)
	require.NoError(t, err)

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: invalidPath,
		Force:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/test",
		Content: "# Test",
		Title:   "Test",
	}

	err = writer.Write(context.Background(), doc)
	assert.Error(t, err)
}

// TestWriter_WriteMultiple_Success tests writing multiple documents
func TestWriter_WriteMultiple_Success(t *testing.T) {
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

// TestWriter_WriteMultiple_ContextCancellation tests context cancellation
func TestWriter_WriteMultiple_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	docs := []*domain.Document{
		{URL: "https://example.com/docs1", Content: "# Doc 1", Title: "Doc 1"},
		{URL: "https://example.com/docs2", Content: "# Doc 2", Title: "Doc 2"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := writer.WriteMultiple(ctx, docs)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestWriter_WriteMultiple_EmptyList tests empty document list
func TestWriter_WriteMultiple_EmptyList(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
	})

	err := writer.WriteMultiple(context.Background(), []*domain.Document{})
	require.NoError(t, err)
}

// TestWriter_GetPath tests path generation
func TestWriter_GetPath(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		flat     bool
		url      string
		contains []string
	}{
		{
			name:    "nested structure",
			baseDir: "/output",
			flat:    false,
			url:     "https://example.com/docs/guide",
			contains: []string{
				"/output",
				"docs",
				".md",
			},
		},
		{
			name:    "flat structure",
			baseDir: "/output",
			flat:    true,
			url:     "https://example.com/docs/guide",
			contains: []string{
				"/output",
				".md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := output.NewWriter(output.WriterOptions{
				BaseDir: tt.baseDir,
				Flat:    tt.flat,
			})

			path := writer.GetPath(tt.url)
			for _, expected := range tt.contains {
				assert.Contains(t, path, expected)
			}
		})
	}
}

// TestWriter_Exists tests file existence checking
func TestWriter_Exists(t *testing.T) {
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

// TestWriter_EnsureBaseDir tests base directory creation
func TestWriter_EnsureBaseDir(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		setup   func(string)
		verify  func(t *testing.T, baseDir string, err error)
	}{
		{
			name:    "create new directory",
			baseDir: filepath.Join(t.TempDir(), "new", "docs"),
			verify: func(t *testing.T, baseDir string, err error) {
				require.NoError(t, err)
				info, err := os.Stat(baseDir)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
		{
			name:    "directory already exists",
			baseDir: filepath.Join(t.TempDir(), "existing"),
			setup: func(baseDir string) {
				err := os.MkdirAll(baseDir, 0755)
				require.NoError(t, err)
			},
			verify: func(t *testing.T, baseDir string, err error) {
				require.NoError(t, err)
				info, err := os.Stat(baseDir)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.baseDir)
			}

			writer := output.NewWriter(output.WriterOptions{
				BaseDir: tt.baseDir,
			})

			err := writer.EnsureBaseDir()
			tt.verify(t, tt.baseDir, err)
		})
	}
}

// TestWriter_Clean tests directory cleaning
func TestWriter_Clean(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, writer *output.Writer)
		verify  func(t *testing.T, writer *output.Writer, err error)
	}{
		{
			name: "remove directory with files",
			setup: func(t *testing.T, writer *output.Writer) {
				doc := &domain.Document{
					URL:     "https://example.com/docs",
					Content: "# Test",
					Title:   "Test",
				}
				err := writer.Write(context.Background(), doc)
				require.NoError(t, err)
			},
			verify: func(t *testing.T, writer *output.Writer, err error) {
				require.NoError(t, err)
				path := writer.GetPath("https://example.com/docs")
				_, err = os.Stat(path)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "clean empty directory",
			setup: func(t *testing.T, writer *output.Writer) {
				// No files created
			},
			verify: func(t *testing.T, writer *output.Writer, err error) {
				require.NoError(t, err)
				_, statErr := os.Stat(writer.GetPath("https://example.com/test"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writer := output.NewWriter(output.WriterOptions{
				BaseDir: tmpDir,
			})

			if tt.setup != nil {
				tt.setup(t, writer)
			}

			err := writer.Clean()
			tt.verify(t, writer, err)
		})
	}
}

// TestWriter_Stats tests statistics gathering
func TestWriter_Stats(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, writer *output.Writer)
		expectedCount  int
		expectedMinSize int64
	}{
		{
			name: "multiple markdown files",
			setup: func(t *testing.T, writer *output.Writer) {
				docs := []*domain.Document{
					{URL: "https://example.com/doc1", Content: "# Doc 1\n\nContent here.", Title: "Doc 1"},
					{URL: "https://example.com/doc2", Content: "# Doc 2\n\nMore content.", Title: "Doc 2"},
					{URL: "https://example.com/doc3", Content: "# Doc 3\n\nEven more.", Title: "Doc 3"},
				}
				err := writer.WriteMultiple(context.Background(), docs)
				require.NoError(t, err)
			},
			expectedCount:  3,
			expectedMinSize: 1,
		},
		{
			name: "with non-markdown files",
			setup: func(t *testing.T, writer *output.Writer) {
				docs := []*domain.Document{
					{URL: "https://example.com/doc1", Content: "# Doc 1", Title: "Doc 1"},
					{URL: "https://example.com/doc2", Content: "# Doc 2", Title: "Doc 2"},
				}
				err := writer.WriteMultiple(context.Background(), docs)
				require.NoError(t, err)

				// Add non-markdown files
				baseDir := filepath.Dir(filepath.Dir(writer.GetPath("https://example.com/doc1")))
				readmePath := filepath.Join(baseDir, "readme.txt")
				err = os.WriteFile(readmePath, []byte("Readme"), 0644)
				require.NoError(t, err)

				jsonPath := filepath.Join(baseDir, "data.json")
				err = os.WriteFile(jsonPath, []byte("{}"), 0644)
				require.NoError(t, err)
			},
			expectedCount:  2, // Only .md files
			expectedMinSize: 1,
		},
		{
			name:           "empty directory",
			setup:          func(t *testing.T, writer *output.Writer) {},
			expectedCount:  0,
			expectedMinSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writer := output.NewWriter(output.WriterOptions{
				BaseDir: tmpDir,
			})

			if tt.setup != nil {
				tt.setup(t, writer)
			}

			count, size, err := writer.Stats()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, count)
			assert.True(t, size >= tt.expectedMinSize)
		})
	}
}

// TestWriter_FlushMetadata tests metadata flushing
func TestWriter_FlushMetadata(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) (*output.Writer, *domain.Document)
		verify    func(t *testing.T, tmpDir string, err error)
	}{
		{
			name: "flush with collector",
			setup: func(t *testing.T) (*output.Writer, *domain.Document) {
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

				return writer, doc
			},
			verify: func(t *testing.T, tmpDir string, err error) {
				// Tested in TestWriter_Write_WithMetadata
				require.NoError(t, err)
			},
		},
		{
			name: "flush without collector",
			setup: func(t *testing.T) (*output.Writer, *domain.Document) {
				tmpDir := t.TempDir()
				writer := output.NewWriter(output.WriterOptions{
					BaseDir:      tmpDir,
					JSONMetadata: false,
					Collector:    nil,
				})

				return writer, nil
			},
			verify: func(t *testing.T, tmpDir string, err error) {
				require.NoError(t, err)
				metadataPath := filepath.Join(tmpDir, "metadata.json")
				_, err = os.Stat(metadataPath)
				assert.True(t, os.IsNotExist(err), "metadata.json should not exist")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, doc := tt.setup(t)
			if doc != nil {
				err := writer.Write(context.Background(), doc)
				require.NoError(t, err)
			}

			err := writer.FlushMetadata()
			tmpDir := writer.GetPath("https://test")
			tmpDir = filepath.Dir(filepath.Dir(tmpDir))
			tt.verify(t, tmpDir, err)
		})
	}
}

// TestWriter_Write_FlatStructure tests flat vs nested output
func TestWriter_Write_FlatStructure(t *testing.T) {
	tests := []struct {
		name           string
		flat           bool
		url            string
		expectedInPath []string
		notExpected    []string
	}{
		{
			name:           "flat structure",
			flat:           true,
			url:            "https://example.com/docs/guide",
			expectedInPath: []string{".md"},
			notExpected:    []string{filepath.Join("example.com", "docs")},
		},
		{
			name:           "nested structure",
			flat:           false,
			url:            "https://example.com/docs/guide",
			expectedInPath: []string{filepath.Join("docs"), ".md"},
			notExpected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			writer := output.NewWriter(output.WriterOptions{
				BaseDir: tmpDir,
				Flat:    tt.flat,
			})

			doc := &domain.Document{
				URL:     tt.url,
				Content: "# Test",
				Title:   "Test",
			}

			err := writer.Write(context.Background(), doc)
			require.NoError(t, err)

			path := writer.GetPath(tt.url)
			for _, expected := range tt.expectedInPath {
				assert.Contains(t, path, expected, "Path should contain %s", expected)
			}
			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, path, notExpected, "Path should not contain %s", notExpected)
			}

			// Verify file exists
			_, err = os.Stat(path)
			require.NoError(t, err)
		})
	}
}
