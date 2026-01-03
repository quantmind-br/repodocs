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

	cleanDoc, err := s.SanitizeDocument(doc)
	if err != nil {
		return "", err
	}

	result, err := cleanDoc.Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

// SanitizeDocument cleans a pre-parsed document in place.
func (s *Sanitizer) SanitizeDocument(doc *goquery.Document) (*goquery.Document, error) {
	if doc == nil {
		return nil, nil
	}

	s.sanitizeSelection(doc.Selection)
	return doc, nil
}

// SanitizeSelection cleans a selection in place.
func (s *Sanitizer) SanitizeSelection(sel *goquery.Selection) (*goquery.Selection, error) {
	if sel == nil {
		return nil, nil
	}

	s.sanitizeSelection(sel)
	return sel, nil
}

func (s *Sanitizer) sanitizeSelection(sel *goquery.Selection) {
	// Remove unwanted tags
	for _, tag := range TagsToRemove {
		findWithRoot(sel, tag).Remove()
	}

	// Remove elements by class
	if s.removeNavigation {
		for _, class := range ClassesToRemove {
			findWithRoot(sel, "."+class).Remove()
			findWithRoot(sel, "[class*='"+class+"']").Remove()
		}

		// Remove elements by ID
		for _, id := range IDsToRemove {
			findWithRoot(sel, "#"+id).Remove()
		}

		findWithRoot(sel, "nav").Remove()
	}

	// Remove hidden elements
	findWithRoot(sel, "[style*='display:none']").Remove()
	findWithRoot(sel, "[style*='display: none']").Remove()
	findWithRoot(sel, "[hidden]").Remove()

	// Normalize URLs if base URL is provided
	if s.baseURL != "" {
		s.normalizeURLsFromSelection(sel)
	}

	// Remove empty paragraphs and divs
	s.removeEmptyElementsFromSelection(sel)
}

// normalizeURLs converts relative URLs to absolute URLs
func (s *Sanitizer) normalizeURLs(doc *goquery.Document) {
	s.normalizeURLsFromSelection(doc.Selection)
}

func (s *Sanitizer) normalizeURLsFromSelection(sel *goquery.Selection) {
	base, err := url.Parse(s.baseURL)
	if err != nil {
		return
	}

	// Normalize href attributes
	findWithRoot(sel, "a[href]").Each(func(_ int, node *goquery.Selection) {
		if href, exists := node.Attr("href"); exists {
			if absoluteURL := resolveURL(base, href); absoluteURL != "" {
				node.SetAttr("href", absoluteURL)
			}
		}
	})

	// Normalize src attributes
	findWithRoot(sel, "[src]").Each(func(_ int, node *goquery.Selection) {
		if src, exists := node.Attr("src"); exists {
			if absoluteURL := resolveURL(base, src); absoluteURL != "" {
				node.SetAttr("src", absoluteURL)
			}
		}
	})

	// Normalize srcset attributes
	findWithRoot(sel, "[srcset]").Each(func(_ int, node *goquery.Selection) {
		if srcset, exists := node.Attr("srcset"); exists {
			node.SetAttr("srcset", normalizeSrcset(base, srcset))
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
	s.removeEmptyElementsFromSelection(doc.Selection)
}

func (s *Sanitizer) removeEmptyElementsFromSelection(sel *goquery.Selection) {
	emptyTags := []string{"p", "div", "span", "section", "article"}
	whitespaceRegex := regexp.MustCompile(`^\s*$`)

	for _, tag := range emptyTags {
		findWithRoot(sel, tag).Each(func(_ int, node *goquery.Selection) {
			text := strings.TrimSpace(node.Text())
			if whitespaceRegex.MatchString(text) && node.Children().Length() == 0 {
				node.Remove()
			}
		})
	}
}
