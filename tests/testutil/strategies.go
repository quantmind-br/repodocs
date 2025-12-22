package testutil

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// NewTestDependencies creates minimal dependencies for testing strategies
func NewTestDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()

	// Create test logger
	logger := NewTestLogger(t)

	// Create test cache
	cache := NewBadgerCache(t)

	// Create fetcher client
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     30 * time.Second,
		MaxRetries:  1,
		EnableCache: true,
		Cache:       cache,
		UserAgent:   "repodocs-test/1.0",
	})
	if err != nil {
		t.Fatalf("Failed to create fetcher: %v", err)
	}

	// Create converter pipeline
	converterPipeline := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	// Create output writer
	tmpDir := TempDir(t)
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Flat:    false,
		Force:   true,
	})

	return &strategies.Dependencies{
		Fetcher:   fetcherClient,
		Renderer:  nil, // Renderer is optional and complex to set up
		Cache:     cache,
		Converter: converterPipeline,
		Writer:    writer,
		Logger:    logger,
	}
}

// NewMinimalDependencies creates minimal dependencies for unit tests that don't need full setup
func NewMinimalDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()

	logger := utils.NewLogger(utils.LoggerOptions{
		Level:  "error",
		Format: "json",
	})

	cache := NewBadgerCache(t)

	return &strategies.Dependencies{
		Fetcher:   nil,
		Renderer:  nil,
		Cache:     cache,
		Converter: nil,
		Writer:    nil,
		Logger:    logger,
	}
}
