package strategies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/quantmind-br/repodocs-go/internal/state"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// Strategy defines the interface for documentation extraction strategies
type Strategy interface {
	// Name returns the strategy name
	Name() string
	// CanHandle returns true if this strategy can handle the given URL
	CanHandle(url string) bool
	// Execute runs the extraction strategy
	Execute(ctx context.Context, url string, opts Options) error
}

// Options contains common options for all strategies
type Options struct {
	domain.CommonOptions
	Output          string
	Concurrency     int
	MaxDepth        int
	Exclude         []string
	NoFolders       bool
	Split           bool
	IncludeAssets   bool
	ContentSelector string
	ExcludeSelector string
	CacheTTL        string
	FilterURL       string
}

// DefaultOptions returns default strategy options
func DefaultOptions() Options {
	return Options{
		CommonOptions: domain.DefaultCommonOptions(),
		Output:        "./docs",
		Concurrency:   5,
		MaxDepth:      3,
		NoFolders:     false,
		Split:         false,
	}
}

// Dependencies contains shared dependencies for all strategies
type Dependencies struct {
	Fetcher          domain.Fetcher
	Renderer         domain.Renderer
	Cache            domain.Cache
	Converter        *converter.Pipeline
	Writer           *output.Writer
	Logger           *utils.Logger
	LLMProvider      domain.LLMProvider
	MetadataEnhancer *llm.MetadataEnhancer
	Collector        *output.MetadataCollector
	HTTPClient       *http.Client
	StateManager     *state.Manager

	rendererOnce sync.Once
	rendererOpts renderer.RendererOptions
	rendererErr  error
}

// NewDependencies creates new dependencies for strategies
func NewDependencies(opts DependencyOptions) (*Dependencies, error) {
	// Create fetcher
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     opts.Timeout,
		MaxRetries:  3,
		EnableCache: opts.EnableCache,
		CacheTTL:    opts.CacheTTL,
		UserAgent:   opts.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	// Create cache if enabled
	var cacheImpl domain.Cache
	if opts.EnableCache {
		cacheImpl, err = cache.NewBadgerCache(cache.Options{
			Directory: opts.CacheDir,
		})
		if err != nil {
			return nil, err
		}
		fetcherClient.SetCache(cacheImpl)
	}

	// Prepare renderer options for lazy initialization
	rendererOpts := renderer.DefaultRendererOptions()
	if opts.RendererTimeout > 0 {
		rendererOpts.Timeout = opts.RendererTimeout
	}
	if opts.Concurrency > 0 {
		rendererOpts.MaxTabs = opts.Concurrency
	}

	// Create renderer eagerly only if explicitly requested
	var rendererImpl domain.Renderer
	if opts.EnableRenderer {
		r, err := renderer.NewRenderer(rendererOpts)
		if err == nil {
			rendererImpl = r
		}
	}

	// Create converter
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "",
		ContentSelector: opts.ContentSelector,
		ExcludeSelector: opts.ExcludeSelector,
	})

	var collector *output.MetadataCollector
	if opts.JSONMetadata {
		collector = output.NewMetadataCollector(output.CollectorOptions{
			BaseDir:   opts.OutputDir,
			SourceURL: opts.SourceURL,
			Enabled:   true,
		})
	}

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      opts.OutputDir,
		Flat:         opts.Flat,
		JSONMetadata: opts.JSONMetadata,
		Force:        opts.Force,
		DryRun:       opts.DryRun,
		Collector:    collector,
	})

	// Create logger
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   "info",
		Format:  "pretty",
		Verbose: opts.Verbose,
	})

	var llmProvider domain.LLMProvider
	var metadataEnhancer *llm.MetadataEnhancer
	if opts.LLMConfig != nil && opts.LLMConfig.EnhanceMetadata && opts.LLMConfig.Provider != "" {
		baseProvider, err := llm.NewProviderFromConfig(opts.LLMConfig)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to create LLM provider, metadata enhancement disabled")
		} else {
			if opts.LLMConfig.RateLimit.Enabled {
				llmProvider = llm.NewRateLimitedProvider(
					baseProvider,
					llm.RateLimitedProviderConfig{
						RequestsPerMinute:        opts.LLMConfig.RateLimit.RequestsPerMinute,
						BurstSize:                opts.LLMConfig.RateLimit.BurstSize,
						MaxRetries:               opts.LLMConfig.RateLimit.MaxRetries,
						InitialDelay:             opts.LLMConfig.RateLimit.InitialDelay,
						MaxDelay:                 opts.LLMConfig.RateLimit.MaxDelay,
						Multiplier:               opts.LLMConfig.RateLimit.Multiplier,
						CircuitBreakerEnabled:    opts.LLMConfig.RateLimit.CircuitBreaker.Enabled,
						FailureThreshold:         opts.LLMConfig.RateLimit.CircuitBreaker.FailureThreshold,
						SuccessThresholdHalfOpen: opts.LLMConfig.RateLimit.CircuitBreaker.SuccessThresholdHalfOpen,
						ResetTimeout:             opts.LLMConfig.RateLimit.CircuitBreaker.ResetTimeout,
					},
					logger,
				)
				logger.Info().
					Str("provider", opts.LLMConfig.Provider).
					Int("requests_per_minute", opts.LLMConfig.RateLimit.RequestsPerMinute).
					Int("burst_size", opts.LLMConfig.RateLimit.BurstSize).
					Msg("LLM metadata enhancement enabled with rate limiting")
			} else {
				llmProvider = baseProvider
				logger.Info().Str("provider", opts.LLMConfig.Provider).Msg("LLM metadata enhancement enabled")
			}
			metadataEnhancer = llm.NewMetadataEnhancer(llmProvider)
		}
	}

	var stateManager *state.Manager
	if opts.Sync && !opts.FullSync {
		stateManager = state.NewManager(state.ManagerOptions{
			BaseDir:   opts.OutputDir,
			SourceURL: opts.SourceURL,
			Logger:    logger,
			Disabled:  false,
		})
		if err := stateManager.Load(context.Background()); err != nil {
			if !errors.Is(err, state.ErrStateNotFound) {
				logger.Warn().Err(err).Msg("Failed to load state, starting fresh")
			}
		}
	}

	return &Dependencies{
		Fetcher:          fetcherClient,
		Renderer:         rendererImpl,
		Cache:            cacheImpl,
		Converter:        converterPipeline,
		Writer:           writer,
		Logger:           logger,
		LLMProvider:      llmProvider,
		MetadataEnhancer: metadataEnhancer,
		Collector:        collector,
		StateManager:     stateManager,
		rendererOpts:     rendererOpts,
	}, nil
}

