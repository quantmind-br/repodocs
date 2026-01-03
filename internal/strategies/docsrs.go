package strategies

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/schollz/progressbar/v3"
)

type DocsRSURL struct {
	CrateName    string
	Version      string
	ModulePath   string
	IsCratePage  bool
	IsSourceView bool
}

type DocsRSMetadata struct {
	CrateName  string
	Version    string
	ModulePath string
	ItemType   string
	Title      string
	Stability  string
}

type DocsRSStrategy struct {
	deps      *Dependencies
	fetcher   domain.Fetcher
	converter *converter.Pipeline
	writer    *output.Writer
	logger    *utils.Logger
	baseHost  string // defaults to "docs.rs", can be overridden for testing
}

var (
	docsRSExcludePaths = []string{
		"/src/",
		"/source/",
		"/all.html",
		"/-/rustdoc.static/",
		"/-/static/",
	}

	docsRSExcludeExtensions = []string{
		".js", ".css", ".svg", ".png", ".ico",
		".woff", ".woff2", ".ttf",
	}

	docsRSExcludeFiles = []string{
		"search-index.js", "sidebar-items.js", "crates.js",
		"aliases.js", "source-script.js", "storage.js", "settings.js",
	}
)

func NewDocsRSStrategy(deps *Dependencies) *DocsRSStrategy {
	if deps == nil {
		return &DocsRSStrategy{baseHost: "docs.rs"}
	}
	return &DocsRSStrategy{
		deps:      deps,
		fetcher:   deps.Fetcher,
		converter: deps.Converter,
		writer:    deps.Writer,
		logger:    deps.Logger,
		baseHost:  "docs.rs",
	}
}

func (s *DocsRSStrategy) Name() string {
	return "docsrs"
}

func (s *DocsRSStrategy) SetFetcher(f domain.Fetcher) {
	s.fetcher = f
}

func (s *DocsRSStrategy) SetBaseHost(host string) {
	s.baseHost = host
}

func (s *DocsRSStrategy) parseURL(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPathWithHost(rawURL, s.baseHost)
}

func (s *DocsRSStrategy) CanHandle(rawURL string) bool {
	parsed, err := parseDocsRSPath(rawURL)
	if err != nil {
		return false
	}

	if parsed.IsSourceView {
		return false
	}

	return parsed.CrateName != ""
}

func parseDocsRSPath(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPathWithHost(rawURL, "docs.rs")
}

func parseDocsRSPathWithHost(rawURL, expectedHost string) (*DocsRSURL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(u.Host, expectedHost) {
		return nil, fmt.Errorf("not a docs.rs URL")
	}

	u.Fragment = ""
	u.RawQuery = ""

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return nil, fmt.Errorf("empty path")
	}

	result := &DocsRSURL{}

	if segments[0] == "crate" {
		result.IsCratePage = true
		if len(segments) >= 2 {
			result.CrateName = segments[1]
		}
		if len(segments) >= 3 {
			result.Version = segments[2]
		} else {
			result.Version = "latest"
		}
		if len(segments) >= 4 && (segments[3] == "source" || segments[3] == "src") {
			result.IsSourceView = true
		}
		return result, nil
	}

	for _, seg := range segments {
		if seg == "src" || seg == "source" {
			result.IsSourceView = true
		}
	}

	result.CrateName = segments[0]

	if len(segments) >= 2 {
		result.Version = segments[1]
	} else {
		result.Version = "latest"
	}

	if len(segments) >= 4 {
		result.ModulePath = strings.Join(segments[3:], "/")
	}

	return result, nil
}

func (s *DocsRSStrategy) shouldCrawl(targetURL string, baseInfo *DocsRSURL) bool {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	if !strings.Contains(u.Host, s.baseHost) {
		return false
	}

	path := u.Path

	for _, excluded := range docsRSExcludePaths {
		if strings.Contains(path, excluded) {
			return false
		}
	}

	for _, ext := range docsRSExcludeExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return false
		}
	}

	baseName := filepath.Base(path)
	for _, file := range docsRSExcludeFiles {
		if baseName == file {
			return false
		}
	}

	targetInfo, err := s.parseURL(targetURL)
	if err != nil {
		return false
	}

	if targetInfo.IsSourceView {
		return false
	}

	if targetInfo.CrateName != baseInfo.CrateName {
		return false
	}

	if targetInfo.Version != baseInfo.Version &&
		targetInfo.Version != "latest" &&
		baseInfo.Version != "latest" {
		return false
	}

	return true
}

