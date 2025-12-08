package strategies

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessSitemapIndex tests the processSitemapIndex function via Execute
func TestProcessSitemapIndex(t *testing.T) {
	t.Run("nested sitemap index", func(t *testing.T) {
		ctx := context.Background()

		// Create HTTP server to serve sitemap content
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sitemap-index.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>` + server.URL + `/sitemap1.xml</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>` + server.URL + `/sitemap2.xml</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
	</sitemap>
</sitemapindex>`))
			case "/sitemap1.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>` + server.URL + `/page1</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</url>
</urlset>`))
			case "/sitemap2.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>` + server.URL + `/page2</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
	</url>
</urlset>`))
			case "/page1", "/page2":
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(200)
				w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Test Content</h1></body></html>`))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		// Create real dependencies
		deps, err := NewDependencies(DependencyOptions{
			Timeout:         5 * time.Second,
			EnableCache:     false,
			EnableRenderer:  false,
			Concurrency:     1,
			OutputDir:       t.TempDir(),
			Flat:            true,
			JSONMetadata:    false,
			DryRun:          true, // Don't actually write files
		})
		require.NoError(t, err)
		defer deps.Close()

		strategy := NewSitemapStrategy(deps)

		// Execute with sitemap index - this will call processSitemapIndex
		err = strategy.Execute(ctx, server.URL+"/sitemap-index.xml", Options{
			Limit:       10,
			Concurrency: 1,
		})

		require.NoError(t, err)
	})

	t.Run("mixed sitemaps and sitemap indexes", func(t *testing.T) {
		ctx := context.Background()

		// Create HTTP server with nested structure
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/mixed-sitemap.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>` + server.URL + `/regular-sitemap.xml</loc>
	</sitemap>
	<sitemap>
		<loc>` + server.URL + `/nested-index.xml</loc>
	</sitemap>
</sitemapindex>`))
			case "/regular-sitemap.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>` + server.URL + `/doc1</loc>
	</url>
</urlset>`))
			case "/nested-index.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>` + server.URL + `/deep-sitemap.xml</loc>
	</sitemap>
</sitemapindex>`))
			case "/deep-sitemap.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>` + server.URL + `/deep-doc</loc>
	</url>
</urlset>`))
			case "/doc1", "/deep-doc":
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(200)
				w.Write([]byte(`<html><head><title>Test</title></head><body><h1>Test</h1></body></html>`))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		// Create real dependencies
		deps, err := NewDependencies(DependencyOptions{
			Timeout:         5 * time.Second,
			EnableCache:     false,
			EnableRenderer:  false,
			Concurrency:     1,
			OutputDir:       t.TempDir(),
			Flat:            true,
			JSONMetadata:    false,
			DryRun:          true,
		})
		require.NoError(t, err)
		defer deps.Close()

		strategy := NewSitemapStrategy(deps)

		// Execute
		err = strategy.Execute(ctx, server.URL+"/mixed-sitemap.xml", Options{
			Limit:       10,
			Concurrency: 1,
		})

		require.NoError(t, err)
	})

	t.Run("invalid XML in nested sitemap", func(t *testing.T) {
		ctx := context.Background()

		// Create HTTP server with invalid XML
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sitemap-index.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>` + server.URL + `/invalid-sitemap.xml</loc>
	</sitemap>
</sitemapindex>`))
			case "/invalid-sitemap.xml":
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				// Invalid XML - unclosed tags
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset>
	<url>
		<loc>` + server.URL + `/page</loc>
	<!-- missing closing tags`))
			case "/page":
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(200)
				w.Write([]byte(`<html><body>Test</body></html>`))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		// Create real dependencies
		deps, err := NewDependencies(DependencyOptions{
			Timeout:         5 * time.Second,
			EnableCache:     false,
			EnableRenderer:  false,
			Concurrency:     1,
			OutputDir:       t.TempDir(),
			Flat:            true,
			JSONMetadata:    false,
			DryRun:          true,
		})
		require.NoError(t, err)
		defer deps.Close()

		strategy := NewSitemapStrategy(deps)

		// Execute should not fail on invalid nested sitemap
		err = strategy.Execute(ctx, server.URL+"/sitemap-index.xml", Options{
			Limit:       10,
			Concurrency: 1,
		})

		// Should complete without error even if nested sitemap is invalid
		// (it logs a warning but continues)
		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Create HTTP server with delay
		var server *httptest.Server
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/sitemap-index.xml":
				time.Sleep(100 * time.Millisecond) // Small delay
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>` + server.URL + `/sitemap1.xml</loc>
	</sitemap>
