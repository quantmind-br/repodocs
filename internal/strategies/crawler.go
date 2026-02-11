package strategies

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/schollz/progressbar/v3"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// CrawlerStrategy crawls websites to extract documentation
type CrawlerStrategy struct {
	deps           *Dependencies
	fetcher        domain.Fetcher
	renderer       domain.Renderer
	converter      *converter.Pipeline
	markdownReader *converter.MarkdownReader
	writer         *output.Writer
	logger         *utils.Logger
}

// crawlContext holds shared state between concurrent crawler callbacks.
type crawlContext struct {
	ctx            context.Context
	baseURL        string
	opts           Options
	visited        *sync.Map
	processedCount *int
	mu             *sync.Mutex
	bar            *progressbar.ProgressBar
	barMu          *sync.Mutex
	excludeRegexps []*regexp.Regexp
	collector      *colly.Collector // for re-injecting JS-discovered links
}

func newCrawlContext(ctx context.Context, baseURL string, opts Options) *crawlContext {
	var excludeRegexps []*regexp.Regexp
	for _, pattern := range opts.Exclude {
		if re, err := regexp.Compile(pattern); err == nil {
			excludeRegexps = append(excludeRegexps, re)
		}
	}

	var processedCount int
	return &crawlContext{
		ctx:            ctx,
		baseURL:        baseURL,
		opts:           opts,
		visited:        &sync.Map{},
		processedCount: &processedCount,
		mu:             &sync.Mutex{},
		bar:            utils.NewProgressBar(-1, utils.DescExtracting),
		barMu:          &sync.Mutex{},
		excludeRegexps: excludeRegexps,
	}
}

func (s *CrawlerStrategy) shouldProcessURL(link, baseURL string, cctx *crawlContext) bool {
	if link == "" {
		return false
	}

	if !utils.IsSameDomain(link, baseURL) {
		return false
	}

	if cctx.opts.FilterURL != "" && !utils.HasBaseURL(link, cctx.opts.FilterURL) {
		return false
	}

	for _, re := range cctx.excludeRegexps {
		if re.MatchString(link) {
			return false
		}
	}

	cctx.mu.Lock()
	if cctx.opts.Limit > 0 && *cctx.processedCount >= cctx.opts.Limit {
		cctx.mu.Unlock()
		return false
	}
	cctx.mu.Unlock()

	if _, exists := cctx.visited.LoadOrStore(link, true); exists {
		return false
	}

	return true
}

func (s *CrawlerStrategy) processMarkdownResponse(body []byte, url string) (*domain.Document, error) {
	doc, err := s.markdownReader.Read(string(body), url)
	if err != nil {
		s.logger.Warn().Err(err).Str("url", url).Msg("Failed to read markdown")
		return nil, err
	}
	return doc, nil
}

func (s *CrawlerStrategy) processHTMLResponse(ctx context.Context, body []byte, url string, opts Options) (*domain.Document, error) {
	html := string(body)

	renderedWithJS := false
	if opts.RenderJS || renderer.NeedsJSRendering(html) {
		if r, err := s.deps.GetRenderer(); err == nil {
			s.renderer = r
			rendered, err := s.renderer.Render(ctx, url, domain.RenderOptions{
				Timeout:     60 * time.Second,
				WaitStable:  2 * time.Second,
				ScrollToEnd: true,
			})
			if err == nil {
				html = rendered
				renderedWithJS = true
			}
		}
	}

	doc, err := s.converter.Convert(ctx, html, url)
	if err != nil {
		s.logger.Warn().Err(err).Str("url", url).Msg("Failed to convert page")
		return nil, err
	}

	doc.RenderedWithJS = renderedWithJS

	return doc, nil
}

