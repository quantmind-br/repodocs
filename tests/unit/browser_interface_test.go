package unit

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// MockRenderer is a mock implementation of the Browser/Renderer interface
type MockRenderer struct {
	ShouldError bool
	HTML        string
}

// Render implements the Browser interface
func (m *MockRenderer) Render(ctx context.Context, url string, opts domain.RenderOptions) (string, error) {
	if m.ShouldError {
		return "", ErrMockRenderer
	}
	return m.HTML, nil
}

// Close implements the Browser interface
func (m *MockRenderer) Close() error {
	return nil
}

var ErrMockRenderer = NewMockError("mock renderer error")

// NewMockError creates a new mock error
func NewMockError(msg string) error {
	return &MockError{message: msg}
}

// MockError is a simple error type for testing
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// TestBrowserInterface tests that the Browser interface can be used with mocks
func TestBrowserInterface(t *testing.T) {
	ctx := context.Background()

	// Test successful render
	mock := &MockRenderer{
		ShouldError: false,
		HTML:        "<html><body>Test</body></html>",
	}

	result, err := mock.Render(ctx, "https://example.com", domain.RenderOptions{
		Timeout:     5 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: true,
	})

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result != "<html><body>Test</body></html>" {
		t.Errorf("Expected HTML to match, got: %s", result)
	}

	// Test error case
	mockWithError := &MockRenderer{
		ShouldError: true,
	}

	_, err = mockWithError.Render(ctx, "https://example.com", domain.RenderOptions{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	// Test Close
	err = mock.Close()
	if err != nil {
		t.Errorf("Expected no error on Close, got: %v", err)
	}
}

// TestDependenciesAcceptRendererInterface tests that Dependencies accepts a Renderer interface
func TestDependenciesAcceptRendererInterface(t *testing.T) {
	// This test demonstrates that the Dependencies struct now accepts
	// any type that implements the domain.Renderer interface

	mockRenderer := &MockRenderer{
		HTML: "<html><body>Mock</body></html>",
	}

	// This should compile without errors, proving that the interface
	// is properly implemented
	var _ domain.Renderer = mockRenderer
}
