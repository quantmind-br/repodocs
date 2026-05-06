package strategies

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs/internal/converter"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/output"
	"github.com/quantmind-br/repodocs/internal/utils"
)

// PkgGoStrategy extracts documentation from pkg.go.dev
type PkgGoStrategy struct {
	deps      *Dependencies
	fetcher   domain.Fetcher
	converter *converter.Pipeline
	writer    *output.Writer
	logger    *utils.Logger
}

// NewPkgGoStrategy creates a new pkg.go.dev strategy
func NewPkgGoStrategy(deps *Dependencies) *PkgGoStrategy {
	if deps == nil {
		return &PkgGoStrategy{}
	}
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

// SetFetcher allows setting a custom fetcher (used for testing)
func (s *PkgGoStrategy) SetFetcher(f domain.Fetcher) {
	s.fetcher = f
}

// CanHandle returns true if this strategy can handle the given URL
func (s *PkgGoStrategy) CanHandle(url string) bool {
	return strings.Contains(url, "pkg.go.dev")
}

// Execute runs the pkg.go.dev extraction strategy
func (s *PkgGoStrategy) Execute(ctx context.Context, url string, opts Options) (*domain.StrategyResult, error) {
	result := domain.NewStrategyResult(s.Name(), url)
	err := s.execute(ctx, url, opts, result)
	result.Finish()
	return result, err
}

func (s *PkgGoStrategy) execute(ctx context.Context, url string, opts Options, result *domain.StrategyResult) error {
	// Check context cancellation early
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.fetcher == nil {
		return fmt.Errorf("pkggo strategy fetcher is nil")
	}
	if s.converter == nil {
		return fmt.Errorf("pkggo strategy converter is nil")
	}
	if s.writer == nil {
		return fmt.Errorf("pkggo strategy writer is nil")
	}
	if s.logger == nil {
		return fmt.Errorf("pkggo strategy logger is nil")
	}

	s.logger.Info().Str("url", url).Msg("Fetching pkg.go.dev documentation")

	// Fetch page
	resp, err := s.fetcher.Get(ctx, url)
	if err != nil {
		result.IncFailed()
		return err
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(resp.Body)))
	if err != nil {
		result.IncFailed()
		return err
	}

	// Extract package name
	packageName := doc.Find("h1.UnitHeader-title").First().Text()
	packageName = strings.TrimSpace(packageName)

	// If split option is enabled, extract sections separately
	if opts.Split {
		return s.extractSections(ctx, doc, url, packageName, opts, result)
	}

	// Extract main documentation content
	content := doc.Find("div.Documentation-content").First()
	if content.Length() == 0 {
		// Fallback to main content area
		content = doc.Find("main").First()
	}

	if content.Length() == 0 {
		result.AddDiagnostic(domain.DiagNoDocuments,
			"No documentation content found on pkg.go.dev page",
			"The page may not contain a Documentation-content div or main element")
		return nil
	}

	contentHTML, err := content.Html()
	if err != nil {
		result.IncFailed()
		return err
	}

	result.IncDiscovered()
	result.IncAttempted()

	// Convert to document
	document, err := s.converter.Convert(ctx, contentHTML, url)
	if err != nil {
		result.IncFailed()
		return err
	}

	// Set metadata
	document.Title = packageName
	document.SourceStrategy = s.Name()
	document.CacheHit = resp.FromCache
	document.FetchedAt = time.Now()

	if !opts.DryRun {
		if s.deps != nil {
			if err := s.deps.WriteDocument(ctx, document); err != nil {
				result.IncFailed()
				return err
			}
			result.IncWritten()
			result.AddBytesWritten(int64(len(document.Content)))
			return nil
		}
		if err := s.writer.Write(ctx, document); err != nil {
			result.IncFailed()
			return err
		}
		result.IncWritten()
		result.AddBytesWritten(int64(len(document.Content)))
		return nil
	}

	return nil
}

// extractSections extracts documentation split by sections
func (s *PkgGoStrategy) extractSections(ctx context.Context, doc *goquery.Document, baseURL, packageName string, opts Options, result *domain.StrategyResult) error {
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

	// Count available sections first
	var availableSections int
	for _, section := range sections {
		content := doc.Find(section.selector).First()
		if content.Length() > 0 {
			sectionHTML, _ := content.Html()
			if strings.TrimSpace(sectionHTML) != "" {
				availableSections++
			}
		}
	}

	if availableSections == 0 {
		result.AddDiagnostic(domain.DiagNoDocuments,
			"No documentation sections found on pkg.go.dev page",
			"The page may have a different structure than expected")
		return nil
	}

	result.AddDiscovered(availableSections)
	result.AddAttempted(availableSections)

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
			result.IncFailed()
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
			result.IncFailed()
			s.logger.Warn().Err(err).Str("section", section.name).Msg("Failed to convert section")
			continue
		}

		// Set metadata
		document.Title = packageName + " - " + section.name
		document.SourceStrategy = s.Name()
		document.FetchedAt = time.Now()

		if !opts.DryRun {
			if s.deps != nil {
				if err := s.deps.WriteDocument(ctx, document); err != nil {
					result.IncFailed()
					s.logger.Warn().Err(err).Str("section", section.name).Msg("Failed to write section")
					continue
				}
			} else {
				if err := s.writer.Write(ctx, document); err != nil {
					result.IncFailed()
					s.logger.Warn().Err(err).Str("section", section.name).Msg("Failed to write section")
					continue
				}
			}
			result.IncWritten()
			result.AddBytesWritten(int64(len(document.Content)))
		}
	}

	s.logger.Info().Msg("pkg.go.dev extraction completed")
	return nil
}
