package git

import "context"

type RepoFetcher interface {
	Fetch(ctx context.Context, info *RepoInfo, branch, destDir string) (*FetchResult, error)
	Name() string
}
