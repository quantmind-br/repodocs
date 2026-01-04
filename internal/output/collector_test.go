package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetadataCollector tests creating a new metadata collector
func TestNewMetadataCollector(t *testing.T) {
	tests := []struct {
		name  string
		opts  CollectorOptions
		check func(t *testing.T, c *MetadataCollector)
	}{
		{
			name: "with all options",
			opts: CollectorOptions{
				BaseDir:   "./output",
				Filename:  "test-metadata.json",
				SourceURL: "https://example.com",
				Strategy:  "crawler",
				Enabled:   true,
			},
			check: func(t *testing.T, c *MetadataCollector) {
				assert.Equal(t, "./output", c.baseDir)
				assert.Equal(t, "test-metadata.json", c.filename)
				assert.Equal(t, "https://example.com", c.sourceURL)
				assert.Equal(t, "crawler", c.strategy)
				assert.True(t, c.enabled)
			},
		},
		{
			name: "with empty filename uses default",
			opts: CollectorOptions{
				BaseDir: "./output",
			},
			check: func(t *testing.T, c *MetadataCollector) {
				assert.Equal(t, "metadata.json", c.filename)
			},
		},
		{
			name: "disabled by default",
			opts: CollectorOptions{
				BaseDir: "./output",
			},
			check: func(t *testing.T, c *MetadataCollector) {
				assert.False(t, c.enabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMetadataCollector(tt.opts)
			if tt.check != nil {
				tt.check(t, c)
			}
		})
	}
}

// TestMetadataCollector_Add tests adding documents to collector
func TestMetadataCollector_Add(t *testing.T) {
	t.Run("adds document when enabled", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir:  "./output",
			Enabled:  true,
			Strategy: "test",
		})

		doc := &domain.Document{
			URL:     "https://example.com/page1",
			Title:   "Test Page",
			Content: "Test content",
		}

		c.Add(doc, "/output/example.com/page1.md")

		assert.Equal(t, 1, c.Count())
	})

	t.Run("does not add when disabled", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: false,
		})

		doc := &domain.Document{
			URL:     "https://example.com/page1",
			Title:   "Test Page",
			Content: "Test content",
		}

		c.Add(doc, "/output/example.com/page1.md")

		assert.Equal(t, 0, c.Count())
	})

	t.Run("handles nil document", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		c.Add(nil, "/output/example.com/page1.md")

		assert.Equal(t, 0, c.Count())
	})

	t.Run("computes relative path correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: tmpDir,
			Enabled: true,
		})

		doc := &domain.Document{
			URL:     "https://example.com/page",
			Title:   "Test",
			Content: "Content",
		}

		filePath := filepath.Join(tmpDir, "example.com", "page.md")
		c.Add(doc, filePath)

		index := c.GetIndex()
		require.Len(t, index.Documents, 1)
		// FilePath should be relative to baseDir
		assert.Contains(t, index.Documents[0].FilePath, "example.com")
		assert.NotContains(t, index.Documents[0].FilePath, tmpDir)
	})

	t.Run("adds multiple documents", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		docs := []*domain.Document{
			{URL: "https://example.com/page1", Title: "Page 1", Content: "Content 1"},
			{URL: "https://example.com/page2", Title: "Page 2", Content: "Content 2"},
			{URL: "https://example.com/page3", Title: "Page 3", Content: "Content 3"},
		}

		for _, doc := range docs {
			c.Add(doc, "/output/example.com/"+doc.URL[len(doc.URL)-1:]+".md")
		}

		assert.Equal(t, 3, c.Count())
	})
}

