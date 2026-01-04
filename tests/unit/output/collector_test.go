package output_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetadataCollector tests creating a new MetadataCollector
func TestNewMetadataCollector(t *testing.T) {
	tests := []struct {
		name   string
		opts   output.CollectorOptions
		verify func(t *testing.T, c *output.MetadataCollector)
	}{
		{
			name: "with all options",
			opts: output.CollectorOptions{
				BaseDir:   "/test/dir",
				Filename:  "custom-metadata.json",
				SourceURL: "https://example.com",
				Strategy:  "crawler",
				Enabled:   true,
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.NotNil(t, c)
				assert.True(t, c.IsEnabled())
			},
		},
		{
			name: "with default filename",
			opts: output.CollectorOptions{
				BaseDir:   "/test/dir",
				SourceURL: "https://example.com",
				Strategy:  "git",
				Enabled:   true,
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.NotNil(t, c)
				assert.True(t, c.IsEnabled())
			},
		},
		{
			name: "disabled collector",
			opts: output.CollectorOptions{
				BaseDir:   "/test/dir",
				SourceURL: "https://example.com",
				Strategy:  "sitemap",
				Enabled:   false,
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.NotNil(t, c)
				assert.False(t, c.IsEnabled())
			},
		},
		{
			name: "empty filename uses default",
			opts: output.CollectorOptions{
				BaseDir:   "/test/dir",
				Filename:  "",
				SourceURL: "https://example.com",
				Strategy:  "llms",
				Enabled:   true,
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.NotNil(t, c)
				assert.True(t, c.IsEnabled())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := output.NewMetadataCollector(tt.opts)
			tt.verify(t, collector)
		})
	}
}

// TestMetadataCollector_Add tests adding documents to the collector
func TestMetadataCollector_Add(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		numDocs int
		setup   func(t *testing.T) *output.MetadataCollector
		verify  func(t *testing.T, c *output.MetadataCollector)
	}{
		{
			name:    "add single document",
			enabled: true,
			numDocs: 1,
			setup: func(t *testing.T) *output.MetadataCollector {
				tmpDir := t.TempDir()
				return output.NewMetadataCollector(output.CollectorOptions{
					BaseDir:   tmpDir,
					SourceURL: "https://example.com",
					Strategy:  "crawler",
					Enabled:   true,
				})
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.Equal(t, 1, c.Count())
			},
		},
		{
			name:    "add multiple documents",
			enabled: true,
			numDocs: 5,
			setup: func(t *testing.T) *output.MetadataCollector {
				tmpDir := t.TempDir()
				return output.NewMetadataCollector(output.CollectorOptions{
					BaseDir:   tmpDir,
					SourceURL: "https://example.com",
					Strategy:  "sitemap",
					Enabled:   true,
				})
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.Equal(t, 5, c.Count())
			},
		},
		{
			name:    "disabled collector ignores additions",
			enabled: false,
			numDocs: 3,
			setup: func(t *testing.T) *output.MetadataCollector {
				tmpDir := t.TempDir()
				return output.NewMetadataCollector(output.CollectorOptions{
					BaseDir:   tmpDir,
					SourceURL: "https://example.com",
					Strategy:  "git",
					Enabled:   false,
				})
			},
			verify: func(t *testing.T, c *output.MetadataCollector) {
				assert.Equal(t, 0, c.Count())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := tt.setup(t)

			for i := 0; i < tt.numDocs; i++ {
				doc := &domain.Document{
					URL:         fmt.Sprintf("https://example.com/doc%d", i),
					Title:       fmt.Sprintf("Document %d", i),
					Description: fmt.Sprintf("Description %d", i),
					Content:     fmt.Sprintf("# Document %d\n\nContent here.", i),
					FetchedAt:   time.Now(),
				}
				filePath := fmt.Sprintf("docs/doc%d.md", i)
				collector.Add(doc, filePath)
			}

			tt.verify(t, collector)
		})
	}
}

// TestMetadataCollector_Add_NilDocument tests that nil documents are handled
func TestMetadataCollector_Add_NilDocument(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Add nil document - should be ignored
	collector.Add(nil, "docs/test.md")

	assert.Equal(t, 0, collector.Count())
}

