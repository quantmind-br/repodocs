package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestrator_Run tests the Run method
func TestOrchestrator_Run(t *testing.T) {
	tests := []struct {
		name         string
		strategyType StrategyType
		url          string
		executeError bool
		expectError  bool
	}{
		{
			name:         "successful execution",
			strategyType: StrategyCrawler,
			url:          "https://example.com/docs",
			executeError: false,
			expectError:  false,
		},
		{
			name:         "strategy execution error",
			strategyType: StrategyCrawler,
			url:          "https://example.com/docs",
			executeError: true,
			expectError:  true,
		},
		{
			name:         "unknown strategy",
			strategyType: StrategyUnknown,
			url:          "ftp://example.com",
			executeError: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Cache: config.CacheConfig{
					Enabled: false,
				},
				Concurrency: config.ConcurrencyConfig{
					Timeout: 10 * time.Second,
					Workers: 1,
				},
				Output: config.OutputConfig{
					Directory: t.TempDir(),
				},
				Logging: config.LoggingConfig{
					Level:  "error",
					Format: "pretty",
				},
			}

			mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
				if tt.executeError {
					return &mockErrorStrategy{name: string(st)}
				}
				return &mockStrategy{name: string(st)}
			}

			orch, err := NewOrchestrator(OrchestratorOptions{
				Config:          cfg,
				StrategyFactory: mockFactory,
			})
			require.NoError(t, err)
			defer orch.Close()

			ctx := context.Background()
			err = orch.Run(ctx, tt.url, OrchestratorOptions{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestOrchestrator_Run_ContextCancellation tests context cancellation
func TestOrchestrator_Run_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "pretty",
		},
	}

	mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		return &mockCancelStrategy{name: string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config:          cfg,
		StrategyFactory: mockFactory,
	})
	require.NoError(t, err)
	defer orch.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = orch.Run(ctx, "https://example.com/docs", OrchestratorOptions{})
	assert.Error(t, err)
}

// TestOrchestrator_Run_VerboseLogging tests verbose logging option
func TestOrchestrator_Run_VerboseLogging(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "pretty",
		},
	}

	mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		return &mockStrategy{name: string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			Verbose: true,
		},
		StrategyFactory: mockFactory,
	})
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()
	err = orch.Run(ctx, "https://example.com/docs", OrchestratorOptions{})
	assert.NoError(t, err)
}

// TestOrchestrator_Run_DryRun tests dry run option
func TestOrchestrator_Run_DryRun(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "pretty",
		},
	}

	mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		return &mockDryRunStrategy{name: string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
		StrategyFactory: mockFactory,
	})
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()
	err = orch.Run(ctx, "https://example.com/docs", OrchestratorOptions{
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	assert.NoError(t, err)
}

// TestOrchestrator_Run_WithLimit tests with limit option
func TestOrchestrator_Run_WithLimit(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "pretty",
		},
	}

	mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		return &mockLimitStrategy{name: string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config:          cfg,
		StrategyFactory: mockFactory,
	})
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()
	err = orch.Run(ctx, "https://example.com/docs", OrchestratorOptions{
		CommonOptions: domain.CommonOptions{
			Limit: 5,
		},
	})
	assert.NoError(t, err)
}

// TestOrchestrator_Run_WithSelectors tests with content/exclude selectors
func TestOrchestrator_Run_WithSelectors(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "pretty",
		},
	}

	mockFactory := func(st StrategyType, deps *strategies.Dependencies) strategies.Strategy {
		return &mockSelectorStrategy{name: string(st)}
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config:          cfg,
		ContentSelector: ".content",
		ExcludeSelector: ".ads",
		StrategyFactory: mockFactory,
	})
	require.NoError(t, err)
	defer orch.Close()

	ctx := context.Background()
	err = orch.Run(ctx, "https://example.com/docs", OrchestratorOptions{
		ContentSelector: ".content",
		ExcludeSelector: ".ads",
	})
	assert.NoError(t, err)
}

// TestOrchestrator_Close_NilDeps tests closing with nil dependencies
func TestOrchestrator_Close_NilDeps(t *testing.T) {
	orch := &Orchestrator{
		deps: nil,
	}
	err := orch.Close()
	assert.NoError(t, err)
}

// TestNewOrchestrator_CacheDirExpansion tests cache directory path expansion
func TestNewOrchestrator_CacheDirExpansion(t *testing.T) {
	tests := []struct {
		name     string
		cacheDir string
	}{
		{
			name:     "tilde expansion",
			cacheDir: "~/.repodocs/cache",
		},
		{
			name:     "explicit cache dir",
			cacheDir: "/tmp/cache",
		},
		{
			name:     "empty cache dir uses default",
			cacheDir: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Cache: config.CacheConfig{
					Enabled:   false,
					Directory: tt.cacheDir,
				},
				Concurrency: config.ConcurrencyConfig{
					Timeout: 10 * time.Second,
					Workers: 1,
				},
				Output: config.OutputConfig{
					Directory: t.TempDir(),
				},
				Logging: config.LoggingConfig{
					Level:  "error",
					Format: "pretty",
				},
			}

			orch, err := NewOrchestrator(OrchestratorOptions{
				Config: cfg,
			})
			require.NoError(t, err)
			orch.Close()
		})
	}
}

