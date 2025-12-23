package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "add https scheme",
			input:    "example.com",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "normalize host to lowercase",
			input:    "https://EXAMPLE.COM",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "remove default http port",
			input:    "http://example.com:80",
			expected: "http://example.com/",
			wantErr:  false,
		},
		{
			name:     "remove default https port",
			input:    "https://example.com:443",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "keep non-default port",
			input:    "https://example.com:8080",
			expected: "https://example.com:8080/",
			wantErr:  false,
		},
		{
			name:     "clean path",
			input:    "https://example.com/docs/../api",
			expected: "https://example.com/api",
			wantErr:  false,
		},
		{
			name:     "remove trailing slash",
			input:    "https://example.com/docs/",
			expected: "https://example.com/docs",
			wantErr:  false,
		},
		{
			name:     "keep root path slash",
			input:    "https://example.com",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "remove fragment",
			input:    "https://example.com/docs#section",
			expected: "https://example.com/docs",
			wantErr:  false,
		},
		{
			name:     "with query params",
			input:    "https://example.com/docs?param=value",
			expected: "https://example.com/docs?param=value",
			wantErr:  false,
		},
		{
			name:     "protocol-relative URL",
			input:    "//example.com/path",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "invalid URL",
			input:    "://invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNormalizeURLWithoutQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove query params",
			input:    "https://example.com/docs?param=value",
			expected: "https://example.com/docs",
		},
		{
			name:     "no query params",
			input:    "https://example.com/docs",
			expected: "https://example.com/docs",
		},
		{
			name:     "multiple query params",
			input:    "https://example.com/docs?a=1&b=2",
			expected: "https://example.com/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeURLWithoutQuery(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		ref      string
		expected string
	}{
		{
			name:     "absolute reference",
			base:     "https://example.com/docs",
			ref:      "https://other.com/page",
			expected: "https://other.com/page",
		},
		{
			name:     "relative reference",
			base:     "https://example.com/docs/",
			ref:      "api",
			expected: "https://example.com/docs/api",
		},
		{
			name:     "parent directory",
			base:     "https://example.com/docs/api",
			ref:      "../page",
			expected: "https://example.com/docs/page",
		},
		{
			name:     "root relative",
			base:     "https://example.com/docs/api",
			ref:      "/page",
			expected: "https://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveURL(tt.base, tt.ref)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple domain",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "with subdomain",
			url:      "https://docs.example.com",
			expected: "docs.example.com",
		},
		{
			name:     "with path",
			url:      "https://example.com/docs",
			expected: "example.com",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDomain(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBaseDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "simple domain",
			url:      "https://example.com",
			expected: "example.com",
		},
		{
			name:     "with www",
			url:      "https://www.example.com",
			expected: "example.com",
		},
		{
			name:     "with subdomain",
			url:      "https://docs.example.com",
			expected: "docs.example.com",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBaseDomain(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "same domain",
			url1:     "https://example.com/page1",
			url2:     "https://example.com/page2",
			expected: true,
		},
		{
			name:     "different domains",
			url1:     "https://example.com",
			url2:     "https://other.com",
			expected: false,
		},
		{
			name:     "case insensitive",
			url1:     "https://EXAMPLE.COM",
			url2:     "https://example.com",
			expected: true,
		},
		{
			name:     "different subdomains",
			url1:     "https://docs.example.com",
			url2:     "https://api.example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSameDomain(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameBaseDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "same base domain",
			url1:     "https://docs.example.com",
			url2:     "https://api.example.com",
			expected: true,
		},
		{
			name:     "different base domains",
			url1:     "https://example.com",
			url2:     "https://other.com",
			expected: false,
		},
		{
			name:     "with www",
			url1:     "https://www.example.com",
			url2:     "https://example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSameBaseDomain(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAbsoluteURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "absolute with https",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "absolute with http",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "relative path",
			url:      "/docs/page",
			expected: false,
		},
		{
			name:     "relative file",
			url:      "page.html",
			expected: false,
		},
		{
			name:     "protocol-relative",
			url:      "//example.com",
			expected: true,
		},
		{
			name:     "invalid URL",
			url:      "://invalid",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbsoluteURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHTTPURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "http",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "https",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "ftp",
			url:      "ftp://example.com",
			expected: false,
		},
		{
			name:     "invalid",
			url:      "not a url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHTTPURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub SSH",
			url:      "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "GitHub HTTPS",
			url:      "https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "GitHub without .git",
			url:      "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "GitLab",
			url:      "https://gitlab.com/user/repo.git",
			expected: true,
		},
		{
			name:     "Bitbucket",
			url:      "https://bitbucket.org/user/repo.git",
			expected: true,
		},
		{
			name:     "regular URL",
			url:      "https://example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSitemapURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "sitemap.xml",
			url:      "https://example.com/sitemap.xml",
			expected: true,
		},
		{
			name:     "sitemap.xml.gz",
			url:      "https://example.com/sitemap.xml.gz",
			expected: true,
		},
		{
			name:     "contains sitemap",
			url:      "https://example.com/sitemap_index.xml",
			expected: true,
		},
		{
			name:     "not a sitemap",
			url:      "https://example.com/page",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSitemapURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsLLMSURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "llms.txt",
			url:      "https://example.com/llms.txt",
			expected: true,
		},
		{
			name:     "llms.txt with path",
			url:      "https://example.com/docs/llms.txt",
			expected: true,
		},
		{
			name:     "not llms.txt",
			url:      "https://example.com/page.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLLMSURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPkgGoDevURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "pkg.go.dev",
			url:      "https://pkg.go.dev/github.com/user/package",
			expected: true,
		},
		{
			name:     "contains pkg.go.dev",
			url:      "https://pkg.go.dev/something",
			expected: true,
		},
		{
			name:     "not pkg.go.dev",
			url:      "https://example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPkgGoDevURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected []string
	}{
		{
			name:     "simple links",
			html:     `<a href="page1.html">Link1</a><a href="page2.html">Link2</a>`,
			baseURL:  "https://example.com/",
			expected: []string{"https://example.com/page1.html", "https://example.com/page2.html"},
		},
		{
			name:     "skip anchors",
			html:     `<a href="#section">Section</a><a href="page.html">Page</a>`,
			baseURL:  "https://example.com/",
			expected: []string{"https://example.com/page.html"},
		},
		{
			name:     "skip javascript",
			html:     `<a href="javascript:void(0)">JS</a><a href="page.html">Page</a>`,
			baseURL:  "https://example.com/",
			expected: []string{"https://example.com/page.html"},
		},
		{
			name:     "skip mailto",
			html:     `<a href="mailto:test@example.com">Email</a><a href="page.html">Page</a>`,
			baseURL:  "https://example.com/",
			expected: []string{"https://example.com/page.html"},
		},
		{
			name:     "absolute links",
			html:     `<a href="https://other.com/page">Other</a>`,
			baseURL:  "https://example.com/",
			expected: []string{"https://other.com/page"},
		},
		{
			name:     "no links",
			html:     `<div>No links here</div>`,
			baseURL:  "https://example.com/",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractLinks(tt.html, tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateOutputDirFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GitHub repository",
			url:      "https://github.com/QwenLM/qwen-code",
			expected: "docs_qwen-code",
		},
		{
			name:     "sitemap URL",
			url:      "https://docs.crawl4ai.com/sitemap.xml",
			expected: "docs_docscrawl4aicom",
		},
		{
			name:     "llms.txt URL",
			url:      "https://docs.factory.ai/llms.txt",
			expected: "docs_docsfactoryai",
		},
		{
			name:     "pkg.go.dev URL",
			url:      "https://pkg.go.dev/github.com/user/package",
			expected: "docs_package",
		},
		{
			name:     "simple URL",
			url:      "https://example.com",
			expected: "docs_examplecom",
		},
		{
			name:     "invalid URL",
			url:      "not a url",
			expected: "docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateOutputDirFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   string
		base     string
		expected bool
	}{
		{
			name:     "same base",
			target:   "https://example.com/docs/api",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "different subdomain",
			target:   "https://api.example.com/page",
			base:     "https://docs.example.com",
			expected: false,
		},
		{
			name:     "empty base",
			target:   "https://example.com/page",
			base:     "",
			expected: true,
		},
		{
			name:     "exact match",
			target:   "https://example.com/docs",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "not a subpath",
			target:   "https://example.com/blog",
			base:     "https://example.com/docs",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasBaseURL(tt.target, tt.base)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		links    []string
		patterns []string
		expected []string
	}{
		{
			name:     "filter by pattern",
			links:    []string{"https://example.com/page1", "https://example.com/page2", "https://other.com/page"},
			patterns: []string{`example\.com`},
			expected: []string{"https://other.com/page"},
		},
		{
			name:     "no filters",
			links:    []string{"https://example.com/page1", "https://example.com/page2"},
			patterns: []string{},
			expected: []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:     "invalid pattern",
			links:    []string{"https://example.com/page"},
			patterns: []string{`[`},
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "multiple patterns",
			links:    []string{"https://example.com/page", "https://other.com/page", "https://test.com/page"},
			patterns: []string{`example`, `test`},
			expected: []string{"https://other.com/page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLinks(tt.links, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
