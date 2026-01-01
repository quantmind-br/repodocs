package fetcher

import (
	"bytes"
	"io"
	"net/http"
)

// StealthTransport is an http.RoundTripper that uses the stealth client
// This allows integration with Colly and other HTTP client libraries
type StealthTransport struct {
	client *Client
}

// NewStealthTransport creates a new StealthTransport
func NewStealthTransport(client *Client) *StealthTransport {
	return &StealthTransport{client: client}
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

// Transport returns the StealthTransport as http.RoundTripper
func (c *Client) Transport() http.RoundTripper {
	return NewStealthTransport(c)
}
