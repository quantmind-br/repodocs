package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/state"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type Processor struct {
	logger *utils.Logger
}

type ProcessorOptions struct {
	Logger *utils.Logger
}

func NewProcessor(opts ProcessorOptions) *Processor {
	return &Processor{logger: opts.Logger}
}

type ProcessOptions struct {
	RepoURL      string
	Branch       string
	FilterPath   string
	Concurrency  int
	Limit        int
	DryRun       bool
	MaxFileSize  int64
	WriteFunc    func(ctx context.Context, doc *domain.Document) error
	StateManager *state.Manager
}

func (p *Processor) FindDocumentationFiles(dir string, filterPath string) ([]string, error) {
	var files []string

	walkDir := dir
	if filterPath != "" {
		walkDir = filepath.Join(dir, filterPath)

		info, err := os.Stat(walkDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("filter path does not exist in repository: %s", filterPath)
			}
			return nil, fmt.Errorf("failed to access filter path: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("filter path is not a directory: %s", filterPath)
		}

		if p.logger != nil {
			p.logger.Debug().Str("filter_path", filterPath).Str("walk_dir", walkDir).Msg("Walking filtered directory")
		}
	}

	err := filepath.WalkDir(walkDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if IgnoreDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if DocumentExtensions[ext] {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func (p *Processor) ProcessFiles(ctx context.Context, files []string, tmpDir string, opts ProcessOptions) error {
	bar := utils.NewProgressBar(len(files), utils.DescExtracting)

	errors := utils.ParallelForEach(ctx, files, opts.Concurrency, func(ctx context.Context, file string) error {
		defer bar.Add(1)

		if err := p.ProcessFile(ctx, file, tmpDir, opts); err != nil {
			if p.logger != nil {
				p.logger.Warn().Err(err).Str("file", file).Msg("Failed to process file")
			}
		}
		return nil
	})

	if err := utils.FirstError(errors); err != nil {
		return err
	}

	if p.logger != nil {
		p.logger.Info().Msg("Git extraction completed")
	}
	return nil
}

func (p *Processor) ProcessFile(ctx context.Context, path, tmpDir string, opts ProcessOptions) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	maxSize := opts.MaxFileSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024
	}
	if maxSize > 0 && info.Size() > maxSize {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	relPath, _ := filepath.Rel(tmpDir, path)
	relPathURL := strings.ReplaceAll(relPath, "\\", "/")
	fileURL := opts.RepoURL + "/blob/" + opts.Branch + "/" + relPathURL

	contentHash := computeHash(content)

	doc := &domain.Document{
		URL:            fileURL,
		Title:          ExtractTitleFromPath(relPath),
		Content:        string(content),
		ContentHash:    contentHash,
		FetchedAt:      time.Now(),
		WordCount:      len(strings.Fields(string(content))),
		CharCount:      len(content),
		SourceStrategy: "git",
		RelativePath:   relPath,
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" && ext != ".mdx" {
		doc.Content = "```\n" + string(content) + "\n```"
	}

	if opts.StateManager != nil {
		opts.StateManager.MarkSeen(fileURL)
		if !opts.StateManager.ShouldProcess(fileURL, contentHash) {
			if p.logger != nil {
				p.logger.Debug().Str("file", relPath).Msg("Skipping unchanged file")
			}
			return nil
		}
	}

	if !opts.DryRun && opts.WriteFunc != nil {
		return opts.WriteFunc(ctx, doc)
	}

	return nil
}

func computeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func ExtractTitleFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}
