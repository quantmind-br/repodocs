package app

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectStrategy tests strategy detection based on URL patterns
func TestDetectStrategy(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected StrategyType
	}{
		// LLMS
		{"llms.txt direct", "https://example.com/llms.txt", StrategyLLMS},
		{"llms.txt with path", "https://example.com/docs/llms.txt", StrategyLLMS},
		{"llms.txt trailing slash", "https://example.com/llms.txt/", StrategyCrawler}, // Ends with /, not llms.txt
		{"llms.txt uppercase", "HTTPS://EXAMPLE.COM/LLMS.TXT", StrategyLLMS},
		{"llms.txt with params", "https://example.com/llms.txt?v=1", StrategyLLMS},

		// PkgGo
		{"pkg.go.dev", "https://pkg.go.dev/github.com/pkg/errors", StrategyPkgGo},
		{"pkg.go.dev uppercase", "HTTPS://PKG.GO.DEV/github.com/pkg/errors", StrategyPkgGo},

		// DocsRS
		{"docs.rs crate", "https://docs.rs/serde", StrategyDocsRS},
		{"docs.rs with version", "https://docs.rs/serde/1.0.0", StrategyDocsRS},
		{"docs.rs full path", "https://docs.rs/serde/1.0.0/serde/", StrategyDocsRS},
		{"docs.rs source view", "https://docs.rs/serde/1.0.0/src/serde/lib.rs", StrategyCrawler},

		// Sitemap
		{"sitemap.xml", "https://example.com/sitemap.xml", StrategySitemap},
		{"sitemap.xml.gz", "https://example.com/sitemap.xml.gz", StrategySitemap},
		{"sitemap with path", "https://example.com/sitemap_index.xml", StrategySitemap},

		// Wiki
		{"GitHub wiki", "https://github.com/owner/repo/wiki", StrategyWiki},
		{"GitHub wiki page", "https://github.com/owner/repo/wiki/Page", StrategyWiki},
		{"wiki clone URL", "https://github.com/owner/repo.wiki.git", StrategyWiki}, // .wiki.git matches wiki pattern

		// Git
		{"git@ URL", "git@github.com:owner/repo.git", StrategyGit},
		{"GitHub .git", "https://github.com/owner/repo.git", StrategyGit},
		{"GitHub non-blob", "https://github.com/owner/repo", StrategyGit},
		{"GitLab .git", "https://gitlab.com/owner/repo.git", StrategyGit},
		{"GitLab non-blob", "https://gitlab.com/owner/repo", StrategyGit},
		{"Bitbucket", "https://bitbucket.org/owner/repo", StrategyGit},

		// Crawler (HTTP URLs)
		{"HTTP URL", "http://example.com/docs", StrategyCrawler},
		{"HTTPS URL", "https://example.com/docs", StrategyCrawler},
		{"docs subdomain", "https://docs.example.com", StrategyCrawler},

		// Unknown (non-HTTP)
		{"local file", "file:///path/to/file", StrategyUnknown},
		{"ftp", "ftp://example.com", StrategyUnknown},
		{"empty", "", StrategyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetectStrategy_EdgeCases tests edge cases in strategy detection
func TestDetectStrategy_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected StrategyType
	}{
		{"GitHub blob is crawler", "https://github.com/owner/repo/blob/main/README.md", StrategyCrawler},
		{"GitLab blob is crawler", "https://gitlab.com/owner/repo/-/blob/main/README.md", StrategyCrawler},
		{"docs.github.com is crawler", "https://docs.github.com", StrategyCrawler},
		{"pages.github.io is crawler", "https://user.github.io", StrategyCrawler},
		{"github.io is crawler", "https://user.github.io/repo", StrategyCrawler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCreateStrategy tests strategy creation
func TestCreateStrategy(t *testing.T) {
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		DryRun:      true,
	})
	require.NoError(t, err)
	defer deps.Close()

	tests := []struct {
		name     string
		strategy StrategyType
	}{
		{"LLMS strategy", StrategyLLMS},
		{"Sitemap strategy", StrategySitemap},
		{"Wiki strategy", StrategyWiki},
		{"Git strategy", StrategyGit},
		{"PkgGo strategy", StrategyPkgGo},
		{"DocsRS strategy", StrategyDocsRS},
		{"Crawler strategy", StrategyCrawler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := CreateStrategy(tt.strategy, deps)
			assert.NotNil(t, strategy)
			assert.Equal(t, string(tt.strategy), strategy.Name())
		})
	}

	t.Run("Unknown strategy", func(t *testing.T) {
		strategy := CreateStrategy(StrategyUnknown, deps)
		assert.Nil(t, strategy)
	})
}