func (s *CrawlerStrategy) processResponse(ctx context.Context, r *colly.Response, cctx *crawlContext) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	contentType := r.Headers.Get("Content-Type")
	currentURL := r.Request.URL.String()
	isMarkdown := converter.IsMarkdownContent(contentType, currentURL)
	isHTML := IsHTMLContentType(contentType)

	if !isMarkdown && !isHTML {
		return
	}

	cctx.mu.Lock()
	if cctx.opts.Limit > 0 && *cctx.processedCount >= cctx.opts.Limit {
		cctx.mu.Unlock()
		return
	}
	*cctx.processedCount++
	cctx.mu.Unlock()

	cctx.barMu.Lock()
	cctx.bar.Add(1)
	cctx.barMu.Unlock()

	if !cctx.opts.Force && s.writer.Exists(currentURL) {
		return
	}

	var doc *domain.Document
	var err error

	if isMarkdown {
		doc, err = s.processMarkdownResponse(r.Body, currentURL)
	} else {
		doc, err = s.processHTMLResponse(ctx, r.Body, currentURL, cctx.opts)
	}

	if err != nil || doc == nil {
		return
	}

	if doc.RenderedWithJS && cctx.collector != nil && len(doc.Links) > 0 {
		var queued int
		for _, link := range doc.Links {
			if s.shouldProcessURL(link, cctx.baseURL, cctx) {
				if err := cctx.collector.Visit(link); err == nil {
					queued++
				}
			}
		}
		if queued > 0 {
			s.logger.Debug().
				Int("queued", queued).
				Int("total", len(doc.Links)).
				Str("url", currentURL).
				Msg("Re-injected JS-rendered links into crawl queue")
		}
	}

	doc.SourceStrategy = s.Name()
	doc.FetchedAt = time.Now()

	if s.deps.StateManager != nil {
		s.deps.StateManager.MarkSeen(currentURL)
		if doc.ContentHash != "" && !s.deps.StateManager.ShouldProcess(currentURL, doc.ContentHash) {
			s.logger.Debug().Str("url", currentURL).Msg("Skipping unchanged page")
			return
		}
	}

	if !cctx.opts.DryRun {
		if err := s.deps.WriteDocument(ctx, doc); err != nil {
			s.logger.Warn().Err(err).Str("url", currentURL).Msg("Failed to write document")
		}
	}
}

// NewCrawlerStrategy creates a new crawler strategy
func NewCrawlerStrategy(deps *Dependencies) *CrawlerStrategy {
	if deps == nil {
		return &CrawlerStrategy{}
	}
	return &CrawlerStrategy{
		deps:           deps,
		fetcher:        deps.Fetcher,
		renderer:       deps.Renderer,
		converter:      deps.Converter,
		markdownReader: converter.NewMarkdownReader(),
		writer:         deps.Writer,
		logger:         deps.Logger,
	}
}

// Name returns the strategy name
func (s *CrawlerStrategy) Name() string {
	return "crawler"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *CrawlerStrategy) CanHandle(url string) bool {
	return utils.IsHTTPURL(url)
}

// SetFetcher allows setting a custom fetcher for testing
func (s *CrawlerStrategy) SetFetcher(f domain.Fetcher) {
	s.fetcher = f
}

func (s *CrawlerStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Starting web crawl")

	if opts.FilterURL != "" {
		s.logger.Info().Str("filter", opts.FilterURL).Msg("URL filter active - only crawling URLs under this path")
	}

	cctx := newCrawlContext(ctx, url, opts)

	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(opts.MaxDepth),
	)

	cctx.collector = c

	c.WithTransport(s.fetcher.Transport())

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: opts.Concurrency,
		RandomDelay: 2 * time.Second,
	})

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if s.shouldProcessURL(link, url, cctx) {
			_ = e.Request.Visit(link)
		}
	})

	c.OnResponse(func(r *colly.Response) {
		s.processResponse(ctx, r, cctx)
	})

	c.OnError(func(r *colly.Response, err error) {
		s.logger.Debug().Err(err).Str("url", r.Request.URL.String()).Msg("Request failed")
	})

	if err := c.Visit(url); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		c.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	s.logger.Info().Int("pages", *cctx.processedCount).Msg("Crawl completed")
	return nil
}

// IsHTMLContentType checks if content type is HTML
func IsHTMLContentType(contentType string) bool {
	if contentType == "" {
		return true
	}
	lower := strings.ToLower(contentType)
	return strings.Contains(lower, "text/html") ||
		strings.Contains(lower, "application/xhtml")
}