// TestMetadataCollector_Flush tests flushing metadata to file
func TestMetadataCollector_Flush(t *testing.T) {
	t.Run("flushes to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir:   tmpDir,
			Filename:  "test-metadata.json",
			SourceURL: "https://example.com",
			Strategy:  "test",
			Enabled:   true,
		})

		doc := &domain.Document{
			URL:     "https://example.com/page1",
			Title:   "Test Page",
			Content: "Test content",
		}

		c.Add(doc, filepath.Join(tmpDir, "example.com", "page1.md"))

		err := c.Flush()
		assert.NoError(t, err)

		// Check file was created
		outputPath := filepath.Join(tmpDir, "test-metadata.json")
		_, err = os.Stat(outputPath)
		assert.NoError(t, err)

		// Verify JSON content
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		var index domain.SimpleMetadataIndex
		err = json.Unmarshal(data, &index)
		assert.NoError(t, err)

		assert.Equal(t, "https://example.com", index.SourceURL)
		assert.Equal(t, "test", index.Strategy)
		assert.Equal(t, 1, index.TotalDocuments)
		assert.Len(t, index.Documents, 1)
		assert.Equal(t, "Test Page", index.Documents[0].Title)
		assert.NotZero(t, index.GeneratedAt)
	})

	t.Run("flush when disabled is no-op", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: false,
		})

		err := c.Flush()
		assert.NoError(t, err)
	})

	t.Run("flush with no documents is no-op", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: tmpDir,
			Enabled: true,
		})

		err := c.Flush()
		assert.NoError(t, err)

		// No file should be created
		files, err := os.ReadDir(tmpDir)
		assert.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("flush preserves all document fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir:   tmpDir,
			SourceURL: "https://example.com",
			Strategy:  "crawler",
			Enabled:   true,
		})

		doc := &domain.Document{
			URL:            "https://example.com/page1",
			Title:          "Test Page",
			Content:        "Test content",
			Summary:        "Test summary",
			Category:       "guide",
			Tags:           []string{"tag1", "tag2"},
			WordCount:      100,
			CharCount:      500,
			FetchedAt:      time.Now(),
			SourceStrategy: "crawler",
		}

		c.Add(doc, filepath.Join(tmpDir, "example.com", "page1.md"))

		err := c.Flush()
		assert.NoError(t, err)

		outputPath := filepath.Join(tmpDir, "metadata.json")
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		var index domain.SimpleMetadataIndex
		err = json.Unmarshal(data, &index)
		assert.NoError(t, err)

		assert.Len(t, index.Documents, 1)
		metadata := index.Documents[0]
		assert.Equal(t, "Test Page", metadata.Title)
		assert.Equal(t, "Test summary", metadata.Summary)
		assert.Equal(t, "guide", metadata.Category)
		assert.Equal(t, []string{"tag1", "tag2"}, metadata.Tags)
		assert.Equal(t, "https://example.com/page1", metadata.URL)
		assert.Equal(t, "crawler", metadata.Source)
		assert.NotZero(t, metadata.FetchedAt)
	})

	t.Run("multiple documents flushed correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: tmpDir,
			Enabled: true,
		})

		docs := []*domain.Document{
			{URL: "https://example.com/page1", Title: "Page 1", Content: "Content 1"},
			{URL: "https://example.com/page2", Title: "Page 2", Content: "Content 2"},
		}

		c.Add(docs[0], filepath.Join(tmpDir, "page1.md"))
		c.Add(docs[1], filepath.Join(tmpDir, "page2.md"))

		err := c.Flush()
		assert.NoError(t, err)

		outputPath := filepath.Join(tmpDir, "metadata.json")
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		var index domain.SimpleMetadataIndex
		err = json.Unmarshal(data, &index)
		assert.NoError(t, err)

		assert.Equal(t, 2, index.TotalDocuments)
		assert.Len(t, index.Documents, 2)
	})
}

// TestMetadataCollector_Count tests counting documents
func TestMetadataCollector_Count(t *testing.T) {
	t.Run("returns zero when empty", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		assert.Equal(t, 0, c.Count())
	})

	t.Run("returns count of added documents", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		for i := 0; i < 5; i++ {
			c.Add(&domain.Document{
				URL:     "https://example.com/page",
				Title:   "Test",
				Content: "Content",
			}, "/output/page.md")
		}

		assert.Equal(t, 5, c.Count())
	})
}

// TestMetadataCollector_GetIndex tests getting the index
func TestMetadataCollector_GetIndex(t *testing.T) {
	t.Run("returns index with documents", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir:   "./output",
			SourceURL: "https://example.com",
			Strategy:  "test",
			Enabled:   true,
		})

		doc := &domain.Document{
			URL:     "https://example.com/page1",
			Title:   "Test Page",
			Content: "Test content",
		}

		c.Add(doc, "/output/page1.md")

		index := c.GetIndex()
		assert.NotNil(t, index)
		assert.Equal(t, "https://example.com", index.SourceURL)
		assert.Equal(t, "test", index.Strategy)
		assert.Equal(t, 1, index.TotalDocuments)
		assert.Len(t, index.Documents, 1)
		assert.NotZero(t, index.GeneratedAt)
	})

	t.Run("returns empty index when no documents", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		index := c.GetIndex()
		assert.NotNil(t, index)
		assert.Equal(t, 0, index.TotalDocuments)
		assert.Empty(t, index.Documents)
		assert.NotZero(t, index.GeneratedAt)
	})
}