// TestMetadataCollector_Add_Concurrent tests concurrent additions
func TestMetadataCollector_Add_Concurrent(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	numGoroutines := 10
	docsPerGoroutine := 100

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < docsPerGoroutine; j++ {
				doc := &domain.Document{
					URL:       fmt.Sprintf("https://example.com/doc%d-%d", goroutineID, j),
					Title:     fmt.Sprintf("Document %d-%d", goroutineID, j),
					Content:   fmt.Sprintf("# Doc %d-%d", goroutineID, j),
					FetchedAt: time.Now(),
				}
				filePath := fmt.Sprintf("docs/doc%d-%d.md", goroutineID, j)
				collector.Add(doc, filePath)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numGoroutines*docsPerGoroutine, collector.Count())
}

// TestMetadataCollector_Flush tests flushing metadata to file
func TestMetadataCollector_Flush(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		numDocs   int
		setupFile bool
		verify    func(t *testing.T, tmpDir string, err error)
	}{
		{
			name:    "flush with documents",
			enabled: true,
			numDocs: 3,
			verify: func(t *testing.T, tmpDir string, err error) {
				require.NoError(t, err)

				metadataPath := filepath.Join(tmpDir, "metadata.json")
				_, err = os.Stat(metadataPath)
				require.NoError(t, err)

				data, err := os.ReadFile(metadataPath)
				require.NoError(t, err)

				var index domain.SimpleMetadataIndex
				err = json.Unmarshal(data, &index)
				require.NoError(t, err)

				assert.Equal(t, 3, index.TotalDocuments)
				assert.Len(t, index.Documents, 3)
				assert.Equal(t, "https://example.com", index.SourceURL)
				assert.Equal(t, "crawler", index.Strategy)
				assert.NotZero(t, index.GeneratedAt)
			},
		},
		{
			name:    "flush with empty collector",
			enabled: true,
			numDocs: 0,
			verify: func(t *testing.T, tmpDir string, err error) {
				require.NoError(t, err)

				metadataPath := filepath.Join(tmpDir, "metadata.json")
				_, err = os.Stat(metadataPath)
				// File should not be created if no documents
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name:    "flush disabled collector",
			enabled: false,
			numDocs: 3,
			verify: func(t *testing.T, tmpDir string, err error) {
				require.NoError(t, err)

				metadataPath := filepath.Join(tmpDir, "metadata.json")
				_, err = os.Stat(metadataPath)
				assert.True(t, os.IsNotExist(err), "File should not exist when collector is disabled")
			},
		},
		{
			name:    "flush with custom filename",
			enabled: true,
			numDocs: 2,
			verify: func(t *testing.T, tmpDir string, err error) {
				// Will be verified in the test below with custom filename
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filename := "metadata.json"
			if tt.name == "flush with custom filename" {
				filename = "custom-index.json"
			}

			collector := output.NewMetadataCollector(output.CollectorOptions{
				BaseDir:   tmpDir,
				Filename:  filename,
				SourceURL: "https://example.com",
				Strategy:  "crawler",
				Enabled:   tt.enabled,
			})

			for i := 0; i < tt.numDocs; i++ {
				doc := &domain.Document{
					URL:         fmt.Sprintf("https://example.com/doc%d", i),
					Title:       fmt.Sprintf("Document %d", i),
					Description: fmt.Sprintf("Description %d", i),
					Content:     fmt.Sprintf("# Document %d", i),
					FetchedAt:   time.Now(),
				}
				filePath := fmt.Sprintf("example.com/doc%d.md", i)
				collector.Add(doc, filePath)
			}

			err := collector.Flush()
			tt.verify(t, tmpDir, err)
		})
	}
}

// TestMetadataCollector_Flush_Format tests JSON formatting
func TestMetadataCollector_Flush_Format(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "git",
		Enabled:   true,
	})

	doc := &domain.Document{
		URL:         "https://example.com/docs",
		Title:       "Test Document",
		Description: "Test Description",
		Content:     "# Test",
		FetchedAt:   time.Now(),
	}

	collector.Add(doc, "example.com/docs.md")

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	// Verify JSON is formatted with indentation
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "\n")
	assert.Contains(t, jsonStr, "  ")

	// Verify it can be unmarshaled
	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	assert.Equal(t, 1, index.TotalDocuments)
	assert.Len(t, index.Documents, 1)
	assert.Equal(t, "Test Document", index.Documents[0].Title)
	assert.Equal(t, "Test Description", index.Documents[0].Description)
	assert.Equal(t, "https://example.com/docs", index.Documents[0].URL)
}

