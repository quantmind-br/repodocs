package converter

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
)

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
		return e.extractWithSelector(html, sourceURL)
	}

	// Otherwise, use readability algorithm
	return e.extractWithReadability(html, sourceURL)
}

// extractWithSelector extracts content using a CSS selector
func (e *ExtractContent) extractWithSelector(html, sourceURL string) (string, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "", err
	}

	// Find the content element
	content := doc.Find(e.selector).First()
	if content.Length() == 0 {
		// Fallback to readability if selector doesn't match
		return e.extractWithReadability(html, sourceURL)
	}

	// Get title
	title := extractTitle(doc)

	// Get content HTML
	contentHTML, err := content.Html()
	if err != nil {
		return "", "", err
	}

	return contentHTML, title, nil
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
	headers := make(map[string][]string)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return headers
	}

	for i := 1; i <= 6; i++ {
		tag := string('h') + string('0'+byte(i)) // h1, h2, ..., h6
		doc.Find(tag).Each(func(_ int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if text != "" {
				headers[tag] = append(headers[tag], text)
			}
		})
	}

	return headers
}

// ExtractLinks extracts all links from HTML
func ExtractLinks(html, baseURL string) []string {
	var links []string

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return links
	}

	base, _ := url.Parse(baseURL)

	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
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
