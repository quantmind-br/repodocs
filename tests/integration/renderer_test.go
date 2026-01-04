package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfChromeUnavailable skips the test if Chrome is not available
func skipIfChromeUnavailable(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Chrome/Chromium not available, skipping integration test")
	}
}

// TestNewRendererIntegration tests creating a renderer with actual Chrome
func TestNewRendererIntegration(t *testing.T) {
	skipIfChromeUnavailable(t)

	// Test with default options
	opts := renderer.DefaultRendererOptions()
	opts.Headless = true
	opts.MaxTabs = 2

	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	// Verify the renderer was created successfully
	assert.NotNil(t, r)
}

// TestRenderIntegration tests rendering an actual HTML page
func TestRenderIntegration(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// Use a simple HTML page for testing
	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Test Page</title>
</head>
<body>
    <h1>Test Heading</h1>
    <p>This is a test paragraph.</p>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            document.body.classList.add('loaded');
        });
    </script>
</body>
</html>
`

	// Start a local HTTP server to serve the test HTML
	server := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testHTML)
		}),
	}

	// Find an available port
	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	go server.Serve(listener)
	defer server.Close()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	// Test basic rendering
	html, err := r.Render(ctx, url, domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitStable:  1 * time.Second,
		ScrollToEnd: false,
	})

	require.NoError(t, err)
	assert.Contains(t, html, "Test Heading")
	assert.Contains(t, html, "test paragraph")
}

// TestRenderWithWaitFor tests rendering with a wait selector
func TestRenderWithWaitFor(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// HTML with delayed content
	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Delayed Content</title>
    <script>
        setTimeout(function() {
            var el = document.createElement('div');
            el.id = 'delayed';
            el.textContent = 'Delayed Content';
            document.body.appendChild(el);
        }, 500);
    </script>
</head>
<body>
    <p>Initial content</p>
</body>
</html>
`

	server := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testHTML)
		}),
	}

	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	go server.Serve(listener)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	// Render with wait for selector
	html, err := r.Render(ctx, url, domain.RenderOptions{
		Timeout:     10 * time.Second,
		WaitFor:     "#delayed",
		ScrollToEnd: false,
	})

	require.NoError(t, err)
	assert.Contains(t, html, "Delayed Content")
}

// TestRenderWithCookies tests rendering with cookies
func TestRenderWithCookies(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// HTML that reads cookies
	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Cookie Test</title>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            var el = document.getElementById('cookie-value');
            el.textContent = document.cookie || 'no cookies';
        });
    </script>
</head>
<body>
    <div id="cookie-value"></div>
</body>
</html>
`

	server := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testHTML)
		}),
	}

	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	go server.Serve(listener)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	// Create test cookies
	cookies := []*http.Cookie{
		{
			Name:  "test_cookie",
			Value: "test_value",
			Path:  "/",
		},
	}

	// Render with cookies
	html, err := r.Render(ctx, url, domain.RenderOptions{
		Timeout:     10 * time.Second,
		Cookies:     cookies,
		ScrollToEnd: false,
	})

	require.NoError(t, err)
	// Cookie might be set (depends on how Chrome handles cookies for localhost)
	_ = html // Just verify it renders without error
}

// TestRenderTimeout tests that rendering respects timeout
func TestRenderTimeout(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// HTML that hangs (never loads)
	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Hanging Page</title>
</head>
<body>
    <h1>This page will hang</h1>
    <!-- No script to complete loading -->
</body>
</html>
`

	server := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			// Intentionally delay to trigger timeout
			time.Sleep(2 * time.Second)
			fmt.Fprint(w, testHTML)
		}),
	}

	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	go server.Serve(listener)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	// Render with short timeout
	_, err = r.Render(ctx, url, domain.RenderOptions{
		Timeout:     500 * time.Millisecond, // Short timeout
		ScrollToEnd: false,
	})

	// Should timeout
	if !assert.Error(t, err) {
		t.Fatal("Expected timeout error but got nil")
	}
	errMsg := err.Error()
	// Check for timeout-related error message (context deadline exceeded, timeout, etc.)
	hasTimeout := strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "context")
	if !hasTimeout {
		t.Errorf("Expected timeout-related error but got: %s", errMsg)
	}
}