func (s *DocsRSStrategy) discoverPages(ctx context.Context, startURL string, baseInfo *DocsRSURL, opts Options) ([]string, error) {
	visited := &sync.Map{}
	var pages []string
	var mu sync.Mutex

	queue := []struct {
		url   string
		depth int
	}{{startURL, 0}}

	visited.Store(startURL, true)
	pages = append(pages, startURL)

	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			return pages, ctx.Err()
		default:
		}

		current := queue[0]
		queue = queue[1:]

		if opts.MaxDepth > 0 && current.depth >= opts.MaxDepth {
			continue
		}

		mu.Lock()
		if opts.Limit > 0 && len(pages) >= opts.Limit {
			mu.Unlock()
			break
		}
		mu.Unlock()

		resp, err := s.fetcher.Get(ctx, current.url)
		if err != nil {
			s.logger.Debug().Err(err).Str("url", current.url).Msg("Failed to fetch for discovery")
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
		if err != nil {
			continue
		}

		doc.Find(".sidebar a[href], #main-content a[href]").Each(func(_ int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" {
				return
			}

			if strings.HasPrefix(href, "#") ||
				strings.HasPrefix(href, "javascript:") ||
				strings.HasPrefix(href, "mailto:") {
				return
			}

			absoluteURL, err := utils.ResolveURL(current.url, href)
			if err != nil {
				return
			}

			normalizedURL, _ := utils.NormalizeURLWithoutQuery(absoluteURL)

			if !s.shouldCrawl(normalizedURL, baseInfo) {
				return
			}

			if _, exists := visited.LoadOrStore(normalizedURL, true); exists {
				return
			}

			mu.Lock()
			pages = append(pages, normalizedURL)
			mu.Unlock()

			queue = append(queue, struct {
				url   string
				depth int
			}{normalizedURL, current.depth + 1})
		})
	}

	return pages, nil
}

func (s *DocsRSStrategy) extractMetadata(doc *goquery.Document, baseInfo *DocsRSURL) *DocsRSMetadata {
	meta := &DocsRSMetadata{
		CrateName:  baseInfo.CrateName,
		Version:    baseInfo.Version,
		ModulePath: baseInfo.ModulePath,
	}

	title := doc.Find(".main-heading h1").First().Text()
	meta.Title = strings.TrimSpace(title)

	bodyClass, _ := doc.Find("body").Attr("class")
	switch {
	case strings.Contains(bodyClass, "struct"):
		meta.ItemType = "struct"
	case strings.Contains(bodyClass, "enum"):
		meta.ItemType = "enum"
	case strings.Contains(bodyClass, "trait"):
		meta.ItemType = "trait"
	case strings.Contains(bodyClass, "fn"):
		meta.ItemType = "function"
	case strings.Contains(bodyClass, "mod"):
		meta.ItemType = "module"
	case strings.Contains(bodyClass, "macro"):
		meta.ItemType = "macro"
	case strings.Contains(bodyClass, "type"):
		meta.ItemType = "type"
	case strings.Contains(bodyClass, "constant"):
		meta.ItemType = "constant"
	default:
		meta.ItemType = "page"
	}

	if doc.Find(".portability.nightly-only").Length() > 0 {
		meta.Stability = "nightly"
	} else if doc.Find(".stab.deprecated").Length() > 0 {
		meta.Stability = "deprecated"
	} else if doc.Find(".stab.unstable").Length() > 0 {
		meta.Stability = "unstable"
	} else {
		meta.Stability = "stable"
	}

	return meta
}

func (s *DocsRSStrategy) applyMetadata(doc *domain.Document, meta *DocsRSMetadata) {
	if meta.Title != "" {
		doc.Title = meta.Title
	}

	doc.Description = fmt.Sprintf("crate:%s version:%s type:%s stability:%s",
		meta.CrateName, meta.Version, meta.ItemType, meta.Stability)

	if meta.ModulePath != "" {
		doc.Description += fmt.Sprintf(" path:%s", meta.ModulePath)
	}

	doc.Tags = []string{
		"docs.rs",
		meta.CrateName,
		meta.ItemType,
		meta.Stability,
	}
}

