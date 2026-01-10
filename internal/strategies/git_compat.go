package strategies

import (
	"context"
	"io"
	"net/http"

	"github.com/quantmind-br/repodocs-go/internal/strategies/git"
)

var DocumentExtensions = git.DocumentExtensions
var IgnoreDirs = git.IgnoreDirs

type repoInfo = git.RepoInfo

type gitURLInfo = git.GitURLInfo

type GitStrategy struct {
	strategy       *git.Strategy
	deps           *Dependencies
	parser         *git.Parser
	archiveFetcher *git.ArchiveFetcher
	processor      *git.Processor
	httpClient     *http.Client
}

func NewGitStrategy(deps *Dependencies) *GitStrategy {
	var gitDeps *git.StrategyDependencies
	var httpClient *http.Client

	if deps != nil {
		gitDeps = &git.StrategyDependencies{
			Writer:     deps.Writer,
			Logger:     deps.Logger,
			HTTPClient: deps.HTTPClient,
			WriteFunc:  deps.WriteDocument,
		}
		httpClient = deps.HTTPClient
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	var logger = deps.Logger
	if deps == nil {
		logger = nil
	}

	return &GitStrategy{
		strategy: git.NewStrategy(gitDeps),
		deps:     deps,
		parser:   git.NewParser(),
		archiveFetcher: git.NewArchiveFetcher(git.ArchiveFetcherOptions{
			HTTPClient: httpClient,
			Logger:     logger,
		}),
		processor: git.NewProcessor(git.ProcessorOptions{
			Logger: logger,
		}),
		httpClient: httpClient,
	}
}

func (s *GitStrategy) Name() string {
	return s.strategy.Name()
}

func (s *GitStrategy) CanHandle(url string) bool {
	return s.strategy.CanHandle(url)
}

func (s *GitStrategy) Execute(ctx context.Context, rawURL string, opts Options) error {
	gitOpts := git.ExecuteOptions{
		Output:      opts.Output,
		Concurrency: opts.Concurrency,
		Limit:       opts.Limit,
		DryRun:      opts.DryRun,
		FilterURL:   opts.FilterURL,
	}
	return s.strategy.Execute(ctx, rawURL, gitOpts)
}

func (s *GitStrategy) detectDefaultBranch(ctx context.Context, url string) (string, error) {
	return git.DetectDefaultBranch(ctx, url)
}

func (s *GitStrategy) buildArchiveURL(info *repoInfo, branch string) string {
	return s.archiveFetcher.BuildArchiveURL(info, branch)
}

func (s *GitStrategy) downloadAndExtract(ctx context.Context, archiveURL, destDir string) error {
	return s.archiveFetcher.DownloadAndExtract(ctx, archiveURL, destDir)
}

func (s *GitStrategy) extractTarGz(r io.Reader, destDir string) error {
	return s.archiveFetcher.ExtractTarGz(r, destDir)
}

func (s *GitStrategy) findDocumentationFiles(dir string, filterPath string) ([]string, error) {
	return s.processor.FindDocumentationFiles(dir, filterPath)
}

func (s *GitStrategy) processFiles(ctx context.Context, files []string, tmpDir, repoURL, branch string, opts Options) error {
	processOpts := git.ProcessOptions{
		RepoURL:     repoURL,
		Branch:      branch,
		Concurrency: opts.Concurrency,
		Limit:       opts.Limit,
		DryRun:      opts.DryRun,
		WriteFunc:   s.deps.WriteDocument,
	}
	return s.processor.ProcessFiles(ctx, files, tmpDir, processOpts)
}

func (s *GitStrategy) processFile(ctx context.Context, path, tmpDir, repoURL, branch string, opts Options) error {
	processOpts := git.ProcessOptions{
		RepoURL:   repoURL,
		Branch:    branch,
		DryRun:    opts.DryRun,
		WriteFunc: s.deps.WriteDocument,
	}
	return s.processor.ProcessFile(ctx, path, tmpDir, processOpts)
}

func (s *GitStrategy) parseGitURLWithPath(rawURL string) (*gitURLInfo, error) {
	return s.parser.ParseURLWithPath(rawURL)
}

func (s *GitStrategy) tryArchiveDownload(ctx context.Context, url, destDir string) (branch, method string, err error) {
	return s.strategy.TryArchiveDownload(ctx, url, destDir)
}

func (s *GitStrategy) cloneRepository(ctx context.Context, url, destDir string) (string, error) {
	return s.strategy.CloneRepository(ctx, url, destDir)
}

func normalizeFilterPath(path string) string {
	return git.NormalizeFilterPath(path)
}

func extractTitleFromPath(path string) string {
	return git.ExtractTitleFromPath(path)
}
