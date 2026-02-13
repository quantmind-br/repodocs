package fetcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultClientOptions tests default client options
func TestDefaultClientOptions(t *testing.T) {
	opts := DefaultClientOptions()

	assert.Equal(t, 90*time.Second, opts.Timeout)
	assert.Equal(t, 3, opts.MaxRetries)
	assert.True(t, opts.EnableCache)
	assert.Equal(t, 24*time.Hour, opts.CacheTTL)
	assert.Empty(t, opts.UserAgent)
	assert.Empty(t, opts.ProxyURL)
}

// TestNewClient tests creating a new client
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    ClientOptions
		check   func(t *testing.T, c *Client)
		wantErr bool
	}{
		{
			name: "with default options",
			opts: DefaultClientOptions(),
			check: func(t *testing.T, c *Client) {
				assert.NotNil(t, c.tlsClient)
				assert.NotNil(t, c.retrier)
			},
			wantErr: false,
		},
		{
			name: "with zero timeout defaults to 90s",
			opts: ClientOptions{
				Timeout: 0,
			},
			check: func(t *testing.T, c *Client) {
				assert.NotNil(t, c)
			},
			wantErr: false,
		},
		{
			name: "with custom user agent",
			opts: ClientOptions{
				UserAgent: "TestAgent/1.0",
			},
			check: func(t *testing.T, c *Client) {
				assert.Equal(t, "TestAgent/1.0", c.userAgent)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, client)
				}
				client.Close()
			}
		})
	}
}

// TestClient_Get tests fetching content
func TestClient_Get(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()

		client, err := NewClient(ClientOptions{EnableCache: false})
		require.NoError(t, err)
		defer client.Close()

		ctx := context.Background()
		resp, err := client.Get(ctx, server.URL)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, []byte("test content"), resp.Body)
		assert.False(t, resp.FromCache)
	})

	t.Run("not found error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client, err := NewClient(ClientOptions{EnableCache: false})
		require.NoError(t, err)
		defer client.Close()

		ctx := context.Background()
		resp, err := client.Get(ctx, server.URL)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("cached response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		}))
		defer server.Close()

		// Create a mock cache
		cache := &mockCache{
			data: []byte("cached content"),
		}

		client, err := NewClient(ClientOptions{
			EnableCache: true,
			Cache:       cache,
		})
		require.NoError(t, err)
		defer client.Close()

		ctx := context.Background()
		resp, err := client.Get(ctx, server.URL)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, []byte("cached content"), resp.Body)
		assert.True(t, resp.FromCache)
	})
}

// TestClient_GetWithHeaders tests fetching with custom headers
func TestClient_GetWithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check custom header
		if r.Header.Get("X-Custom") == "test-value" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("custom header received"))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientOptions{EnableCache: false})
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	headers := map[string]string{"X-Custom": "test-value"}
	resp, err := client.GetWithHeaders(ctx, server.URL, headers)
	assert.NoError(t, err)
	assert.Equal(t, []byte("custom header received"), resp.Body)
}

// TestClient_GetCookies tests getting cookies
func TestClient_GetCookies(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)
	defer client.Close()

	// Get cookies for a URL - may be empty
	cookies := client.GetCookies("https://example.com")
	assert.NotNil(t, cookies)
}

// TestClient_Close tests closing client
func TestClient_Close(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestClient_SetCache tests setting cache
func TestClient_SetCache(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)
	defer client.Close()

	cache := &mockCache{}
	client.SetCache(cache)
	assert.Equal(t, cache, client.cache)
}

// TestClient_SetCacheEnabled tests enabling/disabling cache
func TestClient_SetCacheEnabled(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)
	defer client.Close()

	client.SetCacheEnabled(false)
	assert.False(t, client.cacheEnabled)

	client.SetCacheEnabled(true)
	assert.True(t, client.cacheEnabled)
}

// TestDefaultRetrierOptions tests default retrier options
func TestDefaultRetrierOptions(t *testing.T) {
	opts := DefaultRetrierOptions()

	assert.Equal(t, 3, opts.MaxRetries)
	assert.Equal(t, 1*time.Second, opts.InitialInterval)
	assert.Equal(t, 30*time.Second, opts.MaxInterval)
	assert.Equal(t, 2.0, opts.Multiplier)
}

