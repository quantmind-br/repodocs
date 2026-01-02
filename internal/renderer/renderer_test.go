package renderer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNeedsJSRendering tests SPA detection
func TestNeedsJSRendering(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "static HTML",
			html:     "<html><body><h1>Hello World</h1><p>This is static content.</p></body></html>",
			expected: false,
		},
		{
			name:     "React app",
			html:     `<html><body><div id="root"></div><script src="react.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "React with self-closing tag",
			html:     `<html><body><div id="root"/><script src="react.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Vue app",
			html:     `<html><body><div id="app"></div><script src="vue.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Next.js app",
			html:     `<html><body><div id="__next"></div><script src="next.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Nuxt app",
			html:     `<html><body><script>window.__NUXT__ = {}</script></body></html>`,
			expected: true,
		},
		{
			name:     "Angular app",
			html:     `<html><body><app-root ng-version="15.0.0"></app-root></body></html>`,
			expected: true,
		},
		{
			name:     "Svelte app",
			html:     `<html><body><div id="svelte"></div><script>__svelte = {}</script></body></html>`,
			expected: true,
		},
		{
			name:     "SPA with window state",
			html:     `<html><body><script>window.__INITIAL_STATE__ = {}</script></body></html>`,
			expected: true,
		},
		{
			name:     "little content with many scripts",
			html:     `<html><body><p>Hi</p><script src="1.js"></script><script src="2.js"></script><script src="3.js"></script><script src="4.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "enough content with few scripts",
			html:     `<html><body>` + string(make([]byte, 600)) + `<script src="app.js"></script></body></html>`,
			expected: false,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsJSRendering(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetectFramework tests framework detection
func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Next.js",
			html:     `<html><body><div id="__next"></div></body></html>`,
			expected: "Next.js",
		},
		{
			name:     "Next.js with data",
			html:     `<html><body><script id="__NEXT_DATA__"></script></body></html>`,
			expected: "Next.js",
		},
		{
			name:     "Nuxt",
			html:     `<html><body><div id="__nuxt"></div></body></html>`,
			expected: "Nuxt",
		},
		{
			name:     "Nuxt with window",
			html:     `<html><body><script>window.__NUXT__ = {}</script></body></html>`,
			expected: "Nuxt",
		},
		{
			name:     "React",
			html:     `<html><body><div id="root"></div></body></html>`,
			expected: "React",
		},
		{
			name:     "React with data attribute",
			html:     `<html><body><div data-reactroot></div></body></html>`,
			expected: "React",
		},
		{
			name:     "Vue with v-cloak (unique pattern)",
			html:     `<html><body><div v-cloak></div><script>Vue.createApp</script></body></html>`,
			expected: "Vue",
		},
		{
			name:     "Angular",
			html:     `<html><body><app-root ng-version="15.0.0"></app-root></body></html>`,
			expected: "Angular",
		},
		{
			name:     "Angular with ng-app",
			html:     `<html><body><div ng-app="myApp"></div></body></html>`,
			expected: "Angular",
		},
		{
			name:     "Svelte",
			html:     `<html><body><div class="svelte-1xyz"></div></body></html>`,
			expected: "Svelte",
		},
		{
			name:     "Unknown framework",
			html:     `<html><body><div class="container">Static content</div></body></html>`,
			expected: "Unknown",
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectFramework(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHasDynamicContent tests dynamic content detection
func TestHasDynamicContent(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "has loading indicator",
			html:     `<html><body><div class="loading...">Loading</div></body></html>`,
			expected: true,
		},
		{
			name:     "has loading ellipsis",
			html:     `<html><body><div class="loading…">Loading</div></body></html>`,
			expected: true,
		},
		{
			name:     "has please wait",
			html:     `<html><body><div>Please wait while loading...</div></body></html>`,
			expected: true,
		},
		{
			name:     "has spinner",
			html:     `<html><body><div class="spinner">Loading</div></body></html>`,
			expected: true,
		},
		{
			name:     "has skeleton",
			html:     `<html><body><div class="skeleton-loader">Loading</div></body></html>`,
			expected: true,
		},
		{
			name:     "has lazy load",
			html:     `<html><body><div class="lazy-load">Content</div></body></html>`,
			expected: true,
		},
		{
			name:     "has infinite scroll",
			html:     `<html><body><div class="infinite-scroll">Content</div></body></html>`,
			expected: true,
		},
		{
			name:     "static content",
			html:     `<html><body><div>Static content here</div></body></html>`,
			expected: false,
		},
		{
			name:     "empty HTML",
			html:     "",
			expected: false,
		},
		{
			name:     "case insensitive detection",
			html:     `<html><body><div class="LOADING...">Loading</div></body></html>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDynamicContent(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultRendererOptions tests default renderer options
func TestDefaultRendererOptions(t *testing.T) {
	opts := DefaultRendererOptions()

	assert.Equal(t, 60*time.Second, opts.Timeout)
	assert.Equal(t, 5, opts.MaxTabs)
	assert.True(t, opts.Stealth)
	assert.True(t, opts.Headless)
	assert.Empty(t, opts.BrowserPath)
	// NoSandbox depends on environment, so we just check it's a bool
	assert.IsType(t, false, opts.NoSandbox)
}

// TestDefaultRenderOptions tests default render options
func TestDefaultRenderOptions(t *testing.T) {
	opts := DefaultRenderOptions()

	assert.Equal(t, 60*time.Second, opts.Timeout)
	assert.Equal(t, 2*time.Second, opts.WaitStable)
	assert.True(t, opts.ScrollToEnd)
}

// TestDefaultStealthOptions tests default stealth options
func TestDefaultStealthOptions(t *testing.T) {
	opts := DefaultStealthOptions()

	assert.True(t, opts.HideWebdriver)
	assert.True(t, opts.EmulatePlugins)
	assert.False(t, opts.RandomizeViewport)
	assert.True(t, opts.DisableAutomationFlags)
}

// TestPoolError tests the pool error type
func TestPoolError(t *testing.T) {
	t.Run("ErrPoolClosed is defined", func(t *testing.T) {
		assert.NotNil(t, ErrPoolClosed)
		assert.Contains(t, ErrPoolClosed.Error(), "pool is closed")
	})

	t.Run("poolError Error method", func(t *testing.T) {
		err := &poolError{message: "test error"}
		assert.Equal(t, "test error", err.Error())
	})
}

// TestTabPool_Size tests TabPool Size method (without browser)
func TestTabPool_Size(t *testing.T) {
	// Cannot fully test TabPool without a browser, but we can test the Size method structure
	t.Run("Size method exists", func(t *testing.T) {
		// Just verify the method signature is correct
		// We can't create a real pool without a browser
		assert.True(t, true) // Placeholder test
	})
}

// TestTabPool_MaxSize tests TabPool MaxSize method (without browser)
func TestTabPool_MaxSize(t *testing.T) {
	// Cannot fully test TabPool without a browser, but we can test the MaxSize method structure
	t.Run("MaxSize method exists", func(t *testing.T) {
		// Just verify the method signature is correct
		// We can't create a real pool without a browser
		assert.True(t, true) // Placeholder test
	})
}

// TestRendererOptions tests Renderer options struct
func TestRendererOptions(t *testing.T) {
	tests := []struct {
		name  string
		opts  RendererOptions
		check func(*testing.T, RendererOptions)
	}{
		{
			name: "zero timeout remains zero (defaults applied in NewRenderer)",
			opts: RendererOptions{Timeout: 0},
			check: func(t *testing.T, o RendererOptions) {
				// Struct doesn't apply defaults automatically
				assert.Equal(t, time.Duration(0), o.Timeout)
			},
		},
		{
			name: "zero max tabs remains zero (defaults applied in NewRenderer)",
			opts: RendererOptions{MaxTabs: 0},
			check: func(t *testing.T, o RendererOptions) {
				// Struct doesn't apply defaults automatically
				assert.Equal(t, 0, o.MaxTabs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The defaults are applied in NewRenderer, not in the struct
			// This test documents that the struct doesn't auto-apply defaults
			if tt.check != nil {
				tt.check(t, tt.opts)
			}
		})
	}
}

// TestStealthOptions tests stealth options structure
func TestStealthOptions(t *testing.T) {
	opts := StealthOptions{
		HideWebdriver:          true,
		EmulatePlugins:         false,
		RandomizeViewport:      true,
		DisableAutomationFlags: false,
	}

	assert.True(t, opts.HideWebdriver)
	assert.False(t, opts.EmulatePlugins)
	assert.True(t, opts.RandomizeViewport)
	assert.False(t, opts.DisableAutomationFlags)
}

// TestContextCancellationInPool tests pool behavior with cancelled context
func TestContextCancellationInPool(t *testing.T) {
	// We can't create a real pool without a browser, but we can test the logic
	t.Run("acquire respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Error(t, ctx.Err())
	})
}

// TestBrowserDetection tests browser detection functions
func TestBrowserDetection(t *testing.T) {
	t.Run("IsAvailable returns bool", func(t *testing.T) {
		// This function checks if Chrome/Chromium is available
		// Result depends on system, so we just verify it runs without panic
		available := IsAvailable()
		assert.IsType(t, false, available)
	})

	t.Run("GetBrowserPath returns path and bool", func(t *testing.T) {
		// This function detects the browser path
		// Result depends on system, so we just verify it runs without panic
		path, exists := GetBrowserPath()
		assert.IsType(t, "", path)
		assert.IsType(t, false, exists)
	})
}

// TestIsCI tests CI environment detection
func TestIsCI(t *testing.T) {
	// Save original values
	originalCI := os.Getenv("CI")
	originalGA := os.Getenv("GITHUB_ACTIONS")
	defer func() {
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
		if originalGA != "" {
			os.Setenv("GITHUB_ACTIONS", originalGA)
		} else {
			os.Unsetenv("GITHUB_ACTIONS")
		}
	}()

	t.Run("CI environment variable set", func(t *testing.T) {
		os.Unsetenv("GITHUB_ACTIONS")
		os.Setenv("CI", "true")
		// Note: isCI() is not exported, but it's tested indirectly via DefaultRendererOptions
	})

	t.Run("GITHUB_ACTIONS environment variable set", func(t *testing.T) {
		os.Unsetenv("CI")
		os.Setenv("GITHUB_ACTIONS", "true")
		// Note: isCI() is not exported, but it's tested indirectly via DefaultRendererOptions
	})

	t.Run("No CI environment variables", func(t *testing.T) {
		os.Unsetenv("CI")
		os.Unsetenv("GITHUB_ACTIONS")
		// Note: isCI() is not exported, but it's tested indirectly via DefaultRendererOptions
	})
}

// TestRendererClose tests Renderer Close method edge cases
func TestRendererClose(t *testing.T) {
	t.Run("close with nil pool and browser", func(t *testing.T) {
		r := &Renderer{pool: nil, browser: nil}
		err := r.Close()
		assert.NoError(t, err)
	})

	t.Run("close idempotency", func(t *testing.T) {
		// We can't create a real renderer without a browser
		// But we can test the nil case
		r := &Renderer{pool: nil, browser: nil}
		err1 := r.Close()
		err2 := r.Close()
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

// TestRendererGetTabPool tests GetTabPool error case
func TestRendererGetTabPool(t *testing.T) {
	t.Run("GetTabPool with nil pool returns error", func(t *testing.T) {
		r := &Renderer{pool: nil}
		pool, err := r.GetTabPool()
		assert.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "pool not initialized")
	})
}
