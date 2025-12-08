package strategies

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// DocumentExtensions are file extensions to process
var DocumentExtensions = map[string]bool{
	".md":    true,
	".txt":   true,
	".rst":   true,
	".adoc":  true,
	".asciidoc": true,
}

// IgnoreDirs are directories to skip
var IgnoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
}

// GitStrategy extracts documentation from git repositories
type GitStrategy struct {
	writer *output.Writer
	logger *utils.Logger
}

// NewGitStrategy creates a new git strategy
func NewGitStrategy(deps *Dependencies) *GitStrategy {
	return &GitStrategy{
		writer: deps.Writer,
		logger: deps.Logger,
	}
}

// Name returns the strategy name
func (s *GitStrategy) Name() string {
	return "git"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *GitStrategy) CanHandle(url string) bool {
	return strings.HasPrefix(url, "git@") ||
		strings.HasSuffix(url, ".git") ||
		strings.Contains(url, "github.com") ||
		strings.Contains(url, "gitlab.com") ||
		strings.Contains(url, "bitbucket.org")
}

// Execute runs the git extraction strategy
func (s *GitStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Cloning repository")

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "repodocs-git-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Clone repository (shallow clone for speed)
	cloneOpts := &git.CloneOptions{
		URL:      url,
		Depth:    1, // Shallow clone
		Progress: os.Stdout,
	}

	// Use HTTPS auth if available (for private repos in future)
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: "token",
			Password: token,
		}
	}

	repo, err := git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
	if err != nil {
		return err
	}

	// Get default branch name
	defaultBranch := getDefaultBranch(repo)
	s.logger.Info().Str("branch", defaultBranch).Msg("Repository cloned, processing files")

	// Find all documentation files
	var files []string
	err = filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			name := d.Name()
			if IgnoreDirs[name] {
				return fs.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if DocumentExtensions[ext] {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.logger.Info().Int("count", len(files)).Msg("Found documentation files")

	// Apply limit
	if opts.Limit > 0 && len(files) > opts.Limit {
		files = files[:opts.Limit]
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
	)

	// Process files
	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		bar.Add(1)

		if err := s.processFile(ctx, file, tmpDir, url, defaultBranch, opts); err != nil {
			s.logger.Warn().Err(err).Str("file", file).Msg("Failed to process file")
		}
	}

	s.logger.Info().Msg("Git extraction completed")
	return nil
}

// processFile processes a single documentation file
func (s *GitStrategy) processFile(ctx context.Context, path, tmpDir, repoURL, branch string, opts Options) error {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Skip large files (> 10MB)
	if len(content) > 10*1024*1024 {
		return nil
	}

	// Get relative path for URL
	relPath, _ := filepath.Rel(tmpDir, path)
	// Convert Windows backslashes to forward slashes for URL
	relPathURL := strings.ReplaceAll(relPath, "\\", "/")
	fileURL := repoURL + "/blob/" + branch + "/" + relPathURL

	// Create document
	doc := &domain.Document{
		URL:            fileURL,
		Title:          extractTitleFromPath(relPath),
		Content:        string(content),
		FetchedAt:      time.Now(),
		WordCount:      len(strings.Fields(string(content))),
		CharCount:      len(content),
		SourceStrategy: s.Name(),
		RelativePath:   relPath, // Store relative path for output structure
	}

	// For markdown files, the content is already markdown
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" {
		// Content is already markdown
	} else {
		// For other formats, wrap in code block
		doc.Content = "```\n" + string(content) + "\n```"
	}

	// Write document
	if !opts.DryRun {
		return s.writer.Write(ctx, doc)
	}

	return nil
}

// extractTitleFromPath extracts a title from a file path
func extractTitleFromPath(path string) string {
	// Get filename without extension
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Convert common formats to title case
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}

// getDefaultBranch returns the default branch name from the cloned repository
func getDefaultBranch(repo *git.Repository) string {
	// Try to get HEAD reference
	head, err := repo.Head()
	if err == nil {
		// Extract branch name from refs/heads/branch-name
		refName := head.Name().String()
		if strings.HasPrefix(refName, "refs/heads/") {
			return strings.TrimPrefix(refName, "refs/heads/")
		}
	}

	// Fallback to "main" if we can't determine the branch
	return "main"
}
