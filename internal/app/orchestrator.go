package app

import (
	"context"
	"fmt"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// Orchestrator coordinates the documentation extraction process
type Orchestrator struct {
	config *config.Config
	deps   *strategies.Dependencies
	logger *utils.Logger
}

// OrchestratorOptions contains options for creating an orchestrator
type OrchestratorOptions struct {
	Config          *config.Config
	Verbose         bool
	DryRun          bool
	Force           bool
	RenderJS        bool
	Split           bool
	IncludeAssets   bool
	Limit           int
	ContentSelector string
	ExcludePatterns []string
	FilterURL       string
}

// NewOrchestrator creates a new orchestrator with the given configuration
func NewOrchestrator(opts OrchestratorOptions) (*Orchestrator, error) {
	cfg := opts.Config

	// Create logger
	logLevel := "info"
	logFormat := "pretty"
	if cfg.Logging.Level != "" {
		logLevel = cfg.Logging.Level
	}
	if cfg.Logging.Format != "" {
		logFormat = cfg.Logging.Format
	}
	if opts.Verbose {
		logLevel = "debug"
	}

	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   logLevel,
		Format:  logFormat,
		Verbose: opts.Verbose,
	})

	// Determine cache directory
	cacheDir := cfg.Cache.Directory
	if cacheDir == "" {
		cacheDir = "~/.repodocs/cache"
	}
	cacheDir = utils.ExpandPath(cacheDir)

	// Create dependencies
	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:         cfg.Concurrency.Timeout,
		EnableCache:     cfg.Cache.Enabled,
		CacheTTL:        cfg.Cache.TTL,
		CacheDir:        cacheDir,
		UserAgent:       cfg.Stealth.UserAgent,
		EnableRenderer:  cfg.Rendering.ForceJS || opts.RenderJS,
		RendererTimeout: cfg.Rendering.JSTimeout,
		Concurrency:     cfg.Concurrency.Workers,
		ContentSelector: opts.ContentSelector,
		OutputDir:       cfg.Output.Directory,
		Flat:            cfg.Output.Flat,
		JSONMetadata:    cfg.Output.JSONMetadata,
		Force:           opts.Force || cfg.Output.Overwrite,
		DryRun:          opts.DryRun,
		Verbose:         opts.Verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	return &Orchestrator{
		config: cfg,
		deps:   deps,
		logger: logger,
	}, nil
}

// Run executes the documentation extraction for the given URL
func (o *Orchestrator) Run(ctx context.Context, url string, opts OrchestratorOptions) error {
	startTime := time.Now()

	o.logger.Info().
		Str("url", url).
		Str("output", o.config.Output.Directory).
		Int("concurrency", o.config.Concurrency.Workers).
		Msg("Starting documentation extraction")

	// Detect strategy
	strategyType := DetectStrategy(url)
	o.logger.Debug().
		Str("strategy", string(strategyType)).
		Msg("Detected strategy type")

	if strategyType == StrategyUnknown {
		return fmt.Errorf("unable to determine strategy for URL: %s", url)
	}

	// Create strategy
	strategy := CreateStrategy(strategyType, o.deps)
	if strategy == nil {
		return fmt.Errorf("failed to create strategy for URL: %s", url)
	}

	o.logger.Info().
		Str("strategy", strategy.Name()).
		Msg("Using extraction strategy")

	// Build strategy options
	strategyOpts := strategies.Options{
		Output:          o.config.Output.Directory,
		Concurrency:     o.config.Concurrency.Workers,
		Limit:           opts.Limit,
		MaxDepth:        o.config.Concurrency.MaxDepth,
		Exclude:         append(o.config.Exclude, opts.ExcludePatterns...),
		NoFolders:       o.config.Output.Flat,
		DryRun:          opts.DryRun,
		Verbose:         opts.Verbose,
		Force:           opts.Force || o.config.Output.Overwrite,
		RenderJS:        opts.RenderJS || o.config.Rendering.ForceJS,
		Split:           opts.Split,
		IncludeAssets:   opts.IncludeAssets,
		ContentSelector: opts.ContentSelector,
		FilterURL:       opts.FilterURL,
	}

	// Execute strategy
	if err := strategy.Execute(ctx, url, strategyOpts); err != nil {
		// Check if it was a context cancellation
		if ctx.Err() != nil {
			o.logger.Warn().Msg("Extraction cancelled")
			return ctx.Err()
		}
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	duration := time.Since(startTime)
	o.logger.Info().
		Dur("duration", duration).
		Msg("Documentation extraction completed")

	return nil
}

// Close releases all resources held by the orchestrator
func (o *Orchestrator) Close() error {
	if o.deps != nil {
		return o.deps.Close()
	}
	return nil
}

// GetStrategyName returns the detected strategy name for a URL
func (o *Orchestrator) GetStrategyName(url string) string {
	return string(DetectStrategy(url))
}

// ValidateURL checks if the URL can be processed
func (o *Orchestrator) ValidateURL(url string) error {
	strategyType := DetectStrategy(url)
	if strategyType == StrategyUnknown {
		return fmt.Errorf("unsupported URL format: %s", url)
	}
	return nil
}
