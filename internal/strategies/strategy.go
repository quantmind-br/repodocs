package strategies

import (
	"context"
	"fmt"
	"net/http"
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
	Output          string
	Concurrency     int
	Limit           int
	MaxDepth        int
	Exclude         []string
	NoFolders       bool
	DryRun          bool
	Verbose         bool
	Force           bool
	RenderJS        bool
	Split           bool
	IncludeAssets   bool
	ContentSelector string
	ExcludeSelector string
	CacheTTL        string
	FilterURL       string // Base URL filter - only crawl URLs starting with this path
}

// DefaultOptions returns default strategy options
func DefaultOptions() Options {
	return Options{
		Output:      "./docs",
		Concurrency: 5,
		Limit:       0,
		MaxDepth:    3,
		NoFolders:   false,
		DryRun:      false,
		Verbose:     false,
		Force:       false,
		RenderJS:    false,
		Split:       false,
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
	HTTPClient       *http.Client // Optional custom HTTP client (e.g., for testing)

	// Lazy renderer initialization
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
	if d.Renderer != nil {
		return d.Renderer, nil
	}

	d.rendererOnce.Do(func() {
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

	return d.Writer.Write(ctx, doc)
}

// DependencyOptions contains options for creating dependencies
type DependencyOptions struct {
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
	Force           bool
	DryRun          bool
	Verbose         bool
	LLMConfig       *config.LLMConfig
	SourceURL       string
}
