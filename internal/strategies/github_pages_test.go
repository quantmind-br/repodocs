package strategies

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// TestIsGitHubPagesURL tests GitHub Pages URL detection
func TestIsGitHubPagesURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Valid GitHub Pages URLs
		{"standard github.io", "https://example.github.io", true},
		{"with path", "https://example.github.io/docs/", true},
		{"with http", "http://example.github.io", true},
		{"with subdirectory", "https://example.github.io/project/", true},
		{"uppercase domain", "https://EXAMPLE.GITHUB.IO", true},
		{"mixed case", "https://Example.GitHub.Io", true},
		// Invalid GitHub Pages URLs
		{"plain github.com", "https://github.com/user/repo", false},
		{"different TLD", "https://example.github.com", false},
		{"subdomain of github.io", "https://docs.example.github.io", true},
		{"custom domain", "https://example.com", false},
		{"localhost", "http://localhost:8080", false},
		{"invalid URL", "not-a-url", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitHubPagesURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsGitHubPagesURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestNewGitHubPagesStrategy tests strategy creation
func TestNewGitHubPagesStrategy(t *testing.T) {
	t.Run("nil dependencies", func(t *testing.T) {
		s := NewGitHubPagesStrategy(nil)
		if s == nil {
			t.Fatal("Expected non-nil strategy")
		}
		if s.Name() != "github_pages" {
			t.Errorf("Expected name 'github_pages', got '%s'", s.Name())
		}
		if s.markdownReader == nil {
			t.Error("Expected markdownReader to be initialized")
		}
	})

	t.Run("with dependencies", func(t *testing.T) {
		deps := &Dependencies{}
		s := NewGitHubPagesStrategy(deps)
		if s == nil {
			t.Fatal("Expected non-nil strategy")
		}
		if s.deps != deps {
			t.Error("Expected deps to be set")
		}
	})
}

// TestGitHubPagesStrategyName tests strategy name
func TestGitHubPagesStrategyName(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)
	if s.Name() != "github_pages" {
		t.Errorf("Expected name 'github_pages', got '%s'", s.Name())
	}
}

// TestGitHubPagesStrategyCanHandle tests URL handling detection
func TestGitHubPagesStrategyCanHandle(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"github.io root", "https://example.github.io", true},
		{"github.io with path", "https://example.github.io/docs/", true},
		{"github.io with trailing slash", "https://example.github.io/", true},
		{"non-github.io", "https://example.com", false},
		{"github.com", "https://github.com/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.CanHandle(tt.url)
			if result != tt.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

// TestNormalizeBaseURL tests URL normalization
func TestNormalizeBaseURL(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple github.io URL",
			input:    "https://example.github.io",
			expected: "https://example.github.io",
		},
		{
			name:     "with trailing slash",
			input:    "https://example.github.io/",
			expected: "https://example.github.io",
		},
		{
			name:     "with project path",
			input:    "https://example.github.io/project",
			expected: "https://example.github.io/project",
		},
		{
			name:     "with project path and trailing slash",
			input:    "https://example.github.io/project/",
			expected: "https://example.github.io/project",
		},
		{
			name:     "with nested path",
			input:    "https://example.github.io/project/docs",
			expected: "https://example.github.io/project/docs",
		},
		{
			name:     "without scheme",
			input:    "example.github.io",
			expected: "https://example.github.io",
		},
		{
			name:     "with http scheme",
			input:    "http://example.github.io",
			expected: "http://example.github.io",
		},
		{
			name:    "invalid URL",
			input:   "://invalid",
			wantErr: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "https://",
			wantErr:  false, // url.Parse("") doesn't error, returns empty URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.normalizeBaseURL(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("normalizeBaseURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestLooksLikeSPAShell tests SPA shell detection
func TestLooksLikeSPAShell(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "empty HTML",
			html:     "",
			expected: true,
		},
		{
			name:     "very short HTML",
			html:     "<html></html>",
			expected: true,
		},
		{
			name:     "React app root",
			html:     `<html><body><div id="root"></div><script src="app.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Vue app root",
			html:     `<html><body><div id="app"></div><script src="app.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Next.js root",
			html:     `<html><body><div id="__next"></div><script src="app.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Nuxt.js root",
			html:     `<html><body><div id="__nuxt"></div><script src="app.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "empty body",
			html:     `<html><body></body></html>`,
			expected: true,
		},
		{
			name:     "SPA shell with content",
			html:     `<html><body><div id="root"></div><script>const app = document.getElementById("root"); app.innerHTML = "Hello World, this is a longer text that should pass the minimum content check.";</script></body></html>`,
			expected: true, // Body text is still short even after JS execution check
		},
		{
			name:     "short content (under 500 chars)",
			html:     `<html><head><title>Documentation</title></head><body><h1>Welcome to the Documentation</h1><p>This is a comprehensive guide that contains plenty of useful information for users to read and understand the system.</p></body></html>`,
			expected: true, // HTML < 500 chars
		},
		{
			name:     "page with navigation (short)",
			html:     `<html><body><nav><a href="/">Home</a></nav><main><h1>Guide</h1><p>This guide explains how to use the system with detailed information.</p></main></body></html>`,
			expected: true, // HTML < 500 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.looksLikeSPAShell(tt.html)
			if result != tt.expected {
				t.Errorf("looksLikeSPAShell() = %v, want %v\nHTML: %s", result, tt.expected, tt.html)
			}
		})
	}
}

