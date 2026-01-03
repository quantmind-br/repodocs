package strategies

import (
	"testing"
)

// TestGetDiscoveryProbes verifies all discovery probes are configured
func TestGetDiscoveryProbes(t *testing.T) {
	probes := GetDiscoveryProbes()

	if len(probes) == 0 {
		t.Fatal("Expected at least one discovery probe")
	}

	// Verify each probe has required fields
	for i, probe := range probes {
		if probe.Path == "" {
			t.Errorf("Probe %d: missing Path", i)
		}
		if probe.Name == "" {
			t.Errorf("Probe %d: missing Name", i)
		}
		if probe.Parser == nil {
			t.Errorf("Probe %d: missing Parser function", i)
		}
	}

	// Verify specific probes exist
	probePaths := make(map[string]bool)
	for _, probe := range probes {
		probePaths[probe.Path] = true
	}

	expectedProbes := []string{
		"/llms.txt",
		"/sitemap.xml",
		"/sitemap-0.xml",
		"/sitemap_index.xml",
		"/search/search_index.json",
		"/search-index.json",
		"/index.json",
		"/search.json",
		"/hashmap.json",
	}

	for _, expected := range expectedProbes {
		if !probePaths[expected] {
			t.Errorf("Expected probe for path %s not found", expected)
		}
	}
}

// TestParseLLMsTxt tests llms.txt parsing
func TestParseLLMsTxt(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name        string
		content     string
		wantCount   int
		wantError   bool
		firstURL    string
	}{
		{
			name: "valid llms.txt with markdown links",
			content: `# Project Documentation

- [Getting Started](https://example.github.io/getting-started.md)
- [API Reference](https://example.github.io/api/)
- [Examples](https://example.github.io/examples/)
`,
			wantCount: 3,
			firstURL:  "https://example.github.io/getting-started.md",
		},
		{
			name: "relative URLs",
			content: `- [Home](/)
- [Guide](/guide)
- [API](./api/reference)
`,
			wantCount: 3,
			firstURL:  "https://example.github.io/",
		},
		{
			name:      "empty content",
			content:   "",
			wantError: true,
		},
		{
			name: "no valid links",
			content: `# Just some text

This has no markdown links.
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseLLMsTxt([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}

			if tt.firstURL != "" && len(urls) > 0 && urls[0] != tt.firstURL {
				t.Errorf("Expected first URL %s, got %s", tt.firstURL, urls[0])
			}
		})
	}
}

// TestParseSitemapXML tests sitemap.xml parsing
func TestParseSitemapXML(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name: "valid sitemap",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.github.io/</loc>
  </url>
  <url>
    <loc>https://example.github.io/guide/</loc>
  </url>
  <url>
    <loc>https://example.github.io/api/reference.html</loc>
  </url>
</urlset>`,
			wantCount: 3,
		},
		{
			name:      "empty sitemap",
			content:   `<?xml version="1.0"?><urlset></urlset>`,
			wantError: true,
		},
		{
			name:      "invalid xml",
			content:   `not xml`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseSitemapXML([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}
		})
	}
}

// TestParseSitemapIndexXML tests sitemap index parsing
func TestParseSitemapIndexXML(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name: "valid sitemap index",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>https://example.github.io/sitemap-0.xml</loc>
  </sitemap>
  <sitemap>
    <loc>https://example.github.io/sitemap-1.xml</loc>
  </sitemap>
</sitemapindex>`,
			wantCount: 2,
		},
		{
			name:      "empty index",
			content:   `<?xml version="1.0"?><sitemapindex></sitemapindex>`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseSitemapIndexXML([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}
		})
	}
}

// TestParseMkDocsIndex tests MkDocs search index parsing
func TestParseMkDocsIndex(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name:      "valid MkDocs index",
			content:   `{"docs":[{"location":"index.html","title":"Home","text":"Welcome"},{"location":"api/","title":"API","text":"API docs"}]}`,
			wantCount: 2,
		},
		{
			name:      "with fragments",
			content:   `{"docs":[{"location":"index.html#intro","title":"Home"},{"location":"guide.html#start","title":"Guide"}]}`,
			wantCount: 2,
		},
		{
			name:      "empty docs array",
			content:   `{"docs":[]}`,
			wantError: true,
		},
		{
			name:      "invalid json",
			content:   `{not json}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseMkDocsIndex([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}

			// Verify URLs are properly constructed
			for _, url := range urls {
				if len(url) == 0 {
					t.Error("Got empty URL")
				}
			}
		})
	}
}

