package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestServer is a wrapper around httptest.Server for testing
type TestServer struct {
	*httptest.Server
	mux *http.ServeMux
}

// NewTestServer creates a new test HTTP server
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Cleanup(func() {
		server.Close()
	})

	return &TestServer{
		Server: server,
		mux:    mux,
	}
}

// Handle registers a handler for a specific path
func (ts *TestServer) Handle(t *testing.T, path string, handler http.HandlerFunc) {
	t.Helper()
	ts.mux.HandleFunc(path, handler)
}

// HandleString registers a handler that returns a string response
func (ts *TestServer) HandleString(t *testing.T, path, contentType, body string) {
	t.Helper()
	ts.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	})
}

// HandleHTML registers a handler that returns HTML content
func (ts *TestServer) HandleHTML(t *testing.T, path, htmlBody string) {
	t.Helper()
	ts.HandleString(t, path, "text/html; charset=utf-8", htmlBody)
}

// HandleJSON registers a handler that returns JSON content
func (ts *TestServer) HandleJSON(t *testing.T, path string, data interface{}) {
	t.Helper()
	ts.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	})
}

// Handle404 registers a handler that returns 404 Not Found
func (ts *TestServer) Handle404(t *testing.T, path string) {
	t.Helper()
	ts.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})
}

// Handle500 registers a handler that returns 500 Internal Server Error
func (ts *TestServer) Handle500(t *testing.T, path string) {
	t.Helper()
	ts.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})
}

// NewResponse creates a mock domain.Response
func NewResponse(statusCode int, body string, contentType string) *domain.Response {
	return &domain.Response{
		StatusCode:  statusCode,
		Body:        []byte(body),
		Headers:     make(http.Header),
		ContentType: contentType,
		URL:         "",
		FromCache:   false,
	}
}

// CreateHTTPResponse creates an http.Response from domain.Response
func CreateHTTPResponse(t *testing.T, resp *domain.Response) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: resp.StatusCode,
		Body:       io.NopCloser(strings.NewReader(string(resp.Body))),
		Header:     resp.Headers,
	}
}

// VerifyRequest verifies an HTTP request matches expected values
func VerifyRequest(t *testing.T, req *http.Request, method, path string) {
	t.Helper()
	require.Equal(t, method, req.Method)
	require.Equal(t, path, req.URL.Path)
}
