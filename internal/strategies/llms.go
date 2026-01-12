package strategies

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

type LLMSStrategy struct {
	deps            *Dependencies
	fetcher         domain.Fetcher
	converter       *converter.Pipeline
	markdownReader  *converter.MarkdownReader
	plainTextReader *converter.PlainTextReader
	writer          *output.Writer
	logger          *utils.Logger
}

func NewLLMSStrategy(deps *Dependencies) *LLMSStrategy {
	if deps == nil {
		return &LLMSStrategy{
			markdownReader:  converter.NewMarkdownReader(),
			plainTextReader: converter.NewPlainTextReader(),
		}
	}
	return &LLMSStrategy{
		deps:            deps,
		fetcher:         deps.Fetcher,
		converter:       deps.Converter,
		markdownReader:  converter.NewMarkdownReader(),
		plainTextReader: converter.NewPlainTextReader(),
		writer:          deps.Writer,
		logger:          deps.Logger,
	}
}

// Name returns the strategy name
func (s *LLMSStrategy) Name() string {
	return "llms"
}

// CanHandle returns true if this strategy can handle the given URL
func (s *LLMSStrategy) CanHandle(url string) bool {
	// Only handle HTTP/HTTPS URLs
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return false
	}

	lowerURL := strings.ToLower(url)
	return strings.HasSuffix(lowerURL, "/llms.txt") || strings.HasSuffix(lowerURL, "llms.txt")
}

// Execute runs the LLMS extraction strategy
func (s *LLMSStrategy) Execute(ctx context.Context, url string, opts Options) error {
	// Check context cancellation early
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.fetcher == nil {
		return fmt.Errorf("llms strategy fetcher is nil")
	}
	if s.converter == nil {
		return fmt.Errorf("llms strategy converter is nil")
	}
	if s.writer == nil {
		return fmt.Errorf("llms strategy writer is nil")
	}
	if s.logger == nil {
		return fmt.Errorf("llms strategy logger is nil")
	}

	s.logger.Info().Str("url", url).Msg("Fetching llms.txt")

	if opts.FilterURL != "" {
		s.logger.Info().Str("filter", opts.FilterURL).Msg("URL filter active - only downloading URLs under this path")
	}

	resp, err := s.fetcher.Get(ctx, url)
	if err != nil {
		return err
	}

	links := parseLLMSLinks(string(resp.Body))
	s.logger.Info().Int("count", len(links)).Msg("Found links in llms.txt")

	if opts.FilterURL != "" {
		links = filterLLMSLinks(links, opts.FilterURL)
		s.logger.Info().Int("count", len(links)).Str("filter", opts.FilterURL).Msg("Links after filter")
	}

	if opts.Limit > 0 && len(links) > opts.Limit {
		links = links[:opts.Limit]
	}

	// Create progress bar
	bar := utils.NewProgressBar(len(links), utils.DescExtracting)

	// Process links concurrently
	errors := utils.ParallelForEach(ctx, links, opts.Concurrency, func(ctx context.Context, link domain.LLMSLink) error {
		defer bar.Add(1)

		// Check if already exists
		if !opts.Force && s.writer.Exists(link.URL) {
			return nil
		}

		// Fetch page
		pageResp, err := s.fetcher.Get(ctx, link.URL)
		if err != nil {
			s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to fetch page")
			return nil // Continue with other pages
		}

		var doc *domain.Document
		if converter.IsMarkdownContent(pageResp.ContentType, link.URL) {
			doc, err = s.markdownReader.Read(string(pageResp.Body), link.URL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to read markdown")
				return nil
			}
		} else if converter.IsPlainTextContent(pageResp.ContentType, link.URL) {
			doc, err = s.plainTextReader.Read(string(pageResp.Body), link.URL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to read plain text")
				return nil
			}
		} else {
			doc, err = s.converter.Convert(ctx, string(pageResp.Body), link.URL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to convert page")
				return nil
			}
		}

		// Set metadata
		doc.SourceStrategy = s.Name()
		doc.CacheHit = pageResp.FromCache
		doc.FetchedAt = time.Now()

		// Use title from llms.txt if document title is empty
		if doc.Title == "" && link.Title != "" {
			doc.Title = link.Title
		}

		if !opts.DryRun {
			if s.deps != nil {
				if err := s.deps.WriteDocument(ctx, doc); err != nil {
					s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to write document")
					return nil
				}
			} else {
				if err := s.writer.Write(ctx, doc); err != nil {
					s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to write document")
					return nil
				}
			}
		}

		return nil
	})

	// Check for errors
	if err := utils.FirstError(errors); err != nil {
		return err
	}

	s.logger.Info().Msg("LLMS extraction completed")
	return nil
}

// linkRegex matches markdown links: [Title](url)
var linkRegex = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

func parseLLMSLinks(content string) []domain.LLMSLink {
	links := make([]domain.LLMSLink, 0)

	matches := linkRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) == 3 {
			title := strings.TrimSpace(match[1])
			url := strings.TrimSpace(match[2])

			if url == "" || strings.HasPrefix(url, "#") {
				continue
			}

			links = append(links, domain.LLMSLink{
				Title: title,
				URL:   url,
			})
		}
	}

	return links
}

func filterLLMSLinks(links []domain.LLMSLink, filterURL string) []domain.LLMSLink {
	// Empty filter means no filtering - return all
	if filterURL == "" {
		return links
	}

	filtered := make([]domain.LLMSLink, 0, len(links))
	for _, link := range links {
		// Try HasBaseURL first (works with full URLs)
		if utils.HasBaseURL(link.URL, filterURL) {
			filtered = append(filtered, link)
			continue
		}

		// For path-only filters (e.g., "/docs"), check if URL path contains the filter
		if strings.HasPrefix(filterURL, "/") && strings.Contains(link.URL, filterURL) {
			filtered = append(filtered, link)
		}
	}
	return filtered
}
