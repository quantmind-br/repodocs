package converter

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
	"github.com/rs/zerolog/log"
)

// ErrSelectorNotFound indicates no elements matched the selector.
var ErrSelectorNotFound = errors.New("selector not found")

// ExtractContent extracts the main content from HTML
type ExtractContent struct {
	selector string
}

// ExtractOptions contains options for content extraction
type ExtractOptions struct {
	Selector string // CSS selector for main content
	URL      string // Source URL for resolving relative links
}

// NewExtractContent creates a new content extractor
func NewExtractContent(selector string) *ExtractContent {
	return &ExtractContent{selector: selector}
}

// Extract extracts main content from HTML
func (e *ExtractContent) Extract(html, sourceURL string) (string, string, error) {
	// If a selector is provided, use it directly
	if e.selector != "" {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			return "", "", err
		}

		content, title, err := e.ExtractFromDocument(doc, sourceURL)
		if err == nil {
			return content, title, nil
		}
		if errors.Is(err, ErrSelectorNotFound) {
			return e.extractWithReadability(html, sourceURL)
		}

		return "", "", err
	}

	// Otherwise, use readability algorithm
	return e.extractWithReadability(html, sourceURL)
}

// ExtractFromDocument extracts content using a pre-parsed document.
func (e *ExtractContent) ExtractFromDocument(doc *goquery.Document, sourceURL string) (string, string, error) {
	if e.selector == "" {
		return "", "", fmt.Errorf("extract from document requires selector")
	}

	if doc == nil {
		return "", "", fmt.Errorf("extract from document requires document")
	}

	content := doc.Find(e.selector)
	matchCount := content.Length()

	log.Debug().
		Str("selector", e.selector).
		Int("matches", matchCount).
		Str("url", sourceURL).
		Msg("Content selector applied")

	if matchCount == 0 {
		log.Debug().
			Str("selector", e.selector).
			Str("url", sourceURL).
			Msg("Selector not found, falling back to Readability algorithm")
		return "", "", ErrSelectorNotFound
	}

	title := extractTitle(doc)

	var combined strings.Builder
	content.Each(func(_ int, sel *goquery.Selection) {
		if h, err := sel.Html(); err == nil {
			combined.WriteString(h)
		}
	})

	return combined.String(), title, nil
}

// extractWithSelector extracts content using a CSS selector
func (e *ExtractContent) extractWithSelector(html, sourceURL string) (string, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "", err
	}

	return e.ExtractFromDocument(doc, sourceURL)
}

// extractWithReadability extracts content using the readability algorithm
func (e *ExtractContent) extractWithReadability(html, sourceURL string) (string, string, error) {
	parsedURL, err := url.Parse(sourceURL)
	if err != nil {
		parsedURL = &url.URL{Scheme: "https", Host: "example.com"}
	}

	article, err := readability.FromReader(strings.NewReader(html), parsedURL)
	if err != nil {
		// If readability fails, try to extract the body
		return e.extractBody(html)
	}

	return article.Content, article.Title, nil
}

// extractBody extracts the body content as a fallback
func (e *ExtractContent) extractBody(html string) (string, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html, "", nil
	}

	title := extractTitle(doc)

	// Get body content
	body := doc.Find("body")
	if body.Length() == 0 {
		return html, title, nil
	}

	bodyHTML, err := body.Html()
	if err != nil {
		return html, title, nil
	}

	return bodyHTML, title, nil
}

// extractTitle extracts the page title
func extractTitle(doc *goquery.Document) string {
	// Try <title> tag
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title != "" {
		return title
	}

	// Try <h1> tag
	h1 := strings.TrimSpace(doc.Find("h1").First().Text())
	if h1 != "" {
		return h1
	}

	// Try og:title meta tag
	ogTitle, exists := doc.Find("meta[property='og:title']").Attr("content")
	if exists && ogTitle != "" {
		return ogTitle
	}

	return ""
}

// ExtractDescription extracts the page description
func ExtractDescription(doc *goquery.Document) string {
	// Try meta description
	desc, exists := doc.Find("meta[name='description']").Attr("content")
	if exists && desc != "" {
		return strings.TrimSpace(desc)
	}

	// Try og:description
	ogDesc, exists := doc.Find("meta[property='og:description']").Attr("content")
	if exists && ogDesc != "" {
		return strings.TrimSpace(ogDesc)
	}

	return ""
}

// ExtractHeaders extracts all headers from HTML
func ExtractHeaders(html string) map[string][]string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return map[string][]string{}
	}

	return ExtractHeadersFromDoc(doc)
}

// ExtractHeadersFromDoc extracts headers from a parsed document.
func ExtractHeadersFromDoc(doc *goquery.Document) map[string][]string {
	return extractHeadersFromSelection(doc.Selection)
}

func extractHeadersFromSelection(sel *goquery.Selection) map[string][]string {
	headers := make(map[string][]string)

	for i := 1; i <= 6; i++ {
		tag := string('h') + string('0'+byte(i)) // h1, h2, ..., h6
		findWithRoot(sel, tag).Each(func(_ int, node *goquery.Selection) {
			text := strings.TrimSpace(node.Text())
			if text != "" {
				headers[tag] = append(headers[tag], text)
			}
		})
	}

	return headers
}

// ExtractLinks extracts all links from HTML
func ExtractLinks(html, baseURL string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []string{}
	}

	return ExtractLinksFromDoc(doc, baseURL)
}

// ExtractLinksFromDoc extracts links from a parsed document.
func ExtractLinksFromDoc(doc *goquery.Document, baseURL string) []string {
	return extractLinksFromSelection(doc.Selection, baseURL)
}

func extractLinksFromSelection(sel *goquery.Selection, baseURL string) []string {
	var links []string

	base, _ := url.Parse(baseURL)

	findWithRoot(sel, "a[href]").Each(func(_ int, node *goquery.Selection) {
		href, exists := node.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip anchors, javascript, mailto
		if strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") {
			return
		}

		// Resolve relative URLs
		if base != nil && !strings.HasPrefix(href, "http") {
			refURL, err := url.Parse(href)
			if err == nil {
				href = base.ResolveReference(refURL).String()
			}
		}

		links = append(links, href)
	})

	return links
}

func findWithRoot(sel *goquery.Selection, query string) *goquery.Selection {
	return sel.Filter(query).AddSelection(sel.Find(query))
}
