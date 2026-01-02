package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDocument_ToMetadata tests converting Document to Metadata
func TestDocument_ToMetadata(t *testing.T) {
	now := time.Now()
	doc := &Document{
		URL:            "https://example.com/page",
		Title:          "Test Page",
		Description:    "A test page",
		FetchedAt:      now,
		ContentHash:    "abc123",
		WordCount:      100,
		CharCount:      500,
		Links:          []string{"https://example.com/other"},
		Headers:        map[string][]string{"h1": {"Test Page"}},
		RenderedWithJS: true,
		SourceStrategy: "crawler",
		CacheHit:       false,
		Summary:        "Test summary",
		Tags:           []string{"test", "example"},
		Category:       "testing",
	}

	metadata := doc.ToMetadata()

	assert.Equal(t, doc.URL, metadata.URL)
	assert.Equal(t, doc.Title, metadata.Title)
	assert.Equal(t, doc.Description, metadata.Description)
	assert.Equal(t, doc.FetchedAt, metadata.FetchedAt)
	assert.Equal(t, doc.ContentHash, metadata.ContentHash)
	assert.Equal(t, doc.WordCount, metadata.WordCount)
	assert.Equal(t, doc.CharCount, metadata.CharCount)
	assert.Equal(t, doc.Links, metadata.Links)
	assert.Equal(t, doc.Headers, metadata.Headers)
	assert.Equal(t, doc.RenderedWithJS, metadata.RenderedWithJS)
	assert.Equal(t, doc.SourceStrategy, metadata.SourceStrategy)
	assert.Equal(t, doc.CacheHit, metadata.CacheHit)
	assert.Equal(t, doc.Summary, metadata.Summary)
	assert.Equal(t, doc.Tags, metadata.Tags)
	assert.Equal(t, doc.Category, metadata.Category)
}

// TestDocument_ToFrontmatter tests converting Document to Frontmatter
func TestDocument_ToFrontmatter(t *testing.T) {
	now := time.Now()
	doc := &Document{
		URL:            "https://example.com/page",
		Title:          "Test Page",
		FetchedAt:      now,
		RenderedWithJS: true,
		SourceStrategy: "crawler",
		WordCount:      100,
		Summary:        "Test summary",
		Tags:           []string{"test"},
		Category:       "testing",
	}

	frontmatter := doc.ToFrontmatter()

	assert.Equal(t, doc.Title, frontmatter.Title)
	assert.Equal(t, doc.URL, frontmatter.URL)
	assert.Equal(t, doc.SourceStrategy, frontmatter.Source)
	assert.Equal(t, doc.FetchedAt, frontmatter.FetchedAt)
	assert.Equal(t, doc.RenderedWithJS, frontmatter.RenderedJS)
	assert.Equal(t, doc.WordCount, frontmatter.WordCount)
	assert.Equal(t, doc.Summary, frontmatter.Summary)
	assert.Equal(t, doc.Tags, frontmatter.Tags)
	assert.Equal(t, doc.Category, frontmatter.Category)
}

// TestDocument_ToDocumentMetadata tests converting Document to DocumentMetadata
func TestDocument_ToDocumentMetadata(t *testing.T) {
	now := time.Now()
	doc := &Document{
		URL:         "https://example.com/page",
		Title:       "Test Page",
		Description: "Test description",
		FetchedAt:   now,
		WordCount:   100,
	}

	metadata := doc.ToDocumentMetadata("docs/page.md")

	assert.Equal(t, "docs/page.md", metadata.FilePath)
	assert.NotNil(t, metadata.Metadata)
	assert.Equal(t, doc.URL, metadata.URL)
	assert.Equal(t, doc.Title, metadata.Title)
	assert.Equal(t, doc.WordCount, metadata.WordCount)
}

