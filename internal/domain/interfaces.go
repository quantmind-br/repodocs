package domain

import (
	"context"
	"net/http"
	"time"
)

// Strategy defines the interface for documentation extraction strategies
type Strategy interface {
	// Name returns the strategy name
	Name() string
	// CanHandle returns true if this strategy can handle the given URL
	CanHandle(url string) bool
	// Execute runs the extraction strategy
	Execute(ctx context.Context, url string, opts StrategyOptions) error
}

// StrategyOptions contains options for strategy execution
type StrategyOptions struct {
	Output          string
	Concurrency     int
	Limit           int
	MaxDepth        int
	Exclude         []string
	NoFolders       bool
	DryRun          bool
	Verbose         bool
	Force           bool
	RenderJS        bool
	Split           bool
	IncludeAssets   bool
	ContentSelector string
}

// Fetcher defines the interface for HTTP fetching with stealth capabilities
type Fetcher interface {
	// Get fetches content from a URL
	Get(ctx context.Context, url string) (*Response, error)
	// GetWithHeaders fetches content with custom headers
	GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*Response, error)
	// GetCookies returns cookies for a URL (for sharing with renderer)
	GetCookies(url string) []*http.Cookie
	// Transport returns an http.RoundTripper for integration with other HTTP clients (e.g., colly)
	Transport() http.RoundTripper
	// Close releases resources
	Close() error
}

// Response represents an HTTP response
type Response struct {
	StatusCode  int
	Body        []byte
	Headers     http.Header
	ContentType string
	URL         string
	FromCache   bool
}

// Renderer defines the interface for JavaScript rendering
type Renderer interface {
	// Render fetches and renders a page with JavaScript
	Render(ctx context.Context, url string, opts RenderOptions) (string, error)
	// Close releases browser resources
	Close() error
}

// RenderOptions contains options for page rendering
type RenderOptions struct {
	Timeout     time.Duration
	WaitFor     string        // CSS selector to wait for
	WaitStable  time.Duration // Wait for network idle
	ScrollToEnd bool          // Scroll to load lazy content
	Cookies     []*http.Cookie
}

// Cache defines the interface for content caching
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores a value in cache with TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Has checks if a key exists in cache
	Has(ctx context.Context, key string) bool
	// Delete removes a key from cache
	Delete(ctx context.Context, key string) error
	// Close releases cache resources
	Close() error
}

// Converter defines the interface for HTML to Markdown conversion
type Converter interface {
	// Convert transforms HTML content to a Document
	Convert(ctx context.Context, html string, sourceURL string) (*Document, error)
}

// Writer defines the interface for output writing
type Writer interface {
	// Write saves a document to the output directory
	Write(ctx context.Context, doc *Document) error
}

// LLMProvider defines the interface for LLM interactions
type LLMProvider interface {
	// Name returns the provider name (openai, anthropic, google)
	Name() string
	// Complete sends a request and returns the response
	Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	// Close releases resources
	Close() error
}
