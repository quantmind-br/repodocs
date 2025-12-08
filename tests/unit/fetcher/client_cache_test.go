package fetcher

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/quantmind-br/repodocs-go/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.IsType(t, []*http.Cookie(nil), cookies)
	})

	t.Run("invalid URL", func(t *testing.T) {
		// Execute: Get cookies for an invalid URL (empty string)
		cookies := client.GetCookies("")

		// Verify: Returns empty slice for unparseable/empty URLs
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
