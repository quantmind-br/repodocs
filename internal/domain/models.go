package domain

import "time"

// Document represents a processed documentation page
type Document struct {
	URL            string              `json:"url"`
	Title          string              `json:"title"`
	Description    string              `json:"description,omitempty"`
	Content        string              `json:"-"` // Markdown content (not in JSON)
	HTMLContent    string              `json:"-"` // Original HTML (not in JSON)
	FetchedAt      time.Time           `json:"fetched_at"`
	ContentHash    string              `json:"content_hash"`
	WordCount      int                 `json:"word_count"`
	CharCount      int                 `json:"char_count"`
	Links          []string            `json:"links,omitempty"`
	Headers        map[string][]string `json:"headers,omitempty"` // h1, h2, h3...
	RenderedWithJS bool                `json:"rendered_with_js"`
	SourceStrategy string              `json:"source_strategy"`
	CacheHit       bool                `json:"cache_hit"`
	RelativePath   string              `json:"-"`
	IsRawFile      bool                `json:"-"`

	// LLM-enhanced metadata fields
	Summary  string   `json:"summary,omitempty"`  // AI-generated summary
	Tags     []string `json:"tags,omitempty"`     // AI-generated tags
	Category string   `json:"category,omitempty"` // AI-generated category
}

// Page represents a raw fetched page before conversion
type Page struct {
	URL         string
	Content     []byte
	ContentType string
	StatusCode  int
	FetchedAt   time.Time
	FromCache   bool
	RenderedJS  bool
}

// CacheEntry represents a cached page entry
type CacheEntry struct {
	URL         string    `json:"url"`
	Content     []byte    `json:"content"`
	ContentType string    `json:"content_type"`
	FetchedAt   time.Time `json:"fetched_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// SitemapURL represents a URL entry in a sitemap
type SitemapURL struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"-"`
	LastModStr string    `xml:"lastmod"`
	ChangeFreq string    `xml:"changefreq"`
	Priority   float64   `xml:"priority"`
}

// Sitemap represents a parsed sitemap
type Sitemap struct {
	URLs      []SitemapURL
	Sitemaps  []string // For sitemap index files
	IsIndex   bool
	SourceURL string
}

// LLMSLink represents a link parsed from llms.txt
type LLMSLink struct {
	Title string
	URL   string
}

// Deprecated: Metadata is replaced by SimpleMetadata for JSON output.
// Use SimpleMetadata for cleaner, LLM-evaluation-friendly metadata.
type Metadata struct {
	URL            string              `json:"url"`
	Title          string              `json:"title"`
	Description    string              `json:"description,omitempty"`
	FetchedAt      time.Time           `json:"fetched_at"`
	ContentHash    string              `json:"content_hash"`
	WordCount      int                 `json:"word_count"`
	CharCount      int                 `json:"char_count"`
	Links          []string            `json:"links,omitempty"`
	Headers        map[string][]string `json:"headers,omitempty"`
	RenderedWithJS bool                `json:"rendered_with_js"`
	SourceStrategy string              `json:"source_strategy"`
	CacheHit       bool                `json:"cache_hit"`
	Summary        string              `json:"summary,omitempty"`
	Tags           []string            `json:"tags,omitempty"`
	Category       string              `json:"category,omitempty"`
}

// ToMetadata converts a Document to Metadata
func (d *Document) ToMetadata() *Metadata {
	return &Metadata{
		URL:            d.URL,
		Title:          d.Title,
		Description:    d.Description,
		FetchedAt:      d.FetchedAt,
		ContentHash:    d.ContentHash,
		WordCount:      d.WordCount,
		CharCount:      d.CharCount,
		Links:          d.Links,
		Headers:        d.Headers,
		RenderedWithJS: d.RenderedWithJS,
		SourceStrategy: d.SourceStrategy,
		CacheHit:       d.CacheHit,
		Summary:        d.Summary,
		Tags:           d.Tags,
		Category:       d.Category,
	}
}

// Frontmatter represents YAML frontmatter for markdown files
type Frontmatter struct {
	Title      string    `yaml:"title"`
	URL        string    `yaml:"url"`
	Source     string    `yaml:"source"`
	FetchedAt  time.Time `yaml:"fetched_at"`
	RenderedJS bool      `yaml:"rendered_js"`
	WordCount  int       `yaml:"word_count"`
	Summary    string    `yaml:"summary,omitempty"`
	Tags       []string  `yaml:"tags,omitempty"`
	Category   string    `yaml:"category,omitempty"`
}

