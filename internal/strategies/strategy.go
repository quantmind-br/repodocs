package strategies

import (
	"context"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
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
	Fetcher   *fetcher.Client
	Renderer  domain.Renderer
	Cache     domain.Cache
	Converter *converter.Pipeline
	Writer    *output.Writer
	Logger    *utils.Logger
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

	// Create renderer if needed
	var rendererImpl domain.Renderer
	if opts.EnableRenderer {
		rendererOpts := renderer.DefaultRendererOptions()
		rendererOpts.Timeout = opts.RendererTimeout
		rendererOpts.MaxTabs = opts.Concurrency
		r, err := renderer.NewRenderer(rendererOpts)
		if err != nil {
			// Renderer is optional, continue without it
			rendererImpl = nil
		} else {
			rendererImpl = r
		}
	}

	// Create converter
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL:         "",
		ContentSelector: opts.ContentSelector,
		ExcludeSelector: opts.ExcludeSelector,
	})

	// Create writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir:      opts.OutputDir,
		Flat:         opts.Flat,
		JSONMetadata: opts.JSONMetadata,
		Force:        opts.Force,
		DryRun:       opts.DryRun,
	})

	// Create logger
	logger := utils.NewLogger(utils.LoggerOptions{
		Level:   "info",
		Format:  "pretty",
		Verbose: opts.Verbose,
	})

	return &Dependencies{
		Fetcher:   fetcherClient,
		Renderer:  rendererImpl,
		Cache:     cacheImpl,
		Converter: converterPipeline,
		Writer:    writer,
		Logger:    logger,
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
	return nil
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
}
