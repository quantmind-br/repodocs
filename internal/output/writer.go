package output

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// Writer handles writing documents to the filesystem
type Writer struct {
	baseDir      string
	flat         bool
	jsonMetadata bool
	force        bool
	dryRun       bool
}

// WriterOptions contains options for the writer
type WriterOptions struct {
	BaseDir      string
	Flat         bool
	JSONMetadata bool
	Force        bool
	DryRun       bool
}

// NewWriter creates a new output writer
func NewWriter(opts WriterOptions) *Writer {
	if opts.BaseDir == "" {
		opts.BaseDir = "./docs"
	}

	return &Writer{
		baseDir:      opts.BaseDir,
		flat:         opts.Flat,
		jsonMetadata: opts.JSONMetadata,
		force:        opts.Force,
		dryRun:       opts.DryRun,
	}
}

// Write saves a document to the output directory
func (w *Writer) Write(ctx context.Context, doc *domain.Document) error {
	// Generate path
	path := utils.GeneratePath(w.baseDir, doc.URL, w.flat)

	// Check if file exists
	if !w.force {
		if _, err := os.Stat(path); err == nil {
			// File exists, skip
			return nil
		}
	}

	// Dry run - just return
	if w.dryRun {
		return nil
	}

	// Ensure directory exists
	if err := utils.EnsureDir(path); err != nil {
		return err
	}

	// Add frontmatter
	content, err := converter.AddFrontmatter(doc.Content, doc)
	if err != nil {
		return err
	}

	// Write markdown file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	// Write JSON metadata if enabled
	if w.jsonMetadata {
		jsonPath := utils.JSONPath(path)
		if err := w.writeJSON(jsonPath, doc); err != nil {
			return err
		}
	}

	return nil
}

// writeJSON writes JSON metadata
func (w *Writer) writeJSON(path string, doc *domain.Document) error {
	metadata := doc.ToMetadata()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// WriteMultiple writes multiple documents
func (w *Writer) WriteMultiple(ctx context.Context, docs []*domain.Document) error {
	for _, doc := range docs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := w.Write(ctx, doc); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetPath returns the output path for a URL
func (w *Writer) GetPath(url string) string {
	return utils.GeneratePath(w.baseDir, url, w.flat)
}

// Exists checks if a document already exists
func (w *Writer) Exists(url string) bool {
	path := w.GetPath(url)
	_, err := os.Stat(path)
	return err == nil
}

// EnsureBaseDir creates the base directory if it doesn't exist
func (w *Writer) EnsureBaseDir() error {
	return os.MkdirAll(w.baseDir, 0755)
}

// Clean removes the output directory
func (w *Writer) Clean() error {
	return os.RemoveAll(w.baseDir)
}

// Stats returns statistics about the output directory
func (w *Writer) Stats() (int, int64, error) {
	var count int
	var size int64

	err := filepath.Walk(w.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			count++
			size += info.Size()
		}
		return nil
	})

	return count, size, err
}
