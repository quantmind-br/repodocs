package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

type MetadataCollector struct {
	mu        sync.RWMutex
	documents []*domain.SimpleDocumentMetadata
	sourceURL string
	strategy  string
	baseDir   string
	filename  string
	enabled   bool
}

type CollectorOptions struct {
	BaseDir   string
	Filename  string
	SourceURL string
	Strategy  string
	Enabled   bool
}

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

func (c *MetadataCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.documents)
}

func (c *MetadataCollector) GetIndex() *domain.SimpleMetadataIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buildIndex()
}

func (c *MetadataCollector) IsEnabled() bool {
	return c.enabled
}

func (c *MetadataCollector) SetStrategy(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.strategy = name
}

func (c *MetadataCollector) SetSourceURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sourceURL = url
}