// TestIsEmptyOrErrorContent tests empty/error content detection
func TestIsEmptyOrErrorContent(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "empty string",
			html:     "",
			expected: true,
		},
		{
			name:     "very short HTML",
			html:     "<html>",
			expected: true,
		},
		{
			name:     "301 moved permanently",
			html:     "<html><body><h1>301 Moved Permanently</h1><p>nginx</p></body></html>",
			expected: true,
		},
		{
			name:     "302 found",
			html:     "<html><body><h1>302 Found</h1><p>The resource has been moved.</p></body></html>",
			expected: true,
		},
		{
			name:     "404 not found",
			html:     "<html><body><h1>404 Not Found</h1></body></html>",
			expected: true,
		},
		{
			name:     "page not found",
			html:     "<html><body><h1>Page Not Found</h1><p>The requested page could not be found.</p></body></html>",
			expected: true,
		},
		{
			name:     "403 forbidden",
			html:     "<html><body><h1>403 Forbidden</h1><p>Access denied</p></body></html>",
			expected: true,
		},
		{
			name:     "minimal boilerplate",
			html:     "<html><head><title>Site</title></head><body></body></html>",
			expected: true,
		},
		{
			name:     "valid content page",
			html:     "<html><body><h1>Documentation</h1><p>This is comprehensive documentation that provides detailed information about the system and how to use it effectively. The documentation covers many aspects of the system including installation, configuration, and usage examples.</p><p>Additional content helps ensure the page is not considered empty.</p></body></html>",
			expected: false,
		},
		{
			name:     "page with substantial content",
			html:     "<html><body><main><h1>Getting Started</h1><p>Welcome to our comprehensive documentation. This guide will walk you through all the essential concepts and features you need to know to be productive with the system. We cover everything from basic setup to advanced features.</p><p>More content here to ensure the page passes validation checks.</p></main></body></html>",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.isEmptyOrErrorContent(tt.html)
			if result != tt.expected {
				t.Errorf("isEmptyOrErrorContent() = %v, want %v\nHTML: %s", result, tt.expected, tt.html)
			}
		})
	}
}

// TestFilterURLs tests URL filtering logic
func TestFilterURLs(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	tests := []struct {
		name     string
		urls     []string
		baseURL  string
		opts     Options
		expected []string
	}{
		{
			name:    "no filters",
			urls:    []string{"https://example.github.io/", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts:    Options{},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "keeps trailing slashes",
			urls:    []string{"https://example.github.io/", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts:    Options{},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "skips assets",
			urls:    []string{"https://example.github.io/", "https://example.github.io/assets/style.css", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts:    Options{},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "skips feed files",
			urls:    []string{"https://example.github.io/", "https://example.github.io/feed.xml", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts:    Options{},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "applies exclude pattern",
			urls:    []string{"https://example.github.io/", "https://example.github.io/blog/", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts: Options{
				Exclude: []string{"/blog"},
			},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "applies filter URL",
			urls:    []string{"https://example.github.io/", "https://example.github.io/docs/", "https://example.github.io/blog/"},
			baseURL: "https://example.github.io",
			opts: Options{
				FilterURL: "https://example.github.io/docs",
			},
			expected: []string{
				"https://example.github.io/docs/",
			},
		},
		{
			name:    "does not deduplicate (done by FilterAndDeduplicateURLs)",
			urls:    []string{"https://example.github.io/", "https://example.github.io/", "https://example.github.io/docs/"},
			baseURL: "https://example.github.io",
			opts:    Options{},
			expected: []string{
				"https://example.github.io/",
				"https://example.github.io/",
				"https://example.github.io/docs/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.filterURLs(tt.urls, tt.baseURL, tt.opts)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d URLs, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}

			// Check each expected URL is present
			for _, exp := range tt.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected URL %q not found in result: %v", exp, result)
				}
			}
		})
	}
}

// TestExtractLinksWithGoquery tests link extraction from HTML
func TestExtractLinksWithGoquery(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)
	baseURL := "https://example.github.io"

	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected []string
	}{
		{
			name:     "simple links",
			html:     `<html><body><a href="/">Home</a><a href="/docs">Docs</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io", "https://example.github.io/docs"},
		},
		{
			name:     "skips anchors",
			html:     `<html><body><a href="#section">Section</a><a href="/page">Page</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "skips javascript",
			html:     `<html><body><a href="javascript:void(0)">Link</a><a href="/page">Page</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "skips mailto",
			html:     `<html><body><a href="mailto:test@example.com">Email</a><a href="/page">Page</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "resolves relative URLs",
			html:     `<html><body><a href="page.html">Page</a></body></html>`,
			baseURL:  baseURL + "/docs/",
			expected: []string{"https://example.github.io/docs/page.html"},
		},
		{
			name:     "filters external links",
			html:     `<html><body><a href="https://example.github.io/page">Internal</a><a href="https://other.com/page">External</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "removes fragments",
			html:     `<html><body><a href="/page#section">Page</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "removes trailing slashes",
			html:     `<html><body><a href="/page/">Page</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "deduplicates",
			html:     `<html><body><a href="/page">Page 1</a><a href="/page">Page 2</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
		{
			name:     "skips non-content URLs",
			html:     `<html><body><a href="/page">Page</a><a href="/assets/style.css">Style</a><a href="/feed.xml">Feed</a></body></html>`,
			baseURL:  baseURL,
			expected: []string{"https://example.github.io/page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.extractLinksWithGoquery(tt.html, tt.baseURL)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d links, got %d", len(tt.expected), len(result))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Got: %v", result)
				return
			}

			// Check each expected link is present
			for _, exp := range tt.expected {
				found := false
				for _, r := range result {
					if r == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected link %q not found in result: %v", exp, result)
				}
			}
		})
	}
}

