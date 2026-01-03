package strategies

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

// GitHubPagesStrategy extracts documentation from GitHub Pages sites (*.github.io)
type GitHubPagesStrategy struct {
	deps           *Dependencies
	fetcher        domain.Fetcher
	renderer       domain.Renderer
	converter      *converter.Pipeline
	markdownReader *converter.MarkdownReader
	writer         *output.Writer
	logger         *utils.Logger
}

// NewGitHubPagesStrategy creates a new GitHub Pages strategy
func NewGitHubPagesStrategy(deps *Dependencies) *GitHubPagesStrategy {
	if deps == nil {
		return &GitHubPagesStrategy{
			markdownReader: converter.NewMarkdownReader(),
		}
	}
	return &GitHubPagesStrategy{
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
func (s *GitHubPagesStrategy) Name() string {
	return "github_pages"
}

// CanHandle returns true if URL is a GitHub Pages site
func (s *GitHubPagesStrategy) CanHandle(rawURL string) bool {
	return IsGitHubPagesURL(rawURL)
}

// IsGitHubPagesURL checks if a URL is a GitHub Pages site
func IsGitHubPagesURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Host)
	return strings.HasSuffix(host, ".github.io")
}

// Execute runs the GitHub Pages extraction strategy
func (s *GitHubPagesStrategy) Execute(ctx context.Context, inputURL string, opts Options) error {
	s.logger.Info().
		Str("url", inputURL).
		Msg("Starting GitHub Pages extraction")

	// Normalize base URL
	baseURL, err := s.normalizeBaseURL(inputURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Phase 1: Discover URLs (HTTP-first, browser fallback)
	urls, discoveryMethod, err := s.discoverURLs(ctx, baseURL, opts)
	if err != nil {
		return fmt.Errorf("URL discovery failed: %w", err)
	}

	if len(urls) == 0 {
		s.logger.Warn().Msg("No URLs discovered")
		return nil
	}

	s.logger.Info().
		Int("count", len(urls)).
		Str("method", discoveryMethod).
		Msg("URLs discovered")

	// Filter and deduplicate URLs
	urls = FilterAndDeduplicateURLs(urls, baseURL)

	// Apply filters and limits
	urls = s.filterURLs(urls, baseURL, opts)
	if opts.Limit > 0 && len(urls) > opts.Limit {
		urls = urls[:opts.Limit]
	}

	s.logger.Info().
		Int("count", len(urls)).
		Msg("Processing URLs")

	// Phase 2: Extract content (HTTP-first, browser fallback)
	return s.processURLs(ctx, urls, opts)
}

// discoverURLs finds all URLs using multi-tier discovery
func (s *GitHubPagesStrategy) discoverURLs(ctx context.Context, baseURL string, opts Options) ([]string, string, error) {
	// Tier 1: Try HTTP probes sequentially
	urls, method, err := s.discoverViaHTTPProbes(ctx, baseURL)
	if err == nil && len(urls) > 0 {
		return urls, method, nil
	}

	s.logger.Debug().Err(err).Msg("HTTP discovery failed, falling back to browser crawl")

	// Tier 2: Browser-based discovery (requires renderer)
	if s.renderer == nil {
		return nil, "", fmt.Errorf("no URLs found via HTTP probes and browser renderer is not available")
	}

	urls, err = s.discoverViaBrowser(ctx, baseURL, opts)
	if err != nil {
		return nil, "", fmt.Errorf("browser discovery failed: %w", err)
	}

	return urls, "browser-crawl", nil
}

// discoverViaHTTPProbes tries all HTTP-based discovery methods
func (s *GitHubPagesStrategy) discoverViaHTTPProbes(ctx context.Context, baseURL string) ([]string, string, error) {
	probes := GetDiscoveryProbes()

	for _, probe := range probes {
		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		default:
		}

		probeURL := strings.TrimSuffix(baseURL, "/") + probe.Path

		resp, err := s.fetcher.Get(ctx, probeURL)
		if err != nil {
			s.logger.Debug().Str("probe", probe.Name).Str("url", probeURL).Err(err).Msg("Probe failed")
			continue
		}

		if resp.StatusCode != 200 {
			s.logger.Debug().Str("probe", probe.Name).Int("status", resp.StatusCode).Msg("Probe returned non-200")
			continue
		}

		urls, err := probe.Parser(resp.Body, baseURL)
		if err != nil {
			s.logger.Debug().Str("probe", probe.Name).Err(err).Msg("Failed to parse probe response")
			continue
		}

		if len(urls) > 0 {
			s.logger.Info().
				Str("probe", probe.Name).
				Int("urls", len(urls)).
				Msg("Discovery probe succeeded")
			return urls, probe.Name, nil
		}
	}

	return nil, "", fmt.Errorf("all HTTP probes failed")
}

