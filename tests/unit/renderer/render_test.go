package renderer_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestRender_Success tests rendering a simple page successfully
func TestRender_Success(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	// Create a simple render options
	renderOpts := renderer.DefaultRenderOptions()
	renderOpts.Timeout = 10 * time.Second

	// Test rendering a simple page (using a test URL that doesn't require network)
	// We'll use about:blank which should always be available
	ctx := context.Background()

	// Note: about:blank might not have much content, but it will test the rendering flow
	html, err := r.Render(ctx, "about:blank", renderOpts)
	// We don't assert on the result as about:blank might not have meaningful content
	// But we verify the function completes without error
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_WithTimeout tests rendering with a custom timeout
func TestRender_WithTimeout(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	renderOpts := renderer.DefaultRenderOptions()
	renderOpts.Timeout = 5 * time.Second

	ctx := context.Background()

	html, err := r.Render(ctx, "about:blank", renderOpts)
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_ZeroTimeout tests rendering with zero timeout (should use default)
func TestRender_ZeroTimeout(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	// Create render options with zero timeout
	renderOpts := domain.RenderOptions{
		Timeout:     0,
		WaitStable:  0,
		ScrollToEnd: false,
	}

	ctx := context.Background()

	html, err := r.Render(ctx, "about:blank", renderOpts)
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_WithScrollToEnd tests rendering with scroll to end enabled
func TestRender_WithScrollToEnd(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: true,
	}

	ctx := context.Background()

	html, err := r.Render(ctx, "about:blank", renderOpts)
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_WithCookies tests rendering with cookies
func TestRender_WithCookies(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	// Create cookies with proper domain
	cookies := []*http.Cookie{
		{
			Name:     "testCookie",
			Value:    "testValue",
			Path:     "/",
			Domain:   "example.com",
			Secure:   false,
			HttpOnly: false,
		},
	}

	renderOpts := domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: false,
		Cookies:     cookies,
	}

	ctx := context.Background()

	// Use a local URL that should always be available
	html, err := r.Render(ctx, "about:blank", renderOpts)
	// We test that cookies are passed without error (even if the page doesn't use them)
	// Note: about:blank might reject cookies, but the function should handle it gracefully
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_WithWaitFor tests rendering with wait for selector
func TestRender_WithWaitFor(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: false,
		WaitFor:     "body", // Wait for body element
	}

	ctx := context.Background()

	html, err := r.Render(ctx, "about:blank", renderOpts)
	assert.NoError(t, err)
	assert.NotNil(t, html)
}

// TestRender_WithWaitStable tests rendering with wait stable
func TestRender_WithWaitStable(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: false,
	}

	ctx := context.Background()

	html, err := r.Render(ctx, "about:blank", renderOpts)
	assert.NoError(t, err)
	assert.NotNil(t, html)
}
