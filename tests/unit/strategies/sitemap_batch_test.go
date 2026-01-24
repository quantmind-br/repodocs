package strategies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSitemapIndex_BatchProcessing tests that URLs from first sitemap are processed
// before second sitemap is even fetched (batch-by-batch processing).
func TestSitemapIndex_BatchProcessing(t *testing.T) {
	var mu sync.Mutex
	fetchOrder := []string{}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		mu.Lock()
		fetchOrder = append(fetchOrder, path)
		mu.Unlock()

		if path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>` + server.URL + `/sitemap1.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap2.xml</loc></sitemap>
</sitemapindex>`))
			return
		}

		if path == "/sitemap1.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page1</loc></url>
	<url><loc>` + server.URL + `/page2</loc></url>
</urlset>`))
			return
		}

		if path == "/sitemap2.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page3</loc></url>
	<url><loc>` + server.URL + `/page4</loc></url>
</urlset>`))
			return
		}

		// Serve HTML for page requests
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Content for ` + path + `</h1></body></html>`))
	}))
	defer server.Close()

	// Setup dependencies
	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1 // Sequential processing to ensure order

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)

	// With batch processing, the order should be:
	// 1. sitemap.xml (main index)
	// 2. sitemap1.xml (first nested)
	// 3. page1, page2 (URLs from first sitemap processed before sitemap2 is fetched)
	// 4. sitemap2.xml (second nested)
	// 5. page3, page4 (URLs from second sitemap)

	mu.Lock()
	defer mu.Unlock()

	// Find positions
	sitemap2Pos := -1
	page1Pos := -1
	page2Pos := -1

	for i, path := range fetchOrder {
		switch path {
		case "/sitemap2.xml":
			sitemap2Pos = i
		case "/page1":
			page1Pos = i
		case "/page2":
			page2Pos = i
		}
	}

	// page1 and page2 should be fetched BEFORE sitemap2.xml
	if page1Pos > 0 && sitemap2Pos > 0 {
		assert.Less(t, page1Pos, sitemap2Pos, "page1 should be fetched before sitemap2.xml (batch processing)")
	}
	if page2Pos > 0 && sitemap2Pos > 0 {
		assert.Less(t, page2Pos, sitemap2Pos, "page2 should be fetched before sitemap2.xml (batch processing)")
	}
}

// TestSitemapIndex_GlobalLimitAcrossBatches tests that limit is respected globally
// across all nested sitemaps (e.g., limit=10, sitemap1 has 8, sitemap2 has 5 -> process 8+2=10).
func TestSitemapIndex_GlobalLimitAcrossBatches(t *testing.T) {
	var processedCount atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>` + server.URL + `/sitemap1.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap2.xml</loc></sitemap>
</sitemapindex>`))
			return
		}

		if path == "/sitemap1.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			// 8 URLs in first sitemap
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page1</loc></url>
	<url><loc>` + server.URL + `/page2</loc></url>
	<url><loc>` + server.URL + `/page3</loc></url>
	<url><loc>` + server.URL + `/page4</loc></url>
	<url><loc>` + server.URL + `/page5</loc></url>
	<url><loc>` + server.URL + `/page6</loc></url>
	<url><loc>` + server.URL + `/page7</loc></url>
	<url><loc>` + server.URL + `/page8</loc></url>
</urlset>`))
			return
		}

		if path == "/sitemap2.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			// 5 URLs in second sitemap
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page9</loc></url>
	<url><loc>` + server.URL + `/page10</loc></url>
	<url><loc>` + server.URL + `/page11</loc></url>
	<url><loc>` + server.URL + `/page12</loc></url>
	<url><loc>` + server.URL + `/page13</loc></url>