// TestGetAllStrategies tests getting all strategies
func TestGetAllStrategies(t *testing.T) {
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		DryRun:      true,
	})
	require.NoError(t, err)
	defer deps.Close()

	strategies := GetAllStrategies(deps)
	assert.Len(t, strategies, 7)

	names := make(map[string]bool)
	for _, s := range strategies {
		names[s.Name()] = true
	}

	assert.True(t, names["llms"])
	assert.True(t, names["pkggo"])
	assert.True(t, names["docsrs"])
	assert.True(t, names["sitemap"])
	assert.True(t, names["wiki"])
	assert.True(t, names["git"])
	assert.True(t, names["crawler"])
}

// TestFindMatchingStrategy tests finding a matching strategy for a URL
func TestFindMatchingStrategy(t *testing.T) {
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		DryRun:      true,
	})
	require.NoError(t, err)
	defer deps.Close()

	tests := []struct {
		name             string
		url              string
		expectedStrategy string
	}{
		{"llms.txt URL", "https://example.com/llms.txt", "llms"},
		{"sitemap URL", "https://example.com/sitemap.xml", "sitemap"},
		{"GitHub URL", "https://github.com/owner/repo", "git"},
		{"wiki URL", "https://github.com/owner/repo/wiki", "wiki"},
		{"pkg.go.dev URL", "https://pkg.go.dev/github.com/pkg/errors", "pkggo"},
		{"regular URL", "https://example.com/docs", "crawler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := FindMatchingStrategy(tt.url, deps)
			assert.NotNil(t, strategy)
			assert.Equal(t, tt.expectedStrategy, strategy.Name())
		})
	}

	t.Run("unknown URL", func(t *testing.T) {
		strategy := FindMatchingStrategy("ftp://example.com", deps)
		assert.Nil(t, strategy)
	})
}

// TestNewOrchestrator tests creating a new orchestrator
func TestNewOrchestrator(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "pretty",
		},
	}

	t.Run("valid config", func(t *testing.T) {
		orch, err := NewOrchestrator(OrchestratorOptions{
			Config: cfg,
		})
		require.NoError(t, err)
		assert.NotNil(t, orch)
		assert.NotNil(t, orch.deps)
		assert.NotNil(t, orch.logger)
		assert.NotNil(t, orch.strategyFactory)
	})

	t.Run("nil config", func(t *testing.T) {
		orch, err := NewOrchestrator(OrchestratorOptions{
			Config: nil,
		})
		assert.Error(t, err)
		assert.Nil(t, orch)
	})
}

// TestNewOrchestrator_CustomStrategyFactory tests custom strategy factory injection
func TestNewOrchestrator_CustomStrategyFactory(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "pretty",
		},
	}

	customFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		// Return a mock strategy for testing
		return &mockStrategy{name: "custom_" + string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config:          cfg,
		StrategyFactory: customFactory,
	})
	require.NoError(t, err)
	assert.NotNil(t, orch)

	// Verify custom factory is used by checking strategy name
	strategy := orch.strategyFactory(StrategyCrawler, nil)
	assert.Equal(t, "custom_crawler", strategy.Name())
}

// TestOrchestrator_GetStrategyName tests getting strategy name for a URL
func TestOrchestrator_GetStrategyName(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level: "error",
		},
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)
	defer orch.Close()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"llms.txt", "https://example.com/llms.txt", "llms"},
		{"sitemap", "https://example.com/sitemap.xml", "sitemap"},
		{"wiki", "https://github.com/owner/repo/wiki", "wiki"},
		{"git", "https://github.com/owner/repo", "git"},
		{"crawler", "https://example.com/docs", "crawler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orch.GetStrategyName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestOrchestrator_ValidateURL tests URL validation
func TestOrchestrator_ValidateURL(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level: "error",
		},
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)
	defer orch.Close()

	t.Run("valid URL", func(t *testing.T) {
		err := orch.ValidateURL("https://example.com/docs")
		assert.NoError(t, err)
	})

	t.Run("unknown URL format", func(t *testing.T) {
		err := orch.ValidateURL("ftp://example.com")
		assert.Error(t, err)
	})
}

// TestOrchestrator_Close tests closing the orchestrator
func TestOrchestrator_Close(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: "/tmp",
		},
		Logging: config.LoggingConfig{
			Level: "error",
		},
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	err = orch.Close()
	assert.NoError(t, err)
}

// Mock strategy for testing
type mockStrategy struct {
	name string
}

func (m *mockStrategy) Name() string {
	return m.name
}

func (m *mockStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	return nil
}
