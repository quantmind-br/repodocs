package converter

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/transform"
)

// DetectEncoding detects the character encoding of HTML content
func DetectEncoding(content []byte) string {
	// Try to detect from content-type meta tag or charset attribute
	contentStr := string(content[:min(1024, len(content))])

	// Look for charset in meta tag
	if enc := extractCharsetFromMeta(contentStr); enc != "" {
		return enc
	}

	// Use golang.org/x/net/html/charset for detection
	_, name, _ := charset.DetermineEncoding(content, "")
	if name != "" {
		return name
	}

	// Default to UTF-8
	return "utf-8"
}

// extractCharsetFromMeta extracts charset from meta tag
func extractCharsetFromMeta(html string) string {
	html = strings.ToLower(html)

	// Look for <meta charset="...">
	if idx := strings.Index(html, "charset="); idx != -1 {
		start := idx + 8
		end := start

		// Skip quote if present
		if start < len(html) && (html[start] == '"' || html[start] == '\'') {
			start++
		}

		// Find end of charset value
		for end = start; end < len(html); end++ {
			c := html[end]
			if c == '"' || c == '\'' || c == ';' || c == '>' || c == ' ' {
				break
			}
		}

		if end > start {
			return strings.TrimSpace(html[start:end])
		}
	}

	return ""
}

// ConvertToUTF8 converts content from detected encoding to UTF-8
func ConvertToUTF8(content []byte) ([]byte, error) {
	enc := DetectEncoding(content)

	// Already UTF-8
	if enc == "utf-8" || enc == "utf8" {
		return content, nil
	}

	// Get encoder for the detected charset
	e, err := htmlindex.Get(enc)
	if err != nil {
		// Unknown encoding, return as-is
		return content, nil
	}

	// Decode to UTF-8
	reader := transform.NewReader(bytes.NewReader(content), e.NewDecoder())
	return io.ReadAll(reader)
}

// IsUTF8 checks if content is valid UTF-8
func IsUTF8(content []byte) bool {
	enc := DetectEncoding(content)
	return enc == "utf-8" || enc == "utf8"
}

// GetEncoder returns the encoding for a charset name
func GetEncoder(charsetName string) (encoding.Encoding, error) {
	return htmlindex.Get(charsetName)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
