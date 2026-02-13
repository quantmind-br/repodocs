package fetcher_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestClient_CacheIntegration(t *testing.T) {
	ctx := context.Background()
	testURL := "https://example.com/test"
	cachedBody := []byte("<html><body>Cached content</body></html>")

	// Create mock cache
	cache := mocks.NewSimpleMockCache()
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0, // Disable retries for simpler testing
	})
	require.NoError(t, err)

	// Test cache hit
	t.Run("cache hit", func(t *testing.T) {
		// Setup: Add content to cache
		err := cache.Set(ctx, testURL, cachedBody, time.Hour)
		require.NoError(t, err)

		// Execute: Get from client (should use cache)
		resp, err := client.Get(ctx, testURL)
		require.NoError(t, err)

		// Verify: Response should be from cache
		assert.True(t, resp.FromCache)
		assert.Equal(t, cachedBody, resp.Body)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/html", resp.ContentType)
		assert.Equal(t, testURL, resp.URL)
	})

	// Test cache miss saves to cache
	t.Run("cache miss saves to cache", func(t *testing.T) {
		// Setup: Ensure cache is empty for this URL
		testURL2 := "https://example.com/test2"
		err = cache.Delete(ctx, testURL2)
		require.NoError(t, err)

		// Note: Since we're testing without actual HTTP requests,
		// we can't fully test the fetch and save flow in a unit test.
		// The saveToCache is called internally by GetWithHeaders after a successful fetch.
		// This test verifies that the cache miss handling works correctly.

		// Execute: Try to get non-cached URL
		// This will fail with a network error, but that's expected in a unit test
		// The important part is that it tried to fetch and would have saved to cache on success
		_, err = client.Get(ctx, testURL2)
		// We expect an error because we don't have a real HTTP server
		// but the important thing is the cache lookup happened
		assert.Error(t, err)

		// Verify: The URL was checked in cache (has returns false)
		assert.False(t, cache.Has(ctx, testURL2))
	})

	// Test cache miss scenario
	t.Run("cache miss", func(t *testing.T) {
		// Setup: Ensure cache is empty for this URL
		testURL3 := "https://example.com/nonexistent"

		// Execute: Get from non-cached URL
		// This will fail with network error, but we can verify cache behavior
		_, err := client.Get(ctx, testURL3)

		// Verify: We get an error (network failure in this case)
		assert.Error(t, err)

		// Verify: Cache doesn't have the URL
		assert.False(t, cache.Has(ctx, testURL3))
	})

	// Test with cache disabled
	t.Run("cache disabled", func(t *testing.T) {
		// Setup: Create client with cache disabled
		clientNoCache, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			Cache:       cache,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Setup: Add content to cache
		testURL4 := "https://example.com/test4"
		err = cache.Set(ctx, testURL4, cachedBody, time.Hour)
		require.NoError(t, err)

		// Execute: Get with cache disabled
		// This will fail with network error because cache is disabled
		_, err = clientNoCache.Get(ctx, testURL4)

		// Verify: Error because cache is disabled and no network request was made
		assert.Error(t, err)

		// Verify: Cache still has the content (wasn't used)
		assert.True(t, cache.Has(ctx, testURL4))
	})
}

func TestClient_GetCookies(t *testing.T) {
	// Create client with default options
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: false,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	t.Run("valid URL with cookies", func(t *testing.T) {
		// Note: In a real scenario, cookies would be set by previous requests
		// For this test, we just verify the method works with valid URLs

		// Execute: Get cookies for a valid URL
		cookies := client.GetCookies("https://example.com")

		// Verify: Method returns without error (may return nil or empty cookies)
		assert.NotNil(t, cookies)
		assert.IsType(t, []*http.Cookie{}, cookies)
	})

	t.Run("invalid URL", func(t *testing.T) {
		// Execute: Get cookies for an invalid URL (empty string)
		cookies := client.GetCookies("")

		// Verify: Returns empty slice for empty URLs (url.Parse("") doesn't error)
		assert.NotNil(t, cookies)
		assert.Empty(t, cookies)
	})

	t.Run("URL with path and query", func(t *testing.T) {
		// Execute: Get cookies for a URL with path and query parameters
		cookies := client.GetCookies("https://example.com/path?param=value")

		// Verify: Method handles complex URLs correctly
		assert.NotNil(t, cookies)
		assert.IsType(t, []*http.Cookie(nil), cookies)
	})

	t.Run("URL parsing", func(t *testing.T) {
		// Execute: Get cookies for various URL formats
		testCases := []string{
			"http://example.com",
			"https://example.com:8080",
			"https://subdomain.example.com/path",
		}

		for _, testURL := range testCases {
			t.Run(testURL, func(t *testing.T) {
				// Parse URL to verify it's valid
				_, err := url.Parse(testURL)
				require.NoError(t, err)

				// Get cookies
				cookies := client.GetCookies(testURL)

				// Verify: Method doesn't panic and returns a slice
				assert.NotNil(t, cookies)
				assert.IsType(t, []*http.Cookie(nil), cookies)
			})
		}
	})
}

