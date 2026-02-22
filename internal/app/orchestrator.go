package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/manifest"
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
	Config           *config.Config
	Split            bool
	IncludeAssets    bool
	ContentSelector  string
	ExcludeSelector  string
	ExcludePatterns  []string
	FilterURL        string
	StrategyFactory  func(StrategyType, *strategies.Dependencies) strategies.Strategy
	StrategyOverride string
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
			Sync:     opts.Sync,
			FullSync: opts.FullSync,
			Prune:    opts.Prune,
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

	var strategyType StrategyType
	if opts.StrategyOverride != "" {
		strategyType = StrategyType(opts.StrategyOverride)
		o.logger.Debug().
			Str("strategy", string(strategyType)).
			Msg("Using strategy override from manifest")

		if !IsValidStrategy(strategyType) {
			return fmt.Errorf("unknown strategy override: %s", opts.StrategyOverride)
		}
	} else {
		strategyType = DetectStrategy(url)
		o.logger.Debug().
			Str("strategy", string(strategyType)).
			Msg("Detected strategy type")

		if strategyType == StrategyUnknown {
			return fmt.Errorf("unable to determine strategy for URL: %s", url)
		}
	}

	// Sitemap auto-discovery: when Crawler is selected and no strategy override,
	// probe for sitemaps before falling back to crawling
	if strategyType == StrategyCrawler && opts.StrategyOverride == "" {
		discovery, discoverErr := strategies.DiscoverSitemap(ctx, o.deps.Fetcher, url, o.logger)
		if discoverErr != nil {
			o.logger.Debug().Err(discoverErr).Msg("Sitemap discovery failed, continuing with crawler")
		} else if discovery != nil {
			o.logger.Info().
				Str("sitemap_url", discovery.SitemapURL).
				Str("method", discovery.Method).
				Msg("Discovered sitemap, switching from crawler to sitemap strategy")
			strategyType = StrategySitemap
			url = discovery.SitemapURL
		}
	}

	// Content-based sitemap detection for .xml URLs not caught by URL patterns
	if strategyType == StrategyCrawler && opts.StrategyOverride == "" {
		pathEnd := strings.ToLower(url)
		if idx := strings.IndexAny(pathEnd, "?#"); idx >= 0 {
			pathEnd = pathEnd[:idx]
		}
		if strings.HasSuffix(pathEnd, ".xml") {
			resp, fetchErr := o.deps.Fetcher.Get(ctx, url)
			if fetchErr == nil && resp.StatusCode == 200 && strategies.IsSitemapContent(resp.Body) {
				o.logger.Info().Str("url", url).
					Msg("Content detected as sitemap XML, switching to sitemap strategy")
				strategyType = StrategySitemap
			}
		}
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

	if opts.Prune {
		pruned, err := o.deps.PruneDeletedFiles(ctx)
		if err != nil {
			o.logger.Warn().Err(err).Msg("Failed to prune deleted files")
		} else if pruned > 0 {
			o.logger.Info().Int("pruned", pruned).Msg("Removed deleted pages")
		}
	}

	if err := o.deps.SaveState(ctx); err != nil {
		o.logger.Warn().Err(err).Msg("Failed to save state")
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

// ManifestResult represents the result of processing one manifest source
type ManifestResult struct {
	Source   manifest.Source
	Error    error
	Duration time.Duration
}

// RunManifest executes all sources defined in the manifest
func (o *Orchestrator) RunManifest(
	ctx context.Context,
	manifestCfg *manifest.Config,
	baseOpts OrchestratorOptions,
) error {
	startTime := time.Now()
	totalSources := len(manifestCfg.Sources)

	o.logger.Info().
		Int("sources", totalSources).
		Bool("continue_on_error", manifestCfg.Options.ContinueOnError).
		Str("output", manifestCfg.Options.Output).
		Msg("Starting manifest execution")

	if totalSources == 0 {
		o.logger.Info().
			Dur("total_duration", time.Since(startTime)).
			Int("total", 0).
			Int("success", 0).
			Int("failed", 0).
			Msg("Manifest execution completed")
		return nil
	}

	concurrency := baseOpts.Config.Concurrency.Workers
	if concurrency <= 0 {
		concurrency = 5
	}
	if concurrency > 3 {
		concurrency = 3
	}

	results := make([]ManifestResult, totalSources)
	var resultsMu sync.Mutex
	var firstError error
	var firstErrorMu sync.Mutex

	var cancelCtx context.Context
	var cancel context.CancelFunc
	if manifestCfg.Options.ContinueOnError {
		cancelCtx = ctx
	} else {
		cancelCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	type sourceWithIndex struct {
		source manifest.Source
		index  int
	}

	sourcesWithIndex := make([]sourceWithIndex, totalSources)
	for i, source := range manifestCfg.Sources {
		sourcesWithIndex[i] = sourceWithIndex{source: source, index: i}
	}

	errs := utils.ParallelForEach(cancelCtx, sourcesWithIndex, concurrency, func(ctx context.Context, item sourceWithIndex) error {
		sourceStart := time.Now()
		source := item.source
		idx := item.index

		o.logger.Info().
			Int("source_idx", idx).
			Str("source_url", source.URL).
			Int("total", totalSources).
			Str("strategy", source.Strategy).
			Msg("Processing source")

		opts := o.buildSourceOptions(source, baseOpts)

		err := o.Run(ctx, source.URL, opts)
		sourceDuration := time.Since(sourceStart)

		resultsMu.Lock()
		results[idx] = ManifestResult{
			Source:   source,
			Error:    err,
			Duration: sourceDuration,
		}
		resultsMu.Unlock()

		if err != nil {
			o.logger.Error().
				Err(err).
				Int("source_idx", idx).
				Str("source_url", source.URL).
				Dur("duration", sourceDuration).
				Msg("Source extraction failed")

			if !manifestCfg.Options.ContinueOnError {
				firstErrorMu.Lock()
				if firstError == nil {
					firstError = fmt.Errorf("source %s failed: %w", source.URL, err)
				}
				firstErrorMu.Unlock()
				if cancel != nil {
					cancel()
				}
				return err
			}

			firstErrorMu.Lock()
			if firstError == nil {
				firstError = err
			}
			firstErrorMu.Unlock()
		} else {
			o.logger.Info().
				Int("source_idx", idx).
				Str("source_url", source.URL).
				Dur("duration", sourceDuration).
				Msg("Source extraction completed")
		}

		return nil
	})

	if ctx.Err() != nil {
		o.logger.Warn().Msg("Manifest execution cancelled")
		return ctx.Err()
	}

	if !manifestCfg.Options.ContinueOnError && firstError != nil {
		o.logger.Warn().Msg("Stopping execution (continue_on_error=false)")
		return firstError
	}

	if err := utils.FirstError(errs); err != nil && firstError == nil {
		firstError = err
	}

	duration := time.Since(startTime)
	successCount := 0
	for _, r := range results {
		if r.Error == nil {
			successCount++
		}
	}

	o.logger.Info().
		Dur("total_duration", duration).
		Int("total", totalSources).
		Int("success", successCount).
		Int("failed", totalSources-successCount).
		Msg("Manifest execution completed")

	if firstError != nil {
		return fmt.Errorf("manifest completed with %d/%d failures: %w",
			totalSources-successCount, totalSources, firstError)
	}

	return nil
}

func (o *Orchestrator) buildSourceOptions(source manifest.Source, baseOpts OrchestratorOptions) OrchestratorOptions {
	opts := baseOpts

	if source.Strategy != "" {
		opts.StrategyOverride = source.Strategy
	}

	if source.ContentSelector != "" {
		opts.ContentSelector = source.ContentSelector
	}
	if source.ExcludeSelector != "" {
		opts.ExcludeSelector = source.ExcludeSelector
	}

	if len(source.Exclude) > 0 {
		opts.ExcludePatterns = append(opts.ExcludePatterns, source.Exclude...)
	}

	if source.RenderJS != nil {
		opts.RenderJS = *source.RenderJS
	}

	if source.Limit > 0 {
		opts.Limit = source.Limit
	}

	if source.MaxDepth > 0 {
		o.logger.Debug().
			Int("max_depth", source.MaxDepth).
			Str("url", source.URL).
			Msg("Source max_depth specified but config override not implemented")
	}

	return opts
}
