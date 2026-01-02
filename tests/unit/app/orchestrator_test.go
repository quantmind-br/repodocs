package app_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
)

type testStrategy struct {
	name       string
	canHandle  bool
	execErr    error
	execCalled bool
	lastOpts   strategies.Options
}

func (s *testStrategy) Name() string {
	return s.name
}

func (s *testStrategy) CanHandle(url string) bool {
	return s.canHandle
}

func (s *testStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	s.execCalled = true
	s.lastOpts = opts
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return s.execErr
}

func TestNewOrchestrator_Success(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Cache.Enabled = false // Disable cache to avoid BadgerDB lock issues in tests

	opts := app.OrchestratorOptions{
		Config:  cfg,
		Verbose: false,
	}

	// Act
	orchestrator, err := app.NewOrchestrator(opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, orchestrator)
	assert.NotNil(t, orchestrator.Close)
}

func TestNewOrchestrator_WithOptions(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Output.Directory = "/custom/output"
	cfg.Cache.Enabled = false

	opts := app.OrchestratorOptions{
		Config:          cfg,
		Verbose:         true,
		DryRun:          true,
		RenderJS:        true,
		Split:           true,
		Limit:           10,
		ExcludePatterns: []string{"test/*", "*.tmp"},
		ContentSelector: "#main-content",
	}

	// Act
	orchestrator, err := app.NewOrchestrator(opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, orchestrator)
}

func TestNewOrchestrator_InvalidConfig(t *testing.T) {
	// Arrange
	opts := app.OrchestratorOptions{
		Config: nil,
	}

	// Act
	orchestrator, err := app.NewOrchestrator(opts)

	// Assert
	require.Error(t, err)
	assert.Nil(t, orchestrator)
}

func TestOrchestrator_GetStrategyName(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Cache.Enabled = false // Disable cache to avoid BadgerDB lock issues in tests
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"HTTP URL", "https://example.com", "crawler"},
		{"GitHub URL", "https://github.com/user/repo", "git"},
		{"GitLab URL", "https://gitlab.com/user/repo", "git"},
		{"Sitemap URL", "https://example.com/sitemap.xml", "sitemap"},
		{"llms.txt URL", "https://example.com/llms.txt", "llms"},
		{"pkg.go.dev URL", "https://pkg.go.dev/github.com/example/package", "pkggo"},
	}

	// Act & Assert
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orchestrator.GetStrategyName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrchestrator_Run_NilStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return nil
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.Run(context.Background(), "https://example.com", opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create strategy")
}

func TestOrchestrator_Run_StrategyReceivesCorrectURL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	var capturedURL string
	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   nil,
	}

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return &urlCapturingStrategy{
				testStrategy: mockStrategy,
				capturedURL:  &capturedURL,
			}
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	targetURL := "https://example.com/docs/api"
	err = orchestrator.Run(context.Background(), targetURL, opts)

	require.NoError(t, err)
	assert.Equal(t, targetURL, capturedURL)
}

type urlCapturingStrategy struct {
	*testStrategy
	capturedURL *string
}

func (s *urlCapturingStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	*s.capturedURL = url
	return s.testStrategy.Execute(ctx, url, opts)
}

func TestOrchestrator_Run_StrategyTypeDetection(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	tests := []struct {
		name         string
		url          string
		expectedType app.StrategyType
	}{
		{"HTTP crawler", "https://example.com", app.StrategyCrawler},
		{"Sitemap", "https://example.com/sitemap.xml", app.StrategySitemap},
		{"GitHub repo", "https://github.com/user/repo", app.StrategyGit},
		{"LLMS file", "https://example.com/llms.txt", app.StrategyLLMS},
		{"pkg.go.dev", "https://pkg.go.dev/std", app.StrategyPkgGo},
		{"GitHub Wiki", "https://github.com/user/repo/wiki", app.StrategyWiki},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedType app.StrategyType
			mockStrategy := &testStrategy{name: "mock", canHandle: true}

			opts := app.OrchestratorOptions{
				Config: cfg,
				StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
					capturedType = st
					return mockStrategy
				},
			}

			orchestrator, err := app.NewOrchestrator(opts)
			require.NoError(t, err)
			defer orchestrator.Close()

			_ = orchestrator.Run(context.Background(), tt.url, opts)

			assert.Equal(t, tt.expectedType, capturedType)
		})
	}
}

func TestOrchestrator_ValidateURL(t *testing.T) {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		url      string
		wantErr  bool
		errorMsg string
	}{
		{"Valid HTTP URL", "https://example.com", false, ""},
		{"Valid Git URL", "https://github.com/user/repo", false, ""},
		{"Valid Sitemap URL", "https://example.com/sitemap.xml", false, ""},
		{"Invalid URL", "ftp://example.com", true, "unsupported URL format"},
		{"Empty URL", "", true, "unsupported URL format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orchestrator.ValidateURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrchestrator_Run_Success(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><h1>Hello</h1></body></html>`))
	}))
	defer server.Close()

	// Arrange
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 1

	opts := app.OrchestratorOptions{
		Config: cfg,
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act
	err = orchestrator.Run(context.Background(), server.URL, opts)

	// Assert
	require.NoError(t, err)

	// Verify output
	filename := utils.URLToFilename(server.URL)
	if !strings.HasSuffix(filename, ".md") {
		filename = filepath.Join(filename, "index.md")
	}
	assert.FileExists(t, filepath.Join(tmpDir, filename))
}

