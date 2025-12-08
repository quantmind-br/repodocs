package unit

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestGetAllStrategies tests that GetAllStrategies returns all available strategies
func TestGetAllStrategies(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	allStrategies := app.GetAllStrategies(deps)

	// Verify we get exactly 5 strategies
	assert.Len(t, allStrategies, 5)

	// Verify strategy names
	var names []string
	for _, strategy := range allStrategies {
		names = append(names, strategy.Name())
	}

	// Should contain: crawler, git, sitemap, llms, pkggo
	assert.Contains(t, names, "crawler")
	assert.Contains(t, names, "git")
	assert.Contains(t, names, "sitemap")
	assert.Contains(t, names, "llms")
	assert.Contains(t, names, "pkggo")
}

// TestFindMatchingStrategy tests finding the correct strategy for various URLs
func TestFindMatchingStrategy(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"GitHub repository", "https://github.com/owner/repo", "git"},
		{"GitHub repository with .git", "https://github.com/owner/repo.git", "git"},
		{"GitLab repository", "https://gitlab.com/owner/repo", "git"},
		{"Bitbucket repository", "https://bitbucket.org/owner/repo", "git"},
		{"pkg.go.dev", "https://pkg.go.dev/example.com/module@v1.0.0", "pkggo"},
		{"sitemap.xml", "https://example.com/sitemap.xml", "sitemap"},
		{"llms.txt", "https://example.com/llms.txt", "llms"},
		{"regular website", "https://example.com/docs", "crawler"},
		{"HTTP website", "http://example.com", "crawler"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			strategy := app.FindMatchingStrategy(tc.url, deps)
			assert.NotNil(t, strategy, "Should find a strategy for %s", tc.url)
			assert.Equal(t, tc.expected, strategy.Name(), "Strategy name should match expected for %s", tc.url)
		})
	}
}

// TestFindMatchingStrategy_NoMatch tests when no strategy can handle the URL
func TestFindMatchingStrategy_NoMatch(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	writer := output.NewWriter(output.WriterOptions{BaseDir: "/tmp"})

	deps := &strategies.Dependencies{
		Writer: writer,
		Logger: logger,
	}

	// Empty URL or invalid scheme should not match
	strategy := app.FindMatchingStrategy("", deps)
	assert.Nil(t, strategy)

	// File URLs or other non-HTTP schemes should not match
	strategy = app.FindMatchingStrategy("file:///path/to/file", deps)
	assert.Nil(t, strategy)
}
