package converter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Pipeline orchestrates the HTML to Markdown conversion process
type Pipeline struct {
	sanitizer       *Sanitizer
	extractor       *ExtractContent
	mdConverter     *MarkdownConverter
	excludeSelector string
}

// PipelineOptions contains options for the conversion pipeline
type PipelineOptions struct {
	BaseURL         string
	ContentSelector string
	ExcludeSelector string
}

// NewPipeline creates a new conversion pipeline
func NewPipeline(opts PipelineOptions) *Pipeline {
	sanitizer := NewSanitizer(SanitizerOptions{
		BaseURL:          opts.BaseURL,
		RemoveNavigation: true,
		RemoveComments:   true,
	})

	extractor := NewExtractContent(opts.ContentSelector)

	mdConverter := NewMarkdownConverter(MarkdownOptions{
		Domain:          opts.BaseURL,
		CodeBlockStyle:  "fenced",
		HeadingStyle:    "atx",
		BulletListStyle: "-",
	})

	return &Pipeline{
		sanitizer:       sanitizer,
		extractor:       extractor,
		mdConverter:     mdConverter,
		excludeSelector: opts.ExcludeSelector,
	}
}

// Convert processes HTML content and returns a Document
func (p *Pipeline) Convert(ctx context.Context, html string, sourceURL string) (*domain.Document, error) {
	// Step 1: Convert encoding to UTF-8
	htmlBytes, err := ConvertToUTF8([]byte(html))
	if err != nil {
		return nil, err
	}
	html = string(htmlBytes)

	// Step 2: Extract main content
	content, title, err := p.extractor.Extract(html, sourceURL)
	if err != nil {
		return nil, err
	}

	// Step 2.5: Apply exclusion selector (remove unwanted elements)
	if p.excludeSelector != "" {
		content = p.removeExcluded(content)
	}

	// Step 3: Sanitize HTML
	sanitized, err := p.sanitizer.Sanitize(content)
	if err != nil {
		return nil, err
	}

	// Step 4: Convert to Markdown
	markdown, err := p.mdConverter.Convert(sanitized)
	if err != nil {
		return nil, err
	}

	// Step 5: Extract metadata
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	description := ExtractDescription(doc)
	headers := ExtractHeaders(sanitized)
	links := ExtractLinks(sanitized, sourceURL)

	// Step 6: Calculate statistics
	plainText := StripMarkdown(markdown)
	wordCount := CountWords(plainText)
	charCount := CountChars(plainText)
	contentHash := calculateHash(markdown)

	// Step 7: Build document
	document := &domain.Document{
		URL:            sourceURL,
		Title:          title,
		Description:    description,
		Content:        markdown,
		HTMLContent:    html,
		FetchedAt:      time.Now(),
		ContentHash:    contentHash,
		WordCount:      wordCount,
		CharCount:      charCount,
		Links:          links,
		Headers:        headers,
		RenderedWithJS: false,
		SourceStrategy: "",
		CacheHit:       false,
	}

	return document, nil
}

// calculateHash calculates SHA256 hash of content
func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ConvertHTML is a convenience function for simple HTML to Markdown conversion
func ConvertHTML(html, sourceURL string) (*domain.Document, error) {
	pipeline := NewPipeline(PipelineOptions{
		BaseURL: sourceURL,
	})
	return pipeline.Convert(context.Background(), html, sourceURL)
}

// ConvertHTMLWithSelector converts HTML with a specific content selector
func ConvertHTMLWithSelector(html, sourceURL, selector string) (*domain.Document, error) {
	pipeline := NewPipeline(PipelineOptions{
		BaseURL:         sourceURL,
		ContentSelector: selector,
	})
	return pipeline.Convert(context.Background(), html, sourceURL)
}

// removeExcluded removes elements matching the exclude selector from HTML content
func (p *Pipeline) removeExcluded(html string) string {
	if p.excludeSelector == "" {
		return html
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	doc.Find(p.excludeSelector).Remove()

	result, err := doc.Find("body").Html()
	if err != nil {
		// If body extraction fails, try getting the whole document
		result, err = doc.Html()
		if err != nil {
			return html
		}
	}

	return result
}