// ToFrontmatter converts a Document to Frontmatter
func (d *Document) ToFrontmatter() *Frontmatter {
	return &Frontmatter{
		Title:      d.Title,
		URL:        d.URL,
		Source:     d.SourceStrategy,
		FetchedAt:  d.FetchedAt,
		RenderedJS: d.RenderedWithJS,
		WordCount:  d.WordCount,
		Summary:    d.Summary,
		Tags:       d.Tags,
		Category:   d.Category,
	}
}

// Deprecated: MetadataIndex is replaced by SimpleMetadataIndex for JSON output.
type MetadataIndex struct {
	GeneratedAt    time.Time          `json:"generated_at"`
	SourceURL      string             `json:"source_url"`
	Strategy       string             `json:"strategy"`
	TotalDocuments int                `json:"total_documents"`
	TotalWordCount int                `json:"total_word_count"`
	TotalCharCount int                `json:"total_char_count"`
	Documents      []DocumentMetadata `json:"documents"`
}

// Deprecated: DocumentMetadata is replaced by SimpleDocumentMetadata for JSON output.
type DocumentMetadata struct {
	FilePath string `json:"file_path"`
	*Metadata
}

// ToDocumentMetadata creates a DocumentMetadata from a Document
func (d *Document) ToDocumentMetadata(filePath string) *DocumentMetadata {
	return &DocumentMetadata{
		FilePath: filePath,
		Metadata: d.ToMetadata(),
	}
}

// =============================================================================
// Simple Metadata Types (Simplified JSON output for LLM evaluation)
// =============================================================================

// SimpleMetadata represents simplified document metadata for JSON output
// This is a cleaner structure optimized for LLM evaluation, containing only
// essential fields without technical metadata like content_hash, word_count, etc.
type SimpleMetadata struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	FetchedAt   time.Time `json:"fetched_at"`
	Description string    `json:"description,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Category    string    `json:"category,omitempty"`
}

// SimpleDocumentMetadata adds file_path to SimpleMetadata for document indexing
type SimpleDocumentMetadata struct {
	FilePath string `json:"file_path"`
	*SimpleMetadata
}

// SimpleMetadataIndex represents the consolidated JSON output with simplified metadata
type SimpleMetadataIndex struct {
	GeneratedAt    time.Time                `json:"generated_at"`
	SourceURL      string                   `json:"source_url"`
	Strategy       string                   `json:"strategy"`
	TotalDocuments int                      `json:"total_documents"`
	Documents      []SimpleDocumentMetadata `json:"documents"`
}

// ToSimpleMetadata converts a Document to SimpleMetadata
func (d *Document) ToSimpleMetadata() *SimpleMetadata {
	return &SimpleMetadata{
		Title:       d.Title,
		URL:         d.URL,
		Source:      d.SourceStrategy,
		FetchedAt:   d.FetchedAt,
		Description: d.Description,
		Summary:     d.Summary,
		Tags:        d.Tags,
		Category:    d.Category,
	}
}

// ToSimpleDocumentMetadata creates a SimpleDocumentMetadata from a Document
func (d *Document) ToSimpleDocumentMetadata(filePath string) *SimpleDocumentMetadata {
	return &SimpleDocumentMetadata{
		FilePath:       filePath,
		SimpleMetadata: d.ToSimpleMetadata(),
	}
}

// =============================================================================
// LLM Types
// =============================================================================

// MessageRole represents the role in a conversation
type MessageRole string

const (
	// RoleSystem represents a system message
	RoleSystem MessageRole = "system"
	// RoleUser represents a user message
	RoleUser MessageRole = "user"
	// RoleAssistant represents an assistant message
	RoleAssistant MessageRole = "assistant"
)

// LLMMessage represents a message in the conversation
type LLMMessage struct {
	Role    MessageRole
	Content string
}

// LLMRequest represents a completion request
type LLMRequest struct {
	Messages    []LLMMessage
	MaxTokens   int      // 0 = use provider default
	Temperature *float64 // nil = use provider default
}

// LLMResponse represents the LLM response
type LLMResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        LLMUsage
}

// LLMUsage contains token usage statistics
type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
