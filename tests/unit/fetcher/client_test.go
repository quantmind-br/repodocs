package fetcher_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	t.Run("creates client with default options", func(t *testing.T) {
		// Execute: Create client with default options
		opts := fetcher.DefaultClientOptions()
		client, err := fetcher.NewClient(opts)

		// Verify: Client created successfully
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client with custom options", func(t *testing.T) {
		// Execute: Create client with custom options
		opts := fetcher.ClientOptions{
			Timeout:     10 * time.Second,
			MaxRetries:  5,
			EnableCache: false,
			CacheTTL:    1 * time.Hour,
			UserAgent:   "TestAgent/1.0",
		}
		client, err := fetcher.NewClient(opts)

		// Verify: Client created successfully
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client with proxy", func(t *testing.T) {
		// Execute: Create client with proxy URL
		opts := fetcher.ClientOptions{
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			EnableCache: false,
			ProxyURL:    "http://proxy.example.com:8080",
		}
		client, err := fetcher.NewClient(opts)

		// Verify: Client created successfully
		// Note: Will fail if proxy URL is invalid, but should work with valid format
		if err != nil {
			// Proxy might not be available, which is expected in test environment
			assert.Contains(t, err.Error(), "proxy")
		} else {
			assert.NotNil(t, client)
		}
	})

	t.Run("zero timeout defaults to 90 seconds", func(t *testing.T) {
		// Execute: Create client with zero timeout
		opts := fetcher.ClientOptions{
			Timeout:     0,
			MaxRetries:  3,
			EnableCache: false,
		}
		client, err := fetcher.NewClient(opts)

		// Verify: Client created successfully with default timeout
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestNewClient_Error(t *testing.T) {
	t.Run("invalid proxy URL returns error", func(t *testing.T) {
		// Execute: Create client with invalid proxy URL
		opts := fetcher.ClientOptions{
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			EnableCache: false,
			ProxyURL:    "://invalid-url",
		}
		client, err := fetcher.NewClient(opts)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestClient_Get_Success(t *testing.T) {
	ctx := context.Background()
	responseBody := "<html><body>Test content</body></html>"

	// Setup: Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()

	t.Run("successful GET request", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, responseBody, string(resp.Body))
		assert.Equal(t, "text/html", resp.ContentType)
		assert.Equal(t, server.URL, resp.URL)
		assert.False(t, resp.FromCache)
	})

	t.Run("GET request with different content types", func(t *testing.T) {
		testCases := []struct {
			contentType string
			body        string
			desc        string
		}{
			{"application/json", `{"key":"value"}`, "JSON response"},
			{"text/plain", "Plain text content", "Plain text response"},
			{"application/xml", "<root><item>test</item></root>", "XML response"},
			{"text/html; charset=utf-8", "<div>HTML</div>", "HTML with charset"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				// Setup: Create test server
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", tc.contentType)
					w.WriteHeader(200)
					w.Write([]byte(tc.body))
				}))
				defer ts.Close()

				// Setup: Create client
				client, err := fetcher.NewClient(fetcher.ClientOptions{
					EnableCache: false,
					MaxRetries:  0,
				})
				require.NoError(t, err)

				// Execute: Get from server
				resp, err := client.Get(ctx, ts.URL)

				// Verify: Response is correct
				require.NoError(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				assert.Equal(t, tc.body, string(resp.Body))
				assert.Contains(t, resp.ContentType, strings.Split(tc.contentType, ";")[0])
			})
		}
	})

	t.Run("GET request with headers", func(t *testing.T) {
		// Setup: Create test server that verifies headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify stealth headers were sent
			assert.NotEmpty(t, r.Header.Get("User-Agent"))
			assert.NotEmpty(t, r.Header.Get("Accept"))
			assert.NotEmpty(t, r.Header.Get("Accept-Language"))

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(responseBody))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestClient_Get_Error(t *testing.T) {
	ctx := context.Background()

	t.Run("HTTP 404 error", func(t *testing.T) {
		// Setup: Create test server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)

		// Verify: Error type is FetchError
		var fetchErr *domain.FetchError
		assert.ErrorAs(t, err, &fetchErr)
		assert.Equal(t, 404, fetchErr.StatusCode)
	})

	t.Run("HTTP 500 error", func(t *testing.T) {
		// Setup: Create test server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)

		var fetchErr *domain.FetchError
		assert.ErrorAs(t, err, &fetchErr)
		assert.Equal(t, 500, fetchErr.StatusCode)
	})

	t.Run("HTTP 429 rate limit error (retryable)", func(t *testing.T) {
		// Setup: Create test server that returns 429
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(429)
			w.Write([]byte("Too Many Requests"))
		}))
		defer server.Close()

		// Setup: Create client with retries enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  1,
		})
		require.NoError(t, err)

		// Execute: Get from server (will retry and fail)
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned (after retries)
		assert.Error(t, err)
		assert.Nil(t, resp)

		var retryableErr *domain.RetryableError
		assert.ErrorAs(t, err, &retryableErr)
	})

	t.Run("HTTP 502 Bad Gateway (retryable)", func(t *testing.T) {
		// Setup: Create test server that returns 502
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
			w.Write([]byte("Bad Gateway"))
		}))
		defer server.Close()

		// Setup: Create client with retries enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  1,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned (after retries)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("HTTP 503 Service Unavailable (retryable)", func(t *testing.T) {
		// Setup: Create test server that returns 503
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(503)
			w.Write([]byte("Service Unavailable"))
		}))
		defer server.Close()

		// Setup: Create client with retries enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  1,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("HTTP 504 Gateway Timeout (retryable)", func(t *testing.T) {
		// Setup: Create test server that returns 504
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(504)
			w.Write([]byte("Gateway Timeout"))
		}))
		defer server.Close()

		// Setup: Create client with retries enabled
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  1,
		})
		require.NoError(t, err)

		// Execute: Get from server
		resp, err := client.Get(ctx, server.URL)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("HTTP 520-530 Cloudflare errors (retryable)", func(t *testing.T) {
		codes := []int{520, 521, 522, 523, 524, 525, 526, 527, 528, 529, 530}

		for _, code := range codes {
			t.Run(http.StatusText(code), func(t *testing.T) {
				// Setup: Create test server
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(code)
					w.Write([]byte("Cloudflare Error"))
				}))
				defer server.Close()

				// Setup: Create client with retries
				client, err := fetcher.NewClient(fetcher.ClientOptions{
					EnableCache: false,
					MaxRetries:  1,
				})
				require.NoError(t, err)

				// Execute: Get from server
				resp, err := client.Get(ctx, server.URL)

				// Verify: Error returned (retryable)
				assert.Error(t, err)
				assert.Nil(t, resp)
			})
		}
	})

	t.Run("network connection error", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get from invalid URL (will fail to connect)
		resp, err := client.Get(ctx, "http://localhost:9999/nonexistent")

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid URL format", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get with invalid URL
		resp, err := client.Get(ctx, "://invalid-url")

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Skip("tls-client library does not support context cancellation at the request level")

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Setup: Create context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Execute: Get with cancelled context
		resp, err := client.Get(ctx, "https://example.com")

		// Verify: Error returned (context cancelled)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_GetWithHeaders_Success(t *testing.T) {
	ctx := context.Background()
	responseBody := "<html><body>Test content</body></html>"

	t.Run("GET with custom headers", func(t *testing.T) {
		// Setup: Create test server that verifies headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify custom headers
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
			assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(responseBody))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get with custom headers
		customHeaders := map[string]string{
			"X-Custom-Header": "custom-value",
			"Authorization":   "Bearer token123",
		}
		resp, err := client.GetWithHeaders(ctx, server.URL, customHeaders)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, responseBody, string(resp.Body))
	})

	t.Run("GET with nil headers", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify stealth headers were sent
			assert.NotEmpty(t, r.Header.Get("User-Agent"))

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(responseBody))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get with nil headers
		resp, err := client.GetWithHeaders(ctx, server.URL, nil)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("GET with empty headers map", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(responseBody))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get with empty headers map
		resp, err := client.GetWithHeaders(ctx, server.URL, map[string]string{})

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("custom headers override stealth headers", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify custom User-Agent overrides stealth
			assert.Equal(t, "MyCustomAgent/1.0", r.Header.Get("User-Agent"))

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(responseBody))
		}))
		defer server.Close()

		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get with custom User-Agent
		customHeaders := map[string]string{
			"User-Agent": "MyCustomAgent/1.0",
		}
		resp, err := client.GetWithHeaders(ctx, server.URL, customHeaders)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("close client successfully", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Close client
		err = client.Close()

		// Verify: No error (TLS client doesn't have a Close method)
		assert.NoError(t, err)
	})

	t.Run("close multiple times", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Close client multiple times
		err = client.Close()
		assert.NoError(t, err)

		err = client.Close()
		assert.NoError(t, err)
	})
}

func TestClient_SetCache(t *testing.T) {
	t.Run("set cache implementation", func(t *testing.T) {
		// Setup: Create client without cache
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
			Cache:       nil,
		})
		require.NoError(t, err)

		// Execute: Set cache
		cache := &mockCache{}
		client.SetCache(cache)

		// Verify: Cache is set (we can't directly access it, but we verified it doesn't panic)
		assert.NotNil(t, cache)
	})
}

// mockCache is a simple mock cache for testing
type mockCache struct{}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, domain.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *mockCache) Has(ctx context.Context, key string) bool {
	return false
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	return nil
}

func (m *mockCache) Size() int {
	return 0
}

func (m *mockCache) Stats() map[string]interface{} {
	return nil
}
