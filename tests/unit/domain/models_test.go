package domain_test

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// ToMetadata Tests
// ============================================================================

func TestDocument_ToMetadata(t *testing.T) {
	tests := []struct {
		name     string
		document *domain.Document
		verify   func(t *testing.T, doc *domain.Document, meta *domain.Metadata)
	}{
		{
			name: "full document with all fields",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Test Document",
				Description:    "A test description",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				ContentHash:    "abc123",
				WordCount:      100,
				CharCount:      500,
				Links:          []string{"https://example.com/link1", "https://example.com/link2"},
				Headers:        map[string][]string{"h1": {"Header 1"}, "h2": {"Header 2"}},
				RenderedWithJS: true,
				SourceStrategy: "crawler",
				CacheHit:       true,
				Summary:        "AI-generated summary",
				Tags:           []string{"tag1", "tag2"},
				Category:       "testing",
			},
			verify: func(t *testing.T, doc *domain.Document, meta *domain.Metadata) {
				assert.NotNil(t, meta)
				assert.Equal(t, doc.URL, meta.URL)
				assert.Equal(t, doc.Title, meta.Title)
				assert.Equal(t, doc.Description, meta.Description)
				assert.Equal(t, doc.FetchedAt, meta.FetchedAt)
				assert.Equal(t, doc.ContentHash, meta.ContentHash)
				assert.Equal(t, doc.WordCount, meta.WordCount)
				assert.Equal(t, doc.CharCount, meta.CharCount)
				assert.Equal(t, doc.Links, meta.Links)
				assert.Equal(t, doc.Headers, meta.Headers)
				assert.Equal(t, doc.RenderedWithJS, meta.RenderedWithJS)
				assert.Equal(t, doc.SourceStrategy, meta.SourceStrategy)
				assert.Equal(t, doc.CacheHit, meta.CacheHit)
				assert.Equal(t, doc.Summary, meta.Summary)
				assert.Equal(t, doc.Tags, meta.Tags)
				assert.Equal(t, doc.Category, meta.Category)
			},
		},
		{
			name: "minimal document with only required fields",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Minimal Document",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				ContentHash:    "xyz789",
				WordCount:      0,
				CharCount:      0,
				SourceStrategy: "sitemap",
			},
			verify: func(t *testing.T, doc *domain.Document, meta *domain.Metadata) {
				assert.NotNil(t, meta)
				assert.Equal(t, doc.URL, meta.URL)
				assert.Equal(t, doc.Title, meta.Title)
				assert.Empty(t, meta.Description, "Description should be empty")
				assert.Empty(t, meta.Links, "Links should be empty")
				assert.Empty(t, meta.Headers, "Headers should be empty")
				assert.Empty(t, meta.Tags, "Tags should be empty")
				assert.Empty(t, meta.Category, "Category should be empty")
			},
		},
		{
			name: "document with empty slices and maps",
			document: &domain.Document{
				URL:            "https://example.com/empty",
				Title:          "Empty Document",
				FetchedAt:      time.Now(),
				ContentHash:    "empty123",
				WordCount:      10,
				CharCount:      50,
				Links:          []string{},
				Headers:        map[string][]string{},
				Tags:           []string{},
				SourceStrategy: "git",
			},
			verify: func(t *testing.T, doc *domain.Document, meta *domain.Metadata) {
				assert.NotNil(t, meta)
				assert.NotNil(t, meta.Links, "Links should not be nil")
				assert.NotNil(t, meta.Headers, "Headers should not be nil")
				assert.NotNil(t, meta.Tags, "Tags should not be nil")
				assert.Empty(t, meta.Links)
				assert.Empty(t, meta.Headers)
				assert.Empty(t, meta.Tags)
			},
		},
		{
			name: "document with LLM-enhanced fields",
			document: &domain.Document{
				URL:            "https://example.com/llm",
				Title:          "LLM Enhanced Document",
				FetchedAt:      time.Now(),
				ContentHash:    "llm456",
				WordCount:      200,
				CharCount:      1000,
				SourceStrategy: "crawler",
				Summary:        "This is an AI-generated summary of the document",
				Tags:           []string{"ai", "machine-learning", "documentation"},
				Category:       "Technology",
			},
			verify: func(t *testing.T, doc *domain.Document, meta *domain.Metadata) {
				assert.NotNil(t, meta)
				assert.Equal(t, "This is an AI-generated summary of the document", meta.Summary)
				assert.Equal(t, []string{"ai", "machine-learning", "documentation"}, meta.Tags)
				assert.Equal(t, "Technology", meta.Category)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := tt.document.ToMetadata()
			tt.verify(t, tt.document, meta)
		})
	}
}

