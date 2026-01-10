package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type CloneFetcher struct {
	logger *utils.Logger
}

type CloneFetcherOptions struct {
	Logger *utils.Logger
}

func NewCloneFetcher(opts CloneFetcherOptions) *CloneFetcher {
	return &CloneFetcher{logger: opts.Logger}
}

func (f *CloneFetcher) Name() string {
	return "clone"
}

func (f *CloneFetcher) Fetch(ctx context.Context, info *RepoInfo, branch, destDir string) (*FetchResult, error) {
	if f.logger != nil {
		f.logger.Info().Str("url", info.URL).Msg("Cloning repository")
	}

	cloneOpts := &git.CloneOptions{
		URL:      info.URL,
		Depth:    1,
		Progress: os.Stdout,
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cloneOpts.Auth = &githttp.BasicAuth{
			Username: "token",
			Password: token,
		}
	}

	repo, err := git.PlainCloneContext(ctx, destDir, false, cloneOpts)
	if err != nil {
		return nil, err
	}

	detectedBranch := branch
	head, err := repo.Head()
	if err == nil {
		refName := head.Name().String()
		if strings.HasPrefix(refName, "refs/heads/") {
			detectedBranch = strings.TrimPrefix(refName, "refs/heads/")
		}
	}

	if detectedBranch == "" {
		detectedBranch = "main"
	}

	return &FetchResult{
		LocalPath: destDir,
		Branch:    detectedBranch,
		Method:    "clone",
	}, nil
}

func DetectDefaultBranch(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--symref", url, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ref: refs/heads/") {
			parts := strings.Split(line, "\t")
			if len(parts) >= 1 {
				branch := strings.TrimPrefix(parts[0], "ref: refs/heads/")
				return branch, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}
