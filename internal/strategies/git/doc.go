// Package git implements the git repository extraction strategy.
//
// It supports extracting documentation from GitHub, GitLab, and Bitbucket
// repositories using either archive download (faster) or git clone (fallback).
//
// Architecture:
//   - Strategy: Coordinator implementing strategies.Strategy interface
//   - Parser: URL parsing and platform detection
//   - ArchiveFetcher: HTTP-based tar.gz download and extraction
//   - CloneFetcher: go-git based repository cloning
//   - Processor: File discovery and document conversion
//
// Usage:
//
//	strategy := git.NewStrategy(deps)
//	if strategy.CanHandle(url) {
//	    err := strategy.Execute(ctx, url, opts)
//	}
package git