// ============================================================================
// ToFrontmatter Tests
// ============================================================================

func TestDocument_ToFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		document *domain.Document
		verify   func(t *testing.T, doc *domain.Document, fm *domain.Frontmatter)
	}{
		{
			name: "full document with LLM fields",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Test Document",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				RenderedWithJS: true,
				WordCount:      100,
				SourceStrategy: "crawler",
				Summary:        "AI summary",
				Tags:           []string{"tag1", "tag2"},
				Category:       "test",
			},
			verify: func(t *testing.T, doc *domain.Document, fm *domain.Frontmatter) {
				assert.NotNil(t, fm)
				assert.Equal(t, doc.Title, fm.Title)
				assert.Equal(t, doc.URL, fm.URL)
				assert.Equal(t, doc.SourceStrategy, fm.Source)
				assert.Equal(t, doc.FetchedAt, fm.FetchedAt)
				assert.Equal(t, doc.RenderedWithJS, fm.RenderedJS)
				assert.Equal(t, doc.WordCount, fm.WordCount)
				assert.Equal(t, doc.Summary, fm.Summary)
				assert.Equal(t, doc.Tags, fm.Tags)
				assert.Equal(t, doc.Category, fm.Category)
			},
		},
		{
			name: "document without LLM fields",
			document: &domain.Document{
				URL:            "https://example.com/simple",
				Title:          "Simple Document",
				FetchedAt:      time.Now(),
				RenderedWithJS: false,
				WordCount:      50,
				SourceStrategy: "sitemap",
			},
			verify: func(t *testing.T, doc *domain.Document, fm *domain.Frontmatter) {
				assert.NotNil(t, fm)
				assert.Empty(t, fm.Summary, "Summary should be empty")
				assert.Nil(t, fm.Tags, "Tags should be nil")
				assert.Empty(t, fm.Category, "Category should be empty")
			},
		},
		{
			name: "document from different strategies",
			document: &domain.Document{
				URL:            "https://github.com/test/repo",
				Title:          "Git Document",
				FetchedAt:      time.Now(),
				RenderedWithJS: false,
				WordCount:      150,
				SourceStrategy: "git",
				Summary:        "Git documentation",
				Tags:           []string{"git", "version-control"},
			},
			verify: func(t *testing.T, doc *domain.Document, fm *domain.Frontmatter) {
				assert.NotNil(t, fm)
				assert.Equal(t, "git", fm.Source)
				assert.Equal(t, "Git documentation", fm.Summary)
				assert.Equal(t, []string{"git", "version-control"}, fm.Tags)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := tt.document.ToFrontmatter()
			tt.verify(t, tt.document, fm)
		})
	}
}

// ============================================================================
// ToDocumentMetadata Tests
// ============================================================================