func TestOrchestrator_Run_UnknownStrategy(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Cache.Enabled = false // Disable cache to avoid BadgerDB lock issues in tests
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	err = orchestrator.Run(context.Background(), "ftp://invalid-url", app.OrchestratorOptions{})

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine strategy")
}

func TestOrchestrator_Run_ContextCancellation(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	opts := app.OrchestratorOptions{
		Config: cfg,
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	// Use a valid URL so it picks a strategy (Crawler)
	err = orchestrator.Run(ctx, "https://example.com", opts)

	// Assert
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestOrchestrator_Run_StrategyError(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	expectedErr := fmt.Errorf("mock strategy execution failed")
	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   expectedErr,
	}

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return mockStrategy
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	// Act
	err = orchestrator.Run(context.Background(), "https://example.com", opts)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strategy execution failed")
	assert.Contains(t, err.Error(), "mock strategy execution failed")
}

func TestOrchestrator_Close(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Cache.Enabled = false // Disable cache to avoid BadgerDB lock issues in tests
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	err = orchestrator.Close()

	// Assert
	assert.NoError(t, err)
}

func TestOrchestrator_Close_NilDeps(t *testing.T) {
	// Arrange
	cfg := config.Default()
	cfg.Cache.Enabled = false // Disable cache to avoid BadgerDB lock issues in tests
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act - Close once
	err = orchestrator.Close()

	// Assert
	assert.NoError(t, err)
}

func TestOrchestrator_Run_WithCustomOptions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 3
	cfg.Concurrency.MaxDepth = 5

	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   nil,
	}

	runOpts := app.OrchestratorOptions{
		Config:          cfg,
		Limit:           42,
		DryRun:          true,
		Verbose:         true,
		Force:           true,
		RenderJS:        true,
		Split:           true,
		IncludeAssets:   true,
		ContentSelector: "#content",
		ExcludeSelector: ".nav",
		ExcludePatterns: []string{"*.tmp"},
		FilterURL:       "/docs/",
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return mockStrategy
		},
	}

	orchestrator, err := app.NewOrchestrator(runOpts)
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.Run(context.Background(), "https://example.com", runOpts)

	require.NoError(t, err)
	assert.True(t, mockStrategy.execCalled, "Strategy.Execute should be called")
	assert.Equal(t, 42, mockStrategy.lastOpts.Limit)
	assert.True(t, mockStrategy.lastOpts.DryRun)
	assert.True(t, mockStrategy.lastOpts.Verbose)
	assert.True(t, mockStrategy.lastOpts.Force)
	assert.True(t, mockStrategy.lastOpts.RenderJS)
	assert.True(t, mockStrategy.lastOpts.Split)
	assert.True(t, mockStrategy.lastOpts.IncludeAssets)
	assert.Equal(t, "#content", mockStrategy.lastOpts.ContentSelector)
	assert.Equal(t, ".nav", mockStrategy.lastOpts.ExcludeSelector)
	assert.Contains(t, mockStrategy.lastOpts.Exclude, "*.tmp")
	assert.Equal(t, "/docs/", mockStrategy.lastOpts.FilterURL)
}

