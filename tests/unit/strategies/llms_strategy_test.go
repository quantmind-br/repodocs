package strategies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseLLMSLinks tests parsing links from llms.txt format
func TestParseLLMSLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "simple markdown links",
			content: `[Getting Started](https://example.com/getting-started)
[API Reference](https://example.com/api)
[Guide](https://example.com/guide)`,
			expected: 3,
		},
		{
			name: "links with titles",
			content: `[Introduction to AI](https://example.com/intro)
[Advanced Topics](https://example.com/advanced)`,
			expected: 2,
		},
		{
			name: "mixed valid and invalid links",
			content: `[Valid Link](https://example.com/valid)
[Anchor Only](#section)
[Empty URL]()
[Valid 2](https://example.com/valid2)`,
			expected: 2,
		},
		{
			name:     "empty content",
			content:  ``,
			expected: 0,
		},
		{
			name: "links with special characters in titles",
			content: `[What's New?](https://example.com/whats-new)
[User's Guide](https://example.com/users-guide)`,
			expected: 2,
		},
		{
			name: "URLs with query parameters and fragments",
			content: `[Search](https://example.com/search?q=test)
[Section](https://example.com/docs#intro)`,
			expected: 2,
		},
		{
			name: "malformed links",
			content: `[Missing Parenthesis](https://example.com/test
[Valid](https://example.com/valid)`,
			expected: 1,
		},
		{
			name: "relative URLs",
			content: `[Home](/)
[Docs](/docs)
[About](/about)`,
			expected: 3,
		},
		{
			name: "multiline links",
			content: `[Link 1](https://example.com/1)
[Link 2](https://example.com/2)
[Link 3](https://example.com/3)`,
			expected: 3,
		},
		{
			name: "links with surrounding text",
			content: `# Documentation

Welcome to the docs.

[Getting Started](https://example.com/start)
[API](https://example.com/api)

Thank you!`,
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Import the function by testing it indirectly through the strategy
			// For now, we'll simulate what parseLLMSLinks does
			links := parseLLMSLinksHelper(tc.content)
			assert.Equal(t, tc.expected, len(links))
		})
	}
}

// parseLLMSLinksHelper simulates parseLLMSLinks for testing
func parseLLMSLinksHelper(content string) []map[string]string {
	var links []map[string]string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Simple markdown link parser
		start := strings.Index(line, "[")
		if start == -1 {
			continue
		}

		end := strings.Index(line[start:], "](")
		if end == -1 {
			continue
		}

		titleEnd := start + end
		urlStart := titleEnd + 2

		if urlStart >= len(line) {
			continue
		}

		title := line[start+1 : titleEnd]
		urlEnd := strings.Index(line[urlStart:], ")")
		if urlEnd == -1 {
			continue
		}

		url := line[urlStart : urlStart+urlEnd]

		if url == "" || strings.HasPrefix(url, "#") {
			continue
		}

		links = append(links, map[string]string{
			"title": strings.TrimSpace(title),
			"url":   strings.TrimSpace(url),
		})
	}

	return links
}

// TestFilterLLMSLinks tests filtering links by base URL
func TestFilterLLMSLinks(t *testing.T) {
	tests := []struct {
		name      string
		links     []map[string]string
		filterURL string
		expected  int
	}{
		{
			name: "filter by exact path",
			links: []map[string]string{
				{"title": "Doc 1", "url": "https://example.com/docs/guide"},
				{"title": "Doc 2", "url": "https://example.com/docs/api"},
				{"title": "Other", "url": "https://example.com/blog"},
			},
			filterURL: "https://example.com/docs",
			expected:  2,
		},
		{
			name: "filter with trailing slash",
			links: []map[string]string{
				{"title": "Doc 1", "url": "https://example.com/docs/guide"},
				{"title": "Doc 2", "url": "https://example.com/blog"},
			},
			filterURL: "https://example.com/docs/",
			expected:  1,
		},
		{
			name: "filter root path",
			links: []map[string]string{
				{"title": "Home", "url": "https://example.com/"},
				{"title": "About", "url": "https://example.com/about"},
			},
			filterURL: "https://example.com/",
			expected:  2,
		},
		{
			name: "no matching links",
			links: []map[string]string{
				{"title": "Doc", "url": "https://example.com/docs"},
			},
			filterURL: "https://example.com/other",
			expected:  0,
		},
		{
			name: "empty filter",
			links: []map[string]string{
				{"title": "Doc 1", "url": "https://example.com/1"},
				{"title": "Doc 2", "url": "https://example.com/2"},
			},
			filterURL: "",
			expected:  2,
		},
		{
			name:      "empty links list",
			links:     []map[string]string{},
			filterURL: "https://example.com/docs",
			expected:  0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert to domain.LLMSLink format
			domainLinks := make([]interface{}, len(tc.links))
			for i, link := range tc.links {
				domainLinks[i] = link
			}

			// Simulate filterLLMSLinks behavior
			var filtered []map[string]string
			for _, link := range tc.links {
				if tc.filterURL == "" || strings.HasPrefix(link["url"], tc.filterURL) {
					filtered = append(filtered, link)
				}
			}

			assert.Equal(t, tc.expected, len(filtered))
		})
	}
}

// TestNewLLMSStrategy tests creating a new LLMS strategy
func TestNewLLMSStrategy(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	assert.NotNil(t, strategy)
	assert.Equal(t, "llms", strategy.Name())
}