func TestDocument_ToDocumentMetadata(t *testing.T) {
	tests := []struct {
		name     string
		document *domain.Document
		filePath string
		verify   func(t *testing.T, doc *domain.Document, filePath string, dm *domain.DocumentMetadata)
	}{
		{
			name: "full document metadata",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Test Document",
				Description:    "Test description",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				ContentHash:    "hash123",
				WordCount:      100,
				CharCount:      500,
				Links:          []string{"https://example.com/link1"},
				Headers:        map[string][]string{"h1": {"Header"}},
				RenderedWithJS: true,
				SourceStrategy: "crawler",
				CacheHit:       false,
				Summary:        "AI summary",
				Tags:           []string{"tag1"},
				Category:       "test",
			},
			filePath: "docs/example.com/test-document.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, dm *domain.DocumentMetadata) {
				assert.NotNil(t, dm)
				assert.Equal(t, filePath, dm.FilePath)
				assert.NotNil(t, dm.Metadata)
				assert.Equal(t, doc.URL, dm.Metadata.URL)
				assert.Equal(t, doc.Title, dm.Metadata.Title)
				assert.Equal(t, doc.Description, dm.Metadata.Description)
				assert.Equal(t, doc.ContentHash, dm.Metadata.ContentHash)
				assert.Equal(t, doc.WordCount, dm.Metadata.WordCount)
				assert.Equal(t, doc.CharCount, dm.Metadata.CharCount)
				assert.Equal(t, doc.Summary, dm.Metadata.Summary)
				assert.Equal(t, doc.Tags, dm.Metadata.Tags)
				assert.Equal(t, doc.Category, dm.Metadata.Category)
			},
		},
		{
			name: "minimal document metadata",
			document: &domain.Document{
				URL:            "https://example.com/minimal",
				Title:          "Minimal Doc",
				FetchedAt:      time.Now(),
				ContentHash:    "minimal",
				SourceStrategy: "sitemap",
			},
			filePath: "docs/minimal.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, dm *domain.DocumentMetadata) {
				assert.NotNil(t, dm)
				assert.Equal(t, filePath, dm.FilePath)
				assert.NotNil(t, dm.Metadata)
				assert.Equal(t, doc.URL, dm.Metadata.URL)
				assert.Equal(t, doc.Title, dm.Metadata.Title)
			},
		},
		{
			name: "document with nested file path",
			document: &domain.Document{
				URL:            "https://example.com/nested/path",
				Title:          "Nested Doc",
				FetchedAt:      time.Now(),
				ContentHash:    "nested",
				SourceStrategy: "git",
			},
			filePath: "docs/v1/api/nested/reference.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, dm *domain.DocumentMetadata) {
				assert.NotNil(t, dm)
				assert.Equal(t, "docs/v1/api/nested/reference.md", dm.FilePath)
				assert.Equal(t, "git", dm.Metadata.SourceStrategy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := tt.document.ToDocumentMetadata(tt.filePath)
			tt.verify(t, tt.document, tt.filePath, dm)
		})
	}
}

// ============================================================================
// ToSimpleMetadata Tests
// ============================================================================

