package app_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
)

// TestDetectStrategy_EdgeCases tests edge cases for strategy detection
func TestDetectStrategy_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected app.StrategyType
	}{
		// URL with special characters and encoding
		{"URL with spaces encoded", "https://example.com/docs%20here", app.StrategyCrawler},
		{"URL with query params", "https://example.com/docs?foo=bar&baz=qux", app.StrategyCrawler},
		{"URL with fragment", "https://example.com/docs#section", app.StrategyCrawler},
		{"URL with port", "https://example.com:8080/docs", app.StrategyCrawler},
		{"URL with userinfo", "https://user:pass@example.com/docs", app.StrategyCrawler},

		// Git URLs with various formats
		{"Git with .git in query", "https://github.com/user/repo?query=.git", app.StrategyGit},
		{"Git SSH without .git", "git@github.com:user/repo", app.StrategyGit},
		{"Git SSH with .git", "git@github.com:user/repo.git", app.StrategyGit},
		{"Git with git:// protocol", "git://github.com/user/repo.git", app.StrategyUnknown},
		{"Git with subtree path", "https://github.com/user/repo/tree/main/docs", app.StrategyGit},

		// Sitemap variations
		{"Sitemap with query", "https://example.com/sitemap.xml?page=1", app.StrategySitemap},
		{"Sitemap index", "https://example.com/sitemap_index.xml", app.StrategySitemap},
		{"Sitemap with path and query", "https://example.com/sitemaps/sitemap.xml?ver=2", app.StrategySitemap},
		{"Gzipped sitemap with path", "https://example.com/sitemaps/sitemap.xml.gz", app.StrategySitemap},

		// llms.txt variations
		{"llms.txt with query", "https://example.com/llms.txt?v=1", app.StrategyLLMS},
		{"llms.txt with fragment", "https://example.com/llms.txt#readme", app.StrategyLLMS},
		{"LLMS.TXT uppercase", "https://example.com/LLMS.TXT", app.StrategyLLMS},
		{"llms.txt with trailing slash", "https://example.com/llms.txt/", app.StrategyCrawler},

		// pkg.go.dev variations
		{"pkg.go.dev with query", "https://pkg.go.dev/github.com/user/repo?tab=overview", app.StrategyPkgGo},
		{"pkg.go.dev with fragment", "https://pkg.go.dev/std#section", app.StrategyPkgGo},
		{"pkg.go.dev subpage", "https://pkg.go.dev/github.com/user/repo/tree/main", app.StrategyPkgGo},

		// Wiki variations
		{"GitHub Wiki home", "https://github.com/user/repo/wiki", app.StrategyWiki},
		{"GitHub Wiki page", "https://github.com/user/repo/wiki/Page-Name", app.StrategyWiki},
		{"GitHub Wiki with special chars", "https://github.com/user/repo/wiki/API_Reference_v2.0", app.StrategyWiki},
		{"GitHub Wiki with fragment", "https://github.com/user/repo/wiki/Home#section", app.StrategyWiki},

		// Documentation sites (should be crawler, not git)
		{"GitHub Pages subdomain", "https://project.github.io/docs", app.StrategyCrawler},
		{"GitHub Pages with path", "https://username.github.io/repo/docs", app.StrategyCrawler},
		{"docs.github.com with path", "https://docs.github.com/en/actions/learn-github-actions", app.StrategyCrawler},
		{"pages.github.io", "https://pages.github.io/features", app.StrategyCrawler},

		// Blob/tree views (crawler vs git)
		{"GitHub blob view", "https://github.com/user/repo/blob/develop/README.md", app.StrategyCrawler},
		{"GitHub tree view", "https://github.com/user/repo/tree/develop/docs", app.StrategyGit},
		{"GitLab blob view", "https://gitlab.com/user/repo/-/blob/develop/README.md", app.StrategyCrawler},
		{"GitLab tree view", "https://gitlab.com/user/repo/-/tree/develop/docs", app.StrategyGit},

		// Invalid/unknown protocols
		{"FTP protocol", "ftp://example.com", app.StrategyUnknown},
		{"File protocol", "file:///path/to/file", app.StrategyUnknown},
		{"Custom protocol", "custom://example.com", app.StrategyUnknown},
		{"No protocol", "example.com", app.StrategyUnknown},
		{"Just protocol", "https://", app.StrategyUnknown},

		// Empty and whitespace
		{"Empty string", "", app.StrategyUnknown},
		{"Whitespace only", "   ", app.StrategyUnknown},
		{"Newlines", "\n\n", app.StrategyUnknown},

		// Mixed case protocols
		{"HTTP uppercase", "HTTP://EXAMPLE.COM", app.StrategyCrawler},
		{"HTTPS mixed case", "HtTpS://ExAmPlE.cOm", app.StrategyCrawler},

		// International domains
		{"International domain", "https://例え.jp/llms.txt", app.StrategyLLMS},
		{"Punycode domain", "https://xn--fsq.jp/docs", app.StrategyCrawler},

		// Edge cases with .git in different positions
		{"Repository with git in name", "https://github.com/user/git-repo", app.StrategyGit},
		{"Repository with .git in middle", "https://github.com/user/repo.git.backup", app.StrategyGit},
		{"Repository with git suffix but not git", "https://github.com/user/mygit", app.StrategyGit},

		// Multiple segments
		{"Deep GitHub path without blob", "https://github.com/user/repo/main/docs/guide", app.StrategyGit},
		{"Deep path on regular site", "https://example.com/a/b/c/d/e/f", app.StrategyCrawler},

		// Bitbucket variations
		{"Bitbucket repo", "https://bitbucket.org/user/repo", app.StrategyGit},
		{"Bitbucket repo with .git", "https://bitbucket.org/user/repo.git", app.StrategyGit},
		{"Bitbucket wiki", "https://bitbucket.org/user/repo/wiki/Page", app.StrategyWiki},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

// TestDetectStrategy_PriorityOrder tests that strategy detection follows correct priority
func TestDetectStrategy_PriorityOrder(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected app.StrategyType
		reason   string
	}{
		{
			"llms.txt takes highest priority",
			"https://github.com/user/repo/llms.txt",
			app.StrategyLLMS,
			"Even though it's a GitHub URL, llms.txt should be detected first",
		},
		{
			"pkg.go.dev takes priority over git",
			"https://pkg.go.dev/github.com/user/repo",
			app.StrategyPkgGo,
			"pkg.go.dev contains github.com in path but should be pkggo",
		},
		{
			"Wiki detected before git",
			"https://github.com/user/repo/wiki",
			app.StrategyWiki,
			"Wiki URLs are also git repos but should be detected as wiki",
		},
		{
			"Sitemap takes priority over generic git",
			"https://github.com/user/repo/sitemap.xml",
			app.StrategySitemap,
			"Sitemap should be detected even on GitHub",
		},
		{
			"Git takes priority over crawler",
			"https://github.com/user/repo",
			app.StrategyGit,
			"Git repos should not fall through to crawler",
		},
		{
			"Crawler for docs.github.com",
			"https://docs.github.com/en/actions",
			app.StrategyCrawler,
			"Documentation subdomains should use crawler not git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result, tt.reason)
		})
	}
}