// Close releases all resources
func (d *Dependencies) Close() error {
	if d.Fetcher != nil {
		d.Fetcher.Close()
	}
	if d.Renderer != nil {
		d.Renderer.Close()
	}
	if d.Cache != nil {
		d.Cache.Close()
	}
	if d.LLMProvider != nil {
		d.LLMProvider.Close()
	}
	return nil
}

func (d *Dependencies) FlushMetadata() error {
	if d.Collector != nil {
		return d.Collector.Flush()
	}
	return nil
}

func (d *Dependencies) SaveState(ctx context.Context) error {
	if d.StateManager != nil {
		return d.StateManager.Save(ctx)
	}
	return nil
}

func (d *Dependencies) PruneDeletedFiles(ctx context.Context) (int, error) {
	if d.StateManager == nil || d.StateManager.IsDisabled() {
		return 0, nil
	}

	deleted := d.StateManager.GetDeletedPages()
	if len(deleted) == 0 {
		return 0, nil
	}

	var pruned int
	for _, page := range deleted {
		if err := os.Remove(page.FilePath); err != nil {
			if !os.IsNotExist(err) {
				d.Logger.Warn().Err(err).Str("file", page.FilePath).Msg("Failed to remove deleted page")
				continue
			}
		}
		pruned++
		d.Logger.Info().Str("file", page.FilePath).Msg("Removed deleted page")
	}

	d.StateManager.RemoveDeletedFromState()
	return pruned, nil
}

func (d *Dependencies) GetStateManager() *state.Manager {
	return d.StateManager
}

func (d *Dependencies) SetStrategy(name string) {
	if d.Collector != nil {
		d.Collector.SetStrategy(name)
	}
}

func (d *Dependencies) SetSourceURL(url string) {
	if d.Collector != nil {
		d.Collector.SetSourceURL(url)
	}
}

func (d *Dependencies) GetRenderer() (domain.Renderer, error) {
	d.rendererOnce.Do(func() {
		if d.Renderer != nil {
			return // already set externally
		}
		opts := d.rendererOpts
		if opts.Timeout == 0 {
			opts = renderer.DefaultRendererOptions()
		}
		r, err := renderer.NewRenderer(opts)
		if err != nil {
			d.rendererErr = err
			d.Logger.Debug().Err(err).Msg("Failed to initialize browser renderer on demand")
			return
		}
		d.Renderer = r
		d.Logger.Info().Msg("Browser renderer initialized on demand")
	})

	if d.rendererErr != nil {
		return nil, d.rendererErr
	}
	return d.Renderer, nil
}

// WriteDocument enhances metadata (if configured) and writes the document
func (d *Dependencies) WriteDocument(ctx context.Context, doc *domain.Document) error {
	if d.MetadataEnhancer != nil {
		if err := d.MetadataEnhancer.Enhance(ctx, doc); err != nil {
			d.Logger.Warn().Err(err).Str("url", doc.URL).Msg("Failed to enhance metadata, writing without enhancement")
		}
	}

	if d.Writer == nil {
		return fmt.Errorf("writer is not configured")
	}

	if err := d.Writer.Write(ctx, doc); err != nil {
		return err
	}

	if d.StateManager != nil && doc.ContentHash != "" {
		filePath := d.Writer.GetPath(doc.URL)
		d.StateManager.Update(doc.URL, state.PageState{
			ContentHash: doc.ContentHash,
			FetchedAt:   doc.FetchedAt,
			FilePath:    filePath,
		})
	}

	return nil
}

// DependencyOptions contains options for creating dependencies
type DependencyOptions struct {
	domain.CommonOptions
	Timeout         time.Duration
	EnableCache     bool
	CacheTTL        time.Duration
	CacheDir        string
	UserAgent       string
	EnableRenderer  bool
	RendererTimeout time.Duration
	Concurrency     int
	ContentSelector string
	ExcludeSelector string
	OutputDir       string
	Flat            bool
	JSONMetadata    bool
	LLMConfig       *config.LLMConfig
	SourceURL       string
}
