package renderer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for renderer package with real browser
// These tests verify browser rendering, framework detection, and pool management.
// Tests are skipped if Chrome/Chromium is not available.

func TestRendererIntegration_FrameworkDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("detects React application", func(t *testing.T) {
		// Setup: Create test server with React HTML
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html>
<html><head><title>React App</title></head>
<body><div id="root"></div>
<script src="/static/js/bundle.js"></script>
</body></html>`))
		}))
		defer server.Close()

		// Execute: Detect framework
		framework := renderer.DetectFramework(`<!DOCTYPE html>
<html><body><div id="root"></div></body></html>`)

		// Verify: Framework detected as React
		assert.Equal(t, "React", framework)
	})

	t.Run("detects Next.js application", func(t *testing.T) {
		// Setup: Create test server with Next.js HTML
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Next.js App</title></head>
<body><div id="__next"></div>
<script id="__NEXT_DATA__" type="application/json">{}</script>
</body></html>`))
		}))
		defer server.Close()

		// Execute: Detect framework
		framework := renderer.DetectFramework(`<!DOCTYPE html>
<html><body><div id="__next"></div></body></html>`)

		// Verify: Framework detected as Next.js
		assert.Equal(t, "Next.js", framework)
	})

	t.Run("detects Vue application", func(t *testing.T) {
		// Execute: Detect framework
		framework := renderer.DetectFramework(`<!DOCTYPE html>
<html><body><div id="app" v-cloak>{{ message }}</div></body></html>`)

		// Verify: Framework detected as Vue
		assert.Equal(t, "Vue", framework)
	})

	t.Run("detects Angular application", func(t *testing.T) {
		// Execute: Detect framework
		framework := renderer.DetectFramework(`<!DOCTYPE html>
<html><body><app-root ng-version="15.0.0"></app-root></body></html>`)

		// Verify: Framework detected as Angular
		assert.Equal(t, "Angular", framework)
	})
}

func TestRendererIntegration_BasicRendering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("renders about:blank", func(t *testing.T) {
		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render about:blank
		ctx := context.Background()
		renderOpts := renderer.DefaultRenderOptions()
		renderOpts.Timeout = 10 * time.Second

		html, err := r.Render(ctx, "about:blank", renderOpts)

		// Verify: HTML returned without error
		require.NoError(t, err)
		assert.NotEmpty(t, html)
		assert.Contains(t, html, "<html>")
	})

	t.Run("renders simple HTML page", func(t *testing.T) {
		// Setup: Create test server with simple HTML
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Test Page</title></head>
<body><h1>Hello World</h1><p>This is a test page.</p></body></html>`))
		}))
		defer server.Close()

		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render the page
		ctx := context.Background()
		renderOpts := renderer.DefaultRenderOptions()
		renderOpts.Timeout = 10 * time.Second

		html, err := r.Render(ctx, server.URL, renderOpts)

		// Verify: HTML contains expected content
		require.NoError(t, err)
		assert.NotEmpty(t, html)
		assert.Contains(t, html, "Hello World")
		assert.Contains(t, html, "This is a test page")
	})

	t.Run("renders page with cookies", func(t *testing.T) {
		// Setup: Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify cookies were sent
			cookies := r.Cookies()
			assert.Len(t, cookies, 1, "Should have received 1 cookie")
			if len(cookies) > 0 {
				assert.Equal(t, "testCookie", cookies[0].Name)
				assert.Equal(t, "testValue", cookies[0].Value)
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html><html><body>Cookie test</body></html>`))
		}))
		defer server.Close()

		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Setup: Create cookies
		// Note: Domain must match server URL or be empty for cookie to be sent
		// Parse server URL to get the host
		serverURL, _ := url.Parse(server.URL)
		cookies := []*http.Cookie{
			{
				Name:     "testCookie",
				Value:    "testValue",
				Path:     "/",
				Domain:   serverURL.Hostname(),
				Secure:   false,
				HttpOnly: false,
			},
		}

		// Execute: Render with cookies
		ctx := context.Background()
		renderOpts := domain.RenderOptions{
			Timeout:     10 * time.Second,
			WaitStable:  1 * time.Second,
			ScrollToEnd: false,
			Cookies:     cookies,
		}

		html, err := r.Render(ctx, server.URL, renderOpts)

		// Verify: Render succeeded
		require.NoError(t, err)
		assert.NotEmpty(t, html)
	})
}

func TestRendererIntegration_TabPoolConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("concurrent rendering with pool", func(t *testing.T) {
		// Setup: Create test server
		requestCount := 0
		var mu sync.Mutex

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte(`<!DOCTYPE html>
<html><body><h1>Page ` + string(rune('0'+count)) + `</h1></body></html>`))
		}))
		defer server.Close()

		// Setup: Create renderer with pool
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 3

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render multiple pages concurrently
		ctx := context.Background()
		renderOpts := renderer.DefaultRenderOptions()
		renderOpts.Timeout = 15 * time.Second

		var wg sync.WaitGroup
		numRequests := 5
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := r.Render(ctx, server.URL, renderOpts)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Verify: No errors occurred
		for err := range errors {
			assert.NoError(t, err, "Concurrent render should not error")
		}
	})

	t.Run("pool respects max tabs limit", func(t *testing.T) {
		// Setup: Create renderer with pool of 2
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 2

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		pool, err := r.GetTabPool()
		require.NoError(t, err)

		// Verify: Initial pool size
		assert.Equal(t, 2, pool.MaxSize())
		assert.Equal(t, 2, pool.Size())

		// Execute: Acquire both tabs
		ctx := context.Background()
		tab1, err := pool.Acquire(ctx)
		require.NoError(t, err)
		assert.NotNil(t, tab1)
		assert.Equal(t, 1, pool.Size())

		tab2, err := pool.Acquire(ctx)
		require.NoError(t, err)
		assert.NotNil(t, tab2)
		assert.Equal(t, 0, pool.Size())

		// Release tabs
		pool.Release(tab1)
		assert.Equal(t, 1, pool.Size())

		pool.Release(tab2)
		assert.Equal(t, 2, pool.Size())
	})
}