// TestCreateStrategy_AllTypes tests strategy creation for all types
func TestCreateStrategy_AllTypes(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	strategyTypes := []app.StrategyType{
		app.StrategyLLMS,
		app.StrategySitemap,
		app.StrategyWiki,
		app.StrategyGit,
		app.StrategyPkgGo,
		app.StrategyCrawler,
	}

	for _, strategyType := range strategyTypes {
		t.Run(string(strategyType), func(t *testing.T) {
			strategy := app.CreateStrategy(strategyType, deps)

			assert.NotNil(t, strategy, "Strategy should not be nil for type: %s", strategyType)
			assert.Equal(t, string(strategyType), strategy.Name(), "Strategy name should match type")
		})
	}
}

// TestCreateStrategy_UnknownType tests unknown strategy type
func TestCreateStrategy_UnknownType(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	unknownTypes := []app.StrategyType{
		app.StrategyUnknown,
		app.StrategyType(""),
		app.StrategyType("invalid"),
		app.StrategyType("not-real"),
	}

	for _, strategyType := range unknownTypes {
		t.Run(string(strategyType), func(t *testing.T) {
			strategy := app.CreateStrategy(strategyType, deps)
			assert.Nil(t, strategy, "Strategy should be nil for unknown type: %s", strategyType)
		})
	}
}

// TestCreateStrategy_NilDependencies tests strategy creation with nil dependencies
func TestCreateStrategy_NilDependencies(t *testing.T) {
	// This should not panic
	strategy := app.CreateStrategy(app.StrategyCrawler, nil)

	// Strategy should still be created but may not be fully functional
	assert.NotNil(t, strategy, "Strategy should be created even with nil deps")
	assert.Equal(t, "crawler", strategy.Name())
}

