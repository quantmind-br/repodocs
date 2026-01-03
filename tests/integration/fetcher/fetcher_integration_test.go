package fetcher_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for fetcher package with real HTTP server
// These tests verify the complete HTTP request/response cycle with retry logic,
// stealth headers, and caching integration.

func TestFetcherIntegration_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("complete fetch workflow", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify stealth headers were sent
			assert.NotEmpty(t, r.Header.Get("User-Agent"))
			assert.NotEmpty(t, r.Header.Get("Accept"))
			assert.NotEmpty(t, r.Header.Get("Accept-Language"))
			assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Integration test content</body></html>"))
		}))
		defer server.Close()

		// Setup: Create fetcher client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  2,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content
		resp, err := client.Get(ctx, server.URL)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.ContentType)
		assert.Equal(t, "<html><body>Integration test content</body></html>", string(resp.Body))
		assert.Equal(t, server.URL, resp.URL)
		assert.False(t, resp.FromCache)
	})
}

func TestFetcherIntegration_RetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("retry on 502 error", func(t *testing.T) {
		attempts := 0
		maxAttempts := 3

		// Setup: Create test server that fails first 2 times, then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(502)
				w.Write([]byte("Bad Gateway"))
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Success after retries</body></html>"))
		}))
		defer server.Close()

		// Setup: Create fetcher client with retries
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  3,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content (will retry on 502)
		startTime := time.Now()
		resp, err := client.Get(ctx, server.URL)
		duration := time.Since(startTime)

		// Verify: Response is correct after retries
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "<html><body>Success after retries</body></html>", string(resp.Body))

		// Verify: Multiple attempts were made
		assert.Equal(t, maxAttempts, attempts)

		// Verify: Retry delays occurred (took longer than single request)
		assert.Greater(t, duration, 500*time.Millisecond, "Should have retry delays")
	})

	t.Run("retry on 503 error", func(t *testing.T) {
		attempts := 0

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(503)
				w.Write([]byte("Service Unavailable"))
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Success after 503</body></html>"))
		}))
		defer server.Close()

		// Setup: Create fetcher client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  3,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content
		resp, err := client.Get(ctx, server.URL)

		// Verify: Response is correct after retries
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, 2, attempts)
	})

	t.Run("exhaust retries on persistent 502", func(t *testing.T) {
		// Setup: Create test server that always returns 502
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
			w.Write([]byte("Bad Gateway"))
		}))
		defer server.Close()

		// Setup: Create fetcher client with limited retries
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  2,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content (will exhaust retries)
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned after exhausting retries
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestFetcherIntegration_Caching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("cache hit serves from cache", func(t *testing.T) {
		requestCount := 0

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Cached content</body></html>"))
		}))
		defer server.Close()

		// Setup: Create in-memory cache
		cache := &inMemoryCache{
			data: make(map[string][]byte),
		}

		// Setup: Create fetcher client with cache enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  0,
			EnableCache: true,
			CacheTTL:    1 * time.Hour,
			Cache:       cache,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: First request (cache miss)
		resp1, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 200, resp1.StatusCode)
		assert.False(t, resp1.FromCache)
		assert.Equal(t, 1, requestCount)

		// Execute: Second request (cache hit)
		resp2, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 200, resp2.StatusCode)
		assert.True(t, resp2.FromCache)
		assert.Equal(t, 1, requestCount, "Should not make another HTTP request")
		assert.Equal(t, resp1.Body, resp2.Body)
	})

	t.Run("cache miss after TTL expiry", func(t *testing.T) {
		requestCount := 0

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Content</body></html>"))
		}))
		defer server.Close()

		// Setup: Create cache with very short TTL
		cache := &inMemoryCache{
			data:   make(map[string][]byte),
			expiry: 100 * time.Millisecond,
		}

		// Setup: Create fetcher client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  0,
			EnableCache: true,
			CacheTTL:    100 * time.Millisecond,
			Cache:       cache,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: First request
		resp1, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 1, requestCount)
		assert.False(t, resp1.FromCache)

		// Execute: Second request immediately (cache hit)
		resp2, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 1, requestCount)
		assert.True(t, resp2.FromCache)

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Execute: Third request after expiry (cache miss)
		resp3, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 2, requestCount, "Should make new HTTP request after expiry")
		assert.False(t, resp3.FromCache)
	})

	t.Run("cache disabled bypasses cache", func(t *testing.T) {
		requestCount := 0

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Content</body></html>"))
		}))
		defer server.Close()

		// Setup: Create cache
		cache := &inMemoryCache{
			data: make(map[string][]byte),
		}

		// Setup: Create fetcher client with cache disabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  0,
			EnableCache: false,
			Cache:       cache,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Multiple requests
		resp1, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 1, requestCount)
		assert.False(t, resp1.FromCache)

		resp2, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, 2, requestCount, "Should bypass cache")
		assert.False(t, resp2.FromCache)
	})
}

func TestFetcherIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("multiple concurrent requests", func(t *testing.T) {
		requestCount := 0
		var mu sync.Mutex

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			mu.Unlock()

			// Simulate some processing time
			time.Sleep(50 * time.Millisecond)

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte("<html><body>Concurrent content</body></html>"))
		}))
		defer server.Close()

		// Setup: Create fetcher client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  2,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Make concurrent requests
		numRequests := 10
		var wg sync.WaitGroup
		responses := make([]*domain.Response, numRequests)
		errors := make([]error, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				resp, err := client.Get(ctx, server.URL)
				responses[idx] = resp
				errors[idx] = err
			}(i)
		}

		wg.Wait()

		// Verify: All requests succeeded
		for i := 0; i < numRequests; i++ {
			require.NoError(t, errors[i])
			assert.NotNil(t, responses[i])
			assert.Equal(t, 200, responses[i].StatusCode)
		}

		// Verify: All requests were made
		assert.Equal(t, numRequests, requestCount)
	})
}

func TestFetcherIntegration_WithRealServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("fetch from real test server", func(t *testing.T) {
		// Setup: Create test server with realistic response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify method
			assert.Equal(t, "GET", r.Method)

			// Verify stealth headers
			assert.NotEmpty(t, r.Header.Get("User-Agent"))
			assert.Contains(t, r.Header.Get("User-Agent"), "Mozilla")
			assert.Equal(t, "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8", r.Header.Get("Accept"))
			assert.Equal(t, "gzip, deflate, br", r.Header.Get("Accept-Encoding"))

			// Return realistic HTML (don't set Content-Length manually, let Go handle it)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Integration Test</h1>
    <p>This is a test page for fetcher integration tests.</p>
</body>
</html>`))
		}))
		defer server.Close()

		// Setup: Create fetcher client with production-like settings
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			EnableCache: true,
			CacheTTL:    24 * time.Hour,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content
		resp, err := client.Get(ctx, server.URL)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.ContentType)
		assert.Contains(t, string(resp.Body), "<!DOCTYPE html>")
		assert.Contains(t, string(resp.Body), "Integration Test")
		assert.Equal(t, server.URL, resp.URL)
		assert.False(t, resp.FromCache)

		// Verify: Headers are present
		assert.NotEmpty(t, resp.Headers)
		assert.Equal(t, "text/html; charset=utf-8", resp.Headers.Get("Content-Type"))
	})
}

func TestFetcherIntegration_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("request timeout", func(t *testing.T) {
		// Setup: Create test server with delay
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep longer than client timeout
			time.Sleep(5 * time.Second)
			w.WriteHeader(200)
			w.Write([]byte("Response"))
		}))
		defer server.Close()

		// Setup: Create fetcher client with short timeout
		// Note: tls-client uses seconds (int), so minimum effective timeout is 1 second
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			Timeout:     1 * time.Second,
			MaxRetries:  0,
			EnableCache: false,
		})
		require.NoError(t, err)
		defer client.Close()

		// Execute: Fetch content (should timeout)
		startTime := time.Now()
		resp, err := client.Get(ctx, server.URL)
		duration := time.Since(startTime)

		// Verify: Request timed out
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Less(t, duration, 3*time.Second, "Should timeout before server responds")
	})

	t.Run("context timeout", func(t *testing.T) {
		// Note: tls-client does not respect context.Context for request cancellation.
		// This test documents the expected behavior: context timeout is NOT propagated
		// to the underlying HTTP request. Use ClientOptions.Timeout for request timeouts.
		t.Skip("tls-client does not support context cancellation; use ClientOptions.Timeout instead")
	})
}

// inMemoryCache is a simple in-memory cache for testing
type inMemoryCache struct {
	mu     sync.RWMutex
	data   map[string][]byte
	expiry time.Duration
}

func (c *inMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, ok := c.data[key]; ok {
		return data, nil
	}
	return nil, domain.ErrCacheMiss
}

func (c *inMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.data == nil {
		c.data = make(map[string][]byte)
	}
	c.data[key] = value

	// Simulate expiry if set
	if c.expiry > 0 {
		go func(k string) {
			time.Sleep(c.expiry)
			c.mu.Lock()
			delete(c.data, k)
			c.mu.Unlock()
		}(key)
	}

	return nil
}

func (c *inMemoryCache) Has(ctx context.Context, key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.data[key]
	return ok
}

func (c *inMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	return nil
}

func (c *inMemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string][]byte)
	return nil
}

func (c *inMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string][]byte)
	return nil
}

func (c *inMemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.data)
}

func (c *inMemoryCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"size": len(c.data),
	}
}