// discoverViaBrowser uses browser rendering to crawl and discover URLs
func (s *GitHubPagesStrategy) discoverViaBrowser(ctx context.Context, baseURL string, opts Options) ([]string, error) {
	visited := make(map[string]bool)
	toVisit := []string{baseURL}
	var discovered []string

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}

	for depth := 0; depth < maxDepth && len(toVisit) > 0; depth++ {
		var nextLevel []string

		for _, pageURL := range toVisit {
			if visited[pageURL] {
				continue
			}
			visited[pageURL] = true
			discovered = append(discovered, pageURL)

			// Check limit during discovery
			if opts.Limit > 0 && len(discovered) >= opts.Limit*2 {
				return discovered, nil
			}

			// Render page and extract links
			links, err := s.extractLinksFromRenderedPage(ctx, pageURL, baseURL)
			if err != nil {
				s.logger.Debug().Err(err).Str("url", pageURL).Msg("Failed to extract links")
				continue
			}

			for _, link := range links {
				if !visited[link] {
					nextLevel = append(nextLevel, link)
				}
			}
		}

		toVisit = nextLevel
		s.logger.Debug().Int("depth", depth+1).Int("queued", len(toVisit)).Msg("Browser crawl depth completed")
	}

	return discovered, nil
}

// extractLinksFromRenderedPage renders a page and extracts internal links
func (s *GitHubPagesStrategy) extractLinksFromRenderedPage(ctx context.Context, pageURL, baseURL string) ([]string, error) {
	html, err := s.renderPage(ctx, pageURL)
	if err != nil {
		return nil, err
	}

	return s.extractLinksWithGoquery(html, baseURL)
}

// extractLinksWithGoquery uses goquery for robust HTML parsing
func (s *GitHubPagesStrategy) extractLinksWithGoquery(html, baseURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	parsedBase, _ := url.Parse(baseURL)
	baseHost := parsedBase.Host

	seen := make(map[string]bool)
	var links []string

	// Extract from navigation elements first (most relevant)
	selectors := []string{
		"nav a[href]",
		"[role='navigation'] a[href]",
		".sidebar a[href]",
		".menu a[href]",
		".nav a[href]",
		".toc a[href]",
		"a[href]",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			// Skip non-navigable links
			if strings.HasPrefix(href, "#") ||
				strings.HasPrefix(href, "javascript:") ||
				strings.HasPrefix(href, "mailto:") ||
				strings.HasPrefix(href, "tel:") {
				return
			}

			// Resolve relative URLs
			resolved, err := url.Parse(href)
			if err != nil {
				return
			}

			if !resolved.IsAbs() {
				resolved = parsedBase.ResolveReference(resolved)
			}

			// Filter to same host
			if resolved.Host != baseHost {
				return
			}

			// Normalize: remove fragment, trailing slash
			resolved.Fragment = ""
			normalized := resolved.String()
			normalized = strings.TrimSuffix(normalized, "/")

			if !seen[normalized] && !ShouldSkipGitHubPagesURL(normalized) {
				seen[normalized] = true
				links = append(links, normalized)
			}
		})

		// If we found links in nav elements, prefer those
		if len(links) > 10 && selector != "a[href]" {
			break
		}
	}

	return links, nil
}

// renderPage renders a page using the browser
func (s *GitHubPagesStrategy) renderPage(ctx context.Context, pageURL string) (string, error) {
	return s.renderer.Render(ctx, pageURL, domain.RenderOptions{
		Timeout:     90 * time.Second,
		WaitStable:  3 * time.Second,
		ScrollToEnd: true,
	})
}