// TestMetadataCollector_IsEnabled tests checking if enabled
func TestMetadataCollector_IsEnabled(t *testing.T) {
	t.Run("returns true when enabled", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: true,
		})

		assert.True(t, c.IsEnabled())
	})

	t.Run("returns false when disabled", func(t *testing.T) {
		c := NewMetadataCollector(CollectorOptions{
			BaseDir: "./output",
			Enabled: false,
		})

		assert.False(t, c.IsEnabled())
	})
}

// TestMetadataCollector_SetStrategy tests setting strategy
func TestMetadataCollector_SetStrategy(t *testing.T) {
	c := NewMetadataCollector(CollectorOptions{
		BaseDir:  "./output",
		Strategy: "original",
	})

	assert.Equal(t, "original", c.strategy)

	c.SetStrategy("updated")
	assert.Equal(t, "updated", c.strategy)
}

// TestMetadataCollector_SetSourceURL tests setting source URL
func TestMetadataCollector_SetSourceURL(t *testing.T) {
	c := NewMetadataCollector(CollectorOptions{
		BaseDir:   "./output",
		SourceURL: "https://original.com",
	})

	assert.Equal(t, "https://original.com", c.sourceURL)

	c.SetSourceURL("https://updated.com")
	assert.Equal(t, "https://updated.com", c.sourceURL)
}

// TestMetadataCollector_ConcurrentAccess tests concurrent access safety
func TestMetadataCollector_ConcurrentAccess(t *testing.T) {
	c := NewMetadataCollector(CollectorOptions{
		BaseDir: "./output",
		Enabled: true,
	})

	done := make(chan bool)

	// Concurrent adds
	for i := 0; i < 50; i++ {
		go func() {
			doc := &domain.Document{
				URL:     "https://example.com/page",
				Title:   "Test",
				Content: "Content",
			}
			c.Add(doc, "/output/page.md")
			done <- true
		}()
	}

	// Concurrent counts
	for i := 0; i < 50; i++ {
		go func() {
			_ = c.Count()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have 50 documents and no panic
	assert.Equal(t, 50, c.Count())
}

// TestMetadataCollector_Integration tests integration scenarios
func TestMetadataCollector_Integration(t *testing.T) {
	t.Run("full workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		c := NewMetadataCollector(CollectorOptions{
			BaseDir:   tmpDir,
			Filename:  "metadata.json",
			SourceURL: "https://docs.example.com",
			Strategy:  "crawler",
			Enabled:   true,
		})

		docs := []*domain.Document{
			{
				URL:      "https://docs.example.com/guide/intro",
				Title:    "Introduction",
				Content:  "Welcome to the guide",
				Category: "guide",
				Tags:     []string{"beginner"},
			},
			{
				URL:      "https://docs.example.com/guide/advanced",
				Title:    "Advanced Topics",
				Content:  "Deep dive into features",
				Category: "guide",
				Tags:     []string{"advanced"},
			},
			{
				URL:      "https://docs.example.com/api/overview",
				Title:    "API Overview",
				Content:  "API reference",
				Category: "api",
				Tags:     []string{"reference"},
			},
		}

		for i, doc := range docs {
			filePath := filepath.Join(tmpDir, "docs", "page"+string(rune('1'+i))+".md")
			c.Add(doc, filePath)
		}

		assert.Equal(t, 3, c.Count())

		err := c.Flush()
		assert.NoError(t, err)

		// Verify file exists
		outputPath := filepath.Join(tmpDir, "metadata.json")
		data, err := os.ReadFile(outputPath)
		assert.NoError(t, err)

		var index domain.SimpleMetadataIndex
		err = json.Unmarshal(data, &index)
		assert.NoError(t, err)

		assert.Equal(t, "https://docs.example.com", index.SourceURL)
		assert.Equal(t, "crawler", index.Strategy)
		assert.Equal(t, 3, index.TotalDocuments)
		assert.Len(t, index.Documents, 3)

		// Verify all documents are present
		titles := make([]string, len(index.Documents))
		for i, doc := range index.Documents {
			titles[i] = doc.Title
		}
		assert.Contains(t, titles, "Introduction")
		assert.Contains(t, titles, "Advanced Topics")
		assert.Contains(t, titles, "API Overview")
	})
}