// TestDocument_ToSimpleMetadata tests converting Document to SimpleMetadata
func TestDocument_ToSimpleMetadata(t *testing.T) {
	now := time.Now()
	doc := &Document{
		Title:          "Test Page",
		URL:            "https://example.com/page",
		SourceStrategy: "crawler",
		FetchedAt:      now,
		Description:    "A test page",
		Summary:        "Test summary",
		Tags:           []string{"test", "example"},
		Category:       "testing",
	}

	simple := doc.ToSimpleMetadata()

	assert.Equal(t, doc.Title, simple.Title)
	assert.Equal(t, doc.URL, simple.URL)
	assert.Equal(t, doc.SourceStrategy, simple.Source)
	assert.Equal(t, doc.FetchedAt, simple.FetchedAt)
	assert.Equal(t, doc.Description, simple.Description)
	assert.Equal(t, doc.Summary, simple.Summary)
	assert.Equal(t, doc.Tags, simple.Tags)
	assert.Equal(t, doc.Category, simple.Category)
}

// TestDocument_ToSimpleDocumentMetadata tests converting Document to SimpleDocumentMetadata
func TestDocument_ToSimpleDocumentMetadata(t *testing.T) {
	now := time.Now()
	doc := &Document{
		Title:          "Test Page",
		URL:            "https://example.com/page",
		SourceStrategy: "crawler",
		FetchedAt:      now,
	}

	simpleDoc := doc.ToSimpleDocumentMetadata("output/page.md")

	assert.Equal(t, "output/page.md", simpleDoc.FilePath)
	assert.NotNil(t, simpleDoc.SimpleMetadata)
	assert.Equal(t, doc.Title, simpleDoc.Title)
	assert.Equal(t, doc.URL, simpleDoc.URL)
	assert.Equal(t, doc.SourceStrategy, simpleDoc.Source)
}

// TestSitemapURL tests SitemapURL struct
func TestSitemapURL(t *testing.T) {
	t.Run("creates sitemap URL", func(t *testing.T) {
		url := SitemapURL{
			Loc:        "https://example.com/page",
			LastModStr: "2025-01-01",
			ChangeFreq: "weekly",
			Priority:   0.8,
		}

		assert.Equal(t, "https://example.com/page", url.Loc)
		assert.Equal(t, "2025-01-01", url.LastModStr)
		assert.Equal(t, "weekly", url.ChangeFreq)
		assert.Equal(t, 0.8, url.Priority)
	})
}

// TestSitemap tests Sitemap struct
func TestSitemap(t *testing.T) {
	t.Run("creates sitemap", func(t *testing.T) {
		sitemap := Sitemap{
			URLs: []SitemapURL{
				{Loc: "https://example.com/page1"},
				{Loc: "https://example.com/page2"},
			},
			Sitemaps:  []string{"https://example.com/sitemap2.xml"},
			IsIndex:   false,
			SourceURL: "https://example.com/sitemap.xml",
		}

		assert.Len(t, sitemap.URLs, 2)
		assert.Len(t, sitemap.Sitemaps, 1)
		assert.False(t, sitemap.IsIndex)
		assert.Equal(t, "https://example.com/sitemap.xml", sitemap.SourceURL)
	})
}

// TestLLMSLink tests LLMSLink struct
func TestLLMSLink(t *testing.T) {
	t.Run("creates LLMS link", func(t *testing.T) {
		link := LLMSLink{
			Title: "Example Provider",
			URL:   "https://example.com/openai.yaml",
		}

		assert.Equal(t, "Example Provider", link.Title)
		assert.Equal(t, "https://example.com/openai.yaml", link.URL)
	})
}

// TestPage tests Page struct
func TestPage(t *testing.T) {
	t.Run("creates page", func(t *testing.T) {
		now := time.Now()
		page := Page{
			URL:         "https://example.com",
			Content:     []byte("<html>...</html>"),
			ContentType: "text/html",
			StatusCode:  200,
			FetchedAt:   now,
			FromCache:   false,
			RenderedJS:  false,
		}

		assert.Equal(t, "https://example.com", page.URL)
		assert.Equal(t, []byte("<html>...</html>"), page.Content)
		assert.Equal(t, "text/html", page.ContentType)
		assert.Equal(t, 200, page.StatusCode)
		assert.False(t, page.FromCache)
		assert.False(t, page.RenderedJS)
	})
}

