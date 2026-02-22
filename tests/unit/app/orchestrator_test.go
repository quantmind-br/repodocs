package app_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
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
	execFunc   func(ctx context.Context, url string, opts strategies.Options) error
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
	if s.execFunc != nil {
		return s.execFunc(ctx, url, opts)
	}
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
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			Verbose: false,
		},
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
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			Verbose:  true,
			DryRun:   true,
			RenderJS: true,
			Limit:    10,
		},
		Split:           true,
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
	cfg.Output.Flat = true

	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   nil,
	}

	runOpts := app.OrchestratorOptions{
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			Limit:    42,
			DryRun:   true,
			Verbose:  true,
			Force:    true,
			RenderJS: true,
		},
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
	assert.True(t, mockStrategy.lastOpts.NoFolders) // From cfg.Output.Flat
}

func TestOrchestrator_ValidateURL_EdgeCases(t *testing.T) {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"Valid HTTP", "http://example.com", false},
		{"Valid HTTPS", "https://example.com", false},
		{"Valid GitHub", "https://github.com/user/repo", false},
		{"Valid GitLab", "https://gitlab.com/user/repo", false},
		{"Valid Sitemap", "https://example.com/sitemap.xml", false},
		{"Valid llms.txt", "https://example.com/llms.txt", false},
		{"Valid pkg.go.dev", "https://pkg.go.dev/std", false},
		{"Valid Wiki", "https://github.com/user/repo/wiki", false},
		{"FTP protocol", "ftp://example.com", true},
		{"File protocol", "file:///path/to/file", true},
		{"Empty string", "", true},
		{"Whitespace", "   ", true},
		{"No protocol", "example.com", true},
		{"Invalid chars", "://example.com", true},
		{"Just protocol", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orchestrator.ValidateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported URL format")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrchestrator_Close_Idempotent(t *testing.T) {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Close multiple times - should not panic
	err = orchestrator.Close()
	assert.NoError(t, err)

	err = orchestrator.Close()
	assert.NoError(t, err)

	err = orchestrator.Close()
	assert.NoError(t, err)
}

func TestOrchestrator_GetStrategyName_AllTypes(t *testing.T) {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/llms.txt", "llms"},
		{"https://pkg.go.dev/std", "pkggo"},
		{"https://example.com/sitemap.xml", "sitemap"},
		{"https://github.com/user/repo/wiki", "wiki"},
		{"https://github.com/user/repo", "git"},
		{"https://example.com", "crawler"},
		{"ftp://example.com", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := orchestrator.GetStrategyName(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOrchestrator_Run_ContextTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	// Create a strategy that never completes
	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   nil,
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

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give it a moment to actually timeout
	time.Sleep(10 * time.Millisecond)

	err = orchestrator.Run(ctx, "https://example.com", opts)

	// Should fail with context deadline exceeded or canceled
	assert.Error(t, err)
}

func TestOrchestrator_Run_MultipleURLs(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	// Create a mock strategy
	mockStrategy := &testStrategy{
		name:      "mock",
		canHandle: true,
		execErr:   nil,
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

	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	for _, url := range urls {
		err := orchestrator.Run(context.Background(), url, opts)
		assert.NoError(t, err)
	}
}

func TestOrchestrator_DependsClose(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Directory = t.TempDir() // Use isolated temp dir to avoid lock conflicts
	cfg.Cache.Enabled = true          // Enable cache to ensure it needs closing

	opts := app.OrchestratorOptions{
		Config: cfg,
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)

	// Close should not error
	err = orchestrator.Close()
	assert.NoError(t, err)
}

func TestOrchestrator_StrategyFactoryDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	// Don't provide a strategy factory
	opts := app.OrchestratorOptions{
		Config: cfg,
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	// Run should still work with default factory
	err = orchestrator.Run(context.Background(), "https://example.com", opts)

	// We don't care if it succeeds or fails (might fail without HTTP server)
	// Just verify it doesn't panic and uses some strategy
	_ = err
}

func TestOrchestrator_Run_SitemapDiscoveryViaRobotsTxt(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(200)
			fmt.Fprintf(w, "User-agent: *\nDisallow: /private/\nSitemap: %s/sitemap.xml\n", server.URL)
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(200)
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>%s/page1</loc></url>
</urlset>`, server.URL)
		case "/page1":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(`<html><head><title>Page 1</title></head><body><h1>Page 1</h1><p>Content here.</p></body></html>`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	var capturedType app.StrategyType
	mockStrategy := &testStrategy{name: "sitemap", canHandle: true}

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

	err = orchestrator.Run(context.Background(), server.URL, opts)

	require.NoError(t, err)
	assert.Equal(t, app.StrategySitemap, capturedType,
		"Should switch from crawler to sitemap after discovering sitemap via robots.txt")
}

func TestOrchestrator_Run_ContentBasedSitemapDetection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/data.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(200)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>https://example.com/page1</loc></url>
	<url><loc>https://example.com/page2</loc></url>
</urlset>`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	var capturedType app.StrategyType
	mockStrategy := &testStrategy{name: "sitemap", canHandle: true}

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

	err = orchestrator.Run(context.Background(), server.URL+"/data.xml", opts)

	require.NoError(t, err)
	assert.Equal(t, app.StrategySitemap, capturedType,
		"Should detect sitemap XML content and switch from crawler to sitemap strategy")
}

func TestOrchestrator_Run_NoSitemapFallbackToCrawler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(`<html><body><h1>Home</h1></body></html>`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	var capturedType app.StrategyType
	mockStrategy := &testStrategy{name: "crawler", canHandle: true}

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

	err = orchestrator.Run(context.Background(), server.URL, opts)

	require.NoError(t, err)
	assert.Equal(t, app.StrategyCrawler, capturedType,
		"Should remain crawler when no sitemap is discovered")
}
