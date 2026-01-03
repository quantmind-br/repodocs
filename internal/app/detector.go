package app

import (
	"net/url"
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/strategies"
)

// StrategyType represents the type of extraction strategy
type StrategyType string

const (
	StrategyLLMS        StrategyType = "llms"
	StrategyPkgGo       StrategyType = "pkggo"
	StrategyDocsRS      StrategyType = "docsrs"
	StrategySitemap     StrategyType = "sitemap"
	StrategyWiki        StrategyType = "wiki"
	StrategyGitHubPages StrategyType = "github_pages"
	StrategyGit         StrategyType = "git"
	StrategyCrawler     StrategyType = "crawler"
	StrategyUnknown     StrategyType = "unknown"
)

// DetectStrategy determines the appropriate strategy based on URL patterns
func DetectStrategy(rawURL string) StrategyType {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return StrategyUnknown
	}

	lower := strings.ToLower(rawURL)

	// Check for SSH Git URLs first (git@host:path/repo.git)
	// These don't parse with url.Parse, so handle them before parsing
	if strings.HasPrefix(rawURL, "git@") || strings.HasPrefix(rawURL, "git+ssh://") {
		return StrategyGit
	}

	// Parse URL to strip query and fragment for path-based matching
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If URL parsing fails, do basic checks on the raw string
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			return StrategyCrawler
		}
		return StrategyUnknown
	}

	// Check if the URL has a valid host (for cases like "https://")
	if parsed.Host == "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return StrategyUnknown
	}

	// For path-based matching, use the path without query/fragment
	path := parsed.Path
	lowerPath := strings.ToLower(path)

	// Check for git:// protocol (unsupported)
	if parsed.Scheme == "git" {
		return StrategyUnknown
	}

	// Check for llms.txt first (using path without query/fragment)
	if strings.HasSuffix(lowerPath, "/llms.txt") || strings.HasSuffix(lowerPath, "llms.txt") {
		return StrategyLLMS
	}

	if strings.Contains(lower, "pkg.go.dev") {
		return StrategyPkgGo
	}

	if strings.Contains(lower, "docs.rs") {
		if !strings.Contains(lowerPath, "/src/") && !strings.Contains(lowerPath, "/source/") {
			return StrategyDocsRS
		}
	}

	if strings.HasSuffix(lowerPath, "sitemap.xml") ||
		strings.HasSuffix(lowerPath, "sitemap.xml.gz") ||
		strings.Contains(lowerPath, "sitemap") && strings.HasSuffix(lowerPath, ".xml") {
		return StrategySitemap
	}

	// Check for Wiki (before generic Git) - pass raw URL to support all wiki patterns
	if strategies.IsWikiURL(rawURL) {
		return StrategyWiki
	}

	// Check for GitHub Pages (*.github.io) - after Wiki, before Git
	if strategies.IsGitHubPagesURL(rawURL) {
		return StrategyGitHubPages
	}

	// Check for Git repository
	// Exclude known documentation/pages subdomains
	isDocsSubdomain := strings.Contains(lower, "docs.github.com") ||
		strings.Contains(lower, "pages.github.io") ||
		strings.Contains(lower, "github.io")

	if !isDocsSubdomain && (strings.HasSuffix(lowerPath, ".git") ||
		(strings.Contains(lower, "github.com") && !strings.Contains(lowerPath, "/blob/")) ||
		(strings.Contains(lower, "gitlab.com") && !strings.Contains(lowerPath, "/-/blob/")) ||
		strings.Contains(lower, "bitbucket.org")) {
		return StrategyGit
	}

	// Default to crawler for HTTP URLs
	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		return StrategyCrawler
	}

	return StrategyUnknown
}

func CreateStrategy(strategyType StrategyType, deps *strategies.Dependencies) strategies.Strategy {
	switch strategyType {
	case StrategyLLMS:
		return strategies.NewLLMSStrategy(deps)
	case StrategyPkgGo:
		return strategies.NewPkgGoStrategy(deps)
	case StrategyDocsRS:
		return strategies.NewDocsRSStrategy(deps)
	case StrategySitemap:
		return strategies.NewSitemapStrategy(deps)
	case StrategyWiki:
		return strategies.NewWikiStrategy(deps)
	case StrategyGitHubPages:
		return strategies.NewGitHubPagesStrategy(deps)
	case StrategyGit:
		return strategies.NewGitStrategy(deps)
	case StrategyCrawler:
		return strategies.NewCrawlerStrategy(deps)
	default:
		return nil
	}
}

func GetAllStrategies(deps *strategies.Dependencies) []strategies.Strategy {
	return []strategies.Strategy{
		strategies.NewLLMSStrategy(deps),
		strategies.NewPkgGoStrategy(deps),
		strategies.NewDocsRSStrategy(deps),
		strategies.NewSitemapStrategy(deps),
		strategies.NewWikiStrategy(deps),
		strategies.NewGitHubPagesStrategy(deps),
		strategies.NewGitStrategy(deps),
		strategies.NewCrawlerStrategy(deps),
	}
}

func FindMatchingStrategy(url string, deps *strategies.Dependencies) strategies.Strategy {
	for _, strategy := range GetAllStrategies(deps) {
		if strategy.CanHandle(url) {
			return strategy
		}
	}
	return nil
}
