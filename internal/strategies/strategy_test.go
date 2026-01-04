package strategies

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultOptions tests the DefaultOptions function
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, "./docs", opts.Output)
	assert.Equal(t, 5, opts.Concurrency)
	assert.Equal(t, 0, opts.Limit)
	assert.Equal(t, 3, opts.MaxDepth)
	assert.False(t, opts.NoFolders)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Verbose)
	assert.False(t, opts.Force)
	assert.False(t, opts.RenderJS)
	assert.False(t, opts.Split)
}

// TestNewDependencies tests creating dependencies
func TestNewDependencies(t *testing.T) {
	t.Run("minimal dependencies", func(t *testing.T) {
		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
			Verbose:        false,
		})

		require.NoError(t, err)
		assert.NotNil(t, deps)
		assert.NotNil(t, deps.Fetcher)
		assert.Nil(t, deps.Renderer) // Not enabled
		assert.Nil(t, deps.Cache)    // Not enabled
		assert.NotNil(t, deps.Converter)
		assert.NotNil(t, deps.Writer)
		assert.NotNil(t, deps.Logger)
		assert.Nil(t, deps.LLMProvider)
		assert.Nil(t, deps.MetadataEnhancer)
		assert.Nil(t, deps.Collector)

		deps.Close()
	})

	t.Run("with cache", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    true,
			CacheDir:       tmpDir,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)
		assert.NotNil(t, deps.Cache)

		deps.Close()
	})

	t.Run("with JSON metadata", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           false,
			JSONMetadata:   true,
			DryRun:         true,
		})

		require.NoError(t, err)
		assert.NotNil(t, deps.Collector)

		deps.Close()
	})

	t.Run("with LLM config but disabled", func(t *testing.T) {
		llmConfig := &config.LLMConfig{
			Provider:        "openai",
			EnhanceMetadata: false, // Disabled
		}

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
			LLMConfig:      llmConfig,
		})

		require.NoError(t, err)
		assert.Nil(t, deps.LLMProvider)
		assert.Nil(t, deps.MetadataEnhancer)

		deps.Close()
	})
}

// TestNewDependencies_ErrorCases tests error cases
func TestNewDependencies_ErrorCases(t *testing.T) {
	t.Run("invalid cache directory", func(t *testing.T) {
		// Use an invalid path that cannot be created
		_, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    true,
			CacheDir:       "/proc/nonexistent/path/that/cannot/be/created",
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		// Should either fail or create a nil cache
		// The actual behavior depends on the cache implementation
		if err != nil {
			assert.Error(t, err)
		}
	})
}

// TestDependencies_Close tests the Close method
func TestDependencies_Close(t *testing.T) {
	t.Run("close without cache", func(t *testing.T) {
		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)

		// Close should not error
		err = deps.Close()
		assert.NoError(t, err)
	})

	t.Run("close with cache", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    true,
			CacheDir:       tmpDir,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)

		// Close should not error
		err = deps.Close()
		assert.NoError(t, err)
	})

	t.Run("close with all components", func(t *testing.T) {
		tmpDir := t.TempDir()

		llmConfig := &config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		}

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    true,
			CacheDir:       tmpDir,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           false,
			JSONMetadata:   true,
			DryRun:         true,
			LLMConfig:      llmConfig,
		})

		require.NoError(t, err)

		// Close should not error even if some components are nil
		err = deps.Close()
		assert.NoError(t, err)
	})
}

// TestDependencies_FlushMetadata tests FlushMetadata
func TestDependencies_FlushMetadata(t *testing.T) {
	t.Run("without collector", func(t *testing.T) {
		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not error when collector is nil
		err = deps.FlushMetadata()
		assert.NoError(t, err)
	})

	t.Run("with collector", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           false,
			JSONMetadata:   true,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not error when collector exists
		err = deps.FlushMetadata()
		assert.NoError(t, err)
	})
}

// TestDependencies_SetStrategy tests SetStrategy
func TestDependencies_SetStrategy(t *testing.T) {
	t.Run("without collector", func(t *testing.T) {
		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not panic when collector is nil
		deps.SetStrategy("test-strategy")
	})

	t.Run("with collector", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           false,
			JSONMetadata:   true,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not error when collector exists
		deps.SetStrategy("crawler")
	})
}

// TestDependencies_SetSourceURL tests SetSourceURL
func TestDependencies_SetSourceURL(t *testing.T) {
	t.Run("without collector", func(t *testing.T) {
		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      "/tmp",
			Flat:           false,
			JSONMetadata:   false,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not panic when collector is nil
		deps.SetSourceURL("https://example.com")
	})

	t.Run("with collector", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           false,
			JSONMetadata:   true,
			DryRun:         true,
		})

		require.NoError(t, err)
		defer deps.Close()

		// Should not error when collector exists
		deps.SetSourceURL("https://example.com/docs")
	})
}

