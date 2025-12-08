package testutil

import (
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/require"
)

// NewDocument creates a test document
func NewDocument(t *testing.T) *domain.Document {
	t.Helper()

	return &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Test Document",
		Description:    "Test description",
		Content:        "This is test content",
		HTMLContent:    "<p>This is test content</p>",
		FetchedAt:      time.Now(),
		ContentHash:    "abc123",
		WordCount:      5,
		CharCount:      25,
		Links:          []string{"https://example.com/link1", "https://example.com/link2"},
		Headers:        map[string][]string{"h1": {"Test Document"}, "h2": {"Section 1"}},
		RenderedWithJS: false,
		SourceStrategy: "crawler",
		CacheHit:       false,
	}
}

// NewHTMLDocument creates a test document with HTML content
func NewHTMLDocument(t *testing.T, url, title, htmlContent string) *domain.Document {
	t.Helper()

	return &domain.Document{
		URL:            url,
		Title:          title,
		Description:    "",
		Content:        htmlContent,
		HTMLContent:    htmlContent,
		FetchedAt:      time.Now(),
		ContentHash:    "",
		WordCount:      0,
		CharCount:      len(htmlContent),
		Links:          []string{},
		Headers:        map[string][]string{},
		RenderedWithJS: false,
		SourceStrategy: "crawler",
		CacheHit:       false,
	}
}

// NewMarkdownDocument creates a test document with Markdown content
func NewMarkdownDocument(t *testing.T, url, title, markdown string) *domain.Document {
	t.Helper()

	return &domain.Document{
		URL:            url,
		Title:          title,
		Description:    "",
		Content:        markdown,
		HTMLContent:    "",
		FetchedAt:      time.Now(),
		ContentHash:    "",
		WordCount:      len(strings.Fields(markdown)),
		CharCount:      len(markdown),
		Links:          []string{},
		Headers:        map[string][]string{},
		RenderedWithJS: false,
		SourceStrategy: "git",
		CacheHit:       false,
	}
}

// NewEmptyDocument creates an empty test document
func NewEmptyDocument(t *testing.T) *domain.Document {
	t.Helper()

	return &domain.Document{
		URL:            "",
		Title:          "",
		Description:    "",
		Content:        "",
		HTMLContent:    "",
		FetchedAt:      time.Now(),
		ContentHash:    "",
		WordCount:      0,
		CharCount:      0,
		Links:          []string{},
		Headers:        map[string][]string{},
		RenderedWithJS: false,
		SourceStrategy: "",
		CacheHit:       false,
	}
}

// VerifyDocumentFields verifies common document fields
func VerifyDocumentFields(t *testing.T, doc *domain.Document, expectedURL, expectedTitle string) {
	t.Helper()

	require.NotNil(t, doc)
	require.Equal(t, expectedURL, doc.URL)
	require.Equal(t, expectedTitle, doc.Title)
	require.NotNil(t, doc.Headers)
	require.NotNil(t, doc.Links)
	require.False(t, doc.FetchedAt.IsZero())
}

// CloneDocument creates a copy of a document for testing
func CloneDocument(t *testing.T, doc *domain.Document) *domain.Document {
	t.Helper()

	headersCopy := make(map[string][]string)
	for k, v := range doc.Headers {
		headersCopy[k] = append([]string(nil), v...)
	}

	return &domain.Document{
		URL:            doc.URL,
		Title:          doc.Title,
		Description:    doc.Description,
		Content:        doc.Content,
		HTMLContent:    doc.HTMLContent,
		FetchedAt:      doc.FetchedAt,
		ContentHash:    doc.ContentHash,
		WordCount:      doc.WordCount,
		CharCount:      doc.CharCount,
		Links:          append([]string(nil), doc.Links...),
		Headers:        headersCopy,
		RenderedWithJS: doc.RenderedWithJS,
		SourceStrategy: doc.SourceStrategy,
		CacheHit:       doc.CacheHit,
		RelativePath:   doc.RelativePath,
	}
}
