package strategies

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// SitemapStrategy extracts documentation from sitemap XML files
type SitemapStrategy struct {
	deps           *Dependencies
	fetcher        domain.Fetcher
	renderer       domain.Renderer
	converter      *converter.Pipeline
	markdownReader *converter.MarkdownReader
	writer         *output.Writer
	logger         *utils.Logger
}

// NewSitemapStrategy creates a new sitemap strategy
func NewSitemapStrategy(deps *Dependencies) *SitemapStrategy {
	if deps == nil {
		return &SitemapStrategy{
			markdownReader: converter.NewMarkdownReader(),
		}
	}
	return &SitemapStrategy{
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
func (s *SitemapStrategy) Name() string {
	return "sitemap"
}

// SetFetcher allows setting a custom fetcher (used for testing)
func (s *SitemapStrategy) SetFetcher(f domain.Fetcher) {
	s.fetcher = f
}

// CanHandle returns true if this strategy can handle the given URL
func (s *SitemapStrategy) CanHandle(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, "sitemap.xml") ||
		strings.HasSuffix(lower, "sitemap.xml.gz") ||
		strings.Contains(lower, "sitemap")
}

// Execute runs the sitemap extraction strategy
func (s *SitemapStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Fetching sitemap")

	// Fetch sitemap
	resp, err := s.fetcher.Get(ctx, url)
	if err != nil {
		return err
	}

	// Decompress if gzipped
	content := resp.Body
	if strings.HasSuffix(strings.ToLower(url), ".gz") {
		content, err = decompressGzip(resp.Body)
		if err != nil {
			return err
		}
	}

	// Parse sitemap
	sitemap, err := parseSitemap(content, url)
	if err != nil {
		return err
	}

	// If it's a sitemap index, process each sitemap
	if sitemap.IsIndex {
		return s.processSitemapIndex(ctx, sitemap, opts)
	}

	// Sort by lastmod (most recent first)
	sortURLsByLastMod(sitemap.URLs)

	// Apply limit
	urls := sitemap.URLs
	if opts.Limit > 0 && len(urls) > opts.Limit {
		urls = urls[:opts.Limit]
	}

	s.logger.Info().Int("count", len(urls)).Msg("Processing URLs from sitemap")

	return s.processURLs(ctx, urls, opts)
}

// processSitemapIndex processes a sitemap index file by collecting all URLs first,
// then processing them with a single progress bar for consistent display.
func (s *SitemapStrategy) processSitemapIndex(ctx context.Context, sitemap *domain.Sitemap, opts Options) error {
	s.logger.Info().Int("count", len(sitemap.Sitemaps)).Msg("Processing sitemap index")

	// Collect all URLs from nested sitemaps first
	var allURLs []domain.SitemapURL
	for _, sitemapURL := range sitemap.Sitemaps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		urls, err := s.collectURLsFromSitemap(ctx, sitemapURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL).Msg("Failed to fetch nested sitemap")
			continue
		}
		allURLs = append(allURLs, urls...)
	}

	if len(allURLs) == 0 {
		s.logger.Warn().Msg("No URLs found in sitemap index")
		return nil
	}

	// Sort by lastmod (most recent first)
	sortURLsByLastMod(allURLs)

	// Apply limit
	if opts.Limit > 0 && len(allURLs) > opts.Limit {
		allURLs = allURLs[:opts.Limit]
	}

	s.logger.Info().Int("count", len(allURLs)).Msg("Processing URLs from sitemap index")

	// Process all URLs with a single progress bar
	return s.processURLs(ctx, allURLs, opts)
}

