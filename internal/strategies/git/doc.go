// Package git implements documentation extraction from git repositories.
//
// The package supports GitHub, GitLab, Bitbucket, and generic HTTP(S) git URLs.
// It recognizes repository roots as well as hosted tree URLs that include a
// branch and subdirectory, then limits discovery to that repository-relative
// path when present.
//
// The extraction pipeline is split into small components:
//   - Strategy coordinates parsing, fetching, discovery, processing, and output.
//   - Parser normalizes repository URLs, detects hosting platforms, and extracts
//     branch/subpath information from tree URLs.
//   - ArchiveFetcher uses platform-specific tar.gz archive URLs for the fast
//     path and strips archive root directories during extraction.
//   - CloneFetcher falls back to a shallow go-git clone when archives fail or
//     cannot be used.
//   - Processor walks the fetched repository, ignores dependency/build
//     directories, turns Markdown/MDX and selected config files into
//     domain.Document values, and cooperates with sync state to skip unchanged
//     files.
//
// Strategy first tries archive download for non-SSH URLs, including default
// branch detection and a main-to-master fallback. If archive acquisition fails,
// it clones the repository and processes the local checkout through the same
// Processor path. Output writing is supplied by StrategyDependencies so the
// package stays independent from CLI orchestration details.
//
// Usage:
//
//	strategy := git.NewStrategy(deps)
//	if strategy.CanHandle(url) {
//	    err := strategy.Execute(ctx, url, opts)
//	}
package git
