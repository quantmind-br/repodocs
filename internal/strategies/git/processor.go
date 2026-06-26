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

	"github.com/quantmind-br/repodocs/internal/converter"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/state"
	"github.com/quantmind-br/repodocs/internal/utils"
)

// Processor discovers documentation files in fetched repositories and converts them to documents.
type Processor struct {
	logger *utils.Logger
}

// ProcessorOptions configures a Processor.
type ProcessorOptions struct {
	Logger *utils.Logger
}

// NewProcessor creates a repository documentation processor.
func NewProcessor(opts ProcessorOptions) *Processor {
	return &Processor{logger: opts.Logger}
}

// ProcessOptions controls file processing and output for a fetched repository.
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
	Result       *domain.StrategyResult
}

// FindDocumentationFiles walks dir or filterPath and returns documentation and configuration files.
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
		if DocumentExtensions[ext] || ConfigExtensions[ext] {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ProcessFiles processes files concurrently and writes each resulting document through ProcessOptions.WriteFunc.
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

// ProcessFile converts one repository file into a domain document and writes it when enabled.
func (p *Processor) ProcessFile(ctx context.Context, path, tmpDir string, opts ProcessOptions) error {
	opts.Result.IncAttempted()

	info, err := os.Stat(path)
	if err != nil {
		opts.Result.IncFailed()
		return err
	}
	maxSize := opts.MaxFileSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024
	}
	if maxSize > 0 && info.Size() > maxSize {
		opts.Result.IncSkipped()
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		opts.Result.IncFailed()
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
	switch {
	case ConfigExtensions[ext]:
		doc.IsRawFile = true
	case ext == ".rst":
		md, convErr := converter.ConvertRST(content)
		if convErr != nil {
			if p.logger != nil {
				p.logger.Warn().Err(convErr).Str("file", relPath).Msg("RST conversion failed, falling back to raw")
			}
			doc.Content = "```\n" + string(content) + "\n```"
			doc.WordCount = len(strings.Fields(doc.Content))
			doc.CharCount = len(doc.Content)
		} else {
			doc.Content = string(md)
			doc.WordCount = len(strings.Fields(doc.Content))
			doc.CharCount = len(doc.Content)
		}
	case ext != ".md" && ext != ".mdx":
		doc.Content = "```\n" + string(content) + "\n```"
		doc.WordCount = len(strings.Fields(doc.Content))
		doc.CharCount = len(doc.Content)
	}

	if opts.StateManager != nil {
		opts.StateManager.MarkSeen(fileURL)
		if !opts.StateManager.ShouldProcess(fileURL, contentHash) {
			if p.logger != nil {
				p.logger.Debug().Str("file", relPath).Msg("Skipping unchanged file")
			}
			opts.Result.IncSkipped()
			return nil
		}
	}

	if !opts.DryRun && opts.WriteFunc != nil {
		if err := opts.WriteFunc(ctx, doc); err != nil {
			opts.Result.IncFailed()
			return err
		}
		opts.Result.IncWritten()
		opts.Result.AddBytesWritten(int64(len(doc.Content)))
	} else {
		// Dry-run (or no writer configured): the file was processed but not
		// written. Count it as skipped so URLsAttempted stays consistent with
		// the terminal counters (written + skipped + failed).
		opts.Result.IncSkipped()
	}

	return nil
}

func computeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// ExtractTitleFromPath creates a display title from a repository-relative file path.
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