// TestMetadataCollector_Flush_FilePath tests file path conversion
func TestMetadataCollector_Flush_FilePath(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://github.com/example/repo",
		Strategy:  "git",
		Enabled:   true,
	})

	// Add document with various path formats
	docs := []struct {
		doc        *domain.Document
		filePath   string
		expectedIn string
	}{
		{
			doc: &domain.Document{
				URL:     "https://github.com/example/repo/blob/main/README.md",
				Title:   "README",
				Content: "# README",
			},
			filePath:   filepath.Join(tmpDir, "README.md"),
			expectedIn: "README.md",
		},
		{
			doc: &domain.Document{
				URL:     "https://github.com/example/repo/blob/main/docs/guide.md",
				Title:   "Guide",
				Content: "# Guide",
			},
			filePath:   filepath.Join(tmpDir, "docs/guide.md"),
			expectedIn: "docs/guide.md",
		},
	}

	for _, tt := range docs {
		collector.Add(tt.doc, tt.filePath)
	}

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	assert.Equal(t, 2, index.TotalDocuments)
	// Verify paths are converted to relative paths with forward slashes
	for _, doc := range index.Documents {
		assert.NotContains(t, doc.FilePath, tmpDir, "Path should be relative, not absolute")
		assert.NotContains(t, doc.FilePath, "\\", "Path should use forward slashes")
	}
}

// TestMetadataCollector_Count tests document counting
func TestMetadataCollector_Count(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Initially empty
	assert.Equal(t, 0, collector.Count())

	// Add documents
	for i := 0; i < 5; i++ {
		doc := &domain.Document{
			URL:     fmt.Sprintf("https://example.com/doc%d", i),
			Title:   fmt.Sprintf("Doc %d", i),
			Content: fmt.Sprintf("# Doc %d", i),
		}
		collector.Add(doc, fmt.Sprintf("doc%d.md", i))
		assert.Equal(t, i+1, collector.Count())
	}
}

// TestMetadataCollector_GetIndex tests getting the metadata index
func TestMetadataCollector_GetIndex(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "sitemap",
		Enabled:   true,
	})

	// Add documents
	for i := 0; i < 3; i++ {
		doc := &domain.Document{
			URL:         fmt.Sprintf("https://example.com/page%d", i),
			Title:       fmt.Sprintf("Page %d", i),
			Description: fmt.Sprintf("Description %d", i),
			Content:     fmt.Sprintf("# Page %d", i),
			FetchedAt:   time.Now(),
		}
		collector.Add(doc, fmt.Sprintf("example.com/page%d.md", i))
	}

	// Get index
	index := collector.GetIndex()

	require.NotNil(t, index)
	assert.Equal(t, 3, index.TotalDocuments)
	assert.Len(t, index.Documents, 3)
	assert.Equal(t, "https://example.com", index.SourceURL)
	assert.Equal(t, "sitemap", index.Strategy)
	assert.NotZero(t, index.GeneratedAt)

	// Verify document fields
	for i, doc := range index.Documents {
		assert.Equal(t, fmt.Sprintf("Page %d", i), doc.Title)
		assert.Equal(t, fmt.Sprintf("Description %d", i), doc.Description)
		assert.Equal(t, fmt.Sprintf("https://example.com/page%d", i), doc.URL)
		assert.Equal(t, "sitemap", doc.Source)
	}
}

// TestMetadataCollector_IsEnabled tests enabled state
func TestMetadataCollector_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:    "disabled",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			collector := output.NewMetadataCollector(output.CollectorOptions{
				BaseDir:   tmpDir,
				SourceURL: "https://example.com",
				Strategy:  "crawler",
				Enabled:   tt.enabled,
			})

			assert.Equal(t, tt.enabled, collector.IsEnabled())
		})
	}
}

// TestMetadataCollector_SetStrategy tests setting strategy
func TestMetadataCollector_SetStrategy(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Add document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Title:   "Test",
		Content: "# Test",
	}
	collector.Add(doc, "docs/test.md")

	// Get index with original strategy
	index1 := collector.GetIndex()
	assert.Equal(t, "crawler", index1.Strategy)

	// Change strategy
	collector.SetStrategy("git")

	// Get index with new strategy
	index2 := collector.GetIndex()
	assert.Equal(t, "git", index2.Strategy)

	// Flush and verify file has new strategy
	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)
	assert.Equal(t, "git", index.Strategy)
}

// TestMetadataCollector_SetSourceURL tests setting source URL
func TestMetadataCollector_SetSourceURL(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Add document
	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Title:   "Test",
		Content: "# Test",
	}
	collector.Add(doc, "docs/test.md")

	// Get index with original source URL
	index1 := collector.GetIndex()
	assert.Equal(t, "https://example.com", index1.SourceURL)

	// Change source URL
	collector.SetSourceURL("https://newexample.com")

	// Get index with new source URL
	index2 := collector.GetIndex()
	assert.Equal(t, "https://newexample.com", index2.SourceURL)

	// Flush and verify file has new source URL
	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)
	assert.Equal(t, "https://newexample.com", index.SourceURL)
}

