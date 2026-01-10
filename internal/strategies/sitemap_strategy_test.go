package strategies

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestNewSitemapStrategy tests creating a new sitemap strategy
func TestNewSitemapStrategy(t *testing.T) {
	deps, err := NewDependencies(DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	if err != nil {
		t.Skipf("Failed to create dependencies: %v", err)
		return
	}
	defer deps.Close()

	strategy := NewSitemapStrategy(deps)

	assert.NotNil(t, strategy)
	assert.NotNil(t, strategy.deps)
	assert.NotNil(t, strategy.fetcher)
	assert.NotNil(t, strategy.converter)
	assert.NotNil(t, strategy.markdownReader)
	assert.NotNil(t, strategy.writer)
	assert.NotNil(t, strategy.logger)
}

// TestSitemapStrategy_Name tests the Name method
func TestSitemapStrategy_Name(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewSitemapStrategy(deps)

	assert.Equal(t, "sitemap", strategy.Name())
}

// TestSitemapStrategy_SetFetcher tests the SetFetcher method
func TestSitemapStrategy_SetFetcher(t *testing.T) {
	deps, err := NewDependencies(DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	if err != nil {
		t.Skipf("Failed to create dependencies: %v", err)
		return
	}
	defer deps.Close()

	strategy := NewSitemapStrategy(deps)
	originalFetcher := strategy.fetcher

	// Create a mock fetcher
	mockFetcher := &mockFetcher{}

	strategy.SetFetcher(mockFetcher)

	assert.NotEqual(t, originalFetcher, strategy.fetcher)
	assert.Equal(t, mockFetcher, strategy.fetcher)
}

// TestSitemapStrategy_CanHandle tests the CanHandle method
func TestSitemapStrategy_CanHandle(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewSitemapStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/sitemap.xml", true},
		{"https://example.com/sitemap.xml.gz", true},
		{"https://example.com/sitemap_index.xml", true},
		{"https://example.com/docs/sitemap.xml", true},
		{"https://example.com/sitemap", true},
		{"https://SITEMAP.EXAMPLE.COM/sitemap.xml", true},
		{"https://example.com/sitemap.xml.gz", true},
		{"https://example.com/feed.xml", false},
		{"https://example.com/docs", false},
		{"https://example.com/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}
