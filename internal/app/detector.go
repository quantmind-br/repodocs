package app

import (
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/strategies"
)

// StrategyType represents the type of extraction strategy
type StrategyType string

const (
	StrategyLLMS    StrategyType = "llms"
	StrategySitemap StrategyType = "sitemap"
	StrategyGit     StrategyType = "git"
	StrategyPkgGo   StrategyType = "pkggo"
	StrategyCrawler StrategyType = "crawler"
	StrategyUnknown StrategyType = "unknown"
)

// DetectStrategy determines the appropriate strategy based on URL patterns
func DetectStrategy(url string) StrategyType {
	lower := strings.ToLower(url)

	// Check for llms.txt first
	if strings.HasSuffix(lower, "/llms.txt") || strings.HasSuffix(lower, "llms.txt") {
		return StrategyLLMS
	}

	// Check for pkg.go.dev (before Git, since pkg.go.dev URLs contain github.com paths)
	if strings.Contains(lower, "pkg.go.dev") {
		return StrategyPkgGo
	}

	// Check for sitemap
	if strings.HasSuffix(lower, "sitemap.xml") ||
		strings.HasSuffix(lower, "sitemap.xml.gz") ||
		strings.Contains(lower, "sitemap") && strings.HasSuffix(lower, ".xml") {
		return StrategySitemap
	}

	// Check for Git repository
	// Exclude known documentation/pages subdomains
	isDocsSubdomain := strings.Contains(lower, "docs.github.com") ||
		strings.Contains(lower, "pages.github.io") ||
		strings.Contains(lower, "github.io")

	if !isDocsSubdomain && (strings.HasPrefix(url, "git@") ||
		strings.HasSuffix(lower, ".git") ||
		(strings.Contains(lower, "github.com") && !strings.Contains(lower, "/blob/") && !strings.Contains(lower, "/tree/")) ||
		(strings.Contains(lower, "gitlab.com") && !strings.Contains(lower, "/-/blob/") && !strings.Contains(lower, "/-/tree/")) ||
		strings.Contains(lower, "bitbucket.org")) {
		return StrategyGit
	}

	// Default to crawler for HTTP URLs
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return StrategyCrawler
	}

	return StrategyUnknown
}

// CreateStrategy creates the appropriate strategy based on detected type
func CreateStrategy(strategyType StrategyType, deps *strategies.Dependencies) strategies.Strategy {
	switch strategyType {
	case StrategyLLMS:
		return strategies.NewLLMSStrategy(deps)
	case StrategySitemap:
		return strategies.NewSitemapStrategy(deps)
	case StrategyGit:
		return strategies.NewGitStrategy(deps)
	case StrategyPkgGo:
		return strategies.NewPkgGoStrategy(deps)
	case StrategyCrawler:
		return strategies.NewCrawlerStrategy(deps)
	default:
		return nil
	}
}

// GetAllStrategies returns all available strategies
// Strategies are ordered from most specific to least specific
func GetAllStrategies(deps *strategies.Dependencies) []strategies.Strategy {
	return []strategies.Strategy{
		strategies.NewLLMSStrategy(deps),     // Most specific: /llms.txt
		strategies.NewPkgGoStrategy(deps),    // Specific: pkg.go.dev (must come before Git!)
		strategies.NewSitemapStrategy(deps),  // Specific: sitemap.xml
		strategies.NewGitStrategy(deps),      // General: github.com, gitlab.com, etc
		strategies.NewCrawlerStrategy(deps),  // Catch-all: any HTTP URL
	}
}

// FindMatchingStrategy finds the first strategy that can handle the URL
func FindMatchingStrategy(url string, deps *strategies.Dependencies) strategies.Strategy {
	for _, strategy := range GetAllStrategies(deps) {
		if strategy.CanHandle(url) {
			return strategy
		}
	}
	return nil
}
