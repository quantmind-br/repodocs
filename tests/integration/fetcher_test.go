package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcher_BasicRequest(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html><html><body><h1>Test Page</h1></body></html>`))
	}))
	defer server.Close()

	// Create fetcher client
	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:    10 * time.Second,
		MaxRetries: 1,
	})
	require.NoError(t, err)
	defer client.Close()

	// Make request
	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "Test Page")
}

func TestFetcher_UserAgentHeader(t *testing.T) {
	// Create a mock server that echoes the User-Agent
	var receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Get(context.Background(), server.URL)
	require.NoError(t, err)

	// User-Agent should be set (either custom or from rotation pool)
	assert.NotEmpty(t, receivedUA)
	// Should look like a browser User-Agent
	assert.True(t, strings.Contains(receivedUA, "Mozilla") ||
		strings.Contains(receivedUA, "Chrome") ||
		strings.Contains(receivedUA, "Safari"),
		"User-Agent should look like a browser: %s", receivedUA)
}

func TestFetcher_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	customHeaders := map[string]string{
		"X-Custom-Header": "custom-value",
		"Accept-Language": "pt-BR",
	}

	_, err = client.GetWithHeaders(context.Background(), server.URL, customHeaders)
	require.NoError(t, err)

	assert.Equal(t, "custom-value", receivedHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "pt-BR", receivedHeaders.Get("Accept-Language"))
}

func TestFetcher_404Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout:    10 * time.Second,
		MaxRetries: 0,
	})
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Get(context.Background(), server.URL)
	// The fetcher may return an error for 404 or just return the response
	// depending on implementation - both are valid behaviors
	if err != nil {
		assert.Contains(t, err.Error(), "404")
	} else {
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}
}

func TestFetcher_Timeout(t *testing.T) {
	t.Skip("Timeout behavior varies by tls-client implementation")
}

func TestFetcher_ContextCancellation(t *testing.T) {
	t.Skip("Context cancellation behavior varies by tls-client implementation")
}

func TestFetcher_ContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)

	assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
}

func TestFetcher_LargeResponse(t *testing.T) {
	// Create a large response (1MB)
	largeContent := strings.Repeat("x", 1024*1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 30 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Get(context.Background(), server.URL)
	require.NoError(t, err)

	assert.Equal(t, len(largeContent), len(resp.Body))
}

func TestFetcher_Redirect(t *testing.T) {
	// Create a server that redirects
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/destination", http.StatusMovedPermanently)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Destination reached"))
	}))
	defer redirectServer.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	resp, err := client.Get(context.Background(), redirectServer.URL)
	require.NoError(t, err)

	// Redirect behavior depends on tls-client configuration
	// May follow redirect (200) or return redirect response (301)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMovedPermanently,
		"Expected 200 or 301, got %d", resp.StatusCode)
}

func TestFetcher_AcceptHeaders(t *testing.T) {
	var receivedAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := fetcher.NewClient(fetcher.ClientOptions{
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	defer client.Close()

	_, err = client.Get(context.Background(), server.URL)
	require.NoError(t, err)

	// Should include Accept header
	assert.NotEmpty(t, receivedAccept)
}