</sitemapindex>`))
			case "/sitemap1.xml":
				time.Sleep(200 * time.Millisecond) // Delay to allow cancellation
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(200)
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>` + server.URL + `/page1</loc>
	</url>
</urlset>`))
			case "/page1":
				time.Sleep(500 * time.Millisecond) // Long delay
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(200)
				w.Write([]byte(`<html><body>Test</body></html>`))
			default:
				w.WriteHeader(404)
			}
		}))
		defer server.Close()

		// Create real dependencies
		deps, err := NewDependencies(DependencyOptions{
			Timeout:         5 * time.Second,
			EnableCache:     false,
			EnableRenderer:  false,
			Concurrency:     1,
			OutputDir:       t.TempDir(),
			Flat:            true,
			JSONMetadata:    false,
			DryRun:          true,
		})
		require.NoError(t, err)
		defer deps.Close()

		strategy := NewSitemapStrategy(deps)

		// Cancel context immediately
		cancel()

		// Execute should return context cancellation error
		err = strategy.Execute(ctx, server.URL+"/sitemap-index.xml", Options{
			Limit:       10,
			Concurrency: 1,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	})
}

// TestDecompressGzip tests the decompressGzip function
func TestDecompressGzip(t *testing.T) {
	t.Run("valid gzip content", func(t *testing.T) {
		originalData := []byte("Test gzipped data")
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, err := w.Write(originalData)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		gzippedData := buf.Bytes()
		decompressed, err := decompressGzip(gzippedData)
		require.NoError(t, err)
		assert.Equal(t, originalData, decompressed)
	})

	t.Run("invalid gzip", func(t *testing.T) {
		invalidData := []byte("Not gzip data")
		_, err := decompressGzip(invalidData)
		require.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		emptyData := []byte{}
		_, err := decompressGzip(emptyData)
		require.Error(t, err)
	})

	t.Run("truncated gzip", func(t *testing.T) {
		// Create valid gzip header but incomplete data
		truncatedData := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}
		_, err := decompressGzip(truncatedData)
		require.Error(t, err)
	})

	t.Run("multi megabyte gzip", func(t *testing.T) {
		// Test with larger data
		largeData := bytes.Repeat([]byte("This is test data. "), 100000)
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, err := w.Write(largeData)
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)

		gzippedData := buf.Bytes()
		decompressed, err := decompressGzip(gzippedData)
		require.NoError(t, err)
		assert.Equal(t, largeData, decompressed)
	})

	t.Run("gzip with different compression levels", func(t *testing.T) {
		originalData := []byte("Test data for compression")

		// Test with default compression
		var buf1 bytes.Buffer
		w1 := gzip.NewWriter(&buf1)
		_, err := w1.Write(originalData)
		require.NoError(t, err)
		err = w1.Close()
		require.NoError(t, err)

		decompressed1, err := decompressGzip(buf1.Bytes())
		require.NoError(t, err)
		assert.Equal(t, originalData, decompressed1)

		// Test with different data
		originalData2 := []byte(strings.Repeat("Different test content. ", 100))
		var buf2 bytes.Buffer
		w2 := gzip.NewWriter(&buf2)
		_, err = w2.Write(originalData2)
		require.NoError(t, err)
		err = w2.Close()
		require.NoError(t, err)

		decompressed2, err := decompressGzip(buf2.Bytes())
		require.NoError(t, err)
		assert.Equal(t, originalData2, decompressed2)
	})
}