// TestNewRetrier tests creating a retrier
func TestNewRetrier(t *testing.T) {
	tests := []struct {
		name  string
		opts  RetrierOptions
		check func(t *testing.T, r *Retrier)
	}{
		{
			name: "with valid options",
			opts: RetrierOptions{
				MaxRetries:      5,
				InitialInterval: 2 * time.Second,
				MaxInterval:     60 * time.Second,
				Multiplier:      3.0,
			},
			check: func(t *testing.T, r *Retrier) {
				assert.Equal(t, 5, r.maxRetries)
				assert.Equal(t, 2*time.Second, r.initialInterval)
				assert.Equal(t, 60*time.Second, r.maxInterval)
				assert.Equal(t, 3.0, r.multiplier)
			},
		},
		{
			name: "zero max retries defaults to 3",
			opts: RetrierOptions{
				MaxRetries: 0,
			},
			check: func(t *testing.T, r *Retrier) {
				assert.Equal(t, 3, r.maxRetries)
			},
		},
		{
			name: "zero initial interval defaults to 1s",
			opts: RetrierOptions{
				InitialInterval: 0,
			},
			check: func(t *testing.T, r *Retrier) {
				assert.Equal(t, 1*time.Second, r.initialInterval)
			},
		},
		{
			name: "zero max interval defaults to 30s",
			opts: RetrierOptions{
				MaxInterval: 0,
			},
			check: func(t *testing.T, r *Retrier) {
				assert.Equal(t, 30*time.Second, r.maxInterval)
			},
		},
		{
			name: "zero multiplier defaults to 2",
			opts: RetrierOptions{
				Multiplier: 0,
			},
			check: func(t *testing.T, r *Retrier) {
				assert.Equal(t, 2.0, r.multiplier)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRetrier(tt.opts)
			if tt.check != nil {
				tt.check(t, r)
			}
		})
	}
}

