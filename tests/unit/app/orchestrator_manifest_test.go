package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/manifest"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
)

type manifestTestStrategy struct {
	name      string
	execCalls []string
	execFunc  func(ctx context.Context, url string, opts strategies.Options) error
}

func (s *manifestTestStrategy) Name() string          { return s.name }
func (s *manifestTestStrategy) CanHandle(string) bool { return true }
func (s *manifestTestStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	s.execCalls = append(s.execCalls, url)
	if s.execFunc != nil {
		return s.execFunc(ctx, url, opts)
	}
	return nil
}

func createTestOrchestrator(t *testing.T, strategy strategies.Strategy) *app.Orchestrator {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = t.TempDir()

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return strategy
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	return orchestrator
}

func TestOrchestrator_RunManifest_AllSuccess(t *testing.T) {
	mock := &manifestTestStrategy{name: "mock"}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://example1.com"},
			{URL: "https://example2.com"},
			{URL: "https://example3.com"},
		},
		Options: manifest.Options{
			ContinueOnError: false,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.NoError(t, err)
	assert.Len(t, mock.execCalls, 3)
	assert.Contains(t, mock.execCalls, "https://example1.com")
	assert.Contains(t, mock.execCalls, "https://example2.com")
	assert.Contains(t, mock.execCalls, "https://example3.com")
}

func TestOrchestrator_RunManifest_ContinueOnError_True(t *testing.T) {
	mock := &manifestTestStrategy{
		name: "mock",
		execFunc: func(ctx context.Context, url string, opts strategies.Options) error {
			if url == "https://fail.com" {
				return errors.New("simulated failure")
			}
			return nil
		},
	}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://success1.com"},
			{URL: "https://fail.com"},
			{URL: "https://success2.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failures")
	assert.Len(t, mock.execCalls, 3)
}

func TestOrchestrator_RunManifest_ContinueOnError_False(t *testing.T) {
	mock := &manifestTestStrategy{
		name: "mock",
		execFunc: func(ctx context.Context, url string, opts strategies.Options) error {
			if url == "https://fail.com" {
				return errors.New("simulated failure")
			}
			return nil
		},
	}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://success.com"},
			{URL: "https://fail.com"},
			{URL: "https://never-reached.com"},
		},
		Options: manifest.Options{
			ContinueOnError: false,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://fail.com")
	assert.Len(t, mock.execCalls, 2)
	assert.NotContains(t, mock.execCalls, "https://never-reached.com")
}

func TestOrchestrator_RunManifest_ContextCancellation(t *testing.T) {
	mock := &manifestTestStrategy{name: "mock"}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://example1.com"},
			{URL: "https://example2.com"},
		},
		Options: manifest.Options{Output: t.TempDir()},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(ctx, manifestCfg, app.OrchestratorOptions{Config: cfg})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestOrchestrator_RunManifest_EmptySources(t *testing.T) {
	mock := &manifestTestStrategy{name: "mock"}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{},
		Options: manifest.Options{Output: t.TempDir()},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.NoError(t, err)
	assert.Empty(t, mock.execCalls)
}

func TestOrchestrator_RunManifest_StrategyOverride(t *testing.T) {
	var capturedOpts app.OrchestratorOptions

	mock := &manifestTestStrategy{name: "mock"}
	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = t.TempDir()

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return mock
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://example.com", Strategy: "crawler"},
		},
		Options: manifest.Options{Output: t.TempDir()},
	}

	capturedOpts = opts
	err = orchestrator.RunManifest(context.Background(), manifestCfg, capturedOpts)

	require.NoError(t, err)
	assert.Len(t, mock.execCalls, 1)
}

func TestOrchestrator_RunManifest_SourceOptions(t *testing.T) {
	var capturedOpts strategies.Options

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = t.TempDir()

	mock := &manifestTestStrategy{
		name: "mock",
		execFunc: func(ctx context.Context, url string, opts strategies.Options) error {
			capturedOpts = opts
			return nil
		},
	}

	orchOpts := app.OrchestratorOptions{
		Config:          cfg,
		ContentSelector: "base-selector",
		ExcludePatterns: []string{"base-pattern"},
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return mock
		},
	}

	orchestrator, err := app.NewOrchestrator(orchOpts)
	require.NoError(t, err)
	defer orchestrator.Close()

	renderJS := true
	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{
				URL:             "https://example.com",
				ContentSelector: "source-selector",
				ExcludeSelector: ".exclude-me",
				Exclude:         []string{"source-pattern"},
				RenderJS:        &renderJS,
				Limit:           50,
			},
		},
		Options: manifest.Options{Output: t.TempDir()},
	}

	err = orchestrator.RunManifest(context.Background(), manifestCfg, orchOpts)

	require.NoError(t, err)
	assert.Equal(t, "source-selector", capturedOpts.ContentSelector)
	assert.Equal(t, ".exclude-me", capturedOpts.ExcludeSelector)
	assert.Contains(t, capturedOpts.Exclude, "source-pattern")
	assert.True(t, capturedOpts.RenderJS)
	assert.Equal(t, 50, capturedOpts.Limit)
}

func TestOrchestrator_RunManifest_SingleSource(t *testing.T) {
	mock := &manifestTestStrategy{name: "mock"}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://example.com"},
		},
		Options: manifest.Options{Output: t.TempDir()},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.NoError(t, err)
	assert.Len(t, mock.execCalls, 1)
	assert.Equal(t, "https://example.com", mock.execCalls[0])
}

func TestOrchestrator_RunManifest_AllFail(t *testing.T) {
	mock := &manifestTestStrategy{
		name: "mock",
		execFunc: func(ctx context.Context, url string, opts strategies.Options) error {
			return errors.New("always fails")
		},
	}
	orchestrator := createTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://fail1.com"},
			{URL: "https://fail2.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "2/2 failures")
	assert.Len(t, mock.execCalls, 2)
}

func TestOrchestrator_Run_StrategyOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	var capturedType app.StrategyType
	mock := &manifestTestStrategy{name: "mock"}

	opts := app.OrchestratorOptions{
		Config:           cfg,
		StrategyOverride: "crawler",
		CommonOptions: domain.CommonOptions{
			Verbose: false,
		},
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			capturedType = st
			return mock
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.Run(context.Background(), "https://github.com/user/repo", opts)

	require.NoError(t, err)
	assert.Equal(t, app.StrategyCrawler, capturedType)
}

func TestOrchestrator_Run_InvalidStrategyOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false

	mock := &manifestTestStrategy{name: "mock"}

	opts := app.OrchestratorOptions{
		Config:           cfg,
		StrategyOverride: "invalid_strategy",
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return mock
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	defer orchestrator.Close()

	err = orchestrator.Run(context.Background(), "https://example.com", opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown strategy override")
}
