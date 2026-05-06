package git

import "context"

// RepoFetcher retrieves a repository into a local directory using a named method.
type RepoFetcher interface {
	// Fetch retrieves info at branch into destDir and returns details about the local checkout.
	Fetch(ctx context.Context, info *RepoInfo, branch, destDir string) (*FetchResult, error)
	// Name identifies the fetch method for logs and FetchResult.Method.
	Name() string
}
