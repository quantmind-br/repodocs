package fetcher_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testResponseBody = "<html><body>Test content</body></html>"

func TestNewStealthTransport(t *testing.T) {
	t.Run("creates transport with client", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Create stealth transport
		transport := fetcher.NewStealthTransport(client)

		// Verify: Transport created successfully
		assert.NotNil(t, transport)
	})

	t.Run("creates transport with nil client", func(t *testing.T) {
		// Execute: Create stealth transport with nil client
		transport := fetcher.NewStealthTransport(nil)

		// Verify: Transport created (but will fail when used)
		assert.NotNil(t, transport)
	})
}

func TestStealthTransport_RoundTrip_Success(t *testing.T) {
	ctx := context.Background()
	testResponseBody := "<html><body>Test content</body></html>"

	t.Run("successful GET request", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(testResponseBody))
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "OK", resp.Status)

		// Verify: Body can be read
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, testResponseBody, string(body))
	})

	t.Run("request with custom headers", func(t *testing.T) {
		// Setup: Create test server that verifies headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify custom headers were sent
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
			assert.NotEmpty(t, r.Header.Get("User-Agent"))

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(testResponseBody))
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request with custom headers
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("X-Custom-Header", "custom-value")

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("response headers are preserved", func(t *testing.T) {
		// Setup: Create test server with custom headers
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Custom-Response-Header", "response-value")
			w.WriteHeader(200)
			w.Write([]byte(`{"key":"value"}`))
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Response headers are preserved
		require.NoError(t, err)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.Equal(t, "response-value", resp.Header.Get("X-Custom-Response-Header"))
	})
}

func TestStealthTransport_RoundTrip_Error(t *testing.T) {
	ctx := context.Background()

	t.Run("HTTP 404 error", func(t *testing.T) {
		// Setup: Create test server that returns 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Error returned (404 is not retryable)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("HTTP 500 error", func(t *testing.T) {
		// Setup: Create test server that returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("HTTP 429 retryable error", func(t *testing.T) {
		// Setup: Create test server that returns 429
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(429)
			w.Write([]byte("Too Many Requests"))
		}))
		defer server.Close()

		// Setup: Create client with retries
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  1,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Error returned (after retries)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("network connection error", func(t *testing.T) {
		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request to invalid URL
		req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:9999/nonexistent", nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("context cancellation", func(t *testing.T) {
		t.Skip("tls-client library does not support context cancellation at the request level")

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Error returned
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestStealthTransport_ContentEncoding(t *testing.T) {
	ctx := context.Background()
	testResponseBody := "<html><body>Test content</body></html>"

	t.Run("content-encoding header is removed", func(t *testing.T) {
		// Setup: Create test server with gzipped content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "gzip")

			// Gzip the response body
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write([]byte(testResponseBody))
			gz.Close()

			w.WriteHeader(200)
			w.Write(buf.Bytes())
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Content-Encoding header was removed
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify: Content-Encoding is removed (because body is already decompressed)
		encoding := resp.Header.Get("Content-Encoding")
		assert.Empty(t, encoding, "Content-Encoding should be removed to prevent double decompression")

		// Verify: Body can still be read
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, testResponseBody, string(body))
	})

	t.Run("multiple content-encoding values", func(t *testing.T) {
		// Setup: Create test server with gzipped content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Header().Set("Content-Encoding", "gzip, deflate")

			// Gzip the response body
			var buf bytes.Buffer
			gz := gzip.NewWriter(&buf)
			gz.Write([]byte(testResponseBody))
			gz.Close()

			w.WriteHeader(200)
			w.Write(buf.Bytes())
		}))
		defer server.Close()

		// Setup: Create client and transport
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Perform round trip
		resp, err := transport.RoundTrip(req)

		// Verify: Content-Encoding header was removed
		require.NoError(t, err)
		assert.NotNil(t, resp)

		encoding := resp.Header.Get("Content-Encoding")
		assert.Empty(t, encoding, "Content-Encoding should be removed")

		// Verify: Body can still be read
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.NotEmpty(t, body)
	})
}

func TestClient_Transport(t *testing.T) {
	t.Run("returns stealth transport", func(t *testing.T) {
		// Setup: Create client
		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		// Execute: Get transport from client
		transport := client.Transport()

		// Verify: Transport is returned
		assert.NotNil(t, transport)

		// Verify: Transport implements http.RoundTripper interface
		var _ http.RoundTripper = transport
	})

	t.Run("transport can be used with http.Client", func(t *testing.T) {
		ctx := context.Background()

		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(200)
			w.Write([]byte(testResponseBody))
		}))
		defer server.Close()

		// Setup: Create fetcher client and get transport
		fetcherClient, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcherClient.Transport()

		// Setup: Create standard http.Client with stealth transport
		httpClient := &http.Client{
			Transport: transport,
			Timeout:   10 * time.Second,
		}

		// Setup: Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		// Execute: Make request using http.Client with stealth transport
		resp, err := httpClient.Do(req)

		// Verify: Response is correct
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)

		// Verify: Body can be read
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, testResponseBody, string(body))
		resp.Body.Close()
	})
}

func TestStealthTransport_CloudflareFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("403 with renderer fallback succeeds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			w.Write([]byte("Forbidden"))
		}))
		defer server.Close()

		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		renderedHTML := "<html><body>Rendered by browser</body></html>"
		fallbackCalled := false
		transport := fetcher.NewStealthTransportWithOptions(client, fetcher.StealthTransportOptions{
			RendererFallback: func(ctx context.Context, url string) (string, error) {
				fallbackCalled = true
				return renderedHTML, nil
			},
		})

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)

		require.NoError(t, err)
		assert.True(t, fallbackCalled)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, renderedHTML, string(body))
	})

	t.Run("403 without renderer fallback returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			w.Write([]byte("Forbidden"))
		}))
		defer server.Close()

		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransport(client)

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("403 with failing renderer fallback returns original error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			w.Write([]byte("Forbidden"))
		}))
		defer server.Close()

		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		transport := fetcher.NewStealthTransportWithOptions(client, fetcher.StealthTransportOptions{
			RendererFallback: func(ctx context.Context, url string) (string, error) {
				return "", errors.New("browser crashed")
			},
		})

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "403")
	})

	t.Run("non-403 error skips fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		fallbackCalled := false
		transport := fetcher.NewStealthTransportWithOptions(client, fetcher.StealthTransportOptions{
			RendererFallback: func(ctx context.Context, url string) (string, error) {
				fallbackCalled = true
				return "<html>rendered</html>", nil
			},
		})

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.False(t, fallbackCalled)
	})

	t.Run("404 error skips fallback", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
			w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		client, err := fetcher.NewClient(fetcher.ClientOptions{
			EnableCache: false,
			MaxRetries:  0,
		})
		require.NoError(t, err)

		fallbackCalled := false
		transport := fetcher.NewStealthTransportWithOptions(client, fetcher.StealthTransportOptions{
			RendererFallback: func(ctx context.Context, url string) (string, error) {
				fallbackCalled = true
				return "<html>rendered</html>", nil
			},
		})

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.False(t, fallbackCalled)
	})
}