// filterURLs applies filter and exclude patterns
func (s *GitHubPagesStrategy) filterURLs(urls []string, baseURL string, opts Options) []string {
	var excludeRegexps []*regexp.Regexp
	for _, pattern := range opts.Exclude {
		if re, err := regexp.Compile(pattern); err == nil {
			excludeRegexps = append(excludeRegexps, re)
		}
	}

	var filtered []string
	for _, u := range urls {
		// Apply base URL filter
		if opts.FilterURL != "" && !strings.HasPrefix(u, opts.FilterURL) {
			continue
		}

		// Apply exclude patterns
		excluded := false
		for _, re := range excludeRegexps {
			if re.MatchString(u) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// Skip non-content URLs
		if ShouldSkipGitHubPagesURL(u) {
			continue
		}

		filtered = append(filtered, u)
	}

	return filtered
}

// processURLs processes all URLs using HTTP-first extraction with browser fallback
func (s *GitHubPagesStrategy) processURLs(ctx context.Context, urls []string, opts Options) error {
	bar := progressbar.NewOptions(len(urls),
		progressbar.OptionSetDescription("Extracting"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Limit browser concurrency for stability
	concurrency := opts.Concurrency
	if concurrency > 5 {
		concurrency = 5
	}
	if concurrency <= 0 {
		concurrency = 2
	}

	var mu sync.Mutex
	var processedCount, successCount int

	errors := utils.ParallelForEach(ctx, urls, concurrency, func(ctx context.Context, pageURL string) error {
		defer func() {
			mu.Lock()
			bar.Add(1)
			processedCount++
			mu.Unlock()
		}()

		// Check if already exists
		if !opts.Force && s.writer.Exists(pageURL) {
			return nil
		}

		// HTTP-first fetch with browser fallback
		html, usedBrowser, err := s.fetchOrRenderPage(ctx, pageURL, opts)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to fetch/render page")
			return nil
		}

		// Validate content
		if s.isEmptyOrErrorContent(html) {
			s.logger.Debug().Str("url", pageURL).Msg("Empty or error content, skipping")
			return nil
		}

		// Convert HTML to document
		doc, err := s.converter.Convert(ctx, html, pageURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to convert page")
			return nil
		}

		// Validate converted content
		if len(strings.TrimSpace(doc.Content)) < 50 {
			s.logger.Debug().Str("url", pageURL).Msg("Converted content too short, skipping")
			return nil
		}

		// Set metadata
		doc.SourceStrategy = s.Name()
		doc.FetchedAt = time.Now()
		doc.RenderedWithJS = usedBrowser

		// Write document
		if !opts.DryRun {
			if err := s.deps.WriteDocument(ctx, doc); err != nil {
				s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to write document")
				return nil
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}

		return nil
	})

	if err := utils.FirstError(errors); err != nil {
		return err
	}

	s.logger.Info().
		Int("processed", processedCount).
		Int("success", successCount).
		Int("total", len(urls)).
		Msg("GitHub Pages extraction completed")

	return nil
}

// fetchOrRenderPage attempts HTTP fetch first, falls back to browser rendering if needed
func (s *GitHubPagesStrategy) fetchOrRenderPage(ctx context.Context, pageURL string, opts Options) (string, bool, error) {
	// Try HTTP fetch first (unless RenderJS is forced)
	if !opts.RenderJS {
		resp, err := s.fetcher.Get(ctx, pageURL)
		if err == nil && resp.StatusCode == 200 {
			html := string(resp.Body)

			// Check if content looks like a valid page (not SPA shell)
			if !s.looksLikeSPAShell(html) && !renderer.NeedsJSRendering(html) {
				return html, false, nil
			}

			s.logger.Debug().Str("url", pageURL).Msg("Content appears to be SPA shell, using browser")
		}
	}

	// Fall back to browser rendering
	if s.renderer == nil {
		return "", false, fmt.Errorf("browser renderer not available")
	}

	html, err := s.renderPage(ctx, pageURL)
	if err != nil {
		return "", false, err
	}

	return html, true, nil
}

// looksLikeSPAShell checks if HTML looks like an empty SPA shell
func (s *GitHubPagesStrategy) looksLikeSPAShell(html string) bool {
	// Check for minimal content indicators
	if len(html) < 500 {
		return true
	}

	lower := strings.ToLower(html)

	// Check for empty body or just script tags
	spaIndicators := []string{
		`<div id="app"></div>`,
		`<div id="root"></div>`,
		`<div id="__next"></div>`,
		`<div id="__nuxt"></div>`,
		"<body></body>",
		"<body> </body>",
	}

	for _, indicator := range spaIndicators {
		if strings.Contains(lower, indicator) {
			// Check if there's actual content after the indicator
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
			if err != nil {
				return true
			}

			bodyText := strings.TrimSpace(doc.Find("body").Text())
			if len(bodyText) < 100 {
				return true
			}
		}
	}

	return false
}

// isEmptyOrErrorContent checks if rendered content is empty or an error page
func (s *GitHubPagesStrategy) isEmptyOrErrorContent(html string) bool {
	if len(html) < 300 {
		return true
	}

	lower := strings.ToLower(html)
	errorIndicators := []string{
		"301 moved permanently",
		"302 found",
		"404 not found",
		"page not found",
		"access denied",
		"403 forbidden",
	}

	for _, indicator := range errorIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	// Check for minimal content (just boilerplate)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return true
	}

	// Check if body has meaningful text content
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	if len(bodyText) < 60 {
		return true
	}

	return false
}

// normalizeBaseURL normalizes the input URL to a base URL
func (s *GitHubPagesStrategy) normalizeBaseURL(inputURL string) (string, error) {
	parsed, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	// Keep the path for project subpaths (e.g., /goose/)
	base := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	if parsed.Path != "" && parsed.Path != "/" {
		// Clean path: remove trailing slash, keep structure
		path := strings.TrimSuffix(parsed.Path, "/")
		base += path
	}

	return base, nil
}
