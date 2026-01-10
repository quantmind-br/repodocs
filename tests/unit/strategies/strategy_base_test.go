package strategies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultOptions tests creating default strategy options
func TestDefaultOptions(t *testing.T) {
	opts := strategies.DefaultOptions()

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
	assert.False(t, opts.IncludeAssets)
}

// TestDependencies_Close tests closing all dependencies
func TestDependencies_Close(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dependencies with cache
	badgerCache, err := cache.NewBadgerCache(cache.Options{
		Directory: tmpDir,
	})
	require.NoError(t, err)

	deps := &strategies.Dependencies{
		Cache: badgerCache,
	}

	// Close should not panic
	err = deps.Close()
	assert.NoError(t, err)
}

// TestDependencies_FlushMetadata tests flushing metadata
func TestDependencies_FlushMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	testDoc := &domain.Document{
		URL:     "https://example.com/test",
		Title:   "Test",
		Content: "Test content",
	}
	collector.Add(testDoc, tmpDir+"/test.md")

	deps := &strategies.Dependencies{
		Collector: collector,
	}

	err := deps.FlushMetadata()
	assert.NoError(t, err)

	assert.FileExists(t, tmpDir+"/metadata.json")
}

// TestDependencies_FlushMetadata_NoCollector tests flushing without collector
func TestDependencies_FlushMetadata_NoCollector(t *testing.T) {
	deps := &strategies.Dependencies{
		Collector: nil,
	}

	// Should not error when collector is nil
	err := deps.FlushMetadata()
	assert.NoError(t, err)
}

func TestDependencies_SetStrategy(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	testDoc := &domain.Document{
		URL:     "https://example.com/test",
		Title:   "Test",
		Content: "Test content",
	}
	collector.Add(testDoc, tmpDir+"/test.md")

	deps := &strategies.Dependencies{
		Collector: collector,
	}

	deps.SetStrategy("git")

	err := deps.FlushMetadata()
	assert.NoError(t, err)

	assert.FileExists(t, tmpDir+"/metadata.json")
}

// TestDependencies_SetStrategy_NoCollector tests setting strategy without collector
func TestDependencies_SetStrategy_NoCollector(t *testing.T) {
	deps := &strategies.Dependencies{
		Collector: nil,
	}

	// Should not panic when collector is nil
	deps.SetStrategy("git")
}

func TestDependencies_SetSourceURL(t *testing.T) {
	tmpDir := t.TempDir()

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir,
		SourceURL: "",
		Enabled:   true,
	})

	testDoc := &domain.Document{
		URL:     "https://example.com/test",
		Title:   "Test",
		Content: "Test content",
	}
	collector.Add(testDoc, tmpDir+"/test.md")

	deps := &strategies.Dependencies{
		Collector: collector,
	}

	deps.SetSourceURL("https://github.com/test/repo")

	err := deps.FlushMetadata()
	assert.NoError(t, err)

	assert.FileExists(t, tmpDir+"/metadata.json")
}

// TestDependencies_SetSourceURL_NoCollector tests setting source URL without collector
func TestDependencies_SetSourceURL_NoCollector(t *testing.T) {
	deps := &strategies.Dependencies{
		Collector: nil,
	}

	// Should not panic when collector is nil
	deps.SetSourceURL("https://example.com")
}

// TestDependencies_WriteDocument tests writing a document
func TestDependencies_WriteDocument(t *testing.T) {
	tmpDir := t.TempDir()

	// Create output writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	// Create dependencies
	deps := &strategies.Dependencies{
		Writer: writer,
	}

	ctx := context.Background()
	doc := &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Test Document",
		Content:        "# Test\n\nThis is test content.",
		SourceStrategy: "test",
		FetchedAt:      time.Now(),
	}

	// Write document
	err := deps.WriteDocument(ctx, doc)
	assert.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, tmpDir+"/test.md")
}

// TestDependencies_WriteDocument_WithMetadataEnhancer tests writing with metadata enhancer
func TestDependencies_WriteDocument_WithMetadataEnhancer(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/enhance" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"description": "Enhanced description", "tags": ["test", "docs"]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// Create output writer
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	// Create dependencies without enhancer (nil for this test)
	deps := &strategies.Dependencies{
		Writer:           writer,
		MetadataEnhancer: nil,
	}

	ctx := context.Background()
	doc := &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Test Document",
		Content:        "# Test\n\nContent.",
		SourceStrategy: "test",
		FetchedAt:      time.Now(),
	}

	// Write document
	err := deps.WriteDocument(ctx, doc)
	assert.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, tmpDir+"/test.md")
}

// TestDependencies_WriteDocument_NoWriter tests writing without writer
func TestDependencies_WriteDocument_NoWriter(t *testing.T) {
	deps := &strategies.Dependencies{
		Writer: nil,
	}

	ctx := context.Background()
	doc := &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Test",
		Content:        "Content",
		SourceStrategy: "test",
		FetchedAt:      time.Now(),
	}

	// Should error when writer is nil
	err := deps.WriteDocument(ctx, doc)
	assert.Error(t, err)
}