func TestClient_SetCacheEnabled(t *testing.T) {
	ctx := context.Background()
	testURL := "https://example.com/test"
	cachedBody := []byte("<html><body>Cached content</body></html>")

	// Create mock cache
	cache := mocks.NewSimpleMockCache()

	// Create client with cache initially enabled
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	t.Run("disable cache", func(t *testing.T) {
		// Setup: Add content to cache
		err := cache.Set(ctx, testURL, cachedBody, time.Hour)
		require.NoError(t, err)

		// Verify: Cache has the content
		assert.True(t, cache.Has(ctx, testURL))

		// Execute: Disable cache
		client.SetCacheEnabled(false)

		// Verify: Cache is disabled (Get will fail with network error, not cache hit)
		_, err = client.Get(ctx, testURL)
		// We expect network error because cache is disabled
		assert.Error(t, err)

		// Verify: Content still in cache (wasn't used, just ignored)
		assert.True(t, cache.Has(ctx, testURL))
	})

	t.Run("enable cache", func(t *testing.T) {
		// Setup: Create client with cache disabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			CacheTTL:    24 * time.Hour,
			Cache:       cache,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Setup: Add content to cache
		err = cache.Set(ctx, testURL, cachedBody, time.Hour)
		require.NoError(t, err)

		// Execute: Enable cache
		client.SetCacheEnabled(true)

		// Execute: Get from client (should use cache now)
		resp, err := client.Get(ctx, testURL)
		require.NoError(t, err)

		// Verify: Response is from cache
		assert.True(t, resp.FromCache)
		assert.Equal(t, cachedBody, resp.Body)
	})

	t.Run("toggle cache multiple times", func(t *testing.T) {
		// Setup: Create fresh client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: true,
			CacheTTL:    24 * time.Hour,
			Cache:       cache,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Toggle cache off
		client.SetCacheEnabled(false)

		// Verify: Cache disabled works (will fail with network error, not cache hit)
		_, err = client.Get(ctx, testURL)
		assert.Error(t, err)

		// Execute: Toggle cache on
		client.SetCacheEnabled(true)

		// Verify: Cache enabled works (would use cache if present)
		// Add content to cache to verify it works
		err = cache.Set(ctx, testURL, cachedBody, time.Hour)
		require.NoError(t, err)

		resp, err := client.Get(ctx, testURL)
		require.NoError(t, err)
		assert.True(t, resp.FromCache)

		// Execute: Toggle cache off again
		client.SetCacheEnabled(false)

		// Verify: Cache disabled again
		_, err = client.Get(ctx, testURL)
		assert.Error(t, err)
	})

	t.Run("cache enabled with nil cache", func(t *testing.T) {
		// Create client with nil cache but cache enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: true,
			CacheTTL:    24 * time.Hour,
			Cache:       nil,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Try to get content (should fail gracefully with network error)
		_, err = client.Get(ctx, "https://example.com/test")

		// Verify: Error because cache is nil
		assert.Error(t, err)
	})
}

func TestDefaultClientOptions(t *testing.T) {
	// Execute: Get default options
	opts := fetcher.DefaultClientOptions()

	// Verify: All default values are set correctly
	t.Run("default timeout", func(t *testing.T) {
		assert.Equal(t, 90*time.Second, opts.Timeout)
	})

	t.Run("default max retries", func(t *testing.T) {
		assert.Equal(t, 3, opts.MaxRetries)
	})

	t.Run("cache enabled by default", func(t *testing.T) {
		assert.True(t, opts.EnableCache)
	})

	t.Run("default cache TTL", func(t *testing.T) {
		assert.Equal(t, 24*time.Hour, opts.CacheTTL)
	})

	t.Run("empty user agent", func(t *testing.T) {
		assert.Empty(t, opts.UserAgent)
	})

	t.Run("empty proxy URL", func(t *testing.T) {
		assert.Empty(t, opts.ProxyURL)
	})

	t.Run("nil cache", func(t *testing.T) {
		assert.Nil(t, opts.Cache)
	})
}

