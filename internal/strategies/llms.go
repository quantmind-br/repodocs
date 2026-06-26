package strategies

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/quantmind-br/repodocs/internal/converter"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/output"
	"github.com/quantmind-br/repodocs/internal/utils"
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
func (s *LLMSStrategy) Execute(ctx context.Context, url string, opts Options) (*domain.StrategyResult, error) {
	result := domain.NewStrategyResult(s.Name(), url)
	err := s.execute(ctx, url, opts, result)
	result.Finish()
	return result, err
}

func (s *LLMSStrategy) execute(ctx context.Context, url string, opts Options, result *domain.StrategyResult) error {
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
		// Failing to fetch the llms.txt source itself is a discovery failure;
		// no document was attempted, so do not inflate DocsFailed.
		return err
	}

	links := parseLLMSLinks(string(resp.Body))

	// Resolve relative URLs against the base URL of the llms.txt file
	for i := range links {
		if !strings.HasPrefix(links[i].URL, "http://") && !strings.HasPrefix(links[i].URL, "https://") {
			resolved, err := utils.ResolveURL(url, links[i].URL)
			if err != nil {
				s.logger.Warn().Err(err).Str("url", links[i].URL).Msg("Failed to resolve relative URL")
				continue
			}
			links[i].URL = resolved
		}
	}

	s.logger.Info().Int("count", len(links)).Msg("Found links in llms.txt")

	if opts.FilterURL != "" {
		links = filterLLMSLinks(links, opts.FilterURL)
		s.logger.Info().Int("count", len(links)).Str("filter", opts.FilterURL).Msg("Links after filter")
	}

	if len(links) == 0 {
		result.AddDiagnostic(domain.DiagNoDocuments,
			"No links discovered in llms.txt",
			"The file may be empty or use an unsupported format")
		return nil
	}

	if opts.Limit > 0 && len(links) > opts.Limit {
		links = links[:opts.Limit]
	}

	result.AddDiscovered(len(links))
	result.AddAttempted(len(links))

	// Create progress bar
	bar := utils.NewProgressBar(len(links), utils.DescExtracting)

	// Process links concurrently
	errors := utils.ParallelForEach(ctx, links, opts.Concurrency, func(ctx context.Context, link domain.LLMSLink) error {
		defer bar.Add(1)

		// Check if already exists
		if !opts.Force && s.writer.Exists(link.URL) {
			result.IncSkipped()
			return nil
		}

		// Fetch page
		pageResp, err := s.fetcher.Get(ctx, link.URL)
		if err != nil {
			result.IncFailed()
			s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to fetch page")
			return nil // Continue with other pages
		}

		var doc *domain.Document
		if converter.IsMarkdownContent(pageResp.ContentType, link.URL) {
			doc, err = s.markdownReader.Read(string(pageResp.Body), link.URL)
			if err != nil {
				result.IncFailed()
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to read markdown")
				return nil
			}
		} else if converter.IsPlainTextContent(pageResp.ContentType, link.URL) {
			doc, err = s.plainTextReader.Read(string(pageResp.Body), link.URL)
			if err != nil {
				result.IncFailed()
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to read plain text")
				return nil
			}
		} else {
			doc, err = s.converter.Convert(ctx, string(pageResp.Body), link.URL)
			if err != nil {
				result.IncFailed()
				s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to convert page")
				return nil
			}
		}

		// Set metadata
		doc.SourceStrategy = s.Name()
		doc.CacheHit = pageResp.FromCache
		doc.FetchedAt = time.Now()

		if doc.Title == "" && link.Title != "" {
			doc.Title = link.Title
		}

		if doc.Description == "" && link.Description != "" {
			doc.Description = link.Description
		}

		if !opts.DryRun {
			if s.deps != nil {
				if err := s.deps.WriteDocument(ctx, doc); err != nil {
					result.IncFailed()
					s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to write document")
					return nil
				}
			} else {
				if err := s.writer.Write(ctx, doc); err != nil {
					result.IncFailed()
					s.logger.Warn().Err(err).Str("url", link.URL).Msg("Failed to write document")
					return nil
				}
			}
			result.IncWritten()
			result.AddBytesWritten(int64(len(doc.Content)))
		}

		return nil
	})

	// Check for errors
	if err := utils.FirstError(errors); err != nil {
		return err
	}

	snap := result.Snapshot()
	if snap.URLsAttempted > 0 && snap.DocsWritten == 0 && snap.DocsSkipped == 0 {
		result.AddDiagnostic(domain.DiagAllFetchesFailed,
			"All link fetch/convert attempts failed",
			"Verify the linked pages are accessible and in supported formats")
	}

	s.logger.Info().Msg("LLMS extraction completed")
	return nil
}

// llms.txt is an emerging convention rather than a strictly standardized format,
// so real-world files use both normal Markdown links and bare parenthesized URLs.
// Keep separate expressions so the parser can preserve titles when present while
// still accepting implementations that publish only URLs plus optional descriptions.
// linkRegex matches markdown links: [Title](url) or [Title](url): description.
// It also captures an optional description after the link.
var linkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)(?::\s*(.*))?`)

// bareLinkRegex matches bare URLs in parentheses without anchor text:
// (url) or (url): description
// These appear in some llms.txt implementations (e.g., Google's format).
var bareLinkRegex = regexp.MustCompile(`^\s*(?:-\s*)?\(([^)]+)\)(?::\s*(.*))?`)

func parseLLMSLinks(content string) []domain.LLMSLink {
	links := make([]domain.LLMSLink, 0)
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if matches := linkRegex.FindStringSubmatch(line); matches != nil {
			title := strings.TrimSpace(matches[1])
			url := strings.TrimSpace(matches[2])
			desc := strings.TrimSpace(matches[3])

			if url == "" || strings.HasPrefix(url, "#") {
				continue
			}
			if seen[url] {
				continue
			}
			seen[url] = true

			links = append(links, domain.LLMSLink{
				Title:       title,
				URL:         url,
				Description: desc,
			})
			continue
		}

		if matches := bareLinkRegex.FindStringSubmatch(line); matches != nil {
			url := strings.TrimSpace(matches[1])
			desc := strings.TrimSpace(matches[2])

			if url == "" || strings.HasPrefix(url, "#") {
				continue
			}
			if seen[url] {
				continue
			}
			seen[url] = true

			title := ""
			if desc != "" {
				title = truncateTitle(desc)
			}

			links = append(links, domain.LLMSLink{
				Title:       title,
				URL:         url,
				Description: desc,
			})
		}
	}

	return links
}

func truncateTitle(desc string) string {
	if len(desc) == 0 {
		return ""
	}
	if idx := strings.Index(desc, "."); idx > 0 && idx < 80 {
		return desc[:idx]
	}
	if len(desc) > 80 {
		return desc[:77] + "..."
	}
	return desc
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
