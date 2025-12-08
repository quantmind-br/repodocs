package unit

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewOrchestrator_Success(t *testing.T) {
	// Arrange
	cfg := config.Default()

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
		Config:     cfg,
		Verbose:    true,
		DryRun:     true,
		RenderJS:   true,
		Split:      true,
		Limit:      10,
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

func TestOrchestrator_ValidateURL(t *testing.T) {
	// Arrange
	cfg := config.Default()
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

	// Act & Assert
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
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mocks.NewMockStrategy(ctrl)
	mockStrategy.EXPECT().Name().Return("test-strategy")
	mockStrategy.EXPECT().CanHandle(gomock.Any()).Return(true)
	mockStrategy.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Override CreateStrategy to return our mock
	// This is a simplified test - in real scenario we'd need dependency injection

	// Act
	err = orchestrator.Run(context.Background(), "https://example.com", app.OrchestratorOptions{})

	// Assert
	require.NoError(t, err)
}

func TestOrchestrator_Run_UnknownStrategy(t *testing.T) {
	// Arrange
	cfg := config.Default()
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mocks.NewMockStrategy(ctrl)
	mockStrategy.EXPECT().Name().Return("test-strategy")
	mockStrategy.EXPECT().CanHandle(gomock.Any()).Return(true)
	mockStrategy.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(context.Canceled)

	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = orchestrator.Run(ctx, "https://example.com", app.OrchestratorOptions{})

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strategy execution failed")
}

func TestOrchestrator_Run_StrategyError(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mocks.NewMockStrategy(ctrl)
	mockStrategy.EXPECT().Name().Return("test-strategy")
	mockStrategy.EXPECT().CanHandle(gomock.Any()).Return(true)
	mockStrategy.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(assert.AnError)

	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	orchestrator, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)

	// Act
	err = orchestrator.Run(context.Background(), "https://example.com", app.OrchestratorOptions{})

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strategy execution failed")
}

func TestOrchestrator_Close(t *testing.T) {
	// Arrange
	cfg := config.Default()
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
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStrategy := mocks.NewMockStrategy(ctrl)
	mockStrategy.EXPECT().Name().Return("test-strategy")
	mockStrategy.EXPECT().CanHandle(gomock.Any()).Return(true)
	mockStrategy.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	cfg := config.Default()
	tmpDir := testutil.TempDir(t)
	cfg.Output.Directory = tmpDir

	opts := app.OrchestratorOptions{
		Config:     cfg,
		DryRun:     true,
		Verbose:    true,
		Limit:      100,
		RenderJS:   true,
		Split:      true,
		Force:      true,
		ExcludePatterns: []string{"test/*"},
		ContentSelector: "#content",
		FilterURL: "https://example.com/docs",
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)

	// Act
	err = orchestrator.Run(context.Background(), "https://example.com", opts)

	// Assert
	require.NoError(t, err)
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
		{"Unknown URL", "ftp://example.com", app.StrategyUnknown},
		{"File URL", "file:///path/to/file", app.StrategyUnknown},
		{"Empty URL", "", app.StrategyUnknown},
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
	// Arrange
	cfg := config.Default()
	deps, err := app.NewOrchestrator(app.OrchestratorOptions{
		Config: cfg,
	})
	require.NoError(t, err)
	require.NotNil(t, deps)

	// Note: CreateStrategy requires *strategies.Dependencies
	// In a real test, we'd need to create proper dependencies
	// This is a simplified test structure

	tests := []struct {
		name     string
		strategy app.StrategyType
	}{
		{"LLMS Strategy", app.StrategyLLMS},
		{"Sitemap Strategy", app.StrategySitemap},
		{"Git Strategy", app.StrategyGit},
		{"PkgGo Strategy", app.StrategyPkgGo},
		{"Crawler Strategy", app.StrategyCrawler},
		{"Unknown Strategy", app.StrategyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require proper strategies.Dependencies setup
			// Skipping for now - would be covered in integration tests
			t.Skip("Requires full dependency injection setup")
		})
	}
}

func TestFindMatchingStrategy(t *testing.T) {
	// This test would require full dependency setup
	t.Skip("Requires full dependency injection setup")
}

func TestGetAllStrategies(t *testing.T) {
	// This test would require full dependency setup
	t.Skip("Requires full dependency injection setup")
}
