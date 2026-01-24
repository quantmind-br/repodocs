package converter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	htmlpkg "golang.org/x/net/html"
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

	// Step 2: Parse original HTML once
	origDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// Step 3: Extract main content
	var contentHTML string
	var title string
	var contentSel *goquery.Selection
	usedSelector := false

	if p.extractor.selector != "" {
		contentHTML, title, err = p.extractor.ExtractFromDocument(origDoc, sourceURL)
		if err != nil {
			if errors.Is(err, ErrSelectorNotFound) {
				contentHTML, title, err = p.extractor.extractWithReadability(html, sourceURL)
			} else {
				return nil, err
			}
		} else {
			usedSelector = true
			contentSel = origDoc.Find(p.extractor.selector)
		}
	} else {
		contentHTML, title, err = p.extractor.extractWithReadability(html, sourceURL)
	}
	if err != nil {
		return nil, err
	}

	// Step 3.5: Apply exclusion selector (remove unwanted elements)
	if p.excludeSelector != "" {
		if usedSelector {
			contentSel = p.removeExcludedFromSelection(contentSel)
		} else {
			contentHTML = p.removeExcluded(contentHTML)
		}
	}

	if usedSelector && contentSel == nil {
		return nil, ErrSelectorNotFound
	}

	// Step 4: Sanitize HTML and extract node for conversion
	var contentNode *goquery.Selection
	var headers map[string][]string
	var links []string

	description := ExtractDescription(origDoc)

	if usedSelector {
		sanitizedSel, selErr := p.sanitizer.SanitizeSelection(contentSel)
		if selErr != nil {
			return nil, selErr
		}

		headers = extractHeadersFromSelection(sanitizedSel)
		links = extractLinksFromSelection(sanitizedSel, sourceURL)
		contentNode = sanitizedSel
	} else {
		contentDoc, docErr := goquery.NewDocumentFromReader(strings.NewReader(contentHTML))
		if docErr != nil {
			return nil, docErr
		}

		sanitizedDoc, docErr := p.sanitizer.SanitizeDocument(contentDoc)
		if docErr != nil {
			return nil, docErr
		}

		headers = ExtractHeadersFromDoc(sanitizedDoc)
		links = ExtractLinksFromDoc(sanitizedDoc, sourceURL)
		contentNode = sanitizedDoc.Selection
	}

	// Step 5: Convert to Markdown using DOM node directly (avoids reparsing)
	var markdown string
	if contentNode != nil && contentNode.Length() > 0 {
		nodes := make([]*htmlpkg.Node, 0, contentNode.Length())
		contentNode.Each(func(_ int, s *goquery.Selection) {
			if node := s.Get(0); node != nil {
				nodes = append(nodes, node)
			}
		})
		markdown, err = p.mdConverter.ConvertNodes(nodes)
		if err != nil {
			return nil, err
		}
	}

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

	_ = p.removeExcludedFromSelection(doc.Selection)

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

func (p *Pipeline) removeExcludedFromSelection(sel *goquery.Selection) *goquery.Selection {
	if p.excludeSelector == "" || sel == nil {
		return sel
	}

	findWithRoot(sel, p.excludeSelector).Remove()
	return sel
}