// TestNewOrchestrator_LoggingLevels tests different logging configurations
func TestNewOrchestrator_LoggingLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
		verbose   bool
	}{
		{
			name:      "info level",
			logLevel:  "info",
			logFormat: "pretty",
			verbose:   false,
		},
		{
			name:      "debug level",
			logLevel:  "debug",
			logFormat: "json",
			verbose:   false,
		},
		{
			name:      "error level",
			logLevel:  "error",
			logFormat: "pretty",
			verbose:   false,
		},
		{
			name:      "verbose overrides to debug",
			logLevel:  "info",
			logFormat: "pretty",
			verbose:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Cache: config.CacheConfig{
					Enabled: false,
				},
				Concurrency: config.ConcurrencyConfig{
					Timeout: 10 * time.Second,
					Workers: 1,
				},
				Output: config.OutputConfig{
					Directory: t.TempDir(),
				},
				Logging: config.LoggingConfig{
					Level:  tt.logLevel,
					Format: tt.logFormat,
				},
			}

			orch, err := NewOrchestrator(OrchestratorOptions{
				Config: cfg,
				CommonOptions: domain.CommonOptions{
					Verbose: tt.verbose,
				},
			})
			require.NoError(t, err)
			orch.Close()
		})
	}
}

// TestNewOrchestrator_RendererOptions tests renderer-related options
func TestNewOrchestrator_RendererOptions(t *testing.T) {
	tests := []struct {
		name     string
		renderJS bool
	}{
		{
			name:     "JS rendering enabled",
			renderJS: true,
		},
		{
			name:     "JS rendering disabled",
			renderJS: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Cache: config.CacheConfig{
					Enabled: false,
				},
				Concurrency: config.ConcurrencyConfig{
					Timeout: 10 * time.Second,
					Workers: 1,
				},
				Output: config.OutputConfig{
					Directory: t.TempDir(),
				},
				Logging: config.LoggingConfig{
					Level:  "error",
					Format: "pretty",
				},
			}

			orch, err := NewOrchestrator(OrchestratorOptions{
				Config: cfg,
				CommonOptions: domain.CommonOptions{
					RenderJS: tt.renderJS,
				},
			})
			require.NoError(t, err)
			orch.Close()
		})
	}
}

// TestNewOrchestrator_ForceOption tests force option
func TestNewOrchestrator_ForceOption(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
		Concurrency: config.ConcurrencyConfig{
			Timeout: 10 * time.Second,
			Workers: 1,
		},
		Output: config.OutputConfig{
			Directory: t.TempDir(),
			Overwrite: false,
		},
		Logging: config.LoggingConfig{
			Level:  "error",
			Format: "pretty",
		},
	}

	orch, err := NewOrchestrator(OrchestratorOptions{
		Config: cfg,
		CommonOptions: domain.CommonOptions{
			Force: true,
		},
	})
	require.NoError(t, err)
	orch.Close()
}

// Mock strategies for testing

type mockErrorStrategy struct {
	name string
}

func (m *mockErrorStrategy) Name() string {
	return m.name
}

func (m *mockErrorStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockErrorStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	return fmt.Errorf("mock execution error")
}

type mockCancelStrategy struct {
	name string
}

func (m *mockCancelStrategy) Name() string {
	return m.name
}

func (m *mockCancelStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockCancelStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	<-ctx.Done()
	return ctx.Err()
}

type mockDryRunStrategy struct {
	name string
}

func (m *mockDryRunStrategy) Name() string {
	return m.name
}

func (m *mockDryRunStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockDryRunStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	if !opts.DryRun {
		return fmt.Errorf("expected dry run to be set")
	}
	return nil
}

type mockLimitStrategy struct {
	name string
}

func (m *mockLimitStrategy) Name() string {
	return m.name
}

func (m *mockLimitStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockLimitStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	if opts.Limit == 0 {
		return fmt.Errorf("expected limit to be set")
	}
	return nil
}

type mockSelectorStrategy struct {
	name string
}

func (m *mockSelectorStrategy) Name() string {
	return m.name
}

func (m *mockSelectorStrategy) CanHandle(url string) bool {
	return true
}

func (m *mockSelectorStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	if opts.ContentSelector == "" {
		return fmt.Errorf("expected content selector to be set")
	}
	if opts.ExcludeSelector == "" {
		return fmt.Errorf("expected exclude selector to be set")
	}
	return nil
}
