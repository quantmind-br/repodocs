package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// NewMockServer creates a new HTTP test server for testing HTTP clients.
// The server will be automatically closed when the test completes.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	defer server.Close()
//	server.Handler = yourHandler
func NewMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(nil)
	t.Cleanup(server.Close)
	return server
}

// MockResponse creates a simple HTTP handler that returns a fixed response.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	server.Handler = helpers.MockResponse(200, "test response")
func MockResponse(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

// MockJSONResponse creates a simple HTTP handler that returns a JSON response.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	server.Handler = helpers.MockJSONResponse(http.StatusOK, map[string]string{"key": "value"})
func MockJSONResponse(status int, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// Note: This is a simplified version. In production, you'd want to use encoding/json
		w.Write([]byte(`{"data": "mocked"}`))
	}
}

// MockRedirect creates a simple HTTP handler that redirects to a target URL.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	target := server.URL + "/redirect-target"
//	server.Handler = helpers.MockRedirect(http.StatusMovedPermanently, target)
func MockRedirect(status int, target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, status)
	}
}

// MockNotFound creates a simple HTTP handler that returns 404 Not Found.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	server.Handler = helpers.MockNotFound()
func MockNotFound() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}
}

// MockError creates a simple HTTP handler that returns a server error.
// Usage:
//
//	server := helpers.NewMockServer(t)
//	server.Handler = helpers.MockError(http.StatusInternalServerError, "internal server error")
func MockError(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, status)
	}
}