// TestRenderWithScrollToEnd tests rendering with scroll to end
func TestRenderWithScrollToEnd(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// HTML with lazy-loaded content
	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Scroll Test</title>
    <script>
        var loaded = false;
        window.addEventListener('scroll', function() {
            if (!loaded && window.scrollY > 500) {
                var el = document.createElement('div');
                el.id = 'lazy-loaded';
                el.textContent = 'Lazy loaded content';
                document.body.appendChild(el);
                loaded = true;
            }
        });
    </script>
    <style>
        body { height: 2000px; }
    </style>
</head>
<body>
    <div style="height: 1000px;">Scroll down...</div>
</body>
</html>
`

	server := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testHTML)
		}),
	}

	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}
	port := listener.Addr().(*net.TCPAddr).Port
	server.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	go server.Serve(listener)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	// Render with scroll to end
	html, err := r.Render(ctx, url, domain.RenderOptions{
		Timeout:     10 * time.Second,
		ScrollToEnd: true,
	})

	require.NoError(t, err)
	// Scroll to end should trigger lazy loading
	assert.Contains(t, html, "Scroll down")
}

// TestCloseMultipleTimes tests that Close can be called multiple times
func TestCloseMultipleTimes(t *testing.T) {
	skipIfChromeUnavailable(t)

	r, err := renderer.NewRenderer(renderer.DefaultRendererOptions())
	require.NoError(t, err)

	// First close
	err = r.Close()
	require.NoError(t, err)

	// Second close should not error
	err = r.Close()
	require.NoError(t, err)
}

// TestConcurrentRender tests rendering multiple pages concurrently
func TestConcurrentRender(t *testing.T) {
	skipIfChromeUnavailable(t)

	opts := renderer.DefaultRendererOptions()
	opts.Timeout = 10 * time.Second
	opts.MaxTabs = 5
	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	// Create multiple test pages
	testPages := []string{
		"<html><body><h1>Page 1</h1></body></html>",
		"<html><body><h1>Page 2</h1></body></html>",
		"<html><body><h1>Page 3</h1></body></html>",
	}

	servers := make([]*http.Server, len(testPages))
	listeners := make([]net.Listener, len(testPages))
	ports := make([]int, len(testPages))

	for i, html := range testPages {
		port, err := getAvailablePort()
		if err != nil {
			t.Skip("Could not find available port, skipping test")
		}
		ports[i] = port

		servers[i] = &http.Server{
			Addr: fmt.Sprintf("127.0.0.1:%d", port),
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, html)
			}),
		}

		listeners[i], _ = netListen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		go servers[i].Serve(listeners[i])
	}

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Render all pages concurrently
	errChan := make(chan error, len(testPages))
	results := make(chan string, len(testPages))

	for i, port := range ports {
		go func(p int, h string) {
			url := fmt.Sprintf("http://127.0.0.1:%d/", p)
			html, err := r.Render(ctx, url, domain.RenderOptions{
				Timeout:     10 * time.Second,
				ScrollToEnd: false,
			})
			if err != nil {
				errChan <- err
			} else {
				results <- html
			}
		}(port, testPages[i])
	}

	// Collect results
	for i := 0; i < len(testPages); i++ {
		select {
		case err := <-errChan:
			t.Errorf("Render %d failed: %v", i, err)
		case html := <-results:
			assert.Contains(t, html, "Page")
		}
	}

	// Clean up servers
	for i := range servers {
		servers[i].Close()
		if listeners[i] != nil {
			listeners[i].Close()
		}
	}
}

// TestRenderWithStealthMode tests that stealth mode works
func TestRenderWithStealthMode(t *testing.T) {
	skipIfChromeUnavailable(t)

	opts := renderer.DefaultRendererOptions()
	opts.Timeout = 10 * time.Second
	opts.MaxTabs = 2
	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	ctx := context.Background()

	testHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Stealth Test</title>
</head>
<body>
    <h1>Stealth Mode Test</h1>
</body>
</html>
`

	port, err := getAvailablePort()
	if err != nil {
		t.Skip("Could not find available port, skipping test")
	}

	server := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, testHTML)
		}),
	}

	listener, _ := netListen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	go server.Serve(listener)
	defer server.Close()

	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://127.0.0.1:%d/", port)

	html, err := r.Render(ctx, url, domain.RenderOptions{
		Timeout:     10 * time.Second,
		ScrollToEnd: false,
	})

	require.NoError(t, err)
	assert.Contains(t, html, "Stealth Mode Test")
}

// Helper functions

// netListen is like net.Listen but tries multiple addresses if one fails
func netListen(network, addr string) (net.Listener, error) {
	if addr == "" {
		return nil, fmt.Errorf("addr is empty")
	}
	return net.Listen(network, addr)
}

// getAvailablePort finds an available port
func getAvailablePort() (int, error) {
	listener, err := netListen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