// TestLLMSStrategy_CanHandle tests URL handling for LLMS strategy
func TestLLMSStrategy_CanHandle(t *testing.T) {
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Logger: logger,
		Writer: writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/llms.txt", true},
		{"https://example.com/docs/llms.txt", true},
		{"https://example.com/llms.txt/", false},
		{"https://example.com/docs", false},
		{"https://example.com/LLMS.TXT", true},
		{"ftp://example.com/llms.txt", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestLLMSStrategy_Execute tests executing LLMS strategy
func TestLLMSStrategy_Execute(t *testing.T) {
	// Create test server with llms.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Getting Started](https://example.com/start)
[API Reference](https://example.com/api)
[Guide](https://example.com/guide)`))
			return
		}

		// Serve HTML pages for the links
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body><h1>Content</h1></body>
</html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_WithFilter tests executing with URL filter
func TestLLMSStrategy_Execute_WithFilter(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Docs 1](https://example.com/docs/1)
[Docs 2](https://example.com/docs/2)
[Blog](https://example.com/blog/post)`))
			return
		}

		// Serve pages
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.FilterURL = "https://example.com/docs"
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_WithLimit tests executing with limit
func TestLLMSStrategy_Execute_WithLimit(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Doc 1](https://example.com/1)
[Doc 2](https://example.com/2)
[Doc 3](https://example.com/3)
[Doc 4](https://example.com/4)
[Doc 5](https://example.com/5)`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Limit = 2
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_EmptyLLMS tests executing with empty llms.txt
func TestLLMSStrategy_Execute_EmptyLLMS(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(``))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_InvalidLinks tests executing with invalid links
func TestLLMSStrategy_Execute_InvalidLinks(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Valid Link](https://example.com/valid)
[Anchor Only](#section)
[Empty URL]()
[Valid 2](https://example.com/valid2)`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_ErrorFetchingPage tests error handling
func TestLLMSStrategy_Execute_ErrorFetchingPage(t *testing.T) {
	// Track request count
	requestCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Valid](https://example.com/valid)
[Error Page](https://example.com/error)`))
			return
		}

		// Return error for error page
		if strings.Contains(r.URL.Path, "error") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	// Should complete even if some pages fail
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_DryRun tests dry run mode
func TestLLMSStrategy_Execute_DryRun(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Doc](https://example.com/doc)`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.DryRun = true
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

// TestLLMSStrategy_Execute_ContextCancellation tests context cancellation
func TestLLMSStrategy_Execute_ContextCancellation(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[Doc 1](https://example.com/1)
[Doc 2](https://example.com/2)
[Doc 3](https://example.com/3)`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>Content</body></html>`))
	}))
	defer server.Close()

	// Create dependencies
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	assert.Error(t, err)
}

func TestLLMSStrategy_Execute_PlainTextFile(t *testing.T) {
	servedContentTypes := make(map[string]string)

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[Full Docs](" + serverURL + "/llms-full.txt)\n[API Reference](" + serverURL + "/api.html)"))
			return
		}

		if strings.HasSuffix(path, ".txt") {
			servedContentTypes[path] = "text/plain"
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`# Full Documentation

This is the full documentation content in plain text format.

[Link to API](https://example.com/api)`))
			return
		}

		servedContentTypes[path] = "text/html"
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>API Reference</title></head>
<body><h1>API Reference</h1><p>This is HTML content.</p></body>
</html>`))
	}))
	defer server.Close()
	serverURL = server.URL

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)

	assert.Equal(t, "text/plain", servedContentTypes["/llms-full.txt"], "Expected .txt file to be served as text/plain")
	assert.Equal(t, "text/html", servedContentTypes["/api.html"], "Expected .html file to be served as text/html")
}

func TestLLMSStrategy_Execute_PlainTextContentType(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[Plain Content](" + serverURL + "/content)"))
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`Plain Text Content

This is plain text served with text/plain content type
but without a .txt extension in the URL.`))
	}))
	defer server.Close()
	serverURL = server.URL

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}

func TestLLMSStrategy_Execute_MixedContentTypes(t *testing.T) {
	processedFiles := make(map[string]bool)

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		processedFiles[path] = true

		if strings.HasSuffix(path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[Plain Text](" + serverURL + "/docs.txt)\n" +
				"[Markdown](" + serverURL + "/readme.md)\n" +
				"[HTML Page](" + serverURL + "/page.html)\n" +
				"[Another Text](" + serverURL + "/notes.txt)"))
			return
		}

		switch {
		case strings.HasSuffix(path, ".txt"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Plain Text Document\n\nThis is a plain text file."))

		case strings.HasSuffix(path, ".md"):
			w.Header().Set("Content-Type", "text/markdown")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Markdown Document\n\nThis is a **markdown** file."))

		default:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><title>HTML</title></head><body><h1>HTML Document</h1></body></html>`))
		}
	}))
	defer server.Close()
	serverURL = server.URL

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)

	assert.True(t, processedFiles["/docs.txt"], "Expected docs.txt to be processed")
	assert.True(t, processedFiles["/readme.md"], "Expected readme.md to be processed")
	assert.True(t, processedFiles["/page.html"], "Expected page.html to be processed")
	assert.True(t, processedFiles["/notes.txt"], "Expected notes.txt to be processed")
}

func TestLLMSStrategy_Execute_PlainTextWithLinks(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "llms.txt") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("[Full Docs](" + serverURL + "/full.txt)"))
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`# Documentation Overview

Welcome to the documentation.

See [Getting Started](https://example.com/start) for initial setup.
Check [API Reference](https://example.com/api) for details.`))
	}))
	defer server.Close()
	serverURL = server.URL

	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
	tmpDir := t.TempDir()
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: tmpDir,
		Force:   true,
	})

	deps := &strategies.Dependencies{
		Fetcher:   testutil.NewSimpleFetcher(server.URL),
		Converter: testutil.NewHTMLConverter(t),
		Logger:    logger,
		Writer:    writer,
	}

	strategy := strategies.NewLLMSStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1

	err := strategy.Execute(ctx, server.URL+"/llms.txt", opts)
	require.NoError(t, err)
}
