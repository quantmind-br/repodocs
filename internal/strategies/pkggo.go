package strategies

import (
	"context"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// PkgGoStrategy extracts documentation from pkg.go.dev
type PkgGoStrategy struct {
	deps      *Dependencies
	fetcher   *fetcher.Client
	converter *converter.Pipeline
	writer    *output.Writer
	logger    *utils.Logger
}

// NewPkgGoStrategy creates a new pkg.go.dev strategy
func NewPkgGoStrategy(deps *Dependencies) *PkgGoStrategy {
	return &PkgGoStrategy{
		deps:      deps,
		fetcher:   deps.Fetcher,
		converter: deps.Converter,
		writer:    deps.Writer,
		logger:    deps.Logger,
	}
}

// Name returns the strategy name
func (s *PkgGoStrategy) Name() string {
	return "pkggo"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *PkgGoStrategy) CanHandle(url string) bool {
	return strings.Contains(url, "pkg.go.dev")
}

// Execute runs the pkg.go.dev extraction strategy
func (s *PkgGoStrategy) Execute(ctx context.Context, url string, opts Options) error {
	s.logger.Info().Str("url", url).Msg("Fetching pkg.go.dev documentation")

	// Fetch page
	resp, err := s.fetcher.Get(ctx, url)
	if err != nil {
		return err
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		return err
	}

	// Extract package name
	packageName := doc.Find("h1.UnitHeader-title").First().Text()
	packageName = strings.TrimSpace(packageName)

	// If split option is enabled, extract sections separately
	if opts.Split {
		return s.extractSections(ctx, doc, url, packageName, opts)
	}

	// Extract main documentation content
	content := doc.Find("div.Documentation-content").First()
	if content.Length() == 0 {
		// Fallback to main content area
		content = doc.Find("main").First()
	}

	contentHTML, err := content.Html()
	if err != nil {
		return err
	}

	// Convert to document
	document, err := s.converter.Convert(ctx, contentHTML, url)
	if err != nil {
		return err
	}

	// Set metadata
	document.Title = packageName
	document.SourceStrategy = s.Name()
	document.CacheHit = resp.FromCache
	document.FetchedAt = time.Now()

	if !opts.DryRun {
		return s.deps.WriteDocument(ctx, document)
	}

	return nil
}

// extractSections extracts documentation split by sections
func (s *PkgGoStrategy) extractSections(ctx context.Context, doc *goquery.Document, baseURL, packageName string, opts Options) error {
	sections := []struct {
		selector string
		name     string
	}{
		{"#pkg-overview", "Overview"},
		{"#pkg-index", "Index"},
		{"#pkg-constants", "Constants"},
		{"#pkg-variables", "Variables"},
		{"#pkg-functions", "Functions"},
		{"#pkg-types", "Types"},
	}

	for _, section := range sections {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		content := doc.Find(section.selector).First()
		if content.Length() == 0 {
			continue
		}

		// Get section HTML
		sectionHTML, err := content.Html()
		if err != nil {
			continue
		}

		// Skip empty sections
		if strings.TrimSpace(sectionHTML) == "" {
			continue
		}

		// Create section URL
		sectionURL := baseURL + section.selector

		// Convert to document
		document, err := s.converter.Convert(ctx, sectionHTML, sectionURL)
		if err != nil {
			s.logger.Warn().Err(err).Str("section", section.name).Msg("Failed to convert section")
			continue
		}

		// Set metadata
		document.Title = packageName + " - " + section.name
		document.SourceStrategy = s.Name()
		document.FetchedAt = time.Now()

		if !opts.DryRun {
			if err := s.deps.WriteDocument(ctx, document); err != nil {
				s.logger.Warn().Err(err).Str("section", section.name).Msg("Failed to write section")
			}
		}
	}

	s.logger.Info().Msg("pkg.go.dev extraction completed")
	return nil
}