// TestParseDocusaurusIndex tests Docusaurus search index parsing
func TestParseDocusaurusIndex(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name:      "valid Docusaurus index",
			content:   `[{"url":"/","title":"Home"},{"url":"/docs/intro","title":"Intro"},{"url":"/blog/first","title":"Blog"}]`,
			wantCount: 3,
		},
		{
			name:      "empty array",
			content:   `[]`,
			wantError: true,
		},
		{
			name:      "invalid json",
			content:   `[not json`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseDocusaurusIndex([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}
		})
	}
}

// TestParseHugoIndex tests Hugo index parsing
func TestParseHugoIndex(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name:      "with permalink field",
			content:   `[{"permalink":"https://example.github.io/","title":"Home"},{"permalink":"https://example.github.io/docs/","title":"Docs"}]`,
			wantCount: 2,
		},
		{
			name:      "with url field fallback",
			content:   `[{"url":"/","title":"Home"},{"url":"/about/","title":"About"}]`,
			wantCount: 2,
		},
		{
			name:      "empty array",
			content:   `[]`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseHugoIndex([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}
		})
	}
}

// TestParseGenericSearchIndex tests generic search.json parsing
func TestParseGenericSearchIndex(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
	}{
		{
			name:      "with url field",
			content:   `[{"url":"/","title":"Home"},{"url":"/guide/","title":"Guide"}]`,
			wantCount: 2,
		},
		{
			name:      "with permalink field",
			content:   `[{"permalink":"/"},{"permalink":"/api/"}]`,
			wantCount: 2,
		},
		{
			name:      "with location field",
			content:   `[{"location":"/"},{"location":"/docs/"}]`,
			wantCount: 2,
		},
		{
			name:      "no recognized url fields",
			content:   `[{"title":"Home"},{"title":"Guide"}]`,
			wantError: true,
		},
		{
			name:      "empty array",
			content:   `[]`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseGenericSearchIndex([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}
		})
	}
}

