package git

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/state"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type StrategyDependencies struct {
	Writer       *output.Writer
	Logger       *utils.Logger
	HTTPClient   *http.Client
	WriteFunc    func(ctx context.Context, doc *domain.Document) error
	StateManager *state.Manager
}

type Strategy struct {
	deps             *StrategyDependencies
	parser           *Parser
	archiveFetcher   *ArchiveFetcher
	cloneFetcher     *CloneFetcher
	processor        *Processor
	logger           *utils.Logger
	httpClient       *http.Client
	skipBranchDetect bool
}

func NewStrategy(deps *StrategyDependencies) *Strategy {
	var client *http.Client
	var skipBranchDetect bool

	if deps == nil {
		client = createDefaultHTTPClient()
		return &Strategy{
			httpClient: client,
			parser:     NewParser(),
		}
	}

	client = deps.HTTPClient
	if client == nil {
		client = createDefaultHTTPClient()
	} else {
		skipBranchDetect = true
	}

	logger := deps.Logger

	return &Strategy{
		deps:   deps,
		parser: NewParser(),
		archiveFetcher: NewArchiveFetcher(ArchiveFetcherOptions{
			HTTPClient: client,
			Logger:     logger,
		}),
		cloneFetcher: NewCloneFetcher(CloneFetcherOptions{
			Logger: logger,
		}),
		processor: NewProcessor(ProcessorOptions{
			Logger: logger,
		}),
		logger:           logger,
		httpClient:       client,
		skipBranchDetect: skipBranchDetect,
	}
}

func (s *Strategy) Name() string {
	return "git"
}

func (s *Strategy) CanHandle(url string) bool {
	lower := strings.ToLower(url)

	isDocsSubdomain := strings.Contains(lower, "docs.github.com") ||
		strings.Contains(lower, "pages.github.io") ||
		strings.Contains(lower, "github.io")

	if isDocsSubdomain {
		return false
	}

	if isWikiURL(url) {
		return false
	}

	return strings.HasPrefix(url, "git@") ||
		strings.HasSuffix(lower, ".git") ||
		(strings.Contains(lower, "github.com") && !strings.Contains(lower, "/blob/")) ||
		(strings.Contains(lower, "gitlab.com") && !strings.Contains(lower, "/-/blob/")) ||
		strings.Contains(lower, "bitbucket.org")
}

type ExecuteOptions struct {
	Output      string
	Concurrency int
	Limit       int
	DryRun      bool
	FilterURL   string
}

func (s *Strategy) Execute(ctx context.Context, rawURL string, opts ExecuteOptions) error {
	if s.logger != nil {
		s.logger.Info().Str("url", rawURL).Msg("Starting git extraction")
	}

	urlInfo, err := s.parser.ParseURLWithPath(rawURL)
	if err != nil {
		return fmt.Errorf("failed to parse git URL: %w", err)
	}

	filterPath := urlInfo.SubPath
	if filterPath == "" && opts.FilterURL != "" {
		filterPath = NormalizeFilterPath(opts.FilterURL)
	}

	if filterPath != "" && s.logger != nil {
		s.logger.Info().Str("filter_path", filterPath).Msg("Path filter active")
	}

	tmpDir, err := os.MkdirTemp("", "repodocs-git-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoURL := urlInfo.RepoURL
	branch, method, err := s.TryArchiveDownload(ctx, repoURL, tmpDir)
	if err != nil {
		if s.logger != nil {
			s.logger.Info().Err(err).Msg("Archive download failed, using git clone")
		}
		branch, err = s.CloneRepository(ctx, repoURL, tmpDir)
		if err != nil {
			return fmt.Errorf("failed to acquire repository: %w", err)
		}
		method = "clone"
	}

	if urlInfo.Branch != "" {
		branch = urlInfo.Branch
	}

	if s.logger != nil {
		s.logger.Info().
			Str("method", method).
			Str("branch", branch).
			Msg("Repository acquired successfully")
	}

	files, err := s.processor.FindDocumentationFiles(tmpDir, filterPath)
	if err != nil {
		return err
	}

	if len(files) == 0 && filterPath != "" {
		return fmt.Errorf("no documentation files found under path: %s", filterPath)
	}

	if s.logger != nil {
		s.logger.Info().Int("count", len(files)).Msg("Found documentation files")
	}

	if opts.Limit > 0 && len(files) > opts.Limit {
		files = files[:opts.Limit]
	}

	processOpts := ProcessOptions{
		RepoURL:      repoURL,
		Branch:       branch,
		FilterPath:   filterPath,
		Concurrency:  opts.Concurrency,
		Limit:        opts.Limit,
		DryRun:       opts.DryRun,
		WriteFunc:    s.deps.WriteFunc,
		StateManager: s.deps.StateManager,
	}

	return s.processor.ProcessFiles(ctx, files, tmpDir, processOpts)
}

func (s *Strategy) TryArchiveDownload(ctx context.Context, url, destDir string) (branch, method string, err error) {
	if strings.HasPrefix(url, "git@") {
		return "", "", fmt.Errorf("SSH URLs not supported for archive download")
	}

	info, err := s.parser.ParseURL(url)
	if err != nil {
		return "", "", err
	}

	if !s.skipBranchDetect {
		branch, err = DetectDefaultBranch(ctx, url)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn().Err(err).Msg("Failed to detect branch, using 'main'")
			}
			branch = "main"
		}
	} else {
		branch = "main"
	}

	result, err := s.archiveFetcher.Fetch(ctx, info, branch, destDir)
	if err != nil {
		if branch == "main" {
			if s.logger != nil {
				s.logger.Debug().Msg("Trying 'master' branch")
			}
			result, err = s.archiveFetcher.Fetch(ctx, info, "master", destDir)
			if err == nil {
				return "master", "archive", nil
			}
		}
		return "", "", err
	}

	return result.Branch, result.Method, nil
}

func (s *Strategy) CloneRepository(ctx context.Context, url, destDir string) (string, error) {
	info := &RepoInfo{URL: url}
	result, err := s.cloneFetcher.Fetch(ctx, info, "", destDir)
	if err != nil {
		return "", err
	}
	return result.Branch, nil
}

func isWikiURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "/wiki") ||
		strings.HasSuffix(lower, ".wiki.git")
}

func createDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}