// TestRetrier_Retry tests retry logic
func TestRetrier_Retry(t *testing.T) {
	t.Run("succeeds on first attempt", func(t *testing.T) {
		r := NewRetrier(DefaultRetrierOptions())
		ctx := context.Background()

		attempts := 0
		err := r.Retry(ctx, func() error {
			attempts++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries on retryable error", func(t *testing.T) {
		r := NewRetrier(RetrierOptions{
			MaxRetries:      3,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
		})
		ctx := context.Background()

		attempts := 0
		err := r.Retry(ctx, func() error {
			attempts++
			if attempts < 2 {
				return &domain.RetryableError{
					Err: &domain.FetchError{
						StatusCode: 503,
						Err:        http.ErrHandlerTimeout,
					},
				}
			}
			return nil
		})

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, attempts, 2)
	})

	t.Run("fails after max retries", func(t *testing.T) {
		r := NewRetrier(RetrierOptions{
			MaxRetries:      2,
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			Multiplier:      2.0,
		})
		ctx := context.Background()

		err := r.Retry(ctx, func() error {
			return &domain.RetryableError{
				Err: &domain.FetchError{
					StatusCode: 503,
					Err:        http.ErrHandlerTimeout,
				},
			}
		})

		assert.Error(t, err)
	})
}

// TestRetryWithValue tests retry with value return
func TestRetryWithValue(t *testing.T) {
	r := NewRetrier(DefaultRetrierOptions())
	ctx := context.Background()

	attempts := 0
	result, err := RetryWithValue(ctx, r, func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", &domain.RetryableError{
				Err: &domain.FetchError{
					StatusCode: 503,
					Err:        http.ErrHandlerTimeout,
				},
			}
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.GreaterOrEqual(t, attempts, 2)
}

// TestShouldRetryStatus tests status code retry logic
func TestShouldRetryStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"429 Too Many Requests", 429, true},
		{"502 Bad Gateway", 502, true},
		{"503 Service Unavailable", 503, true},
		{"504 Gateway Timeout", 504, true},
		{"520 Cloudflare", 520, true},
		{"525 Cloudflare", 525, true},
		{"530 Cloudflare", 530, true},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"500 Internal Server Error", 500, false},
		{"200 OK", 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetryStatus(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseRetryAfter tests parsing retry-after header
func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{"seconds value", "120", 120 * time.Second},
		{"empty string", "", 0},
		{"zero value", "0", 0},
		{"large value", "3600", 3600 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRetryAfter(tt.header)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRandomUserAgent tests random user agent generation
func TestRandomUserAgent(t *testing.T) {
	ua := RandomUserAgent()
	assert.NotEmpty(t, ua)
	assert.Contains(t, ua, "Mozilla")
}

// TestRandomAcceptLanguage tests random accept-language generation
func TestRandomAcceptLanguage(t *testing.T) {
	lang := RandomAcceptLanguage()
	assert.NotEmpty(t, lang)
	assert.Contains(t, lang, "en")
}

// TestRandomSecChUaPlatform tests random platform generation
func TestRandomSecChUaPlatform(t *testing.T) {
	platform := RandomSecChUaPlatform()
	assert.NotEmpty(t, platform)
}

// TestStealthHeaders tests stealth header generation
func TestStealthHeaders(t *testing.T) {
	t.Run("with custom user agent", func(t *testing.T) {
		headers := StealthHeaders("TestAgent/1.0")
		assert.Equal(t, "TestAgent/1.0", headers["User-Agent"])
		assert.NotEmpty(t, headers["Accept"])
		assert.NotEmpty(t, headers["Accept-Language"])
	})

	t.Run("with empty user agent generates random", func(t *testing.T) {
		headers := StealthHeaders("")
		assert.NotEmpty(t, headers["User-Agent"])
	})

	t.Run("Chrome headers added for Chrome UA", func(t *testing.T) {
		headers := StealthHeaders("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		assert.NotEmpty(t, headers["Sec-CH-UA"])
		assert.NotEmpty(t, headers["Sec-CH-UA-Mobile"])
		assert.NotEmpty(t, headers["Sec-CH-UA-Platform"])
	})

	t.Run("no Chrome headers for Firefox UA", func(t *testing.T) {
		headers := StealthHeaders("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0")
		_, hasCHUA := headers["Sec-CH-UA"]
		_, hasMobile := headers["Sec-CH-UA-Mobile"]
		_, hasPlatform := headers["Sec-CH-UA-Platform"]
		assert.False(t, hasCHUA || hasMobile || hasPlatform)
	})
}

// TestRandomDelay tests random delay generation
func TestRandomDelay(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 500 * time.Millisecond

		delay := RandomDelay(min, max)
		assert.GreaterOrEqual(t, delay, min)
		assert.LessOrEqual(t, delay, max)
	})

	t.Run("min equals max returns min", func(t *testing.T) {
		delay := RandomDelay(100*time.Millisecond, 100*time.Millisecond)
		assert.Equal(t, 100*time.Millisecond, delay)
	})

	t.Run("min greater than max returns min", func(t *testing.T) {
		delay := RandomDelay(500*time.Millisecond, 100*time.Millisecond)
		assert.Equal(t, 500*time.Millisecond, delay)
	})
}

// TestNewStealthTransport tests creating transport
func TestNewStealthTransport(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)
	defer client.Close()

	transport := NewStealthTransport(client)
	assert.NotNil(t, transport)
	assert.Equal(t, client, transport.client)
}

// TestClient_Transport tests getting transport from client
func TestClient_Transport(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	require.NoError(t, err)
	defer client.Close()

	transport := client.Transport()
	assert.NotNil(t, transport)
	assert.IsType(t, &StealthTransport{}, transport)
}

// Mock implementations for testing

type mockCache struct {
	data []byte
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.data != nil {
		return m.data, nil
	}
	return nil, domain.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data = value
	return nil
}

func (m *mockCache) Has(ctx context.Context, key string) bool {
	return m.data != nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	m.data = nil
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func (m *mockCache) Clear() error {
	m.data = nil
	return nil
}

func (m *mockCache) Size() int64 {
	if m.data != nil {
		return 1
	}
	return 0
}