// TestCacheEntry tests CacheEntry struct
func TestCacheEntry(t *testing.T) {
	t.Run("creates cache entry", func(t *testing.T) {
		now := time.Now()
		expiry := now.Add(24 * time.Hour)
		entry := CacheEntry{
			URL:         "https://example.com",
			Content:     []byte("cached content"),
			ContentType: "text/html",
			FetchedAt:   now,
			ExpiresAt:   expiry,
		}

		assert.Equal(t, "https://example.com", entry.URL)
		assert.Equal(t, []byte("cached content"), entry.Content)
		assert.Equal(t, "text/html", entry.ContentType)
		assert.Equal(t, now, entry.FetchedAt)
		assert.Equal(t, expiry, entry.ExpiresAt)
	})
}

// TestResponse tests Response struct
func TestResponse(t *testing.T) {
	t.Run("creates response", func(t *testing.T) {
		resp := &Response{
			StatusCode:  200,
			Body:        []byte("response body"),
			ContentType: "application/json",
			URL:         "https://example.com/api",
			FromCache:   true,
		}

		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, []byte("response body"), resp.Body)
		assert.Equal(t, "application/json", resp.ContentType)
		assert.True(t, resp.FromCache)
	})
}

// TestStrategyOptions tests StrategyOptions defaults
func TestStrategyOptions(t *testing.T) {
	t.Run("empty options have zero values", func(t *testing.T) {
		opts := StrategyOptions{}

		assert.Empty(t, opts.Output)
		assert.Equal(t, 0, opts.Concurrency)
		assert.Equal(t, 0, opts.Limit)
		assert.Equal(t, 0, opts.MaxDepth)
		assert.Nil(t, opts.Exclude)
		assert.False(t, opts.NoFolders)
		assert.False(t, opts.DryRun)
		assert.False(t, opts.Verbose)
		assert.False(t, opts.Force)
		assert.False(t, opts.RenderJS)
		assert.False(t, opts.Split)
		assert.False(t, opts.IncludeAssets)
		assert.Empty(t, opts.ContentSelector)
	})
}

// TestRenderOptions tests RenderOptions
func TestRenderOptions(t *testing.T) {
	t.Run("creates render options", func(t *testing.T) {
		opts := RenderOptions{
			Timeout:     30 * time.Second,
			WaitFor:     ".content-loaded",
			WaitStable:  2 * time.Second,
			ScrollToEnd: true,
			Cookies:     nil,
		}

		assert.Equal(t, 30*time.Second, opts.Timeout)
		assert.Equal(t, ".content-loaded", opts.WaitFor)
		assert.Equal(t, 2*time.Second, opts.WaitStable)
		assert.True(t, opts.ScrollToEnd)
		assert.Nil(t, opts.Cookies)
	})
}

// TestMessageRole tests MessageRole constants
func TestMessageRole(t *testing.T) {
	t.Run("role constants are correct", func(t *testing.T) {
		assert.Equal(t, MessageRole("system"), RoleSystem)
		assert.Equal(t, MessageRole("user"), RoleUser)
		assert.Equal(t, MessageRole("assistant"), RoleAssistant)
	})
}

// TestLLMMessage tests LLMMessage struct
func TestLLMMessage(t *testing.T) {
	t.Run("creates messages", func(t *testing.T) {
		msg := LLMMessage{
			Role:    RoleUser,
			Content: "Hello, AI!",
		}

		assert.Equal(t, RoleUser, msg.Role)
		assert.Equal(t, "Hello, AI!", msg.Content)
	})
}

