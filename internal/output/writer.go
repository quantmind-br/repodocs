package output

import (
	"context"
	"os"
	"path/filepath"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type Writer struct {
	baseDir      string
	flat         bool
	jsonMetadata bool
	force        bool
	dryRun       bool
	collector    *MetadataCollector
}

type WriterOptions struct {
	BaseDir      string
	Flat         bool
	JSONMetadata bool
	Force        bool
	DryRun       bool
	Collector    *MetadataCollector
}

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
		collector:    opts.Collector,
	}
}

// Write saves a document to the output directory
func (w *Writer) Write(ctx context.Context, doc *domain.Document) error {
	var path string
	if doc.IsRawFile && doc.RelativePath != "" {
		path = utils.GenerateRawPathFromRelative(w.baseDir, doc.RelativePath, w.flat)
	} else if doc.RelativePath != "" {
		path = utils.GeneratePathFromRelative(w.baseDir, doc.RelativePath, w.flat)
	} else {
		path = utils.GeneratePath(w.baseDir, doc.URL, w.flat)
	}

	if !w.force {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}

	if w.dryRun {
		return nil
	}

	if err := utils.EnsureDir(path); err != nil {
		return err
	}

	var content string
	if doc.IsRawFile {
		content = doc.Content
	} else {
		var err error
		content, err = converter.AddFrontmatter(doc.Content, doc)
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	if w.jsonMetadata && w.collector != nil {
		w.collector.Add(doc, path)
	}

	return nil
}

func (w *Writer) FlushMetadata() error {
	if w.collector != nil {
		return w.collector.Flush()
	}
	return nil
}

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