func TestDocument_ToSimpleMetadata(t *testing.T) {
	tests := []struct {
		name     string
		document *domain.Document
		verify   func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata)
	}{
		{
			name: "full document with all optional fields",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Test Document",
				Description:    "Test description",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				SourceStrategy: "crawler",
				Summary:        "AI-generated summary",
				Tags:           []string{"tag1", "tag2", "tag3"},
				Category:       "Technology",
			},
			verify: func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata) {
				assert.NotNil(t, sm)
				assert.Equal(t, doc.Title, sm.Title)
				assert.Equal(t, doc.URL, sm.URL)
				assert.Equal(t, doc.SourceStrategy, sm.Source)
				assert.Equal(t, doc.FetchedAt, sm.FetchedAt)
				assert.Equal(t, doc.Description, sm.Description)
				assert.Equal(t, doc.Summary, sm.Summary)
				assert.Equal(t, doc.Tags, sm.Tags)
				assert.Equal(t, doc.Category, sm.Category)
			},
		},
		{
			name: "minimal document with only required fields",
			document: &domain.Document{
				URL:            "https://example.com/simple",
				Title:          "Simple Document",
				FetchedAt:      time.Now(),
				SourceStrategy: "sitemap",
			},
			verify: func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata) {
				assert.NotNil(t, sm)
				assert.Equal(t, doc.Title, sm.Title)
				assert.Equal(t, doc.URL, sm.URL)
				assert.Equal(t, doc.SourceStrategy, sm.Source)
				assert.Equal(t, doc.FetchedAt, sm.FetchedAt)
				assert.Empty(t, sm.Description)
				assert.Empty(t, sm.Summary)
				assert.Nil(t, sm.Tags)
				assert.Empty(t, sm.Category)
			},
		},
		{
			name: "document with different strategies",
			document: &domain.Document{
				URL:            "https://github.com/org/repo",
				Title:          "GitHub Document",
				FetchedAt:      time.Now(),
				SourceStrategy: "git",
				Description:    "Git documentation",
				Tags:           []string{"git", "docs"},
				Category:       "Development",
			},
			verify: func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata) {
				assert.NotNil(t, sm)
				assert.Equal(t, "git", sm.Source)
				assert.Equal(t, "Git documentation", sm.Description)
				assert.Equal(t, []string{"git", "docs"}, sm.Tags)
				assert.Equal(t, "Development", sm.Category)
			},
		},
		{
			name: "document from llms strategy",
			document: &domain.Document{
				URL:            "https://example.com/llms",
				Title:          "LLMS Document",
				FetchedAt:      time.Now(),
				SourceStrategy: "llms",
				Summary:        "This is from llms.txt",
				Tags:           []string{"llms", "index"},
			},
			verify: func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata) {
				assert.NotNil(t, sm)
				assert.Equal(t, "llms", sm.Source)
				assert.Equal(t, "This is from llms.txt", sm.Summary)
				assert.Equal(t, []string{"llms", "index"}, sm.Tags)
			},
		},
		{
			name: "document with empty tags",
			document: &domain.Document{
				URL:            "https://example.com/no-tags",
				Title:          "No Tags Document",
				FetchedAt:      time.Now(),
				SourceStrategy: "crawler",
				Tags:           []string{},
			},
			verify: func(t *testing.T, doc *domain.Document, sm *domain.SimpleMetadata) {
				assert.NotNil(t, sm)
				assert.NotNil(t, sm.Tags, "Tags should not be nil")
				assert.Empty(t, sm.Tags, "Tags should be empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.document.ToSimpleMetadata()
			tt.verify(t, tt.document, sm)
		})
	}
}

// ============================================================================
// ToSimpleDocumentMetadata Tests
// ============================================================================

