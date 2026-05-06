package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// MetadataCollector aggregates document metadata and writes a JSON metadata index.
type MetadataCollector struct {
	mu        sync.RWMutex
	documents []*domain.SimpleDocumentMetadata
	sourceURL string
	strategy  string
	baseDir   string
	filename  string
	enabled   bool
}

// CollectorOptions configures metadata collection output, source context, and enablement.
type CollectorOptions struct {
	BaseDir   string
	Filename  string
	SourceURL string
	Strategy  string
	Enabled   bool
}

// NewMetadataCollector creates a metadata collector with the supplied options.
func NewMetadataCollector(opts CollectorOptions) *MetadataCollector {
	filename := opts.Filename
	if filename == "" {
		filename = "metadata.json"
	}
	return &MetadataCollector{
		documents: make([]*domain.SimpleDocumentMetadata, 0),
		sourceURL: opts.SourceURL,
		strategy:  opts.Strategy,
		baseDir:   opts.BaseDir,
		filename:  filename,
		enabled:   opts.Enabled,
	}
}

// Add records metadata for doc using filePath relative to the collector base directory.
func (c *MetadataCollector) Add(doc *domain.Document, filePath string) {
	if !c.enabled || doc == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	relPath, err := filepath.Rel(c.baseDir, filePath)
	if err != nil {
		relPath = filePath
	}
	relPath = filepath.ToSlash(relPath)

	metadata := doc.ToSimpleDocumentMetadata(relPath)
	// Use the collector's strategy as the source, overriding the document's SourceStrategy
	metadata.Source = c.strategy
	c.documents = append(c.documents, metadata)
}

// Flush writes the collected metadata index to disk when collection is enabled.
func (c *MetadataCollector) Flush() error {
	if !c.enabled || len(c.documents) == 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	index := c.buildIndex()

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	outputPath := filepath.Join(c.baseDir, c.filename)
	return os.WriteFile(outputPath, data, 0644)
}

func (c *MetadataCollector) buildIndex() *domain.SimpleMetadataIndex {
	docs := make([]domain.SimpleDocumentMetadata, len(c.documents))

	for i, doc := range c.documents {
		docs[i] = *doc
	}

	return &domain.SimpleMetadataIndex{
		GeneratedAt:    time.Now(),
		SourceURL:      c.sourceURL,
		Strategy:       c.strategy,
		TotalDocuments: len(c.documents),
		Documents:      docs,
	}
}

// Count returns the number of documents collected so far.
func (c *MetadataCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.documents)
}

// GetIndex returns an in-memory metadata index for the collected documents.
func (c *MetadataCollector) GetIndex() *domain.SimpleMetadataIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buildIndex()
}

// IsEnabled reports whether metadata collection is active.
func (c *MetadataCollector) IsEnabled() bool {
	return c.enabled
}

// SetStrategy updates the strategy name stored in future metadata indexes.
func (c *MetadataCollector) SetStrategy(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strategy = name
}

// SetSourceURL updates the source URL stored in future metadata indexes.
func (c *MetadataCollector) SetSourceURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sourceURL = url
}
