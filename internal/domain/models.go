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
	RelativePath   string              `json:"-"` // Relative path for Git-sourced files (used for output structure)
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

// Metadata represents document metadata for JSON output
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
	}
}