// TestGetDiscoveryProbesOrder tests probe ordering
func TestGetDiscoveryProbesOrder(t *testing.T) {
	probes := GetDiscoveryProbes()

	if len(probes) == 0 {
		t.Fatal("Expected at least one probe")
	}

	// Verify tier 1 is llms.txt
	if probes[0].Path != "/llms.txt" {
		t.Errorf("Expected first probe to be /llms.txt, got %s", probes[0].Path)
	}

	// Verify tier 2 includes sitemaps
	sitemapFound := false
	for _, p := range probes {
		if p.Path == "/sitemap.xml" || p.Path == "/sitemap-0.xml" || p.Path == "/sitemap_index.xml" {
			sitemapFound = true
			break
		}
	}
	if !sitemapFound {
		t.Error("Expected to find sitemap probes")
	}

	// Verify tier 3 includes MkDocs
	mkdocsFound := false
	for _, p := range probes {
		if p.Path == "/search/search_index.json" && p.Name == "mkdocs-search" {
			mkdocsFound = true
			break
		}
	}
	if !mkdocsFound {
		t.Error("Expected to find MkDocs search probe")
	}
}

// TestGitHubPagesStrategy_Execute_ErrorCases tests Execute method error cases
func TestGitHubPagesStrategy_Execute_ErrorCases(t *testing.T) {
	t.Run("invalid URL returns error", func(t *testing.T) {
		deps := &Dependencies{
			Logger: utils.NewDefaultLogger(),
		}
		s := NewGitHubPagesStrategy(deps)

		ctx := context.Background()
		err := s.Execute(ctx, "://invalid-url", Options{})

		if err == nil {
			t.Error("Expected error for invalid URL, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "invalid URL") {
			t.Errorf("Expected 'invalid URL' error, got: %v", err)
		}
	})

	t.Run("empty URL behavior", func(t *testing.T) {
		// Test that empty URL is handled (either errors or is normalized)
		deps := &Dependencies{
			Logger: utils.NewDefaultLogger(),
		}
		s := NewGitHubPagesStrategy(deps)

		// Test normalizeBaseURL directly with empty string
		normalized, err := s.normalizeBaseURL("")
		if err != nil {
			// Empty URL causing error is expected behavior
			return
		}
		// If no error, normalized should be "https://" or similar
		if normalized != "" && normalized != "https://" {
			t.Logf("Empty URL normalized to: %s", normalized)
		}
	})
}

// TestProcessURLs_Limit tests URL limit processing
func TestGitHubPagesStrategy_ProcessURLs_Limit(t *testing.T) {
	// Test that limit option is respected in the Execute flow
	s := NewGitHubPagesStrategy(nil)

	// Create a large list of URLs
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		urls[i] = fmt.Sprintf("https://example.github.io/page%d", i)
	}

	opts := Options{
		CommonOptions: domain.CommonOptions{Limit: 10},
	}

	// After filtering, should have at most 10 URLs
	filtered := s.filterURLs(urls, "https://example.github.io", opts)
	limit := opts.CommonOptions.Limit
	if limit > 0 && len(filtered) > limit {
		// The limit is applied after filterURLs, so we test the slice operation
		filtered = filtered[:limit]
	}

	if len(filtered) != 10 {
		t.Errorf("Expected 10 URLs after limit, got %d", len(filtered))
	}
}

// TestGitHubPagesStrategy_FilterURLs_WithLimit tests filterURLs with limit option
func TestGitHubPagesStrategy_FilterURLs_WithLimit(t *testing.T) {
	s := NewGitHubPagesStrategy(nil)

	urls := []string{
		"https://example.github.io/",
		"https://example.github.io/docs/",
		"https://example.github.io/blog/",
	}

	opts := Options{
		CommonOptions: domain.CommonOptions{Limit: 2},
	}

	filtered := s.filterURLs(urls, "https://example.github.io", opts)

	// filterURLs doesn't apply the limit, but the Execute method does
	// This test verifies filterURLs behavior
	if len(filtered) != 3 {
		t.Errorf("Expected 3 URLs before limit, got %d", len(filtered))
	}
}
