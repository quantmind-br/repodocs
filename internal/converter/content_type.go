package converter

import "strings"

// IsMarkdownContent checks if the content type or URL indicates markdown content.
// It checks both the Content-Type header and the URL extension.
func IsMarkdownContent(contentType, url string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/markdown") ||
		strings.Contains(ct, "text/x-markdown") ||
		strings.Contains(ct, "application/markdown") {
		return true
	}

	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		return false
	}

	lowerURL := strings.ToLower(url)

	if idx := strings.Index(lowerURL, "?"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}

	if idx := strings.Index(lowerURL, "#"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}

	return strings.HasSuffix(lowerURL, ".md") ||
		strings.HasSuffix(lowerURL, ".markdown") ||
		strings.HasSuffix(lowerURL, ".mdown")
}

// IsHTMLContent checks if the content type indicates HTML content.
// Returns true for empty content type (assumes HTML for backward compatibility).
func IsHTMLContent(contentType string) bool {
	if contentType == "" {
		return true
	}
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "application/xhtml")
}
