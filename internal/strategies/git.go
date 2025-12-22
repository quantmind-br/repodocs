package strategies

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// DocumentExtensions are file extensions to process
var DocumentExtensions = map[string]bool{
	".md":       true,
	".txt":      true,
	".rst":      true,
	".adoc":     true,
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
// Uses archive download as primary method (faster) with git clone as fallback
type GitStrategy struct {
	writer     *output.Writer
	logger     *utils.Logger
	httpClient *http.Client
}

// NewGitStrategy creates a new git strategy
func NewGitStrategy(deps *Dependencies) *GitStrategy {
	return &GitStrategy{
		writer: deps.Writer,
		logger: deps.Logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// Name returns the strategy name
func (s *GitStrategy) Name() string {
	return "git"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *GitStrategy) CanHandle(url string) bool {
	lower := strings.ToLower(url)

	// Exclude known documentation/pages subdomains
	isDocsSubdomain := strings.Contains(lower, "docs.github.com") ||
		strings.Contains(lower, "pages.github.io") ||
		strings.Contains(lower, "github.io")

	if isDocsSubdomain {
		return false
	}

	// Check if it's a Git repository URL
	return strings.HasPrefix(url, "git@") ||
		strings.HasSuffix(lower, ".git") ||
		(strings.Contains(lower, "github.com") && !strings.Contains(lower, "/blob/") && !strings.Contains(lower, "/tree/")) ||
		(strings.Contains(lower, "gitlab.com") && !strings.Contains(lower, "/-/blob/") && !strings.Contains(lower, "/-/tree/")) ||
		strings.Contains(lower, "bitbucket.org")
}

// Execute runs the git extraction strategy
// It tries archive download first (faster), falls back to git clone if needed
func (s *GitStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Starting git extraction")

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "repodocs-git-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try archive download first (faster)
	branch, method, err := s.tryArchiveDownload(ctx, url, tmpDir)
	if err != nil {
		// Fallback to git clone
		s.logger.Info().Err(err).Msg("Archive download failed, using git clone")
		branch, err = s.cloneRepository(ctx, url, tmpDir)
		if err != nil {
			return fmt.Errorf("failed to acquire repository: %w", err)
		}
		method = "clone"
	}

	s.logger.Info().
		Str("method", method).
		Str("branch", branch).
		Msg("Repository acquired successfully")

	// Find documentation files
	files, err := s.findDocumentationFiles(tmpDir)
	if err != nil {
		return err
	}

	s.logger.Info().Int("count", len(files)).Msg("Found documentation files")

	// Apply limit
	if opts.Limit > 0 && len(files) > opts.Limit {
		files = files[:opts.Limit]
	}

	// Process files in parallel
	return s.processFiles(ctx, files, tmpDir, url, branch, opts)
}

// repoInfo contains parsed repository information
type repoInfo struct {
	platform string // github, gitlab, bitbucket
	owner    string
	repo     string
}

// tryArchiveDownload attempts to download and extract repository as archive
// Returns branch name, method used ("archive"), and error if failed
func (s *GitStrategy) tryArchiveDownload(ctx context.Context, url, destDir string) (branch, method string, err error) {
	// SSH URLs not supported for archive download
	if strings.HasPrefix(url, "git@") {
		return "", "", fmt.Errorf("SSH URLs not supported for archive download")
	}

	// Parse URL
	info, err := s.parseGitURL(url)
	if err != nil {
		return "", "", err
	}

	// Detect default branch
	branch, err = s.detectDefaultBranch(ctx, url)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to detect branch, using 'main'")
		branch = "main"
	}

	// Build archive URL
	archiveURL := s.buildArchiveURL(info, branch)
	s.logger.Debug().Str("archive_url", archiveURL).Msg("Downloading archive")

	// Download and extract
	if err := s.downloadAndExtract(ctx, archiveURL, destDir); err != nil {
		// If failed with 'main', try 'master'
		if branch == "main" {
			s.logger.Debug().Msg("Trying 'master' branch")
			archiveURL = s.buildArchiveURL(info, "master")
			if err2 := s.downloadAndExtract(ctx, archiveURL, destDir); err2 == nil {
				return "master", "archive", nil
			}
		}
		return "", "", err
	}

	return branch, "archive", nil
}

// parseGitURL extracts owner and repo from various git URL formats
func (s *GitStrategy) parseGitURL(url string) (*repoInfo, error) {
	patterns := []struct {
		platform string
		regex    *regexp.Regexp
	}{
		{"github", regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+)`)},
		{"gitlab", regexp.MustCompile(`gitlab\.com[:/]([^/]+)/([^/.]+)`)},
		{"bitbucket", regexp.MustCompile(`bitbucket\.org[:/]([^/]+)/([^/.]+)`)},
	}

	for _, p := range patterns {
		if matches := p.regex.FindStringSubmatch(url); len(matches) == 3 {
			return &repoInfo{
				platform: p.platform,
				owner:    matches[1],
				repo:     strings.TrimSuffix(matches[2], ".git"),
			}, nil
		}
	}

	return nil, fmt.Errorf("unsupported git URL format: %s", url)
}

// detectDefaultBranch uses git ls-remote to find the default branch
func (s *GitStrategy) detectDefaultBranch(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--symref", url, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	// Output format: "ref: refs/heads/master\tHEAD"
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ref: refs/heads/") {
			// Split by tab first, then extract branch from first part
			parts := strings.Split(line, "\t")
			if len(parts) >= 1 {
				// parts[0] = "ref: refs/heads/master"
				branch := strings.TrimPrefix(parts[0], "ref: refs/heads/")
				return branch, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// buildArchiveURL constructs the archive download URL for the platform
func (s *GitStrategy) buildArchiveURL(info *repoInfo, branch string) string {
	switch info.platform {
	case "github":
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			info.owner, info.repo, branch)
	case "gitlab":
		return fmt.Sprintf("https://gitlab.com/%s/%s/-/archive/%s/%s-%s.tar.gz",
			info.owner, info.repo, branch, info.repo, branch)
	case "bitbucket":
		return fmt.Sprintf("https://bitbucket.org/%s/%s/get/%s.tar.gz",
			info.owner, info.repo, branch)
	default:
		// Fallback to GitHub format
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz",
			info.owner, info.repo, branch)
	}
}

// downloadAndExtract downloads a tar.gz archive and extracts it
func (s *GitStrategy) downloadAndExtract(ctx context.Context, archiveURL, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", archiveURL, nil)
	if err != nil {
		return err
	}

	// Add authentication if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("archive not found (404)")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("authentication required (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	return s.extractTarGz(resp.Body, destDir)
}

// extractTarGz extracts a tar.gz archive to destDir
func (s *GitStrategy) extractTarGz(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader failed: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read failed: %w", err)
		}

		// Skip the root directory (GitHub adds repo-branch/ prefix)
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relativePath := parts[1]

		targetPath := filepath.Join(destDir, relativePath)

		// Security check: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}

			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file failed: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("copy failed: %w", err)
			}
			f.Close()
		}
	}

	return nil
}

// cloneRepository clones the repository using git (fallback method)
func (s *GitStrategy) cloneRepository(ctx context.Context, url, destDir string) (string, error) {
	s.logger.Info().Str("url", url).Msg("Cloning repository")

	cloneOpts := &git.CloneOptions{
		URL:      url,
		Depth:    1, // Shallow clone
		Progress: os.Stdout,
	}

	// Use HTTPS auth if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneOpts.Auth = &githttp.BasicAuth{
			Username: "token",
			Password: token,
		}
	}

	repo, err := git.PlainCloneContext(ctx, destDir, false, cloneOpts)
	if err != nil {
		return "", err
	}

	// Get default branch name
	head, err := repo.Head()
	if err == nil {
		refName := head.Name().String()
		if strings.HasPrefix(refName, "refs/heads/") {
			return strings.TrimPrefix(refName, "refs/heads/"), nil
		}
	}

	return "main", nil
}

// findDocumentationFiles walks the directory and finds all documentation files
func (s *GitStrategy) findDocumentationFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored directories
		if d.IsDir() {
			if IgnoreDirs[d.Name()] {
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

	return files, err
}

// processFiles processes all documentation files in parallel
func (s *GitStrategy) processFiles(ctx context.Context, files []string, tmpDir, repoURL, branch string, opts Options) error {
	// Create progress bar
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Processing"),
		progressbar.OptionShowCount(),
	)

	// Process files in parallel using existing infrastructure
	errors := utils.ParallelForEach(ctx, files, opts.Concurrency, func(ctx context.Context, file string) error {
		defer bar.Add(1)

		if err := s.processFile(ctx, file, tmpDir, repoURL, branch, opts); err != nil {
			s.logger.Warn().Err(err).Str("file", file).Msg("Failed to process file")
		}
		return nil
	})

	// Check for critical errors (context cancellation)
	if err := utils.FirstError(errors); err != nil {
		return err
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
		RelativePath:   relPath,
	}

	// For markdown files, the content is already markdown
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".md" {
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