// collectURLsFromSitemap fetches and parses a sitemap, returning its URLs.
// For sitemap indexes, it recursively collects URLs from all nested sitemaps.
func (s *SitemapStrategy) collectURLsFromSitemap(ctx context.Context, url string) ([]domain.SitemapURL, error) {
	resp, err := s.fetcher.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	// Decompress if gzipped
	content := resp.Body
	if strings.HasSuffix(strings.ToLower(url), ".gz") {
		content, err = decompressGzip(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	// Parse sitemap
	sitemap, err := parseSitemap(content, url)
	if err != nil {
		return nil, err
	}

	// If it's a nested index, recursively collect URLs
	if sitemap.IsIndex {
		var allURLs []domain.SitemapURL
		for _, nestedURL := range sitemap.Sitemaps {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			urls, err := s.collectURLsFromSitemap(ctx, nestedURL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", nestedURL).Msg("Failed to fetch nested sitemap")
				continue
			}
			allURLs = append(allURLs, urls...)
		}
		return allURLs, nil
	}

	return sitemap.URLs, nil
}

func (s *SitemapStrategy) processURLs(ctx context.Context, urls []domain.SitemapURL, opts Options) error {
	bar := utils.NewProgressBar(len(urls), utils.DescExtracting)

	errors := utils.ParallelForEach(ctx, urls, opts.Concurrency, func(ctx context.Context, sitemapURL domain.SitemapURL) error {
		defer bar.Add(1)

		if !opts.Force && s.writer.Exists(sitemapURL.Loc) {
			return nil
		}

		pageResp, err := s.fetcher.Get(ctx, sitemapURL.Loc)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to fetch page")
			return nil
		}

		var doc *domain.Document
		if converter.IsMarkdownContent(pageResp.ContentType, sitemapURL.Loc) {
			doc, err = s.markdownReader.Read(string(pageResp.Body), sitemapURL.Loc)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to read markdown")
				return nil
			}
		} else {
			html := string(pageResp.Body)

			if opts.RenderJS || renderer.NeedsJSRendering(html) {
				if r, err := s.deps.GetRenderer(); err == nil {
					s.renderer = r
					rendered, err := s.renderer.Render(ctx, sitemapURL.Loc, domain.RenderOptions{
						Timeout:     60 * time.Second,
						WaitStable:  2 * time.Second,
						ScrollToEnd: true,
					})
					if err == nil {
						html = rendered
					}
				}
			}

			doc, err = s.converter.Convert(ctx, html, sitemapURL.Loc)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to convert page")
				return nil
			}
		}

		doc.SourceStrategy = s.Name()
		doc.CacheHit = pageResp.FromCache
		doc.FetchedAt = time.Now()

		if !opts.DryRun {
			if err := s.deps.WriteDocument(ctx, doc); err != nil {
				s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to write document")
				return nil
			}
		}

		return nil
	})

	if err := utils.FirstError(errors); err != nil {
		return err
	}

	s.logger.Info().Msg("Sitemap extraction completed")
	return nil
}

// sitemapXML represents the XML structure of a sitemap
type sitemapXML struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

// sitemapIndexXML represents the XML structure of a sitemap index
type sitemapIndexXML struct {
	XMLName  xml.Name          `xml:"sitemapindex"`
	Sitemaps []sitemapLocation `xml:"sitemap"`
}

type sitemapLocation struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// parseSitemap parses sitemap XML content
func parseSitemap(content []byte, sourceURL string) (*domain.Sitemap, error) {
	// Try to parse as sitemap index first
	var index sitemapIndexXML
	if err := xml.Unmarshal(content, &index); err == nil && len(index.Sitemaps) > 0 {
		var sitemaps []string
		for _, sm := range index.Sitemaps {
			sitemaps = append(sitemaps, sm.Loc)
		}
		return &domain.Sitemap{
			IsIndex:   true,
			Sitemaps:  sitemaps,
			SourceURL: sourceURL,
		}, nil
	}

	// Parse as regular sitemap
	var sitemap sitemapXML
	if err := xml.Unmarshal(content, &sitemap); err != nil {
		return nil, err
	}

	var urls []domain.SitemapURL
	for _, u := range sitemap.URLs {
		lastMod, _ := parseLastMod(u.LastMod)
		urls = append(urls, domain.SitemapURL{
			Loc:        u.Loc,
			LastMod:    lastMod,
			LastModStr: u.LastMod,
			ChangeFreq: u.ChangeFreq,
		})
	}

	return &domain.Sitemap{
		URLs:      urls,
		IsIndex:   false,
		SourceURL: sourceURL,
	}, nil
}

// parseLastMod parses a lastmod date string
func parseLastMod(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}

// sortURLsByLastMod sorts URLs by lastmod date (most recent first)
func sortURLsByLastMod(urls []domain.SitemapURL) {
	sort.Slice(urls, func(i, j int) bool {
		return urls[i].LastMod.After(urls[j].LastMod)
	})
}

// decompressGzip decompresses gzip content
func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