func TestDocument_ToSimpleDocumentMetadata(t *testing.T) {
	tests := []struct {
		name     string
		document *domain.Document
		filePath string
		verify   func(t *testing.T, doc *domain.Document, filePath string, sdm *domain.SimpleDocumentMetadata)
	}{
		{
			name: "full simple document metadata",
			document: &domain.Document{
				URL:            "https://example.com/docs",
				Title:          "Test Document",
				Description:    "Test description",
				FetchedAt:      time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
				SourceStrategy: "crawler",
				Summary:        "AI summary",
				Tags:           []string{"tag1", "tag2"},
				Category:       "Testing",
			},
			filePath: "docs/test-document.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, sdm *domain.SimpleDocumentMetadata) {
				assert.NotNil(t, sdm)
				assert.Equal(t, filePath, sdm.FilePath)
				assert.NotNil(t, sdm.SimpleMetadata)
				assert.Equal(t, doc.Title, sdm.SimpleMetadata.Title)
				assert.Equal(t, doc.URL, sdm.SimpleMetadata.URL)
				assert.Equal(t, doc.SourceStrategy, sdm.SimpleMetadata.Source)
				assert.Equal(t, doc.FetchedAt, sdm.SimpleMetadata.FetchedAt)
				assert.Equal(t, doc.Description, sdm.SimpleMetadata.Description)
				assert.Equal(t, doc.Summary, sdm.SimpleMetadata.Summary)
				assert.Equal(t, doc.Tags, sdm.SimpleMetadata.Tags)
				assert.Equal(t, doc.Category, sdm.SimpleMetadata.Category)
			},
		},
		{
			name: "minimal simple document metadata",
			document: &domain.Document{
				URL:            "https://example.com/minimal",
				Title:          "Minimal Document",
				FetchedAt:      time.Now(),
				SourceStrategy: "sitemap",
			},
			filePath: "docs/minimal.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, sdm *domain.SimpleDocumentMetadata) {
				assert.NotNil(t, sdm)
				assert.Equal(t, filePath, sdm.FilePath)
				assert.NotNil(t, sdm.SimpleMetadata)
				assert.Equal(t, doc.Title, sdm.SimpleMetadata.Title)
				assert.Equal(t, doc.URL, sdm.SimpleMetadata.URL)
				assert.Empty(t, sdm.SimpleMetadata.Description)
				assert.Nil(t, sdm.SimpleMetadata.Tags)
			},
		},
		{
			name: "document with nested path",
			document: &domain.Document{
				URL:            "https://pkg.go.dev/github.com/pkg/errors",
				Title:          "Package Documentation",
				FetchedAt:      time.Now(),
				SourceStrategy: "pkggo",
				Description:    "Go package error handling",
				Tags:           []string{"go", "errors"},
				Category:       "Go",
			},
			filePath: "docs/pkg.go.dev/github.com/pkg/errors.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, sdm *domain.SimpleDocumentMetadata) {
				assert.NotNil(t, sdm)
				assert.Equal(t, "pkggo", sdm.SimpleMetadata.Source)
				assert.Contains(t, sdm.FilePath, "pkg.go.dev")
				assert.Equal(t, "Go package error handling", sdm.SimpleMetadata.Description)
				assert.Equal(t, []string{"go", "errors"}, sdm.SimpleMetadata.Tags)
				assert.Equal(t, "Go", sdm.SimpleMetadata.Category)
			},
		},
		{
			name: "document from git strategy with relative path",
			document: &domain.Document{
				URL:            "https://github.com/test/repo",
				Title:          "README",
				FetchedAt:      time.Now(),
				SourceStrategy: "git",
				Summary:        "Project README",
				Tags:           []string{"readme", "getting-started"},
			},
			filePath: "docs/README.md",
			verify: func(t *testing.T, doc *domain.Document, filePath string, sdm *domain.SimpleDocumentMetadata) {
				assert.NotNil(t, sdm)
				assert.Equal(t, "git", sdm.SimpleMetadata.Source)
				assert.Equal(t, "docs/README.md", sdm.FilePath)
				assert.Equal(t, "Project README", sdm.SimpleMetadata.Summary)
				assert.Equal(t, []string{"readme", "getting-started"}, sdm.SimpleMetadata.Tags)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdm := tt.document.ToSimpleDocumentMetadata(tt.filePath)
			tt.verify(t, tt.document, tt.filePath, sdm)
		})
	}
}

// ============================================================================
// Conversion Consistency Tests
// ============================================================================