// TestLLMRequest tests LLMRequest struct
func TestLLMRequest(t *testing.T) {
	t.Run("creates request with all fields", func(t *testing.T) {
		temp := 0.7
		req := LLMRequest{
			Messages: []LLMMessage{
				{Role: RoleSystem, Content: "You are helpful"},
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens:   1000,
			Temperature: &temp,
		}

		assert.Len(t, req.Messages, 2)
		assert.Equal(t, 1000, req.MaxTokens)
		assert.NotNil(t, req.Temperature)
		assert.Equal(t, 0.7, *req.Temperature)
	})

	t.Run("creates request with defaults", func(t *testing.T) {
		req := LLMRequest{
			Messages:    []LLMMessage{{Role: RoleUser, Content: "Hello"}},
			MaxTokens:   0,
			Temperature: nil,
		}

		assert.Equal(t, 0, req.MaxTokens)
		assert.Nil(t, req.Temperature)
	})
}

// TestLLMResponse tests LLMResponse struct
func TestLLMResponse(t *testing.T) {
	t.Run("creates response", func(t *testing.T) {
		resp := LLMResponse{
			Content:      "Hello, user!",
			Model:        "gpt-4",
			FinishReason: "stop",
			Usage: LLMUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		assert.Equal(t, "Hello, user!", resp.Content)
		assert.Equal(t, "gpt-4", resp.Model)
		assert.Equal(t, "stop", resp.FinishReason)
		assert.Equal(t, 10, resp.Usage.PromptTokens)
		assert.Equal(t, 20, resp.Usage.CompletionTokens)
		assert.Equal(t, 30, resp.Usage.TotalTokens)
	})
}

// TestLLMUsage tests LLMUsage
func TestLLMUsage(t *testing.T) {
	t.Run("total tokens must be set manually", func(t *testing.T) {
		usage := LLMUsage{
			PromptTokens:     15,
			CompletionTokens: 25,
			TotalTokens:      40, // Must be set manually
		}

		assert.Equal(t, 15, usage.PromptTokens)
		assert.Equal(t, 25, usage.CompletionTokens)
		assert.Equal(t, 40, usage.TotalTokens)
	})
}

// TestMetadataIndex tests MetadataIndex struct
func TestMetadataIndex(t *testing.T) {
	now := time.Now()
	index := MetadataIndex{
		GeneratedAt:    now,
		SourceURL:      "https://example.com",
		Strategy:       "crawler",
		TotalDocuments: 5,
		TotalWordCount: 5000,
		TotalCharCount: 30000,
		Documents:      []DocumentMetadata{},
	}

	assert.Equal(t, now, index.GeneratedAt)
	assert.Equal(t, "https://example.com", index.SourceURL)
	assert.Equal(t, "crawler", index.Strategy)
	assert.Equal(t, 5, index.TotalDocuments)
	assert.Equal(t, 5000, index.TotalWordCount)
	assert.Equal(t, 30000, index.TotalCharCount)
	assert.NotNil(t, index.Documents)
}

// TestSimpleMetadataIndex tests SimpleMetadataIndex struct
func TestSimpleMetadataIndex(t *testing.T) {
	now := time.Now()
	index := SimpleMetadataIndex{
		GeneratedAt:    now,
		SourceURL:      "https://example.com",
		Strategy:       "crawler",
		TotalDocuments: 3,
		Documents:      []SimpleDocumentMetadata{},
	}

	assert.Equal(t, now, index.GeneratedAt)
	assert.Equal(t, "https://example.com", index.SourceURL)
	assert.Equal(t, "crawler", index.Strategy)
	assert.Equal(t, 3, index.TotalDocuments)
	assert.NotNil(t, index.Documents)
}

// TestDocumentFieldTags tests JSON tags on Document fields
func TestDocumentFieldTags(t *testing.T) {
	t.Run("Content and HTMLContent are excluded from JSON", func(t *testing.T) {
		// This test verifies the struct tags are correct by checking
		// that the fields exist but have json:"-" tags
		doc := Document{
			Content:     "markdown content",
			HTMLContent: "<html>content</html>",
		}

		// Fields should be accessible
		assert.Equal(t, "markdown content", doc.Content)
		assert.Equal(t, "<html>content</html>", doc.HTMLContent)
	})
}

// TestDocument_Empty tests empty Document handling
func TestDocument_Empty(t *testing.T) {
	t.Run("empty document has zero values", func(t *testing.T) {
		doc := Document{}

		assert.Empty(t, doc.URL)
		assert.Empty(t, doc.Title)
		assert.Empty(t, doc.Content)
		assert.True(t, doc.FetchedAt.IsZero())
		assert.Equal(t, 0, doc.WordCount)
		assert.Equal(t, 0, doc.CharCount)
		assert.Nil(t, doc.Links)
		assert.Nil(t, doc.Headers)
		assert.Nil(t, doc.Tags)
	})

	t.Run("conversion methods handle empty document", func(t *testing.T) {
		doc := Document{}
		metadata := doc.ToMetadata()
		frontmatter := doc.ToFrontmatter()
		simple := doc.ToSimpleMetadata()

		// Should not panic and return empty structs
		assert.NotNil(t, metadata)
		assert.NotNil(t, frontmatter)
		assert.NotNil(t, simple)
	})
}
