package strategies

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
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

// Execute runs the crawler extraction strategy
func (s *CrawlerStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Starting web crawl")

	// Log filter if set
	if opts.FilterURL != "" {
		s.logger.Info().Str("filter", opts.FilterURL).Msg("URL filter active - only crawling URLs under this path")
	}

	// Create visited URL tracker
	visited := sync.Map{}
	var processedCount int
	var mu sync.Mutex

	// Compile exclude patterns
	var excludeRegexps []*regexp.Regexp
	for _, pattern := range opts.Exclude {
		if re, err := regexp.Compile(pattern); err == nil {
			excludeRegexps = append(excludeRegexps, re)
		}
	}

	// Create colly collector
	c := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(opts.MaxDepth),
	)

	// Set transport from fetcher for stealth
	c.WithTransport(s.fetcher.Transport())

	// Configure rate limiting
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: opts.Concurrency,
		RandomDelay: 2 * time.Second,
	})

	// Create progress bar (unknown total - uses spinner mode)
	bar := utils.NewProgressBar(-1, utils.DescExtracting)
	var barMu sync.Mutex

	// Handle links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link == "" {
			return
		}

		// Check if within same domain
		if !utils.IsSameDomain(link, url) {
			return
		}

		// Check base URL filter - only crawl URLs that start with the filter path
		if opts.FilterURL != "" && !utils.HasBaseURL(link, opts.FilterURL) {
			return
		}

		// Check exclude patterns
		for _, re := range excludeRegexps {
			if re.MatchString(link) {
				return
			}
		}

		// Check limit
		mu.Lock()
		if opts.Limit > 0 && processedCount >= opts.Limit {
			mu.Unlock()
			return
		}
		mu.Unlock()

		// Check if already visited
		if _, exists := visited.LoadOrStore(link, true); exists {
			return
		}

		// Visit the link
		_ = e.Request.Visit(link)
	})

	// Handle page responses
	c.OnResponse(func(r *colly.Response) {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		contentType := r.Headers.Get("Content-Type")
		currentURL := r.Request.URL.String()
		isMarkdown := converter.IsMarkdownContent(contentType, currentURL)
		isHTML := isHTMLContentType(contentType)

		if !isMarkdown && !isHTML {
			return
		}

		mu.Lock()
		if opts.Limit > 0 && processedCount >= opts.Limit {
			mu.Unlock()
			return
		}
		processedCount++
		mu.Unlock()

		barMu.Lock()
		bar.Add(1)
		barMu.Unlock()

		if !opts.Force && s.writer.Exists(currentURL) {
			return
		}

		var doc *domain.Document
		var err error

		if isMarkdown {
			doc, err = s.markdownReader.Read(string(r.Body), currentURL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", currentURL).Msg("Failed to read markdown")
				return
			}
		} else {
			html := string(r.Body)

			if opts.RenderJS || renderer.NeedsJSRendering(html) {
				if r, err := s.deps.GetRenderer(); err == nil {
					s.renderer = r
					rendered, err := s.renderer.Render(ctx, currentURL, domain.RenderOptions{
						Timeout:     60 * time.Second,
						WaitStable:  2 * time.Second,
						ScrollToEnd: true,
					})
					if err == nil {
						html = rendered
					}
				}
			}

			// Convert to document
			doc, err = s.converter.Convert(ctx, html, currentURL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", currentURL).Msg("Failed to convert page")
				return
			}
		}

		// Set metadata
		doc.SourceStrategy = s.Name()
		doc.FetchedAt = time.Now()

		if !opts.DryRun {
			if err := s.deps.WriteDocument(ctx, doc); err != nil {
				s.logger.Warn().Err(err).Str("url", currentURL).Msg("Failed to write document")
			}
		}
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		s.logger.Debug().Err(err).Str("url", r.Request.URL.String()).Msg("Request failed")
	})

	// Start crawling
	if err := c.Visit(url); err != nil {
		return err
	}

	// Handle context cancellation
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

	s.logger.Info().Int("pages", processedCount).Msg("Crawl completed")
	return nil
}

// isHTMLContentType checks if content type is HTML
func isHTMLContentType(contentType string) bool {
	if contentType == "" {
		return true
	}
	lower := strings.ToLower(contentType)
	return strings.Contains(lower, "text/html") ||
		strings.Contains(lower, "application/xhtml")
}