func TestDocument_ConversionConsistency(t *testing.T) {
	baseTime := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)

	doc := &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Consistency Test",
		Description:    "Testing conversion consistency",
		FetchedAt:      baseTime,
		ContentHash:    "consistent123",
		WordCount:      42,
		CharCount:      210,
		SourceStrategy: "crawler",
		Summary:        "Test summary",
		Tags:           []string{"test", "consistency"},
		Category:       "Testing",
	}

	t.Run("ToMetadata and ToDocumentMetadata consistency", func(t *testing.T) {
		meta := doc.ToMetadata()
		dm := doc.ToDocumentMetadata("docs/test.md")

		assert.Equal(t, meta.URL, dm.Metadata.URL)
		assert.Equal(t, meta.Title, dm.Metadata.Title)
		assert.Equal(t, meta.Summary, dm.Metadata.Summary)
		assert.Equal(t, meta.Tags, dm.Metadata.Tags)
		assert.Equal(t, meta.Category, dm.Metadata.Category)
	})

	t.Run("ToSimpleMetadata and ToSimpleDocumentMetadata consistency", func(t *testing.T) {
		sm := doc.ToSimpleMetadata()
		sdm := doc.ToSimpleDocumentMetadata("docs/test.md")

		assert.Equal(t, sm.Title, sdm.SimpleMetadata.Title)
		assert.Equal(t, sm.URL, sdm.SimpleMetadata.URL)
		assert.Equal(t, sm.Summary, sdm.SimpleMetadata.Summary)
		assert.Equal(t, sm.Tags, sdm.SimpleMetadata.Tags)
		assert.Equal(t, sm.Category, sdm.SimpleMetadata.Category)
	})

	t.Run("ToFrontmatter uses same fields", func(t *testing.T) {
		fm := doc.ToFrontmatter()
		sm := doc.ToSimpleMetadata()

		assert.Equal(t, doc.Title, fm.Title)
		assert.Equal(t, sm.Title, fm.Title)

		assert.Equal(t, doc.URL, fm.URL)
		assert.Equal(t, sm.URL, fm.URL)

		assert.Equal(t, doc.SourceStrategy, fm.Source)
		assert.Equal(t, sm.Source, fm.Source)

		assert.Equal(t, doc.Summary, fm.Summary)
		assert.Equal(t, sm.Summary, fm.Summary)

		assert.Equal(t, doc.Tags, fm.Tags)
		assert.Equal(t, sm.Tags, fm.Tags)

		assert.Equal(t, doc.Category, fm.Category)
		assert.Equal(t, sm.Category, fm.Category)
	})
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

func TestDocument_NilReceiver(t *testing.T) {
	var doc *domain.Document

	t.Run("ToMetadata handles nil", func(t *testing.T) {
		// This should panic if nil checks are not in place
		// But since Go doesn't automatically check for nil receivers, this test documents behavior
		assert.Panics(t, func() {
			doc.ToMetadata()
		}, "ToMetadata should panic on nil receiver")
	})

	t.Run("ToFrontmatter handles nil", func(t *testing.T) {
		assert.Panics(t, func() {
			doc.ToFrontmatter()
		}, "ToFrontmatter should panic on nil receiver")
	})

	t.Run("ToSimpleMetadata handles nil", func(t *testing.T) {
		assert.Panics(t, func() {
			doc.ToSimpleMetadata()
		}, "ToSimpleMetadata should panic on nil receiver")
	})
}

func TestDocument_EmptyFields(t *testing.T) {
	doc := &domain.Document{
		URL:            "",
		Title:          "",
		Description:    "",
		FetchedAt:      time.Time{},
		ContentHash:    "",
		SourceStrategy: "",
		Tags:           nil,
	}

	t.Run("ToMetadata with empty fields", func(t *testing.T) {
		meta := doc.ToMetadata()
		assert.NotNil(t, meta)
		assert.Empty(t, meta.URL)
		assert.Empty(t, meta.Title)
		assert.Empty(t, meta.Description)
		assert.True(t, meta.FetchedAt.IsZero())
		assert.Empty(t, meta.ContentHash)
		assert.Nil(t, meta.Tags)
	})

	t.Run("ToSimpleMetadata with empty fields", func(t *testing.T) {
		sm := doc.ToSimpleMetadata()
		assert.NotNil(t, sm)
		assert.Empty(t, sm.URL)
		assert.Empty(t, sm.Title)
		assert.Empty(t, sm.Source)
		assert.True(t, sm.FetchedAt.IsZero())
		assert.Nil(t, sm.Tags)
	})
}
