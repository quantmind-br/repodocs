package app

import (
	"context"
	"fmt"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// Orchestrator coordinates the documentation extraction process
type Orchestrator struct {
	config          *config.Config
	deps            *strategies.Dependencies
	logger          *utils.Logger
	strategyFactory func(StrategyType, *strategies.Dependencies) strategies.Strategy
}

// OrchestratorOptions contains options for creating an orchestrator
type OrchestratorOptions struct {
	domain.CommonOptions
	Config          *config.Config
	Split           bool
	IncludeAssets   bool
	ContentSelector string
	ExcludeSelector string
	ExcludePatterns []string
	FilterURL       string
	StrategyFactory func(StrategyType, *strategies.Dependencies) strategies.Strategy
}

// NewOrchestrator creates a new orchestrator with the given configuration
func NewOrchestrator(opts OrchestratorOptions) (*Orchestrator, error) {
	cfg := opts.Config

	// Validate config
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

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
		CommonOptions: domain.CommonOptions{
			Verbose:  opts.Verbose,
			DryRun:   opts.DryRun,
			Force:    opts.Force || cfg.Output.Overwrite,
			RenderJS: opts.RenderJS,
			Limit:    opts.Limit,
		},
		Timeout:         cfg.Concurrency.Timeout,
		EnableCache:     cfg.Cache.Enabled,
		CacheTTL:        cfg.Cache.TTL,
		CacheDir:        cacheDir,
		UserAgent:       cfg.Stealth.UserAgent,
		EnableRenderer:  cfg.Rendering.ForceJS || opts.RenderJS,
		RendererTimeout: cfg.Rendering.JSTimeout,
		Concurrency:     cfg.Concurrency.Workers,
		ContentSelector: opts.ContentSelector,
		ExcludeSelector: opts.ExcludeSelector,
		OutputDir:       cfg.Output.Directory,
		Flat:            cfg.Output.Flat,
		JSONMetadata:    cfg.Output.JSONMetadata,
		LLMConfig:       &cfg.LLM,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create dependencies: %w", err)
	}

	// Set default strategy factory if none provided
	strategyFactory := opts.StrategyFactory
	if strategyFactory == nil {
		strategyFactory = func(st StrategyType, d *strategies.Dependencies) strategies.Strategy {
			return CreateStrategy(st, d)
		}
	}

	return &Orchestrator{
		config:          cfg,
		deps:            deps,
		logger:          logger,
		strategyFactory: strategyFactory,
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

	// Create strategy using strategy factory (allows injection for testing)
	strategy := o.strategyFactory(strategyType, o.deps)
	if strategy == nil {
		return fmt.Errorf("failed to create strategy for URL: %s", url)
	}

	o.logger.Info().
		Str("strategy", strategy.Name()).
		Msg("Using extraction strategy")

	o.deps.SetSourceURL(url)
	o.deps.SetStrategy(strategy.Name())

	strategyOpts := strategies.Options{
		CommonOptions: domain.CommonOptions{
			Verbose:  opts.Verbose,
			DryRun:   opts.DryRun,
			Force:    opts.Force || o.config.Output.Overwrite,
			RenderJS: opts.RenderJS || o.config.Rendering.ForceJS,
			Limit:    opts.Limit,
		},
		Output:          o.config.Output.Directory,
		Concurrency:     o.config.Concurrency.Workers,
		MaxDepth:        o.config.Concurrency.MaxDepth,
		Exclude:         append(o.config.Exclude, opts.ExcludePatterns...),
		NoFolders:       o.config.Output.Flat,
		Split:           opts.Split,
		IncludeAssets:   opts.IncludeAssets,
		ContentSelector: opts.ContentSelector,
		ExcludeSelector: opts.ExcludeSelector,
		FilterURL:       opts.FilterURL,
	}

	if err := strategy.Execute(ctx, url, strategyOpts); err != nil {
		if ctx.Err() != nil {
			o.logger.Warn().Msg("Extraction cancelled")
			return ctx.Err()
		}
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	if err := o.deps.FlushMetadata(); err != nil {
		o.logger.Warn().Err(err).Msg("Failed to flush metadata")
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
