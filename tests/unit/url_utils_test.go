package unit

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Add https scheme when missing",
			input:    "example.com",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "Lowercase host",
			input:    "https://EXAMPLE.COM/path",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Remove default http port",
			input:    "http://example.com:80/path",
			expected: "http://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Remove default https port",
			input:    "https://example.com:443/path",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Keep non-default port",
			input:    "https://example.com:8080/path",
			expected: "https://example.com:8080/path",
			wantErr:  false,
		},
		{
			name:     "Add root path when missing",
			input:    "https://example.com",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "Remove trailing slash",
			input:    "https://example.com/path/",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Keep root path slash",
			input:    "https://example.com/",
			expected: "https://example.com/",
			wantErr:  false,
		},
		{
			name:     "Clean double slashes in path",
			input:    "https://example.com//path//to//page",
			expected: "https://example.com/path/to/page",
			wantErr:  false,
		},
		{
			name:     "Remove fragment",
			input:    "https://example.com/path#section",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Keep query parameters",
			input:    "https://example.com/path?foo=bar&baz=qux",
			expected: "https://example.com/path?foo=bar&baz=qux",
			wantErr:  false,
		},
		{
			name:     "Complex URL normalization",
			input:    "HTTPS://WWW.EXAMPLE.COM:443/Path/To/Page/?query=VALUE#fragment",
			expected: "https://www.example.com/Path/To/Page?query=VALUE",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.NormalizeURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestNormalizeURLWithoutQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Remove query parameters",
			input:    "https://example.com/path?foo=bar",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "URL without query stays the same",
			input:    "https://example.com/path",
			expected: "https://example.com/path",
			wantErr:  false,
		},
		{
			name:     "Remove query and fragment",
			input:    "https://example.com/path?foo=bar#section",
			expected: "https://example.com/path",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.NormalizeURLWithoutQuery(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		ref      string
		expected string
		wantErr  bool
	}{
		{
			name:     "Absolute reference unchanged",
			base:     "https://example.com/path",
			ref:      "https://other.com/page",
			expected: "https://other.com/page",
			wantErr:  false,
		},
		{
			name:     "Relative path",
			base:     "https://example.com/docs/",
			ref:      "page.html",
			expected: "https://example.com/docs/page.html",
			wantErr:  false,
		},
		{
			name:     "Root relative path",
			base:     "https://example.com/docs/guide/",
			ref:      "/api/reference",
			expected: "https://example.com/api/reference",
			wantErr:  false,
		},
		{
			name:     "Parent directory reference",
			base:     "https://example.com/docs/guide/",
			ref:      "../api/",
			expected: "https://example.com/docs/api/",
			wantErr:  false,
		},
		{
			name:     "Protocol-relative URL",
			base:     "https://example.com/",
			ref:      "//cdn.example.com/script.js",
			expected: "https://cdn.example.com/script.js",
			wantErr:  false,
		},
		{
			name:    "Invalid base URL",
			base:    "://invalid",
			ref:     "/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.ResolveURL(tt.base, tt.ref)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple domain",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "Domain with port",
			input:    "https://example.com:8080/path",
			expected: "example.com:8080",
		},
		{
			name:     "Domain with subdomain",
			input:    "https://www.example.com/path",
			expected: "www.example.com",
		},
		{
			name:     "Invalid URL returns empty",
			input:    "://invalid",
			expected: "",
		},
		{
			name:     "Empty string returns empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GetDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBaseDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple domain",
			input:    "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "Strip www subdomain",
			input:    "https://www.example.com/path",
			expected: "example.com",
		},
		{
			name:     "Keep other subdomains",
			input:    "https://api.example.com/path",
			expected: "api.example.com",
		},
		{
			name:     "Invalid URL returns empty",
			input:    "://invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GetBaseDomain(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameDomain(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "Same domain",
			url1:     "https://example.com/path1",
			url2:     "https://example.com/path2",
			expected: true,
		},
		{
			name:     "Different subdomains",
			url1:     "https://www.example.com/path",
			url2:     "https://api.example.com/path",
			expected: false,
		},
		{
			name:     "Different domains",
			url1:     "https://example.com/path",
			url2:     "https://other.com/path",
			expected: false,
		},
		{
			name:     "Case insensitive",
			url1:     "https://EXAMPLE.COM/path",
			url2:     "https://example.com/path",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsSameDomain(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameBaseDomain(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{
			name:     "Same domain",
			url1:     "https://example.com/path1",
			url2:     "https://example.com/path2",
			expected: true,
		},
		{
			name:     "www vs non-www",
			url1:     "https://www.example.com/path",
			url2:     "https://example.com/path",
			expected: true,
		},
		{
			name:     "Different domains",
			url1:     "https://example.com/path",
			url2:     "https://other.com/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsSameBaseDomain(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAbsoluteURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "HTTPS URL",
			input:    "https://example.com/path",
			expected: true,
		},
		{
			name:     "HTTP URL",
			input:    "http://example.com/path",
			expected: true,
		},
		{
			name:     "Protocol-relative URL",
			input:    "//example.com/path",
			expected: true,
		},
		{
			name:     "Relative path",
			input:    "/path/to/page",
			expected: false,
		},
		{
			name:     "Relative file",
			input:    "page.html",
			expected: false,
		},
		{
			name:     "Parent relative",
			input:    "../page.html",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsAbsoluteURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "GitHub HTTPS",
			input:    "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "GitHub with .git suffix",
			input:    "https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "GitHub SSH",
			input:    "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "GitLab",
			input:    "https://gitlab.com/user/repo",
			expected: true,
		},
		{
			name:     "Bitbucket",
			input:    "https://bitbucket.org/user/repo",
			expected: true,
		},
		{
			name:     "Regular website",
			input:    "https://example.com/path",
			expected: false,
		},
		{
			name:     "Random .git URL",
			input:    "https://example.com/file.git",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsGitURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSitemapURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Standard sitemap.xml",
			input:    "https://example.com/sitemap.xml",
			expected: true,
		},
		{
			name:     "Compressed sitemap",
			input:    "https://example.com/sitemap.xml.gz",
			expected: true,
		},
		{
			name:     "Sitemap with prefix",
			input:    "https://example.com/sitemap-posts.xml",
			expected: true,
		},
		{
			name:     "Sitemap in subdirectory",
			input:    "https://example.com/sitemaps/sitemap.xml",
			expected: true,
		},
		{
			name:     "Case insensitive",
			input:    "https://example.com/SITEMAP.XML",
			expected: true,
		},
		{
			name:     "Regular URL",
			input:    "https://example.com/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsSitemapURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsLLMSURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Standard llms.txt",
			input:    "https://example.com/llms.txt",
			expected: true,
		},
		{
			name:     "llms.txt in subdirectory",
			input:    "https://example.com/docs/llms.txt",
			expected: true,
		},
		{
			name:     "Case insensitive",
			input:    "https://example.com/LLMS.TXT",
			expected: true,
		},
		{
			name:     "Regular URL",
			input:    "https://example.com/path",
			expected: false,
		},
		{
			name:     "Similar but not llms.txt",
			input:    "https://example.com/llms.json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsLLMSURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPkgGoDevURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Standard pkg.go.dev URL",
			input:    "https://pkg.go.dev/github.com/user/package",
			expected: true,
		},
		{
			name:     "pkg.go.dev std library",
			input:    "https://pkg.go.dev/std",
			expected: true,
		},
		{
			name:     "pkg.go.dev with version",
			input:    "https://pkg.go.dev/github.com/user/package@v1.0.0",
			expected: true,
		},
		{
			name:     "Regular URL",
			input:    "https://example.com/path",
			expected: false,
		},
		{
			name:     "Go playground",
			input:    "https://go.dev/play/",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsPkgGoDevURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected []string
	}{
		{
			name:     "Extract absolute links",
			html:     `<a href="https://example.com/page1">Link 1</a><a href="https://example.com/page2">Link 2</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:     "Resolve relative links",
			html:     `<a href="/docs/api">API</a><a href="guide.html">Guide</a>`,
			baseURL:  "https://example.com/docs/",
			expected: []string{"https://example.com/docs/api", "https://example.com/docs/guide.html"},
		},
		{
			name:     "Skip anchor links",
			html:     `<a href="#section">Section</a><a href="/page">Page</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "Skip javascript links",
			html:     `<a href="javascript:void(0)">Click</a><a href="/page">Page</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "Skip mailto links",
			html:     `<a href="mailto:test@example.com">Email</a><a href="/page">Page</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "Skip tel links",
			html:     `<a href="tel:+1234567890">Call</a><a href="/page">Page</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
		{
			name:     "Empty HTML",
			html:     "",
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:     "No links",
			html:     `<p>Just some text</p>`,
			baseURL:  "https://example.com",
			expected: []string{},
		},
		{
			name:     "Single quoted href",
			html:     `<a href='https://example.com/page'>Link</a>`,
			baseURL:  "https://example.com",
			expected: []string{"https://example.com/page"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ExtractLinks(tt.html, tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterLinks(t *testing.T) {
	tests := []struct {
		name            string
		links           []string
		excludePatterns []string
		expected        []string
	}{
		{
			name:            "No patterns - all links pass",
			links:           []string{"https://example.com/page1", "https://example.com/page2"},
			excludePatterns: []string{},
			expected:        []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:            "Exclude by exact pattern",
			links:           []string{"https://example.com/api/v1", "https://example.com/docs"},
			excludePatterns: []string{".*/api/.*"},
			expected:        []string{"https://example.com/docs"},
		},
		{
			name:            "Exclude multiple patterns",
			links:           []string{"https://example.com/api/v1", "https://example.com/docs", "https://example.com/admin"},
			excludePatterns: []string{".*/api/.*", ".*/admin"},
			expected:        []string{"https://example.com/docs"},
		},
		{
			name:            "Invalid regex pattern ignored",
			links:           []string{"https://example.com/page1", "https://example.com/page2"},
			excludePatterns: []string{"[invalid"},
			expected:        []string{"https://example.com/page1", "https://example.com/page2"},
		},
		{
			name:            "Empty links",
			links:           []string{},
			excludePatterns: []string{".*/api/.*"},
			expected:        []string{},
		},
		{
			name:            "Exclude all",
			links:           []string{"https://example.com/api/v1", "https://example.com/api/v2"},
			excludePatterns: []string{".*"},
			expected:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FilterLinks(tt.links, tt.excludePatterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHTTPURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "HTTPS URL",
			input:    "https://example.com/path",
			expected: true,
		},
		{
			name:     "HTTP URL",
			input:    "http://example.com/path",
			expected: true,
		},
		{
			name:     "FTP URL",
			input:    "ftp://example.com/path",
			expected: false,
		},
		{
			name:     "Relative path",
			input:    "/path/to/page",
			expected: false,
		},
		{
			name:     "Invalid URL",
			input:    "://invalid",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsHTTPURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
