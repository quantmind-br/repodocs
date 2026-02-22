package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Client is a stealth HTTP client using tls-client
type Client struct {
	tlsClient    tls_client.HttpClient
	userAgent    string
	retrier      *Retrier
	cache        domain.Cache
	cacheEnabled bool
	cacheTTL     time.Duration
}

// ClientOptions contains options for creating a Client
type ClientOptions struct {
	Timeout     time.Duration
	MaxRetries  int
	EnableCache bool
	CacheTTL    time.Duration
	Cache       domain.Cache
	UserAgent   string
	ProxyURL    string
}

// DefaultClientOptions returns default client options
func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		Timeout:     90 * time.Second,
		MaxRetries:  3,
		EnableCache: true,
		CacheTTL:    24 * time.Hour,
		UserAgent:   "",
		ProxyURL:    "",
	}
}

// NewClient creates a new stealth HTTP client
func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 90 * time.Second
	}

	tlsTimeout := opts.Timeout * 3
	if tlsTimeout < 3*time.Minute {
		tlsTimeout = 3 * time.Minute
	}

	tlsOpts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(int(tlsTimeout.Seconds())),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithNotFollowRedirects(),
	}

	if opts.ProxyURL != "" {
		tlsOpts = append(tlsOpts, tls_client.WithProxyUrl(opts.ProxyURL))
	}

	tlsClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), tlsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create tls client: %w", err)
	}

	// Create retrier
	retrier := NewRetrier(RetrierOptions{
		MaxRetries:      opts.MaxRetries,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	})

	return &Client{
		tlsClient:    tlsClient,
		userAgent:    opts.UserAgent,
		retrier:      retrier,
		cache:        opts.Cache,
		cacheEnabled: opts.EnableCache,
		cacheTTL:     opts.CacheTTL,
	}, nil
}

// Get fetches content from a URL
func (c *Client) Get(ctx context.Context, url string) (*domain.Response, error) {
	return c.GetWithHeaders(ctx, url, nil)
}

// GetWithHeaders fetches content with custom headers
func (c *Client) GetWithHeaders(ctx context.Context, url string, extraHeaders map[string]string) (*domain.Response, error) {
	// Check cache first
	if c.cacheEnabled && c.cache != nil {
		cached, err := c.getFromCache(ctx, url)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	// Perform request with retry
	var resp *domain.Response
	err := c.retrier.Retry(ctx, func() error {
		var err error
		resp, err = c.doRequest(ctx, url, extraHeaders)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Cache the response
	if c.cacheEnabled && c.cache != nil && resp != nil {
		_ = c.saveToCache(ctx, url, resp)
	}

	return resp, nil
}

// doRequest performs the actual HTTP request
func (c *Client) doRequest(ctx context.Context, targetURL string, extraHeaders map[string]string) (*domain.Response, error) {
	// Create request using fhttp (tls-client's http package)
	req, err := fhttp.NewRequest(fhttp.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply stealth headers
	headers := StealthHeaders(c.userAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Apply extra headers
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	// Perform request
	resp, err := c.tlsClient.Do(req)
	if err != nil {
		return nil, &domain.FetchError{
			URL: targetURL,
			Err: fmt.Errorf("request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode >= 400 {
		if ShouldRetryStatus(resp.StatusCode) {
			return nil, &domain.RetryableError{
				Err:        &domain.FetchError{URL: targetURL, StatusCode: resp.StatusCode, Err: fmt.Errorf("HTTP %d", resp.StatusCode)},
				RetryAfter: int(ParseRetryAfter(resp.Header.Get("Retry-After")).Seconds()),
			}
		}
		return nil, &domain.FetchError{
			URL:        targetURL,
			StatusCode: resp.StatusCode,
			Err:        fmt.Errorf("HTTP %d", resp.StatusCode),
		}
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Convert fhttp.Header to http.Header
	httpHeaders := make(http.Header)
	for k, v := range resp.Header {
		httpHeaders[k] = v
	}

	return &domain.Response{
		StatusCode:  resp.StatusCode,
		Body:        body,
		Headers:     httpHeaders,
		ContentType: resp.Header.Get("Content-Type"),
		URL:         targetURL,
		FromCache:   false,
	}, nil
}

// GetCookies returns cookies for a URL (for sharing with renderer)
func (c *Client) GetCookies(rawURL string) []*http.Cookie {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	cookies := c.tlsClient.GetCookies(parsedURL)
	result := make([]*http.Cookie, len(cookies))
	for i, cookie := range cookies {
		result[i] = &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   cookie.Domain,
			Expires:  cookie.Expires,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
		}
	}
	return result
}

// Close releases client resources
func (c *Client) Close() error {
	// TLS client doesn't have a Close method, but we keep this for interface compliance
	return nil
}

// getFromCache retrieves a response from cache
func (c *Client) getFromCache(ctx context.Context, url string) (*domain.Response, error) {
	if c.cache == nil {
		return nil, domain.ErrCacheMiss
	}

	data, err := c.cache.Get(ctx, url)
	if err != nil {
		return nil, err
	}

	return &domain.Response{
		StatusCode:  200,
		Body:        data,
		ContentType: "text/html",
		URL:         url,
		FromCache:   true,
	}, nil
}

// saveToCache saves a response to cache
func (c *Client) saveToCache(ctx context.Context, url string, resp *domain.Response) error {
	if c.cache == nil {
		return nil
	}
	return c.cache.Set(ctx, url, resp.Body, c.cacheTTL)
}

// SetCache sets the cache implementation
func (c *Client) SetCache(cache domain.Cache) {
	c.cache = cache
}

// SetCacheEnabled enables or disables caching
func (c *Client) SetCacheEnabled(enabled bool) {
	c.cacheEnabled = enabled
}