func TestDetectStrategy(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected app.StrategyType
	}{
		{"HTTP URL", "https://example.com", app.StrategyCrawler},
		{"GitHub URL", "https://github.com/user/repo", app.StrategyGit},
		{"GitLab URL", "https://gitlab.com/user/repo", app.StrategyGit},
		{"Bitbucket URL", "https://bitbucket.org/user/repo", app.StrategyGit},
		{"Git URL with .git", "https://github.com/user/repo.git", app.StrategyGit},
		{"SSH Git URL", "git@github.com:user/repo.git", app.StrategyGit},
		{"Sitemap XML", "https://example.com/sitemap.xml", app.StrategySitemap},
		{"Sitemap XML GZ", "https://example.com/sitemap.xml.gz", app.StrategySitemap},
		{"Sitemap with path", "https://example.com/sitemaps/sitemap.xml", app.StrategySitemap},
		{"llms.txt root", "https://example.com/llms.txt", app.StrategyLLMS},
		{"llms.txt with path", "https://example.com/docs/llms.txt", app.StrategyLLMS},
		{"pkg.go.dev URL", "https://pkg.go.dev/github.com/example/package", app.StrategyPkgGo},
		{"pkg.go.dev std", "https://pkg.go.dev/std", app.StrategyPkgGo},
		{"GitHub Wiki", "https://github.com/user/repo/wiki", app.StrategyWiki},
		{"GitHub Wiki page", "https://github.com/user/repo/wiki/Some-Page", app.StrategyWiki},
		{"Unknown URL", "ftp://example.com", app.StrategyUnknown},
		{"File URL", "file:///path/to/file", app.StrategyUnknown},
		{"Empty URL", "", app.StrategyUnknown},
		// GitHub documentation sites should use crawler, not git
		{"docs.github.com", "https://docs.github.com/en/get-started", app.StrategyCrawler},
		{"docs.github.com with locale", "https://docs.github.com/pt/copilot/concepts/agents/about-copilot-cli", app.StrategyCrawler},
		{"github.io pages", "https://user.github.io/project", app.StrategyCrawler},
		{"pages.github.io", "https://pages.github.io/docs", app.StrategyCrawler},
		{"GitHub blob view", "https://github.com/user/repo/blob/main/README.md", app.StrategyCrawler},
		{"GitHub tree view", "https://github.com/user/repo/tree/main/docs", app.StrategyGit},
		{"GitLab blob view", "https://gitlab.com/user/repo/-/blob/main/README.md", app.StrategyCrawler},
		{"GitLab tree view", "https://gitlab.com/user/repo/-/tree/main/docs", app.StrategyGit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectStrategy_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected app.StrategyType
	}{
		{"HTTPS uppercase", "HTTPS://EXAMPLE.COM", app.StrategyCrawler},
		{"GitHub uppercase", "HTTPS://GITHUB.COM/USER/REPO", app.StrategyGit},
		{"Sitemap uppercase", "HTTPS://EXAMPLE.COM/SITEMAP.XML", app.StrategySitemap},
		{"llms.txt uppercase", "HTTPS://EXAMPLE.COM/LLMS.TXT", app.StrategyLLMS},
		{"pkg.go.dev mixed case", "https://PKG.GO.DEV/github.com/example/package", app.StrategyPkgGo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.DetectStrategy(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateStrategy(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	tests := []struct {
		name         string
		strategyType app.StrategyType
		expectNil    bool
	}{
		{"LLMS Strategy", app.StrategyLLMS, false},
		{"Sitemap Strategy", app.StrategySitemap, false},
		{"Wiki Strategy", app.StrategyWiki, false},
		{"Git Strategy", app.StrategyGit, false},
		{"PkgGo Strategy", app.StrategyPkgGo, false},
		{"Crawler Strategy", app.StrategyCrawler, false},
		{"Unknown Strategy", app.StrategyUnknown, true},
		{"Invalid Strategy", app.StrategyType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := app.CreateStrategy(tt.strategyType, deps)

			if tt.expectNil {
				assert.Nil(t, strategy, "Expected nil strategy for %s", tt.strategyType)
			} else {
				assert.NotNil(t, strategy, "Expected non-nil strategy for %s", tt.strategyType)

				// Verify the strategy has the correct name
				expectedName := string(tt.strategyType)
				assert.Equal(t, expectedName, strategy.Name(), "Strategy name mismatch")
			}
		})
	}
}

func TestGetAllStrategies(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	strategies := app.GetAllStrategies(deps)

	// Should return all 6 strategy types
	assert.Len(t, strategies, 6, "Expected 6 strategies")

	// Verify all strategies are non-nil
	for i, strategy := range strategies {
		assert.NotNil(t, strategy, "Strategy at index %d should not be nil", i)
	}

	// Verify strategy names
	expectedNames := []string{"llms", "sitemap", "wiki", "git", "pkggo", "crawler"}
	actualNames := make([]string, len(strategies))
	for i, strategy := range strategies {
		actualNames[i] = strategy.Name()
	}

	assert.ElementsMatch(t, expectedNames, actualNames, "Strategy names don't match expected")
}

func TestFindMatchingStrategy(t *testing.T) {
	deps := testutil.NewMinimalDependencies(t)

	tests := []struct {
		name             string
		url              string
		expectedStrategy string // strategy name, or empty for nil
	}{
		{"LLMS txt file", "https://example.com/llms.txt", "llms"},
		{"pkg.go.dev URL", "https://pkg.go.dev/github.com/user/repo", "pkggo"},
		{"Sitemap XML", "https://example.com/sitemap.xml", "sitemap"},
		{"GitHub repo", "https://github.com/user/repo", "git"},
		{"GitHub Wiki", "https://github.com/user/repo/wiki", "wiki"},
		{"Regular website", "https://example.com/docs", "crawler"},
		{"docs.github.com", "https://docs.github.com/en/actions", "crawler"},
		{"Unknown protocol", "ftp://example.com", ""},
		{"Empty URL", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := app.FindMatchingStrategy(tt.url, deps)

			if tt.expectedStrategy == "" {
				assert.Nil(t, strategy, "Expected nil strategy for URL: %s", tt.url)
			} else {
				require.NotNil(t, strategy, "Expected non-nil strategy for URL: %s", tt.url)
				assert.Equal(t, tt.expectedStrategy, strategy.Name(),
					"Strategy name mismatch for URL: %s", tt.url)
			}
		})
	}
}
