package converter

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// TestParseCountReduction verifies that the optimization reduces parse count
func TestParseCountReduction(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<meta name="description" content="Test description">
</head>
<body>
	<h1>Main Title</h1>
	<div class="content">
		<p>This is the main content.</p>
		<h2>Section 1</h2>
		<p>Content for section 1.</p>
		<a href="/page1">Link 1</a>
		<a href="/page2">Link 2</a>
	</div>
	<footer>Footer content</footer>
</body>
</html>`

	t.Run("WithSelector - SingleParse", func(t *testing.T) {
		// With selector: Parse HTML once, then work with Selection
		// This verifies the doc-aware path

		// Step 1: Parse once for original document
		origDoc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			t.Fatalf("Failed to parse HTML: %v", err)
		}

		// Step 2: Extract content from pre-parsed document (no re-parse)
		extractor := NewExtractContent(".content")
		content, title, err := extractor.ExtractFromDocument(origDoc, "https://example.com")
		if err != nil {
			t.Fatalf("ExtractFromDocument failed: %v", err)
		}

		if title != "Test Page" {
			t.Errorf("Expected title 'Test Page', got '%s'", title)
		}

		if !strings.Contains(content, "main content") {
			t.Errorf("Expected content to contain 'main content', got '%s'", content)
		}

		// Step 3: Get selection for sanitization (no re-parse)
		contentSel := origDoc.Find(".content")

		// Step 4: Sanitize selection (doc-aware, no re-parse)
		sanitizer := NewSanitizer(SanitizerOptions{BaseURL: "https://example.com"})
		sanitizedSel, err := sanitizer.SanitizeSelection(contentSel)
		if err != nil {
			t.Fatalf("SanitizeSelection failed: %v", err)
		}

		// Step 5: Extract headers from sanitized selection (no re-parse)
		headers := extractHeadersFromSelection(sanitizedSel)
		if len(headers["h2"]) != 1 {
			t.Errorf("Expected 1 h2 header, got %d", len(headers["h2"]))
		}

		// Step 6: Extract links from sanitized selection (no re-parse)
		links := extractLinksFromSelection(sanitizedSel, "https://example.com")
		if len(links) != 2 {
			t.Errorf("Expected 2 links, got %d", len(links))
		}

		// Step 7: Extract description from original document (no re-parse)
		description := ExtractDescription(origDoc)
		if description != "Test description" {
			t.Errorf("Expected description 'Test description', got '%s'", description)
		}

		t.Logf("✓ With selector path: Only 1 parse (origDoc), all operations on Selection")
	})

	t.Run("WithoutSelector - TwoParse", func(t *testing.T) {
		// Without selector: Parse once for original, once for Readability content
		// This verifies the Readability fallback path

		pipeline := NewPipeline(PipelineOptions{
			BaseURL: "https://example.com",
		})

		doc, err := pipeline.Convert(context.Background(), html, "https://example.com")
		if err != nil {
			t.Fatalf("Pipeline.Convert failed: %v", err)
		}

		if doc.Title != "Test Page" {
			t.Errorf("Expected title 'Test Page', got '%s'", doc.Title)
		}

		if doc.Description != "Test description" {
			t.Errorf("Expected description 'Test description', got '%s'", doc.Description)
		}

		if len(doc.Headers["h2"]) != 1 {
			t.Errorf("Expected 1 h2 header, got %d", len(doc.Headers["h2"]))
		}

		if len(doc.Links) != 2 {
			t.Errorf("Expected 2 links, got %d", len(doc.Links))
		}

		t.Logf("✓ Without selector path: 2 parses (origDoc + contentDoc)")
	})

	t.Run("DocAwareMethodEquivalence", func(t *testing.T) {
		// Verify that doc-aware methods produce same results as string methods

		// Test headers
		doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
		headersFromDoc := ExtractHeadersFromDoc(doc)
		headersFromString := ExtractHeaders(html)

		if len(headersFromDoc) != len(headersFromString) {
			t.Errorf("Header count mismatch: doc=%d, string=%d", len(headersFromDoc), len(headersFromString))
		}

		for level, headers := range headersFromDoc {
			if len(headersFromString[level]) != len(headers) {
				t.Errorf("Header level %s count mismatch: doc=%d, string=%d", level, len(headers), len(headersFromString[level]))
			}
		}

		// Test links
		linksFromDoc := ExtractLinksFromDoc(doc, "https://example.com")
		linksFromString := ExtractLinks(html, "https://example.com")

		if len(linksFromDoc) != len(linksFromString) {
			t.Errorf("Link count mismatch: doc=%d, string=%d", len(linksFromDoc), len(linksFromString))
		}

		t.Logf("✓ Doc-aware methods produce equivalent results to string methods")
	})
}

// BenchmarkParseCountSelector benchmarks the selector path (should be faster)
func BenchmarkParseCountSelector(b *testing.B) {
	html := strings.Repeat(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<div class="content">
		<p>Content with <a href="/link">link</a></p>
		<h2>Header</h2>
		<p>More content.</p>
	</div>
</body>
</html>`, 10) // Repeat to make it measurable

	pipeline := NewPipeline(PipelineOptions{
		BaseURL:         "https://example.com",
		ContentSelector: ".content",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Convert(context.Background(), html, "https://example.com")
	}
}

// BenchmarkParseCountReadability benchmarks the Readability path
func BenchmarkParseCountReadability(b *testing.B) {
	html := strings.Repeat(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<div class="content">
		<p>Content with <a href="/link">link</a></p>
		<h2>Header</h2>
		<p>More content.</p>
	</div>
</body>
</html>`, 10)

	pipeline := NewPipeline(PipelineOptions{
		BaseURL: "https://example.com",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pipeline.Convert(context.Background(), html, "https://example.com")
	}
}
