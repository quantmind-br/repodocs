package converter

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// TagsToRemove are HTML tags that should be completely removed
var TagsToRemove = []string{
	"script",
	"style",
	"noscript",
	"iframe",
	"object",
	"embed",
	"applet",
	"form",
	"input",
	"button",
	"select",
	"textarea",
	"footer",
	"header",
	"aside",
	"advertisement",
	"banner",
}

// ClassesToRemove are CSS classes that indicate non-content elements
var ClassesToRemove = []string{
	"sidebar",
	"navigation",
	"nav",
	"menu",
	"footer",
	"header",
	"banner",
	"advertisement",
	"ad",
	"social",
	"share",
	"comment",
	"comments",
	"related",
	"recommended",
}

// IDsToRemove are element IDs that indicate non-content elements
var IDsToRemove = []string{
	"sidebar",
	"navigation",
	"nav",
	"menu",
	"footer",
	"header",
	"banner",
	"advertisement",
	"comments",
}

// Sanitizer cleans HTML content for conversion
type Sanitizer struct {
	baseURL          string
	removeNavigation bool
	removeComments   bool
}

// SanitizerOptions contains options for the sanitizer
type SanitizerOptions struct {
	BaseURL          string
	RemoveNavigation bool
	RemoveComments   bool
}

// NewSanitizer creates a new sanitizer
func NewSanitizer(opts SanitizerOptions) *Sanitizer {
	return &Sanitizer{
		baseURL:          opts.BaseURL,
		removeNavigation: opts.RemoveNavigation,
		removeComments:   opts.RemoveComments,
	}
}

// Sanitize cleans HTML content
func (s *Sanitizer) Sanitize(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove unwanted tags
	for _, tag := range TagsToRemove {
		doc.Find(tag).Remove()
	}

	// Remove elements by class
	if s.removeNavigation {
		for _, class := range ClassesToRemove {
			doc.Find("." + class).Remove()
			doc.Find("[class*='" + class + "']").Remove()
		}

		// Remove elements by ID
		for _, id := range IDsToRemove {
			doc.Find("#" + id).Remove()
		}

		doc.Find("nav").Remove()
	}

	// Remove hidden elements
	doc.Find("[style*='display:none']").Remove()
	doc.Find("[style*='display: none']").Remove()
	doc.Find("[hidden]").Remove()

	// Normalize URLs if base URL is provided
	if s.baseURL != "" {
		s.normalizeURLs(doc)
	}

	// Remove empty paragraphs and divs
	s.removeEmptyElements(doc)

	// Get cleaned HTML
	result, err := doc.Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

// normalizeURLs converts relative URLs to absolute URLs
func (s *Sanitizer) normalizeURLs(doc *goquery.Document) {
	base, err := url.Parse(s.baseURL)
	if err != nil {
		return
	}

	// Normalize href attributes
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			if absoluteURL := resolveURL(base, href); absoluteURL != "" {
				sel.SetAttr("href", absoluteURL)
			}
		}
	})

	// Normalize src attributes
	doc.Find("[src]").Each(func(_ int, sel *goquery.Selection) {
		if src, exists := sel.Attr("src"); exists {
			if absoluteURL := resolveURL(base, src); absoluteURL != "" {
				sel.SetAttr("src", absoluteURL)
			}
		}
	})

	// Normalize srcset attributes
	doc.Find("[srcset]").Each(func(_ int, sel *goquery.Selection) {
		if srcset, exists := sel.Attr("srcset"); exists {
			sel.SetAttr("srcset", normalizeSrcset(base, srcset))
		}
	})
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base *url.URL, ref string) string {
	// Skip empty, fragment, javascript, mailto, and data URLs
	if ref == "" || strings.HasPrefix(ref, "#") ||
		strings.HasPrefix(ref, "javascript:") ||
		strings.HasPrefix(ref, "mailto:") ||
		strings.HasPrefix(ref, "data:") {
		return ref
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}

	return base.ResolveReference(refURL).String()
}

// normalizeSrcset normalizes URLs in srcset attribute
func normalizeSrcset(base *url.URL, srcset string) string {
	parts := strings.Split(srcset, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		tokens := strings.Fields(part)
		if len(tokens) > 0 {
			tokens[0] = resolveURL(base, tokens[0])
			parts[i] = strings.Join(tokens, " ")
		}
	}
	return strings.Join(parts, ", ")
}

// removeEmptyElements removes empty block elements
func (s *Sanitizer) removeEmptyElements(doc *goquery.Document) {
	emptyTags := []string{"p", "div", "span", "section", "article"}
	whitespaceRegex := regexp.MustCompile(`^\s*$`)

	for _, tag := range emptyTags {
		doc.Find(tag).Each(func(_ int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if whitespaceRegex.MatchString(text) && sel.Children().Length() == 0 {
				sel.Remove()
			}
		})
	}
}