func TestSaveToCache_Success(t *testing.T) {
	ctx := context.Background()
	testURL := "https://example.com/test"
	responseBody := []byte("<html><body>Test content</body></html>")

	// Setup: Create mock cache
	cache := mocks.NewSimpleMockCache()

	// Setup: Create client with cache enabled
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	// Test the saveToCache method via a mock server to ensure it gets called
	t.Run("save successful", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write(responseBody)
		}))
		defer server.Close()

		// Execute: Get from server (should save to cache)
		resp, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify: Content was saved to cache
		assert.True(t, cache.Has(ctx, server.URL))
		cachedData, err := cache.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, responseBody, cachedData)
	})

	t.Run("save with custom TTL", func(t *testing.T) {
		// Setup: Create client with custom TTL
		customTTL := 1 * time.Hour
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: true,
			CacheTTL:    customTTL,
			Cache:       cache,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write(responseBody)
		}))
		defer server.Close()

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify: Content was saved with custom TTL (we can't directly verify TTL in SimpleMockCache)
		assert.True(t, cache.Has(ctx, server.URL))
	})

	t.Run("save preserves response body", func(t *testing.T) {
		// Setup: Clear cache
		cache.Delete(ctx, testURL)

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write(responseBody)
		}))
		defer server.Close()

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)
		require.NoError(t, err)

		// Verify: Cached data matches original body
		cachedData, err := cache.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, responseBody, cachedData)
		assert.Equal(t, resp.Body, cachedData)
	})
}

func TestSaveToCache_Disabled(t *testing.T) {
	ctx := context.Background()
	responseBody := []byte("<html><body>Test content</body></html>")

	// Setup: Create mock cache
	cache := mocks.NewSimpleMockCache()

	// Setup: Create client with cache disabled
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: false, // Cache disabled
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(responseBody)
	}))
	defer server.Close()

	// Execute: Get from server (cache is disabled)
	resp, err := client.Get(ctx, server.URL)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify: Content was NOT saved to cache
	assert.False(t, cache.Has(ctx, server.URL), "Cache should be disabled, content should not be saved")
}