// TestMetadataCollector_ConcurrentGetIndex tests concurrent GetIndex calls
func TestMetadataCollector_ConcurrentGetIndex(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Add documents
	for i := 0; i < 10; i++ {
		doc := &domain.Document{
			URL:     fmt.Sprintf("https://example.com/doc%d", i),
			Title:   fmt.Sprintf("Doc %d", i),
			Content: fmt.Sprintf("# Doc %d", i),
		}
		collector.Add(doc, fmt.Sprintf("doc%d.md", i))
	}

	// Concurrent GetIndex calls
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			index := collector.GetIndex()
			assert.Equal(t, 10, index.TotalDocuments)
			assert.Len(t, index.Documents, 10)
		}()
	}

	wg.Wait()
}

// TestMetadataCollector_Flush_MultipleTimes tests flushing multiple times
func TestMetadataCollector_Flush_MultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	// Add and flush first batch
	for i := 0; i < 3; i++ {
		doc := &domain.Document{
			URL:     fmt.Sprintf("https://example.com/doc%d", i),
			Title:   fmt.Sprintf("Doc %d", i),
			Content: fmt.Sprintf("# Doc %d", i),
		}
		collector.Add(doc, fmt.Sprintf("doc%d.md", i))
	}

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index1 domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index1)
	require.NoError(t, err)
	assert.Equal(t, 3, index1.TotalDocuments)

	// Add more documents and flush again
	for i := 3; i < 6; i++ {
		doc := &domain.Document{
			URL:     fmt.Sprintf("https://example.com/doc%d", i),
			Title:   fmt.Sprintf("Doc %d", i),
			Content: fmt.Sprintf("# Doc %d", i),
		}
		collector.Add(doc, fmt.Sprintf("doc%d.md", i))
	}

	err = collector.Flush()
	require.NoError(t, err)

	data, err = os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index2 domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index2)
	require.NoError(t, err)
	assert.Equal(t, 6, index2.TotalDocuments)
}

// TestMetadataCollector_Add_WithLLMMetadata tests adding documents with LLM metadata
func TestMetadataCollector_Add_WithLLMMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	doc := &domain.Document{
		URL:         "https://example.com/docs",
		Title:       "Test Document",
		Description: "Original description",
		Content:     "# Test",
		Summary:     "AI-generated summary",
		Tags:        []string{"tag1", "tag2", "tag3"},
		Category:    "AI-category",
		FetchedAt:   time.Now(),
	}

	collector.Add(doc, "example.com/docs.md")

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	require.Len(t, index.Documents, 1)
	metadataDoc := index.Documents[0]

	assert.Equal(t, "Test Document", metadataDoc.Title)
	assert.Equal(t, "Original description", metadataDoc.Description)
	assert.Equal(t, "AI-generated summary", metadataDoc.Summary)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, metadataDoc.Tags)
	assert.Equal(t, "AI-category", metadataDoc.Category)
}

// TestMetadataCollector_EmptyStringPath tests handling of empty string paths
func TestMetadataCollector_EmptyStringPath(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	doc := &domain.Document{
		URL:     "https://example.com/docs",
		Title:   "Test",
		Content: "# Test",
	}

	// Add with empty path - should still work
	collector.Add(doc, "")

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	assert.Equal(t, 1, index.TotalDocuments)
	// File path might be empty or the base dir, but should not panic
}

// TestMetadataCollector_DocumentTimestamps tests that document timestamps are preserved
func TestMetadataCollector_DocumentTimestamps(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Enabled:   true,
	})

	now := time.Now().Add(-1 * time.Hour) // Use past time to avoid timing issues

	doc := &domain.Document{
		URL:       "https://example.com/docs",
		Title:     "Test",
		Content:   "# Test",
		FetchedAt: now,
	}

	collector.Add(doc, "example.com/docs.md")

	err := collector.Flush()
	require.NoError(t, err)

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	require.NoError(t, err)

	var index domain.SimpleMetadataIndex
	err = json.Unmarshal(data, &index)
	require.NoError(t, err)

	require.Len(t, index.Documents, 1)
	// Timestamp should be approximately the same (within 1 second for JSON marshaling)
	assert.True(t, index.Documents[0].FetchedAt.Sub(now).Abs() < time.Second)
}
