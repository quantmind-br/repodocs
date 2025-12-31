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
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// SitemapStrategy extracts documentation from sitemap XML files
type SitemapStrategy struct {
	deps      *Dependencies
	fetcher   *fetcher.Client
	renderer  domain.Renderer
	converter *converter.Pipeline
	writer    *output.Writer
	logger    *utils.Logger
}

// NewSitemapStrategy creates a new sitemap strategy
func NewSitemapStrategy(deps *Dependencies) *SitemapStrategy {
	return &SitemapStrategy{
		deps:      deps,
		fetcher:   deps.Fetcher,
		renderer:  deps.Renderer,
		converter: deps.Converter,
		writer:    deps.Writer,
		logger:    deps.Logger,
	}
}

// Name returns the strategy name
func (s *SitemapStrategy) Name() string {
	return "sitemap"
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

	// Create progress bar
	bar := progressbar.NewOptions(len(urls),
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process URLs concurrently
	errors := utils.ParallelForEach(ctx, urls, opts.Concurrency, func(ctx context.Context, sitemapURL domain.SitemapURL) error {
		defer bar.Add(1)

		// Check if already exists
		if !opts.Force && s.writer.Exists(sitemapURL.Loc) {
			return nil
		}

		// Fetch page
		var html string
		var fromCache bool

		pageResp, err := s.fetcher.Get(ctx, sitemapURL.Loc)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to fetch page")
			return nil
		}
		html = string(pageResp.Body)
		fromCache = pageResp.FromCache

		// Check if JS rendering is needed
		if opts.RenderJS || renderer.NeedsJSRendering(html) {
			if s.renderer != nil {
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

		// Convert to document
		doc, err := s.converter.Convert(ctx, html, sitemapURL.Loc)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL.Loc).Msg("Failed to convert page")
			return nil
		}

		// Set metadata
		doc.SourceStrategy = s.Name()
		doc.CacheHit = fromCache
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

// processSitemapIndex processes a sitemap index file
func (s *SitemapStrategy) processSitemapIndex(ctx context.Context, sitemap *domain.Sitemap, opts Options) error {
	s.logger.Info().Int("count", len(sitemap.Sitemaps)).Msg("Processing sitemap index")

	for _, sitemapURL := range sitemap.Sitemaps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.Execute(ctx, sitemapURL, opts); err != nil {
			s.logger.Warn().Err(err).Str("url", sitemapURL).Msg("Failed to process nested sitemap")
		}
	}

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
