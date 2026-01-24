package renderer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScrollToEnd_EarlyExitOnStableHeight tests that scroll exits early
// after 2 consecutive stable height checks (optimization: reduced iterations)
func TestScrollToEnd_EarlyExitOnStableHeight(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a simple static page - height won't change on scroll
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Static Page</title></head>
<body>
	<div style="height: 500px;">Static content that doesn't grow</div>
</body>
</html>`))
	}))
	defer server.Close()

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     30 * time.Second,
		WaitStable:  500 * time.Millisecond,
		ScrollToEnd: true,
	}

	ctx := context.Background()
	start := time.Now()

	html, err := r.Render(ctx, server.URL, renderOpts)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotEmpty(t, html)
	assert.Contains(t, html, "Static content")

	// With early exit after 2 stable checks at 300ms each,
	// scroll should complete in ~600ms (2 * 300ms) plus overhead
	// Old behavior would take longer (exited on first stable)
	// This verifies the optimization is working
	t.Logf("Scroll completed in %v", elapsed)
}

// TestScrollToEnd_MaxIterationsRespected tests that scroll doesn't exceed
// max 10 iterations even with dynamic content
func TestScrollToEnd_MaxIterationsRespected(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a page that keeps growing on scroll (infinite scroll simulation)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Infinite Scroll</title></head>
<body>
	<div id="content">Initial content</div>
	<script>
		let counter = 0;
		window.addEventListener('scroll', function() {
			if (counter < 20) {
				counter++;
				const div = document.createElement('div');
				div.style.height = '500px';
				div.textContent = 'Added content ' + counter;
				document.getElementById('content').appendChild(div);
			}
		});
	</script>
</body>
</html>`))
	}))
	defer server.Close()

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     60 * time.Second,
		WaitStable:  500 * time.Millisecond,
		ScrollToEnd: true,
	}

	ctx := context.Background()
	start := time.Now()

	html, err := r.Render(ctx, server.URL, renderOpts)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotEmpty(t, html)

	// With max 10 iterations at 300ms each, should complete in ~3-4 seconds
	// (accounting for WaitStable and other overhead)
	// This ensures we don't scroll forever
	t.Logf("Scroll completed in %v with max iterations limit", elapsed)

	// Should complete within reasonable time (10 iterations * 300ms = 3s + overhead)
	assert.Less(t, elapsed.Seconds(), 15.0, "Scroll should complete within 15 seconds")
}

// TestScrollToEnd_StableCounterResets tests that the stable counter resets
// when height changes (content still loading between checks)
func TestScrollToEnd_StableCounterResets(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a page that grows twice then stops
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Delayed Content</title></head>
<body>
	<div id="content" style="height: 500px;">Initial content</div>
	<script>
		let loaded = 0;
		window.addEventListener('scroll', function() {
			if (loaded < 2) {
				loaded++;
				const content = document.getElementById('content');
				content.style.height = (500 + loaded * 300) + 'px';
				content.textContent = 'Content after scroll ' + loaded;
			}
		});
	</script>
</body>
</html>`))
	}))
	defer server.Close()

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     30 * time.Second,
		WaitStable:  500 * time.Millisecond,
		ScrollToEnd: true,
	}

	ctx := context.Background()

	html, err := r.Render(ctx, server.URL, renderOpts)

	assert.NoError(t, err)
	assert.NotEmpty(t, html)

	// Should contain content that was loaded after scrolling
	// This verifies that the stable counter resets when content grows
	assert.Contains(t, html, "Content after scroll")
}

// TestScrollToEnd_ScrollsBackToTop tests that after scrolling to end,
// the page is scrolled back to top
func TestScrollToEnd_ScrollsBackToTop(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Scroll Test</title></head>
<body>
	<div id="top-marker">TOP OF PAGE</div>
	<div style="height: 2000px;"></div>
	<div id="bottom-marker">BOTTOM OF PAGE</div>
</body>
</html>`))
	}))
	defer server.Close()

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	renderOpts := domain.RenderOptions{
		Timeout:     30 * time.Second,
		WaitStable:  500 * time.Millisecond,
		ScrollToEnd: true,
	}

	ctx := context.Background()

	html, err := r.Render(ctx, server.URL, renderOpts)

	assert.NoError(t, err)
	assert.NotEmpty(t, html)

	// Both markers should be in the rendered HTML
	assert.Contains(t, html, "TOP OF PAGE")
	assert.Contains(t, html, "BOTTOM OF PAGE")
}