// TestGetAllStrategies_Ordering tests that GetAllStrategies returns strategies in expected order
func TestGetAllStrategies_Ordering(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	strategies := app.GetAllStrategies(deps)

	// Should have exactly 7 strategies
	assert.Len(t, strategies, 7, "Should have exactly 7 strategies")

	// Check expected order (priority order for detection)
	// Order must match DetectStrategy priority: llms > pkggo > docsrs > sitemap > wiki > git > crawler
	// pkggo must come before git because pkg.go.dev URLs contain github.com in the path
	expectedOrder := []string{"llms", "pkggo", "docsrs", "sitemap", "wiki", "git", "crawler"}
	actualNames := make([]string, len(strategies))

	for i, strategy := range strategies {
		assert.NotNil(t, strategy, "Strategy at index %d should not be nil", i)
		actualNames[i] = strategy.Name()
	}

	assert.Equal(t, expectedOrder, actualNames, "Strategies should be in priority order")
}

// TestGetAllStrategies_Properties tests properties of all returned strategies
func TestGetAllStrategies_Properties(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	strategies := app.GetAllStrategies(deps)

	// All strategies should have names
	for i, strategy := range strategies {
		t.Run(strategy.Name(), func(t *testing.T) {
			assert.NotEmpty(t, strategy.Name(), "Strategy at index %d should have a name", i)

			// All strategies should implement CanHandle
			// We don't test the actual behavior here, just that it doesn't panic
			strategy.CanHandle("https://example.com")
		})
	}
}

// TestFindMatchingStrategy_StrategyOrder tests that FindMatchingStrategy uses correct priority
func TestFindMatchingStrategy_StrategyOrder(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	tests := []struct {
		name             string
		url              string
		expectedStrategy string
	}{
		{"llms.txt file", "https://example.com/llms.txt", "llms"},
		{"pkg.go.dev URL", "https://pkg.go.dev/github.com/user/repo", "pkggo"},
		{"Sitemap", "https://example.com/sitemap.xml", "sitemap"},
		{"GitHub Wiki", "https://github.com/user/repo/wiki", "wiki"},
		{"GitHub repo", "https://github.com/user/repo", "git"},
		{"Regular website", "https://example.com/docs", "crawler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := app.FindMatchingStrategy(tt.url, deps)

			require.NotNil(t, strategy, "Should find a matching strategy for: %s", tt.url)
			assert.Equal(t, tt.expectedStrategy, strategy.Name(),
				"Should return correct strategy for URL: %s", tt.url)
		})
	}
}

// TestFindMatchingStrategy_NoMatch tests URLs that don't match any strategy
func TestFindMatchingStrategy_NoMatch(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	unmatchableURLs := []string{
		"ftp://example.com",
		"file:///path/to/file",
		"",
		"not-a-url",
		"custom://protocol",
	}

	for _, url := range unmatchableURLs {
		t.Run(url, func(t *testing.T) {
			strategy := app.FindMatchingStrategy(url, deps)
			assert.Nil(t, strategy, "Should not find a matching strategy for: %s", url)
		})
	}
}

// TestFindMatchingStrategy_FirstMatch tests that first matching strategy is returned
func TestFindMatchingStrategy_FirstMatch(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	// This URL could potentially match multiple strategies
	// but should return the first (highest priority) one
	url := "https://github.com/user/repo/wiki"

	strategy := app.FindMatchingStrategy(url, deps)

	require.NotNil(t, strategy)
	// Should be wiki, not git, because wiki is checked first in GetAllStrategies
	assert.Equal(t, "wiki", strategy.Name())
}

// TestStrategyType_String tests string representation of strategy types
func TestStrategyType_String(t *testing.T) {
	types := map[app.StrategyType]string{
		app.StrategyLLMS:    "llms",
		app.StrategySitemap: "sitemap",
		app.StrategyWiki:    "wiki",
		app.StrategyGit:     "git",
		app.StrategyPkgGo:   "pkggo",
		app.StrategyCrawler: "crawler",
		app.StrategyUnknown: "unknown",
	}

	for strategyType, expectedString := range types {
		t.Run(expectedString, func(t *testing.T) {
			actualString := string(strategyType)
			assert.Equal(t, expectedString, actualString)
		})
	}
}
