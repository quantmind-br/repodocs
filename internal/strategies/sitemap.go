package strategies

import (
	"compress/gzip"
	"context"
	"encoding/xml"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs/internal/converter"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/output"
	"github.com/quantmind-br/repodocs/internal/renderer"
	"github.com/quantmind-br/repodocs/internal/utils"
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

	// Apply URL filter
	if opts.FilterURL != "" {
		urls = filterSitemapURLs(urls, opts.FilterURL)
		s.logger.Info().Str("filter", opts.FilterURL).Int("filtered_count", len(urls)).Msg("URLs after filter")
	}

	s.logger.Info().Int("count", len(urls)).Msg("Processing URLs from sitemap")

	return s.processURLs(ctx, urls, opts)
}

// processSitemapIndex processes a sitemap index file batch-by-batch.
// Each nested sitemap's URLs are processed immediately before fetching the next sitemap.
func (s *SitemapStrategy) processSitemapIndex(ctx context.Context, sitemap *domain.Sitemap, opts Options) error {
	s.logger.Info().Int("count", len(sitemap.Sitemaps)).Msg("Processing sitemap index")

	// Log filter if set
	if opts.FilterURL != "" {
		s.logger.Info().Str("filter", opts.FilterURL).Msg("URL filter active - only processing URLs under this path")
	}

	// Process each nested sitemap batch-by-batch
	totalProcessed := 0
	for _, sitemapURL := range sitemap.Sitemaps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		urls, err := s.collectURLsFromSitemap(ctx, sitemapURL, opts.FilterURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL).Msg("Failed to fetch nested sitemap")
			continue
		}

		if len(urls) == 0 {
			continue
		}

		// Apply remaining limit for this batch
		if opts.Limit > 0 {
			remaining := opts.Limit - totalProcessed
			if remaining <= 0 {
				break
			}
			if len(urls) > remaining {
				urls = urls[:remaining]
			}
		}

		s.logger.Info().Int("count", len(urls)).Str("sitemap", sitemapURL).Msg("Processing URLs from nested sitemap")

		// Process immediately
		if err := s.processURLs(ctx, urls, opts); err != nil {
			return err
		}
		totalProcessed += len(urls)
	}

	if totalProcessed == 0 {
		s.logger.Warn().Msg("No URLs found in sitemap index")
	}

	return nil
}

// collectURLsFromSitemap fetches and parses a sitemap, returning its URLs.
// For sitemap indexes, it recursively collects URLs from all nested sitemaps.
func (s *SitemapStrategy) collectURLsFromSitemap(ctx context.Context, url string, filterURL string) ([]domain.SitemapURL, error) {
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

			urls, err := s.collectURLsFromSitemap(ctx, nestedURL, filterURL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", nestedURL).Msg("Failed to fetch nested sitemap")
				continue
			}
			allURLs = append(allURLs, urls...)
		}
		// Apply filter after collecting from all nested sitemaps
		if filterURL != "" {
			allURLs = filterSitemapURLs(allURLs, filterURL)
		}
		return allURLs, nil
	}

	// Apply filter to URLs from this sitemap
	if filterURL != "" {
		sitemap.URLs = filterSitemapURLs(sitemap.URLs, filterURL)
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

// parseSitemap parses sitemap content. Supports XML urlset, XML sitemapindex,
// and the plain-text variant of the Sitemaps protocol (one URL per line).
func parseSitemap(content []byte, sourceURL string) (*domain.Sitemap, error) {
	// Plain-text sitemaps are advertised with a .txt extension. Skip XML
	// parsing entirely — `encoding/xml` returns io.EOF on non-XML input,
	// which previously bubbled up as "strategy execution failed: EOF".
	if strings.HasSuffix(strings.ToLower(sourceURL), ".txt") {
		return parseTextSitemap(content, sourceURL)
	}

	// Try to parse as sitemap index first
	var index sitemapIndexXML
	if indexErr := xml.Unmarshal(content, &index); indexErr == nil && len(index.Sitemaps) > 0 {
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
		// XML parsing failed — fall back to text sitemap if the content
		// looks like a URL list (served with the wrong extension/content-type).
		if looksLikeTextSitemap(content) {
			return parseTextSitemap(content, sourceURL)
		}
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

// parseTextSitemap parses a plain-text sitemap: one URL per line, with
// blank lines and `#` comments ignored. This is part of the Sitemaps
// protocol (https://www.sitemaps.org/protocol.html#otherformats).
func parseTextSitemap(content []byte, sourceURL string) (*domain.Sitemap, error) {
	urls := make([]domain.SitemapURL, 0)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") {
			continue
		}
		urls = append(urls, domain.SitemapURL{Loc: line})
	}
	return &domain.Sitemap{
		URLs:      urls,
		IsIndex:   false,
		SourceURL: sourceURL,
	}, nil
}

// looksLikeTextSitemap reports whether content appears to be a plain-text
// URL list rather than XML. It requires at least one non-comment, non-empty
// line beginning with http:// or https://.
func looksLikeTextSitemap(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			return true
		}
		// First non-blank, non-comment line isn't a URL — not a text sitemap.
		return false
	}
	return false
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

// filterSitemapURLs filters URLs based on the provided filter URL.
// Only URLs that have the filter URL as a prefix are included.
func filterSitemapURLs(urls []domain.SitemapURL, filterURL string) []domain.SitemapURL {
	if filterURL == "" {
		return urls
	}

	var filtered []domain.SitemapURL
	for _, u := range urls {
		if strings.HasPrefix(u.Loc, filterURL) {
			filtered = append(filtered, u)
		}
	}
	return filtered
}