func (s *DocsRSStrategy) processPage(ctx context.Context, pageURL string, baseInfo *DocsRSURL, opts Options) error {
	delay := time.Duration(500+rand.Intn(1000)) * time.Millisecond
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
	}

	if !opts.Force && s.writer.Exists(pageURL) {
		return nil
	}

	resp, err := s.fetcher.Get(ctx, pageURL)
	if err != nil {
		s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to fetch page")
		return nil
	}

	htmlDoc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to parse HTML")
		return nil
	}

	meta := s.extractMetadata(htmlDoc, baseInfo)

	doc, err := s.converter.Convert(ctx, string(resp.Body), pageURL)
	if err != nil {
		s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to convert page")
		return nil
	}

	s.applyMetadata(doc, meta)
	doc.SourceStrategy = s.Name()
	doc.CacheHit = resp.FromCache
	doc.FetchedAt = time.Now()

	if !opts.DryRun {
		if err := s.deps.WriteDocument(ctx, doc); err != nil {
			s.logger.Warn().Err(err).Str("url", pageURL).Msg("Failed to write document")
			return nil
		}
	}

	return nil
}

func (s *DocsRSStrategy) buildStartURL(info *DocsRSURL) string {
	if info.IsCratePage {
		return fmt.Sprintf("https://docs.rs/crate/%s/%s", info.CrateName, info.Version)
	}
	return fmt.Sprintf("https://docs.rs/%s/%s/%s/", info.CrateName, info.Version, info.CrateName)
}

func (s *DocsRSStrategy) Execute(ctx context.Context, rawURL string, opts Options) error {
	s.logger.Info().Str("url", rawURL).Msg("Starting docs.rs extraction")

	if s.fetcher == nil {
		return fmt.Errorf("docsrs strategy fetcher is nil")
	}
	if s.converter == nil {
		return fmt.Errorf("docsrs strategy converter is nil")
	}
	if s.writer == nil {
		return fmt.Errorf("docsrs strategy writer is nil")
	}

	baseInfo, err := s.parseURL(rawURL)
	if err != nil {
		return fmt.Errorf("invalid docs.rs URL: %w", err)
	}

	var startURL string
	if s.baseHost == "docs.rs" {
		startURL = s.buildStartURL(baseInfo)
	} else {
		startURL = rawURL
	}
	s.logger.Info().
		Str("crate", baseInfo.CrateName).
		Str("version", baseInfo.Version).
		Str("start_url", startURL).
		Msg("Parsed docs.rs URL")

	pages, err := s.discoverPages(ctx, startURL, baseInfo, opts)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	s.logger.Info().Int("count", len(pages)).Msg("Discovered pages")

	if opts.Limit > 0 && len(pages) > opts.Limit {
		pages = pages[:opts.Limit]
		s.logger.Info().Int("limit", opts.Limit).Msg("Applied page limit")
	}

	bar := progressbar.NewOptions(len(pages),
		progressbar.OptionSetDescription("Extracting docs.rs"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	errors := utils.ParallelForEach(ctx, pages, opts.Concurrency, func(ctx context.Context, pageURL string) error {
		defer bar.Add(1)
		return s.processPage(ctx, pageURL, baseInfo, opts)
	})

	if err := utils.FirstError(errors); err != nil {
		return err
	}

	s.logger.Info().Int("pages", len(pages)).Msg("docs.rs extraction completed")
	return nil
}

func (s *DocsRSStrategy) ExtractMetadataForTest(doc *goquery.Document, baseInfo *DocsRSURL) *DocsRSMetadata {
	return s.extractMetadata(doc, baseInfo)
}

func (s *DocsRSStrategy) ShouldCrawlForTest(targetURL string, baseInfo *DocsRSURL) bool {
	return s.shouldCrawl(targetURL, baseInfo)
}

func (s *DocsRSStrategy) BuildStartURLForTest(info *DocsRSURL) string {
	return s.buildStartURL(info)
}

func ParseDocsRSPathForTest(rawURL string) (*DocsRSURL, error) {
	return parseDocsRSPath(rawURL)
}
