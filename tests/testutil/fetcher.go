package testutil

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// SimpleFetcher provides deterministic responses for tests without real network access.
type SimpleFetcher struct {
	baseURL string
}

// NewSimpleFetcher returns a fetcher that reads from the provided base URL and
// serves static HTML for any other URL.
func NewSimpleFetcher(baseURL string) *SimpleFetcher {
	return &SimpleFetcher{baseURL: strings.TrimRight(baseURL, "/")}
}

// Get fetches a URL or returns a static HTML response for non-base URLs.
func (f *SimpleFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	if strings.HasPrefix(url, f.baseURL) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return &domain.Response{
			StatusCode:  resp.StatusCode,
			Body:        body,
			Headers:     resp.Header,
			ContentType: resp.Header.Get("Content-Type"),
			URL:         url,
			FromCache:   false,
		}, nil
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "text/html; charset=utf-8")

	return &domain.Response{
		StatusCode:  http.StatusOK,
		Body:        []byte("<html><body>Content</body></html>"),
		Headers:     headers,
		ContentType: "text/html; charset=utf-8",
		URL:         url,
		FromCache:   false,
	}, nil
}

// GetWithHeaders fetches content with headers (ignored in this mock).
func (f *SimpleFetcher) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return f.Get(ctx, url)
}

// GetCookies returns nil cookies for tests.
func (f *SimpleFetcher) GetCookies(url string) []*http.Cookie {
	return nil
}

// Transport returns the default HTTP transport.
func (f *SimpleFetcher) Transport() http.RoundTripper {
	return http.DefaultTransport
}

// Close releases resources (no-op).
func (f *SimpleFetcher) Close() error {
	return nil
}