</urlset>`))
			return
		}

		// Count page requests
		if strings.HasPrefix(path, "/page") {
			processedCount.Add(1)
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Content</h1></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Limit = 10 // Limit: 8 from sitemap1 + 2 from sitemap2 = 10 total
	opts.Concurrency = 1
	opts.Force = true // Force processing even if file exists

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)

	// Should process exactly 10 pages: 8 from first + 2 from second
	assert.Equal(t, int32(10), processedCount.Load(), "Should process exactly 10 pages (8 from sitemap1 + 2 from sitemap2)")
}

// TestSitemapIndex_ErrorInNestedDoesNotBlockOthers tests that an error in one
// nested sitemap doesn't prevent processing other nested sitemaps.
func TestSitemapIndex_ErrorInNestedDoesNotBlockOthers(t *testing.T) {
	var processedCount atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>` + server.URL + `/sitemap1.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap-error.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap2.xml</loc></sitemap>
</sitemapindex>`))
			return
		}

		if path == "/sitemap1.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page1</loc></url>
</urlset>`))
			return
		}

		if path == "/sitemap-error.xml" {
			// Return 500 error for this sitemap
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if path == "/sitemap2.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page2</loc></url>
</urlset>`))
			return
		}

		if strings.HasPrefix(path, "/page") {
			processedCount.Add(1)
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Content</h1></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1
	opts.Force = true

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)

	// Should process pages from sitemap1 and sitemap2, despite error in sitemap-error
	assert.Equal(t, int32(2), processedCount.Load(), "Should process 2 pages (from sitemap1 and sitemap2, skipping errored sitemap)")
}

// TestSitemapIndex_ContextCancellationStopsProcessing tests that context cancellation
// stops processing immediately during batch processing.
func TestSitemapIndex_ContextCancellationStopsProcessing(t *testing.T) {
	var sitemapsFetched atomic.Int32
	cancelAfterFirstSitemap := make(chan struct{})

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>` + server.URL + `/sitemap1.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap2.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap3.xml</loc></sitemap>
</sitemapindex>`))
			return
		}

		if path == "/sitemap1.xml" {
			sitemapsFetched.Add(1)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page1</loc></url>
</urlset>`))
			return
		}

		if path == "/page1" {
			// Signal to cancel context after first batch is processed
			close(cancelAfterFirstSitemap)
			// Small delay to allow cancellation to propagate
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><body><h1>Page 1</h1></body></html>`))
			return
		}

		if strings.HasPrefix(path, "/sitemap") {
			sitemapsFetched.Add(1)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page-never</loc></url>
</urlset>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Content</h1></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cancel after first batch
	go func() {
		<-cancelAfterFirstSitemap
		cancel()
	}()

	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1
	opts.Force = true

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	// Should return context error
	if err != nil {
		assert.Contains(t, err.Error(), "context canceled")
	}

	// Should NOT have fetched all 3 sitemaps - only the first one and maybe the second
	// if cancellation was slow
	assert.LessOrEqual(t, sitemapsFetched.Load(), int32(2), "Should stop fetching sitemaps after context cancellation")
}

// TestSitemapIndex_EmptyBatchContinues tests that empty nested sitemaps are skipped
// and processing continues to the next one.
func TestSitemapIndex_EmptyBatchContinues(t *testing.T) {
	var processedCount atomic.Int32

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap><loc>` + server.URL + `/sitemap-empty.xml</loc></sitemap>
	<sitemap><loc>` + server.URL + `/sitemap2.xml</loc></sitemap>
</sitemapindex>`))
			return
		}

		if path == "/sitemap-empty.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`))
			return
		}

		if path == "/sitemap2.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url><loc>` + server.URL + `/page1</loc></url>
	<url><loc>` + server.URL + `/page2</loc></url>
</urlset>`))
			return
		}

		if strings.HasPrefix(path, "/page") {
			processedCount.Add(1)
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Content</h1></body></html>`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	deps := setupSitemapTestDependencies(t, tmpDir)

	strategy := strategies.NewSitemapStrategy(deps)

	ctx := context.Background()
	opts := strategies.DefaultOptions()
	opts.Output = tmpDir
	opts.Concurrency = 1
	opts.Force = true

	err := strategy.Execute(ctx, server.URL+"/sitemap.xml", opts)
	require.NoError(t, err)

	// Should process 2 pages from sitemap2, skipping empty sitemap
	assert.Equal(t, int32(2), processedCount.Load(), "Should process 2 pages from sitemap2")
}