func TestRendererIntegration_TimeoutHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("context cancellation propagates to acquire", func(t *testing.T) {
		// Setup: Create renderer with single tab
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		pool, err := r.GetTabPool()
		require.NoError(t, err)

		// Acquire the only tab
		ctx := context.Background()
		tab1, err := pool.Acquire(ctx)
		require.NoError(t, err)
		defer pool.Release(tab1)

		// Try to acquire another tab with timeout
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		_, err = pool.Acquire(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("render respects timeout", func(t *testing.T) {
		// Setup: Create test server with delay
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Longer than render timeout
			w.WriteHeader(200)
			w.Write([]byte(`<html><body>Delayed response</body></html>`))
		}))
		defer server.Close()

		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render with short timeout
		ctx := context.Background()
		renderOpts := renderer.DefaultRenderOptions()
		renderOpts.Timeout = 1 * time.Second // Shorter than server delay

		startTime := time.Now()
		_, err = r.Render(ctx, server.URL, renderOpts)
		duration := time.Since(startTime)

		// Verify: Request timed out
		assert.Error(t, err)
		assert.Less(t, duration, 3*time.Second, "Should timeout quickly, not wait for full server delay")
	})
}

func TestRendererIntegration_ScrollToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("scrolls to end of page", func(t *testing.T) {
		// Setup: Create test server with tall page
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			html := `<!DOCTYPE html>
<html><head><style>body { height: 10000px; }</style></head>
<body><h1>Tall Page</h1><div id="bottom" style="position: absolute; bottom: 0;">Bottom of page</div></body></html>`
			w.Write([]byte(html))
		}))
		defer server.Close()

		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render with scroll to end
		ctx := context.Background()
		renderOpts := renderer.DefaultRenderOptions()
		renderOpts.Timeout = 10 * time.Second
		renderOpts.ScrollToEnd = true

		html, err := r.Render(ctx, server.URL, renderOpts)

		// Verify: Page rendered
		require.NoError(t, err)
		assert.NotEmpty(t, html)
		assert.Contains(t, html, "Tall Page")
	})
}

func TestRendererIntegration_WaitForSelector(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("waits for specific selector", func(t *testing.T) {
		// Setup: Create test server with dynamically loaded content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			html := `<!DOCTYPE html>
<html><body>
<h1>Page Title</h1>
<div id="dynamic-content" style="display: none;">Dynamic Content Loaded</div>
<script>
setTimeout(function() {
	document.getElementById('dynamic-content').style.display = 'block';
}, 100);
</script>
</body></html>`
			w.Write([]byte(html))
		}))
		defer server.Close()

		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 1

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)
		defer r.Close()

		// Execute: Render with wait for selector
		ctx := context.Background()
		renderOpts := domain.RenderOptions{
			Timeout:     10 * time.Second,
			WaitStable:  500 * time.Millisecond,
			ScrollToEnd: false,
			WaitFor:     "#dynamic-content",
		}

		html, err := r.Render(ctx, server.URL, renderOpts)

		// Verify: Dynamic content loaded
		require.NoError(t, err)
		assert.NotEmpty(t, html)
		assert.Contains(t, html, "Dynamic Content Loaded")
	})
}

func TestRendererIntegration_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("close releases all resources", func(t *testing.T) {
		// Setup: Create renderer
		opts := renderer.DefaultRendererOptions()
		opts.MaxTabs = 3

		r, err := renderer.NewRenderer(opts)
		require.NoError(t, err)

		pool, err := r.GetTabPool()
		require.NoError(t, err)
		assert.Equal(t, 3, pool.Size())

		// Execute: Close renderer
		err = r.Close()
		assert.NoError(t, err)

		// Verify: Pool is closed
		assert.Equal(t, 0, pool.Size())

		// Close again should be idempotent
		err = r.Close()
		assert.NoError(t, err)
	})
}

func TestRendererIntegration_DefaultOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping integration test")
	}

	t.Run("uses default renderer options", func(t *testing.T) {
		// Execute: Get default options
		opts := renderer.DefaultRendererOptions()

		// Verify: Default values are set
		assert.Equal(t, 60*time.Second, opts.Timeout)
		assert.Equal(t, 5, opts.MaxTabs)
		assert.True(t, opts.Stealth)
		assert.True(t, opts.Headless)
		assert.Empty(t, opts.BrowserPath)
	})

	t.Run("uses default render options", func(t *testing.T) {
		// Execute: Get default options
		opts := renderer.DefaultRenderOptions()

		// Verify: Default values are set
		assert.Equal(t, 60*time.Second, opts.Timeout)
		assert.Equal(t, 2*time.Second, opts.WaitStable)
		assert.True(t, opts.ScrollToEnd)
	})

	t.Run("uses default stealth options", func(t *testing.T) {
		// Execute: Get default options
		opts := renderer.DefaultStealthOptions()

		// Verify: Default values are set
		assert.True(t, opts.HideWebdriver)
		assert.True(t, opts.EmulatePlugins)
		assert.False(t, opts.RandomizeViewport)
		assert.True(t, opts.DisableAutomationFlags)
	})
}
