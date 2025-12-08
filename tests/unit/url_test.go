package unit

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOutputDirFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		// GitHub repositories
		{
			name:     "GitHub repo",
			url:      "https://github.com/QwenLM/qwen-code",
			expected: "docs_qwen-code",
		},
		{
			name:     "GitHub repo with .git suffix",
			url:      "https://github.com/owner/repo.git",
			expected: "docs_repo",
		},
		{
			name:     "GitHub repo with trailing slash",
			url:      "https://github.com/owner/my-repo/",
			expected: "docs_my-repo",
		},

		// GitLab repositories
		{
			name:     "GitLab repo",
			url:      "https://gitlab.com/group/project",
			expected: "docs_project",
		},

		// Bitbucket repositories
		{
			name:     "Bitbucket repo",
			url:      "https://bitbucket.org/team/repo",
			expected: "docs_repo",
		},

		// Sitemap URLs
		{
			name:     "Sitemap URL",
			url:      "https://docs.crawl4ai.com/sitemap.xml",
			expected: "docs_docscrawl4aicom",
		},

		// LLMS.txt URLs
		{
			name:     "LLMS.txt URL",
			url:      "https://docs.factory.ai/llms.txt",
			expected: "docs_docsfactoryai",
		},

		// pkg.go.dev URLs
		{
			name:     "pkg.go.dev URL",
			url:      "https://pkg.go.dev/github.com/user/package",
			expected: "docs_package",
		},
		{
			name:     "pkg.go.dev with nested package",
			url:      "https://pkg.go.dev/golang.org/x/tools/gopls",
			expected: "docs_gopls",
		},

		// Generic URLs
		{
			name:     "Simple domain",
			url:      "https://example.com",
			expected: "docs_examplecom",
		},
		{
			name:     "Domain with www",
			url:      "https://www.example.com",
			expected: "docs_examplecom",
		},
		{
			name:     "Domain with subdomain",
			url:      "https://api.docs.example.com",
			expected: "docs_apidocsexamplecom",
		},
		{
			name:     "Domain with path",
			url:      "https://docs.langchain.com/docs/",
			expected: "docs_docslangchaincom",
		},

		// Edge cases
		{
			name:     "URL with port",
			url:      "http://localhost:8080/docs",
			expected: "docs_localhost",
		},
		{
			name:     "Invalid URL without scheme",
			url:      "not-a-url",
			expected: "docs", // Falls back to default for parse errors
		},
		{
			name:     "Empty string",
			url:      "",
			expected: "docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GenerateOutputDirFromURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		base     string
		expected bool
	}{
		// Basic cases
		{
			name:     "Exact match",
			target:   "https://example.com/docs",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "Target is subpath",
			target:   "https://example.com/docs/api",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "Target is deeper subpath",
			target:   "https://example.com/docs/api/v1/users",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "Target is different path",
			target:   "https://example.com/blog",
			base:     "https://example.com/docs",
			expected: false,
		},
		{
			name:     "Target is root, base has path",
			target:   "https://example.com/",
			base:     "https://example.com/docs",
			expected: false,
		},
		{
			name:     "Similar prefix but different path",
			target:   "https://example.com/docs-old",
			base:     "https://example.com/docs",
			expected: false,
		},
		{
			name:     "Empty base allows all",
			target:   "https://example.com/anything",
			base:     "",
			expected: true,
		},
		{
			name:     "Different host",
			target:   "https://other.com/docs",
			base:     "https://example.com/docs",
			expected: false,
		},
		{
			name:     "Base is root",
			target:   "https://example.com/docs/api",
			base:     "https://example.com/",
			expected: true,
		},
		{
			name:     "Trailing slashes handled",
			target:   "https://example.com/docs/",
			base:     "https://example.com/docs",
			expected: true,
		},
		{
			name:     "Case insensitive host",
			target:   "https://EXAMPLE.COM/docs/api",
			base:     "https://example.com/docs",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.HasBaseURL(tt.target, tt.base)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeForDirName(t *testing.T) {
	// Test through GenerateOutputDirFromURL since sanitizeForDirName is not exported
	tests := []struct {
		name     string
		url      string
		contains string
	}{
		{
			name:     "Removes dots",
			url:      "https://example.com",
			contains: "examplecom",
		},
		{
			name:     "Preserves hyphens",
			url:      "https://my-site.com",
			contains: "my-site",
		},
		{
			name:     "Preserves underscores",
			url:      "https://my_site.com",
			contains: "my_site",
		},
		{
			name:     "Lowercase conversion",
			url:      "https://MyDomain.COM",
			contains: "mydomaincom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.GenerateOutputDirFromURL(tt.url)
			assert.Contains(t, result, tt.contains)
		})
	}
}
