package strategies

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/converter"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLLMSStrategy tests creating a new LLMS strategy
func TestNewLLMSStrategy(t *testing.T) {
	deps, err := NewDependencies(DependencyOptions{
		Timeout:     10 * time.Second,
		EnableCache: false,
		OutputDir:   "/tmp",
		Flat:        false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	assert.NotNil(t, strategy)
	assert.NotNil(t, strategy.deps)
	assert.NotNil(t, strategy.fetcher)
	assert.NotNil(t, strategy.converter)
	assert.NotNil(t, strategy.markdownReader)
	assert.NotNil(t, strategy.writer)
	assert.NotNil(t, strategy.logger)
}

// TestLLMSStrategy_Name tests the Name method
func TestLLMSStrategy_Name(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewLLMSStrategy(deps)

	assert.Equal(t, "llms", strategy.Name())
}

// TestLLMSStrategy_CanHandle tests the CanHandle method
func TestLLMSStrategy_CanHandle(t *testing.T) {
	deps := &Dependencies{
		Converter: converter.NewPipeline(converter.PipelineOptions{}),
		Writer:    output.NewWriter(output.WriterOptions{BaseDir: "/tmp"}),
		Logger:    utils.NewLogger(utils.LoggerOptions{Level: "error"}),
	}

	strategy := NewLLMSStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/llms.txt", true},
		{"https://example.com/docs/llms.txt", true},
		{"http://example.com/llms.txt", true},
		{"https://example.com/llms.txt/", false},            // trailing slash makes it not match
		{"https://example.com/llms.txt?query=param", false}, // query params break the match
		{"https://example.com/docs", false},
		{"https://example.com/lms.txt", false},
		{"https://example.com/llm.txt", false},
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

// TestLLMSStrategy_Execute_Success tests successful execution
func TestLLMSStrategy_Execute_Success(t *testing.T) {
	var serverURL string
	var fetchedPages []string
	var server *httptest.Server

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			// Return llms.txt with absolute URLs pointing to this server
			llmsContent := fmt.Sprintf(`[Home](%s/)
[Getting Started](%s/getting-started)
[API Reference](%s/api)
[Guide](%s/guide)
`, serverURL, serverURL, serverURL, serverURL)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			fetchedPages = append(fetchedPages, r.URL.Path)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><title>Page</title></head><body><h1>Content</h1></body></html>`))
		}
	}))
	serverURL = server.URL
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
	assert.Len(t, fetchedPages, 4)
}

// TestLLMSStrategy_Execute_WithFilter tests URL filtering
func TestLLMSStrategy_Execute_WithFilter(t *testing.T) {
	llmsContent := `[Home](https://example.com/)
[API](https://example.com/api/v1)
[Guide](https://example.com/docs/guide)
[Blog](https://example.com/blog/post)
`

	var fetchedPages []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			fetchedPages = append(fetchedPages, r.URL.String())
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
		FilterURL:   "/docs",
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)

	// Only docs should be fetched
	for _, page := range fetchedPages {
		assert.Contains(t, page, "/docs")
	}
}

// TestLLMSStrategy_Execute_WithLimit tests limit functionality
func TestLLMSStrategy_Execute_WithLimit(t *testing.T) {
	llmsContent := `[Page1](https://example.com/1)
[Page2](https://example.com/2)
[Page3](https://example.com/3)
[Page4](https://example.com/4)
`

	var fetchedPages int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			fetchedPages++
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  2,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
	assert.LessOrEqual(t, fetchedPages, 2)
}

// TestLLMSStrategy_Execute_WithMarkdown tests markdown page handling
func TestLLMSStrategy_Execute_WithMarkdown(t *testing.T) {
	llmsContent := `[Home](https://example.com/)
[Guide](https://example.com/guide.md)
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			w.Header().Set("Content-Type", "text/markdown")
			w.Write([]byte(`# Guide

This is a markdown guide.
`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
}

// TestLLMSStrategy_Execute_ContextCancellation tests context cancellation
func TestLLMSStrategy_Execute_ContextCancellation(t *testing.T) {
	llmsContent := `[Page1](https://example.com/1)
[Page2](https://example.com/2)
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			time.Sleep(200 * time.Millisecond)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.Error(t, err)
}

// TestLLMSStrategy_Execute_EmptyLLMS tests empty llms.txt
func TestLLMSStrategy_Execute_EmptyLLMS(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
}

// TestParseLLMSLinks tests parsing llms.txt content
func TestParseLLMSLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []domain.LLMSLink
	}{
		{
			name: "simple links",
			content: `[Home](https://example.com/)
[Guide](https://example.com/guide)`,
			expected: []domain.LLMSLink{
				{Title: "Home", URL: "https://example.com/"},
				{Title: "Guide", URL: "https://example.com/guide"},
			},
		},
		{
			name: "links with titles",
			content: `[Getting Started](https://example.com/start)
[API Reference](https://example.com/api)`,
			expected: []domain.LLMSLink{
				{Title: "Getting Started", URL: "https://example.com/start"},
				{Title: "API Reference", URL: "https://example.com/api"},
			},
		},
		{
			name:     "empty content",
			content:  "",
			expected: []domain.LLMSLink{},
		},
		{
			name:     "only text",
			content:  "Just some text",
			expected: []domain.LLMSLink{},
		},
		{
			name: "ignore anchor links",
			content: `[Home](https://example.com/)
[Section](#section)
[Page](https://example.com/page)`,
			expected: []domain.LLMSLink{
				{Title: "Home", URL: "https://example.com/"},
				{Title: "Page", URL: "https://example.com/page"},
			},
		},
		{
			name: "ignore empty URLs",
			content: `[Home](https://example.com/)
[Empty]()
[Page](https://example.com/page)`,
			expected: []domain.LLMSLink{
				{Title: "Home", URL: "https://example.com/"},
				{Title: "Page", URL: "https://example.com/page"},
			},
		},
		{
			name: "multiline content",
			content: `# Documentation Index

[Getting Started](https://example.com/start)
[API Reference](https://example.com/api)

## Advanced Topics
[Guide](https://example.com/guide)
`,
			expected: []domain.LLMSLink{
				{Title: "Getting Started", URL: "https://example.com/start"},
				{Title: "API Reference", URL: "https://example.com/api"},
				{Title: "Guide", URL: "https://example.com/guide"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLLMSLinks(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFilterLLMSLinks tests filtering LLMS links
func TestFilterLLMSLinks(t *testing.T) {
	links := []domain.LLMSLink{
		{Title: "Home", URL: "https://example.com/"},
		{Title: "API", URL: "https://example.com/api/v1"},
		{Title: "Guide", URL: "https://example.com/docs/guide"},
		{Title: "Blog", URL: "https://example.com/blog/post"},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{
			name:     "filter by /docs",
			filter:   "/docs",
			expected: 1,
		},
		{
			name:     "filter by /api",
			filter:   "/api",
			expected: 1,
		},
		{
			name:     "filter by root",
			filter:   "/",
			expected: 4,
		},
		{
			name:     "filter with no matches",
			filter:   "/nonexistent",
			expected: 0,
		},
		{
			name:     "empty filter",
			filter:   "",
			expected: 4, // Empty filter means return all links
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterLLMSLinks(links, tt.filter)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

// TestLLMSStrategy_Execute_RelativeURLs tests that relative URLs in llms.txt are resolved against the base URL
func TestLLMSStrategy_Execute_RelativeURLs(t *testing.T) {
	var fetchedPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			llmsContent := `# Documentation

[Introduction](/docs/get-started/introduction.md): Getting started guide.
[Installation](/docs/get-started/installation.md): Install instructions.
[API Reference](/docs/api/reference.md): Full API docs.
`
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			fetchedPaths = append(fetchedPaths, r.URL.Path)
			w.Header().Set("Content-Type", "text/markdown")
			w.Write([]byte("# Test Page\n\nContent here."))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/docs/llms.txt", opts)
	assert.NoError(t, err)
	assert.Len(t, fetchedPaths, 3, "All 3 relative URLs should have been resolved and fetched")

	for _, p := range fetchedPaths {
		assert.True(t, strings.HasPrefix(p, "/docs/"), "Fetched path should start with /docs/: %s", p)
	}
}

// TestLLMSStrategy_Execute_TitleFallback tests using llms.txt title when page has no title
func TestLLMSStrategy_Execute_TitleFallback(t *testing.T) {
	llmsContent := `[Custom Title](https://example.com/page)
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else {
			// Return HTML without title
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Content without title</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
}

// TestLLMSStrategy_Execute_FetchError continues on error
func TestLLMSStrategy_Execute_FetchError(t *testing.T) {
	llmsContent := `[Good](https://example.com/good)
[Bad](https://example.com/bad)
[Also Good](https://example.com/also-good)
`

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(llmsContent))
		} else if strings.Contains(r.URL.Path, "bad") {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><body>Good content</body></html>`))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps, err := NewDependencies(DependencyOptions{
		Timeout:        5 * time.Second,
		EnableCache:    false,
		EnableRenderer: false,
		Concurrency:    1,
		OutputDir:      tmpDir,
		Flat:           true,
		JSONMetadata:   false,
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	strategy := NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			DryRun: true,
		},
		Concurrency: 1,
	}

	// Should not fail, should continue processing other pages
	err = strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.NoError(t, err)
}
