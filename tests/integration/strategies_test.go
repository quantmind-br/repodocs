package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/stretchr/testify/assert"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("Fixture not found: %s", name)
	}
	return string(data)
}

func TestDetectStrategy_LLMS(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://example.com/llms.txt", app.StrategyLLMS},
		{"https://docs.example.com/llms.txt", app.StrategyLLMS},
		{"https://example.com/path/to/llms.txt", app.StrategyLLMS},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_Sitemap(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://example.com/sitemap.xml", app.StrategySitemap},
		{"https://example.com/sitemap.xml.gz", app.StrategySitemap},
		{"https://example.com/sitemap-posts.xml", app.StrategySitemap},
		{"https://example.com/sitemaps/sitemap.xml", app.StrategySitemap},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_Git(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://github.com/user/repo", app.StrategyGit},
		{"https://github.com/user/repo.git", app.StrategyGit},
		{"git@github.com:user/repo.git", app.StrategyGit},
		{"https://gitlab.com/user/repo", app.StrategyGit},
		{"https://bitbucket.org/user/repo", app.StrategyGit},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_PkgGo(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://pkg.go.dev/github.com/user/package", app.StrategyPkgGo},
		{"https://pkg.go.dev/golang.org/x/net", app.StrategyPkgGo},
		{"https://pkg.go.dev/std", app.StrategyPkgGo},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_Crawler(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://example.com", app.StrategyCrawler},
		{"https://docs.example.com/guide", app.StrategyCrawler},
		{"http://localhost:8080", app.StrategyCrawler},
		{"https://example.com/docs/getting-started", app.StrategyCrawler},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_Unknown(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"ftp://example.com/file.txt", app.StrategyUnknown},
		{"file:///path/to/file", app.StrategyUnknown},
		{"not-a-url", app.StrategyUnknown},
		{"", app.StrategyUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_GitHubBlobExcluded(t *testing.T) {
	// GitHub blob URLs should be treated as crawler, not git
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"https://github.com/user/repo/blob/main/README.md", app.StrategyCrawler},
		{"https://github.com/user/repo/tree/main/src", app.StrategyCrawler},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectStrategy_PkgGoBeforeGit(t *testing.T) {
	// pkg.go.dev URLs contain github.com but should be detected as PkgGo
	url := "https://pkg.go.dev/github.com/user/package"
	result := app.DetectStrategy(url)
	assert.Equal(t, app.StrategyPkgGo, result, "pkg.go.dev should be detected before GitHub")
}

func TestDetectStrategy_CaseInsensitive(t *testing.T) {
	tests := []struct {
		url      string
		expected app.StrategyType
	}{
		{"HTTPS://EXAMPLE.COM/SITEMAP.XML", app.StrategySitemap},
		{"https://GitHub.com/User/Repo", app.StrategyGit},
		{"https://PKG.GO.DEV/package", app.StrategyPkgGo},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := app.DetectStrategy(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateStrategy_ReturnsCorrectType(t *testing.T) {
	// Note: CreateStrategy requires Dependencies, which we can't fully test here
	// without creating all the dependencies. Just test that nil deps don't panic.
	t.Skip("Requires full dependency injection setup")
}

func TestGetAllStrategies_Count(t *testing.T) {
	// Note: GetAllStrategies requires Dependencies
	t.Skip("Requires full dependency injection setup")
}