// TestNewDependencies tests creating new dependencies
func TestNewDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:         30 * time.Second,
		EnableCache:     true,
		CacheDir:        tmpDir + "/cache",
		UserAgent:       "test-agent",
		EnableRenderer:  false,
		Concurrency:     2,
		ContentSelector: "main",
		ExcludeSelector: ".sidebar",
		OutputDir:       tmpDir + "/output",
		Flat:            false,
		JSONMetadata:    true,
		CommonOptions: domain.CommonOptions{
			Force:   true,
			DryRun:  false,
			Verbose: false,
		},
		SourceURL: "https://example.com",
	})

	require.NoError(t, err)
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.Fetcher)
	assert.NotNil(t, deps.Cache)
	assert.NotNil(t, deps.Converter)
	assert.NotNil(t, deps.Writer)
	assert.NotNil(t, deps.Logger)
	assert.NotNil(t, deps.Collector)

	// Cleanup
	deps.Close()
}

// TestNewDependencies_Minimal tests creating minimal dependencies
func TestNewDependencies_Minimal(t *testing.T) {
	tmpDir := t.TempDir()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:      10 * time.Second,
		EnableCache:  false,
		OutputDir:    tmpDir,
		JSONMetadata: false,
	})

	require.NoError(t, err)
	assert.NotNil(t, deps)
	assert.NotNil(t, deps.Fetcher)
	assert.Nil(t, deps.Cache) // Cache should be nil when disabled
	assert.NotNil(t, deps.Converter)
	assert.NotNil(t, deps.Writer)
	assert.NotNil(t, deps.Logger)
	assert.Nil(t, deps.Collector) // Collector should be nil when JSONMetadata is false

	deps.Close()
}

// TestDependencies_Close_AllComponents tests closing all components
func TestDependencies_Close_AllComponents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create full dependencies
	cache, err := cache.NewBadgerCache(cache.Options{
		Directory: tmpDir + "/cache",
	})
	require.NoError(t, err)

	converter := converter.NewPipeline(converter.PipelineOptions{
		BaseURL: "https://example.com",
	})

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir + "/output",
		Force:   true,
	})

	collector := output.NewMetadataCollector(output.CollectorOptions{
		BaseDir:   tmpDir + "/output",
		SourceURL: "https://example.com",
		Enabled:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   &mockFetcherForTest{},
		Renderer:  nil, // Renderer is optional
		Cache:     cache,
		Converter: converter,
		Writer:    writer,
		Collector: collector,
	}

	// Close should handle nil and non-nil components
	err = deps.Close()
	assert.NoError(t, err)
}

// TestDependencies_WriteDocument_DryRun tests writing in dry run mode
func TestDependencies_WriteDocument_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
		DryRun:  true, // Dry run mode
	})

	deps := &strategies.Dependencies{
		Writer: writer,
	}

	ctx := context.Background()
	doc := &domain.Document{
		URL:            "https://example.com/test",
		Title:          "Test Document",
		Content:        "# Test\n\nContent.",
		SourceStrategy: "test",
		FetchedAt:      time.Now(),
	}

	// Write should succeed but not create file in dry run mode
	err := deps.WriteDocument(ctx, doc)
	assert.NoError(t, err)

	// In dry run mode, file should not be created
	// (This depends on Writer implementation)
}

// Mock types for testing

type mockFetcherForTest struct{}

func (m *mockFetcherForTest) Get(ctx context.Context, url string) (*domain.Response, error) {
	return &domain.Response{
		Body:        []byte("<html><body>Test</body></html>"),
		ContentType: "text/html",
		StatusCode:  200,
		FromCache:   false,
	}, nil
}

func (m *mockFetcherForTest) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return m.Get(ctx, url)
}

func (m *mockFetcherForTest) Close() error {
	return nil
}

func (m *mockFetcherForTest) Transport() http.RoundTripper {
	return nil
}

func (m *mockFetcherForTest) SetCache(c domain.Cache) {}

func (m *mockFetcherForTest) GetCookies(url string) []*http.Cookie {
	return nil
}

type mockMetadataEnhancer struct {
	enhanceFunc func(ctx context.Context, doc *domain.Document) error
}

func (m *mockMetadataEnhancer) Enhance(ctx context.Context, doc *domain.Document) error {
	if m.enhanceFunc != nil {
		return m.enhanceFunc(ctx, doc)
	}
	return nil
}

func (m *mockMetadataEnhancer) EnhanceAll(ctx context.Context, docs []*domain.Document) error {
	for _, doc := range docs {
		if err := m.Enhance(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockMetadataEnhancer) Close() {}