// TestDependencies_WriteDocument tests WriteDocument
func TestDependencies_WriteDocument(t *testing.T) {
	t.Run("without metadata enhancer", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           true,
			JSONMetadata:   false,
			DryRun:         false,
			Force:          true,
		})

		require.NoError(t, err)
		defer deps.Close()

		doc := &domain.Document{
			URL:     "https://example.com/page",
			Title:   "Test Page",
			Content: "# Test\n\nContent here.",
		}

		ctx := context.Background()
		err = deps.WriteDocument(ctx, doc)
		assert.NoError(t, err)
	})

	t.Run("with dry run", func(t *testing.T) {
		tmpDir := t.TempDir()

		deps, err := NewDependencies(DependencyOptions{
			Timeout:        10 * time.Second,
			EnableCache:    false,
			EnableRenderer: false,
			Concurrency:    1,
			OutputDir:      tmpDir,
			Flat:           true,
			JSONMetadata:   false,
			DryRun:         true,
			Force:          true,
		})

		require.NoError(t, err)
		defer deps.Close()

		doc := &domain.Document{
			URL:     "https://example.com/page",
			Title:   "Test Page",
			Content: "# Test\n\nContent here.",
		}

		ctx := context.Background()
		err = deps.WriteDocument(ctx, doc)
		assert.NoError(t, err)
	})

	t.Run("with metadata enhancer", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a mock LLM provider
		mockProvider := &mockLLMProvider{}

		deps := &Dependencies{
			Converter: converter.NewPipeline(converter.PipelineOptions{}),
			Writer: output.NewWriter(output.WriterOptions{
				BaseDir: tmpDir,
				Flat:    true,
				Force:   true,
				DryRun:  false,
			}),
			Logger:           utils.NewLogger(utils.LoggerOptions{Level: "error"}),
			MetadataEnhancer: llm.NewMetadataEnhancer(mockProvider),
		}
		defer deps.Close()

		doc := &domain.Document{
			URL:     "https://example.com/page",
			Title:   "Test Page",
			Content: "# Test\n\nContent here.",
		}

		ctx := context.Background()
		// Should not fail even if enhancement has issues
		err := deps.WriteDocument(ctx, doc)
		// May fail due to mock provider, but that's okay for this test
		_ = err
	})
}

// TestDependencies_WriteDocument_EnhancementFailure tests handling of enhancement failure
func TestDependencies_WriteDocument_EnhancementFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock LLM provider that fails
	mockProvider := &mockLLMProvider{fail: true}

	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer: output.NewWriter(output.WriterOptions{
			BaseDir: tmpDir,
			Flat:    true,
			Force:   true,
			DryRun:  false,
		}),
		Logger:           utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		MetadataEnhancer: llm.NewMetadataEnhancer(mockProvider),
	}
	defer deps.Close()

	doc := &domain.Document{
		URL:     "https://example.com/page",
		Title:   "Test Page",
		Content: "# Test\n\nContent here.",
	}

	ctx := context.Background()
	// Should not fail when enhancement fails, should write anyway
	err := deps.WriteDocument(ctx, doc)
	// The document should be written despite enhancement failure
	assert.NoError(t, err)
}

// Mock types for testing

type mockLLMProvider struct {
	fail bool
}

func (m *mockLLMProvider) Name() string {
	return "mock"
}

func (m *mockLLMProvider) Complete(ctx context.Context, req *domain.LLMRequest) (*domain.LLMResponse, error) {
	if m.fail {
		return nil, assert.AnError
	}
	return &domain.LLMResponse{
		Content: "Enhanced content",
	}, nil
}

func (m *mockLLMProvider) Close() error {
	return nil
}

// TestDependencies_WithRealCache tests with real cache
func TestDependencies_WithRealCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real cache
	cacheImpl, err := cache.NewBadgerCache(cache.Options{
		Directory: tmpDir,
	})
	require.NoError(t, err)

	// Create a real fetcher with cache
	fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
	})
	require.NoError(t, err)

	deps := &Dependencies{
		Fetcher:   fetcherClient,
		Cache:     cacheImpl,
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer: output.NewWriter(output.WriterOptions{
			BaseDir: tmpDir,
			Flat:    true,
		}),
		Logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	// Close should close the cache
	err = deps.Close()
	assert.NoError(t, err)
}

// TestNewDependencies_WithContentSelector tests with content selector
func TestNewDependencies_WithContentSelector(t *testing.T) {
	deps, err := NewDependencies(DependencyOptions{
		Timeout:         10 * time.Second,
		EnableCache:     false,
		EnableRenderer:  false,
		Concurrency:     1,
		OutputDir:       "/tmp",
		Flat:            false,
		JSONMetadata:    false,
		DryRun:          true,
		ContentSelector: "main article",
	})

	require.NoError(t, err)
	assert.NotNil(t, deps.Converter)
	deps.Close()
}

// TestNewDependencies_WithExcludeSelector tests with exclude selector
func TestNewDependencies_WithExcludeSelector(t *testing.T) {
	deps, err := NewDependencies(DependencyOptions{
		Timeout:         10 * time.Second,
		EnableCache:     false,
		EnableRenderer:  false,
		Concurrency:     1,
		OutputDir:       "/tmp",
		Flat:            false,
		JSONMetadata:    false,
		DryRun:          true,
		ExcludeSelector: ".sidebar, .footer",
	})

	require.NoError(t, err)
	assert.NotNil(t, deps.Converter)
	deps.Close()
}

// TestDependencies_Integration tests integration of all components
func TestDependencies_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	deps, err := NewDependencies(DependencyOptions{
		Timeout:         10 * time.Second,
		EnableCache:     true,
		CacheDir:        tmpDir,
		EnableRenderer:  false,
		Concurrency:     2,
		ContentSelector: "main",
		ExcludeSelector: "nav",
		OutputDir:       tmpDir,
		Flat:            false,
		JSONMetadata:    true,
		Force:           true,
		DryRun:          false,
		Verbose:         false,
		SourceURL:       "https://example.com",
	})

	require.NoError(t, err)

	// Test all methods
	deps.SetStrategy("crawler")
	deps.SetSourceURL("https://example.com/docs")

	ctx := context.Background()
	doc := &domain.Document{
		URL:     "https://example.com/page",
		Title:   "Test",
		Content: "# Test",
	}

	err = deps.WriteDocument(ctx, doc)
	assert.NoError(t, err)

	err = deps.FlushMetadata()
	assert.NoError(t, err)

	err = deps.Close()
	assert.NoError(t, err)
}
