package fetcher

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
)

// RendererFallback renders a URL using a headless browser, returning HTML or error.
// Used as a fallback when the HTTP fetcher encounters a Cloudflare challenge (403).
type RendererFallback func(ctx context.Context, url string) (string, error)

// StealthTransportOptions configures optional StealthTransport behavior.
type StealthTransportOptions struct {
	RendererFallback RendererFallback
	Logger           *utils.Logger
}

// StealthTransport is an http.RoundTripper that uses the stealth client
// This allows integration with Colly and other HTTP client libraries
type StealthTransport struct {
	client           *Client
	rendererFallback RendererFallback
	logger           *utils.Logger
}

// NewStealthTransport creates a new StealthTransport
func NewStealthTransport(client *Client) *StealthTransport {
	return &StealthTransport{client: client}
}

// NewStealthTransportWithOptions creates a StealthTransport with optional renderer fallback.
func NewStealthTransportWithOptions(client *Client, opts StealthTransportOptions) *StealthTransport {
	return &StealthTransport{
		client:           client,
		rendererFallback: opts.RendererFallback,
		logger:           opts.Logger,
	}
}

// RoundTrip implements http.RoundTripper
func (t *StealthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Extract headers from request
	extraHeaders := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			extraHeaders[k] = v[0]
		}
	}

	// Use the stealth client to make the request
	resp, err := t.client.GetWithHeaders(req.Context(), req.URL.String(), extraHeaders)
	if err != nil {
		// Attempt renderer fallback on HTTP 403 (Cloudflare Managed Challenge)
		if t.rendererFallback != nil {
			var fetchErr *domain.FetchError
			if errors.As(err, &fetchErr) && fetchErr.StatusCode == 403 {
				return t.tryRendererFallback(req, err)
			}
		}
		return nil, err
	}

	// Convert domain.Response to http.Response
	// IMPORTANT: We must strip Content-Encoding header because we are returning
	// the already decompressed body. If we leave it, the caller (e.g. Colly)
	// will try to decompress it again and fail with "gzip: invalid header".
	resp.Headers.Del("Content-Encoding")

	return &http.Response{
		Status: http.StatusText(resp.StatusCode),

		StatusCode:    resp.StatusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        resp.Headers,
		Body:          io.NopCloser(bytes.NewReader(resp.Body)),
		ContentLength: int64(len(resp.Body)),
		Request:       req,
	}, nil
}

// tryRendererFallback attempts to render the page using a headless browser
// when the HTTP fetcher encounters a 403 (likely Cloudflare challenge).
func (t *StealthTransport) tryRendererFallback(req *http.Request, originalErr error) (*http.Response, error) {
	if t.logger != nil {
		t.logger.Info().Str("url", req.URL.String()).Msg("HTTP 403 detected, attempting headless browser fallback")
	}

	html, err := t.rendererFallback(req.Context(), req.URL.String())
	if err != nil {
		if t.logger != nil {
			t.logger.Warn().Err(err).Str("url", req.URL.String()).Msg("Renderer fallback failed")
		}
		return nil, originalErr
	}

	if t.logger != nil {
		t.logger.Info().Str("url", req.URL.String()).Int("bytes", len(html)).Msg("Renderer fallback succeeded")
	}

	body := []byte(html)
	return &http.Response{
		Status:        "OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}, nil
}

// Transport returns the StealthTransport as http.RoundTripper
func (c *Client) Transport() http.RoundTripper {
	return NewStealthTransport(c)
}

// TransportWithOptions returns a StealthTransport configured with the given options.
func (c *Client) TransportWithOptions(opts StealthTransportOptions) http.RoundTripper {
	return NewStealthTransportWithOptions(c, opts)
}
