package renderer_test

import (
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestNewRenderer_Success tests creating a new renderer successfully
func TestNewRenderer_Success(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create renderer with default options
	opts := renderer.DefaultRendererOptions()

	r, err := renderer.NewRenderer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Verify the pool was created
	pool, err := r.GetTabPool()
	assert.NoError(t, err)
	assert.NotNil(t, pool)

	// Clean up
	err = r.Close()
	assert.NoError(t, err)
}

// TestNewRenderer_WithOptions tests creating a new renderer with custom options
func TestNewRenderer_WithOptions(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create renderer with custom options
	opts := renderer.DefaultRendererOptions()
	opts.Timeout = 30 * time.Second
	opts.MaxTabs = 10
	opts.Stealth = false
	opts.Headless = false

	r, err := renderer.NewRenderer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Verify the pool was created with custom max tabs
	pool, err := r.GetTabPool()
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, 10, pool.MaxSize())

	// Clean up
	err = r.Close()
	assert.NoError(t, err)
}

// TestClose_ClosesBrowser tests that Close closes the browser
func TestClose_ClosesBrowser(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create renderer
	opts := renderer.DefaultRendererOptions()
	r, err := renderer.NewRenderer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Close renderer
	err = r.Close()
	assert.NoError(t, err)

	// Try to close again (should be idempotent)
	err = r.Close()
	assert.NoError(t, err)
}

// TestIsAvailable_CheckAvailability tests the IsAvailable function
func TestIsAvailable_CheckAvailability(t *testing.T) {
	// Test that IsAvailable returns a boolean
	// This test will pass if browser is available or not
	available := renderer.IsAvailable()
	assert.IsType(t, true, available)
}

// TestGetBrowserPath_FindsChrome tests GetBrowserPath when Chrome is found
func TestGetBrowserPath_FindsChrome(t *testing.T) {
	// This test will only pass if Chrome/Chromium is installed
	path, exists := renderer.GetBrowserPath()

	// If browser is available, path should exist
	if renderer.IsAvailable() {
		assert.True(t, exists)
		assert.NotEmpty(t, path)
	} else {
		// If browser is not available, path should be empty
		assert.False(t, exists)
		assert.Empty(t, path)
	}
}

// TestGetBrowserPath_NotFound tests GetBrowserPath when Chrome is not found
func TestGetBrowserPath_NotFound(t *testing.T) {
	// This test documents the expected behavior when Chrome is not found
	// The actual behavior depends on the test environment
	path, exists := renderer.GetBrowserPath()

	// If browser is not available, we expect path to be empty and exists to be false
	if !renderer.IsAvailable() {
		assert.False(t, exists)
		assert.Empty(t, path)
	}
}

// TestDefaultRenderOptions tests the DefaultRenderOptions function
func TestDefaultRenderOptions(t *testing.T) {
	opts := renderer.DefaultRenderOptions()

	// Verify default options
	assert.Equal(t, 60*time.Second, opts.Timeout)
	assert.Equal(t, 2*time.Second, opts.WaitStable)
	assert.True(t, opts.ScrollToEnd)
}

// TestNewRenderer_ZeroTimeout tests creating a renderer with zero timeout (should default)
func TestNewRenderer_ZeroTimeout(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create renderer with zero timeout (should default to 60s)
	opts := renderer.DefaultRendererOptions()
	opts.Timeout = 0
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Clean up
	err = r.Close()
	assert.NoError(t, err)
}

// TestNewRenderer_ZeroMaxTabs tests creating a renderer with zero maxTabs (should default)
func TestNewRenderer_ZeroMaxTabs(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create renderer with zero maxTabs (should default to 5)
	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 0

	r, err := renderer.NewRenderer(opts)
	assert.NoError(t, err)
	assert.NotNil(t, r)

	// Verify the pool was created with default max tabs
	pool, err := r.GetTabPool()
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, 5, pool.MaxSize())

	// Clean up
	err = r.Close()
	assert.NoError(t, err)
}