// TestParseVitePressHashmap tests VitePress hashmap parsing
func TestParseVitePressHashmap(t *testing.T) {
	baseURL := "https://example.github.io"

	tests := []struct {
		name      string
		content   string
		wantCount int
		wantError bool
		firstURL  string
	}{
		{
			name:      "valid hashmap",
			content:   `{"guide_getting-started.md":"hash1","api_reference.md":"hash2","index.md":"hash3"}`,
			wantCount: 3,
			// Don't check firstURL because map iteration order is not deterministic
			firstURL: "",
		},
		{
			name:      "empty hashmap",
			content:   `{}`,
			wantError: true,
		},
		{
			name:      "invalid json",
			content:   `{not json}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := ParseVitePressHashmap([]byte(tt.content), baseURL)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(urls) != tt.wantCount {
				t.Errorf("Expected %d URLs, got %d", tt.wantCount, len(urls))
			}

			if tt.firstURL != "" && len(urls) > 0 && urls[0] != tt.firstURL {
				t.Errorf("Expected first URL %s, got %s", tt.firstURL, urls[0])
			}
		})
	}
}

// TestFilterAndDeduplicateURLs tests URL filtering and deduplication
func TestFilterAndDeduplicateURLs(t *testing.T) {
	tests := []struct {
		name     string
		urls     []string
		baseURL  string
		wantLen  int
		contains []string
		excludes []string
	}{
		{
			name:    "filters external host",
			urls:    []string{"https://example.github.io/", "https://other.github.io/"},
			baseURL: "https://example.github.io",
			wantLen: 1,
			contains: []string{"https://example.github.io"},
			excludes: []string{"https://other.github.io"},
		},
		{
			name:    "deduplicates URLs",
			urls:    []string{"https://example.github.io/", "https://example.github.io/", "https://example.github.io/page"},
			baseURL: "https://example.github.io",
			wantLen: 2,
		},
		{
			name:    "removes fragments",
			urls:    []string{"https://example.github.io/#intro", "https://example.github.io/"},
			baseURL: "https://example.github.io",
			wantLen: 1,
		},
		{
			name:    "removes trailing slashes",
			urls:    []string{"https://example.github.io/page/", "https://example.github.io/page"},
			baseURL: "https://example.github.io",
			wantLen: 1,
		},
		{
			name:     "handles parse errors gracefully",
			urls:     []string{"https://example.github.io/", "http://[invalid-brackets]"},
			baseURL:  "https://example.github.io",
			wantLen:  1,
			contains: []string{"https://example.github.io"},
		},
		{
			name:    "handles empty URL (parses to empty, included)",
			urls:    []string{"https://example.github.io/", ""},
			baseURL: "https://example.github.io",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterAndDeduplicateURLs(tt.urls, tt.baseURL)

			if len(result) != tt.wantLen {
				t.Errorf("Expected %d URLs, got %d", tt.wantLen, len(result))
			}

			for _, contain := range tt.contains {
				found := false
				for _, url := range result {
					if url == contain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find URL %s in result", contain)
				}
			}

			for _, exclude := range tt.excludes {
				for _, url := range result {
					if url == exclude {
						t.Errorf("Did not expect to find URL %s in result", exclude)
					}
				}
			}
		})
	}
}

// TestShouldSkipGitHubPagesURL tests URL skipping logic
func TestShouldSkipGitHubPagesURL(t *testing.T) {
	tests := []struct {
		url     string
		wantSkip bool
	}{
		// Assets to skip
		{"https://example.github.io/assets/style.css", true},
		{"https://example.github.io/static/app.js", true},
		{"https://example.github.io/_next/static", true},
		{"https://example.github.io/_nuxt/app.js", true},
		{"https://example.github.io/img/logo.png", true},
		{"https://example.github.io/images/photo.jpg", true},
		{"https://example.github.io/media/file.mp4", true},
		{"https://example.github.io/css/style.css", true},
		{"https://example.github.io/js/app.js", true},
		{"https://example.github.io/fonts/font.woff", true},
		{"https://example.github.io/page/file.svg", true},
		{"https://example.github.io/file.ico", true},
		{"https://example.github.io/file.webp", true},
		{"https://example.github.io/file.pdf", true},
		{"https://example.github.io/feed.xml", true},
		{"https://example.github.io/rss.xml", true},
		// Content to keep
		{"https://example.github.io/", false},
		{"https://example.github.io/guide/", false},
		{"https://example.github.io/api/reference", false},
		{"https://example.github.io/docs/getting-started", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := ShouldSkipGitHubPagesURL(tt.url)
			if result != tt.wantSkip {
				t.Errorf("ShouldSkipGitHubPagesURL(%q) = %v, want %v", tt.url, result, tt.wantSkip)
			}
		})
	}
}

// TestResolveDiscoveryURL tests URL resolution
func TestResolveDiscoveryURL(t *testing.T) {
	baseURL := "https://example.github.io/docs/"

	tests := []struct {
		name     string
		href     string
		baseURL  string
		expected string
	}{
		{
			name:     "absolute URL",
			href:     "https://other.github.io/page",
			baseURL:  baseURL,
			expected: "https://other.github.io/page",
		},
		{
			name:     "absolute path",
			href:     "/guide",
			baseURL:  baseURL,
			expected: "https://example.github.io/guide",
		},
		{
			name:     "relative path",
			href:     "intro.html",
			baseURL:  baseURL,
			expected: "https://example.github.io/docs/intro.html",
		},
		{
			name:     "relative with parent",
			href:     "../other",
			baseURL:  baseURL,
			expected: "https://example.github.io/other",
		},
		{
			name:     "same directory",
			href:     "./page",
			baseURL:  baseURL,
			expected: "https://example.github.io/docs/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveDiscoveryURL(tt.href, tt.baseURL)
			if result != tt.expected {
				t.Errorf("resolveDiscoveryURL(%q, %q) = %q, want %q", tt.href, tt.baseURL, result, tt.expected)
			}
		})
	}
}
