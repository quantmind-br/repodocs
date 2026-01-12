package converter

import "strings"

func IsMarkdownContent(contentType, url string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/markdown") ||
		strings.Contains(ct, "text/x-markdown") ||
		strings.Contains(ct, "application/markdown") {
		return true
	}

	lowerURL := strings.ToLower(url)

	if idx := strings.Index(lowerURL, "?"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}

	if idx := strings.Index(lowerURL, "#"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}

	if strings.HasSuffix(lowerURL, ".md") ||
		strings.HasSuffix(lowerURL, ".mdx") ||
		strings.HasSuffix(lowerURL, ".markdown") ||
		strings.HasSuffix(lowerURL, ".mdown") {
		return true
	}

	return false
}

// IsPlainTextContent checks if the content is plain text.
// Returns true for text/plain content type or .txt URL extension.
// Query strings and fragments are stripped before checking the extension.
func IsPlainTextContent(contentType, url string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/plain") {
		return true
	}

	lowerURL := strings.ToLower(url)

	if idx := strings.Index(lowerURL, "?"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}
	if idx := strings.Index(lowerURL, "#"); idx != -1 {
		lowerURL = lowerURL[:idx]
	}

	return strings.HasSuffix(lowerURL, ".txt")
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