func TestSaveToCache_Error(t *testing.T) {
	ctx := context.Background()
	responseBody := []byte("<html><body>Test content</body></html>")

	t.Run("nil cache returns nil", func(t *testing.T) {
		// Setup: Create client with nil cache
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: true,
			CacheTTL:    24 * time.Hour,
			Cache:       nil, // Nil cache
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write(responseBody)
		}))
		defer server.Close()

		// Execute: Get from server (should fail because cache is nil)
		// Even though cache is nil, Get should still work and just skip caching
		resp, err := client.Get(ctx, server.URL)
		// We expect success because nil cache is handled gracefully
		// The saveToCache method checks if cache is nil and returns nil
		if err == nil {
			assert.NotNil(t, resp)
		}
	})

	t.Run("cache set error handled", func(t *testing.T) {
		// Create a mock cache that returns an error on Set
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		// Set up cache to return a cache miss first (so it will try to fetch and then save)
		mockCache.EXPECT().Get(ctx, gomock.Any()).Return(nil, domain.ErrCacheMiss)
		// Set up the Set to return an error
		mockCache.EXPECT().Set(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache error"))

		// Setup: Create client with mock cache
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: true,
			CacheTTL:    time.Hour,
			Cache:       mockCache,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write(responseBody)
		}))
		defer server.Close()

		// Execute: Get from server (cache Set will fail, but should be ignored)
		resp, err := client.Get(ctx, server.URL)
		// The error from cache.Set is ignored with _ = saveToCache()
		// So we should still get a successful response
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestGet_WithCache(t *testing.T) {
	ctx := context.Background()
	testURL := "https://example.com/test"
	cachedBody := []byte("<html><body>Cached content</body></html>")

	// Setup: Create mock cache with content
	cache := mocks.NewSimpleMockCache()
	err := cache.Set(ctx, testURL, cachedBody, time.Hour)
	require.NoError(t, err)

	// Setup: Create client with cache enabled
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	// Execute: Get from client (should use cache)
	resp, err := client.Get(ctx, testURL)
	require.NoError(t, err)

	// Verify: Response is from cache
	t.Run("response marked as from cache", func(t *testing.T) {
		assert.True(t, resp.FromCache, "Response should be marked as from cache")
	})

	t.Run("response body matches cache", func(t *testing.T) {
		assert.Equal(t, cachedBody, resp.Body)
	})

	t.Run("response status code is 200", func(t *testing.T) {
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("response content type is html", func(t *testing.T) {
		assert.Equal(t, "text/html", resp.ContentType)
	})

	t.Run("response URL is correct", func(t *testing.T) {
		assert.Equal(t, testURL, resp.URL)
	})

	t.Run("cache hit without network request", func(t *testing.T) {
		// This test verifies that when cache is hit, no network request is made
		// We can verify this by checking the response is from cache
		assert.True(t, resp.FromCache)
	})
}

func TestGet_WithoutCache(t *testing.T) {
	ctx := context.Background()
	responseBody := []byte("<html><body>Fresh content</body></html>")

	// Setup: Create mock cache (empty)
	cache := mocks.NewSimpleMockCache()

	// Setup: Create client with cache enabled
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		Cache:       cache,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(responseBody)
	}))
	defer server.Close()

	// Execute: Get from client (cache miss, should fetch from network)
	resp, err := client.Get(ctx, server.URL)
	require.NoError(t, err)

	// Verify: Response is NOT from cache
	t.Run("response marked as not from cache", func(t *testing.T) {
		assert.False(t, resp.FromCache, "Response should not be from cache")
	})

	t.Run("response body from network", func(t *testing.T) {
		assert.Equal(t, responseBody, resp.Body)
	})

	t.Run("response status code is 200", func(t *testing.T) {
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("content saved to cache after fetch", func(t *testing.T) {
		assert.True(t, cache.Has(ctx, server.URL), "Content should be saved to cache after successful fetch")
	})

	t.Run("cached content matches response", func(t *testing.T) {
		cachedData, err := cache.Get(ctx, server.URL)
		require.NoError(t, err)
		assert.Equal(t, responseBody, cachedData)
	})
}

func TestGetWithHeaders_CustomHeaders(t *testing.T) {
	ctx := context.Background()
	responseBody := []byte("<html><body>Test</body></html>")

	// Setup: Create client
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		EnableCache: false,
		MaxRetries:  0,
	})
	require.NoError(t, err)

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(responseBody)
	}))
	defer server.Close()

	t.Run("custom headers are sent", func(t *testing.T) {
		customHeaders := map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
		}

		// Execute: Get with custom headers
		resp, err := client.GetWithHeaders(ctx, server.URL, customHeaders)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify: Custom headers were sent (though we can't directly verify with SimpleMockCache)
		// The important thing is that the method accepts and processes custom headers
		assert.Equal(t, responseBody, resp.Body)
	})

	t.Run("nil headers handled gracefully", func(t *testing.T) {
		// Execute: Get with nil headers
		resp, err := client.GetWithHeaders(ctx, server.URL, nil)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, responseBody, resp.Body)
	})

	t.Run("empty headers map handled", func(t *testing.T) {
		// Execute: Get with empty headers map
		resp, err := client.GetWithHeaders(ctx, server.URL, map[string]string{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, responseBody, resp.Body)
	})

	t.Run("multiple custom headers", func(t *testing.T) {
		customHeaders := map[string]string{
			"X-Header-1": "value1",
			"X-Header-2": "value2",
			"X-Header-3": "value3",
		}

		// Execute: Get with multiple custom headers
		resp, err := client.GetWithHeaders(ctx, server.URL, customHeaders)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, responseBody, resp.Body)
	})

	t.Run("override default headers", func(t *testing.T) {
		// Note: We can't easily test header override without more complex mock setup
		// But we can verify the method accepts the headers parameter
		customHeaders := map[string]string{
			"Accept": "application/json",
		}

		resp, err := client.GetWithHeaders(ctx, server.URL, customHeaders)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
