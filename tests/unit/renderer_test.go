package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestDefaultRendererOptions tests the default renderer options
func TestDefaultRendererOptions(t *testing.T) {
	opts := renderer.DefaultRendererOptions()

	assert.Equal(t, 60*time.Second, opts.Timeout)
	assert.Equal(t, 5, opts.MaxTabs)
	assert.True(t, opts.Stealth)
	assert.True(t, opts.Headless)
	assert.Empty(t, opts.BrowserPath)
}

// TestDefaultRenderOptions tests the default render options
func TestDefaultRenderOptions(t *testing.T) {
	opts := renderer.DefaultRenderOptions()

	assert.Equal(t, 60*time.Second, opts.Timeout)
	assert.Equal(t, 2*time.Second, opts.WaitStable)
	assert.True(t, opts.ScrollToEnd)
}

// TestDetectFramework tests framework detection
func TestDetectFramework(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "React",
			html:     `<div id="root"></div>`,
			expected: "React",
		},
		{
			name:     "Vue - note: id=app also matches React, so we use more specific patterns",
			html:     `<div id="app" v-cloak></div>`,
			expected: "Vue",
		},
		{
			name:     "Next.js",
			html:     `<div id="__next"></div>`,
			expected: "Next.js",
		},
		{
			name:     "Nuxt",
			html:     `window.__NUXT__={}`,
			expected: "Nuxt",
		},
		{
			name:     "Angular",
			html:     `<app-root></app-root>`,
			expected: "Angular",
		},
		{
			name:     "Svelte",
			html:     `__svelte`,
			expected: "Svelte",
		},
		{
			name:     "Unknown",
			html:     `<html><body>Generic HTML</body></html>`,
			expected: "Unknown",
		},
		{
			name:     "React DevTools",
			html:     `__REACT_DEVTOOLS_GLOBAL_HOOK__`,
			expected: "React",
		},
		{
			name:     "Vue v-cloak",
			html:     `v-cloak`,
			expected: "Vue",
		},
		{
			name:     "Next.js Data",
			html:     `__NEXT_DATA__`,
			expected: "Next.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.DetectFramework(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHasDynamicContent tests detection of dynamic content indicators
func TestHasDynamicContent(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "Loading text",
			html:     `<p>Loading...</p>`,
			expected: true,
		},
		{
			name:     "Loading with ellipsis",
			html:     `<p>Loading…</p>`,
			expected: true,
		},
		{
			name:     "Please wait",
			html:     `<div>Please wait</div>`,
			expected: true,
		},
		{
			name:     "Spinner",
			html:     `<div class="spinner"></div>`,
			expected: true,
		},
		{
			name:     "Skeleton loader",
			html:     `<div class="skeleton"></div>`,
			expected: true,
		},
		{
			name:     "Lazy load",
			html:     `<img data-lazyload="true">`,
			expected: true,
		},
		{
			name:     "lazyload attribute",
			html:     `<div class="lazyload"></div>`,
			expected: true,
		},
		{
			name:     "Infinite scroll",
			html:     `<div class="infinite-scroll"></div>`,
			expected: true,
		},
		{
			name:     "Static content",
			html:     `<p>Welcome to our site</p>`,
			expected: false,
		},
		{
			name:     "Normal page",
			html:     `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.HasDynamicContent(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNeedsJSRendering tests the heuristic for detecting SPA pages
func TestNeedsJSRendering(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected bool
	}{
		{
			name:     "React root",
			html:     `<div id="root"></div>`,
			expected: true,
		},
		{
			name:     "Vue app",
			html:     `<div id="app" v-cloak></div>`,
			expected: true,
		},
		{
			name:     "Next.js",
			html:     `<div id="__next"></div>`,
			expected: true,
		},
		{
			name:     "Nuxt",
			html:     `<div id="__nuxt"></div>`,
			expected: true,
		},
		{
			name:     "Angular",
			html:     `<app-root></app-root>`,
			expected: true,
		},
		{
			name:     "Svelte",
			html:     `__svelte`,
			expected: true,
		},
		{
			name:     "Static content with sufficient text",
			html:     `<html><body><h1>Title</h1><p>Content here with substantial text</p><p>More content</p></body></html>`,
			expected: false,
		},
		{
			name:     "Minimal content with many scripts",
			html:     `<html><body><script src="1.js"></script><script src="2.js"></script><script src="3.js"></script><script src="4.js"></script></body></html>`,
			expected: true,
		},
		{
			name:     "Empty body - no scripts, so doesn't need rendering",
			html:     `<html><body></body></html>`,
			expected: false,
		},
		{
			name:     "Server-rendered React with content",
			html:     `<div id="root"><h1>Server Rendered</h1><p>Substantial content here</p></div>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.NeedsJSRendering(tt.html)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStealthOptions tests stealth options
func TestStealthOptions(t *testing.T) {
	opts := renderer.DefaultStealthOptions()

	assert.True(t, opts.HideWebdriver)
	assert.True(t, opts.EmulatePlugins)
	assert.False(t, opts.RandomizeViewport)
	assert.True(t, opts.DisableAutomationFlags)
}

// TestIsAvailable tests the browser availability check
func TestIsAvailable(t *testing.T) {
	// This test will vary depending on system
	// Just verify the function exists and returns a boolean
	available := renderer.IsAvailable()
	// Result depends on whether Chrome is installed
	// We don't assert on the value, just that it doesn't panic
	_ = available
}

// TestGetBrowserPath tests browser path detection
func TestGetBrowserPath(t *testing.T) {
	path, exists := renderer.GetBrowserPath()

	// Just verify the function works
	// The actual values depend on the system
	_ = path
	_ = exists
}

// TestRendererInterfaceCompliance tests that a mock implements domain.Renderer
func TestRendererInterfaceCompliance(t *testing.T) {
	mock := &MockBrowserRenderer{}

	// This will fail at compile time if MockRenderer doesn't implement domain.Renderer
	var _ domain.Renderer = mock

	// Test basic methods work
	assert.Equal(t, "mock-browser", mock.Name())
	assert.True(t, mock.CanHandle("https://example.com"))
}

// MockBrowserRenderer is a mock that implements domain.Renderer
type MockBrowserRenderer struct {
	html   string
	err    error
	called bool
}

func (m *MockBrowserRenderer) Name() string {
	return "mock-browser"
}

func (m *MockBrowserRenderer) CanHandle(url string) bool {
	return true
}

func (m *MockBrowserRenderer) Execute(ctx context.Context, url string, opts domain.StrategyOptions) error {
	m.called = true
	return m.err
}

func (m *MockBrowserRenderer) Render(ctx context.Context, url string, opts domain.RenderOptions) (string, error) {
	return m.html, m.err
}

func (m *MockBrowserRenderer) Close() error {
	return nil
}

// TestDetectFrameworkReactPatterns tests React-specific detection patterns
func TestDetectFrameworkReactPatterns(t *testing.T) {
	patterns := []string{
		`<div id="root"></div>`,
		`<div id="root"/>`,
		`<div id="app"></div>`,
		`<div id="app"/>`,
		`data-reactroot`,
		`__REACT_DEVTOOLS_GLOBAL_HOOK__`,
	}

	for _, pattern := range patterns {
		t.Run("React pattern: "+pattern, func(t *testing.T) {
			result := renderer.DetectFramework(pattern)
			assert.Equal(t, "React", result)
		})
	}
}

// TestDetectFrameworkVuePatterns tests Vue-specific detection patterns
func TestDetectFrameworkVuePatterns(t *testing.T) {
	patterns := []struct {
		pattern  string
		expected string
	}{
		{`<div id="app" v-cloak></div>`, "Vue"}, // More specific than id="app"
		{`__VUE__`, "Vue"},
		{`v-cloak`, "Vue"},
		{`Vue.createApp`, "Vue"},
	}

	for _, tt := range patterns {
		t.Run("Vue pattern: "+tt.pattern, func(t *testing.T) {
			result := renderer.DetectFramework(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetectFrameworkNextPatterns tests Next.js-specific detection patterns
func TestDetectFrameworkNextPatterns(t *testing.T) {
	patterns := []string{
		`<div id="__next"></div>`,
		`<div id="__next"/>`,
		`__NEXT_DATA__`,
		`_next/static`,
	}

	for _, pattern := range patterns {
		t.Run("Next.js pattern: "+pattern, func(t *testing.T) {
			result := renderer.DetectFramework(pattern)
			assert.Equal(t, "Next.js", result)
		})
	}
}

// TestDetectFrameworkPriority tests that framework detection respects priority
func TestDetectFrameworkPriority(t *testing.T) {
	// Next.js should be detected before React
	html := `<div id="__next"></div><div id="root"></div>`
	result := renderer.DetectFramework(html)
	assert.Equal(t, "Next.js", result)

	// Nuxt should be detected before Vue
	html = `window.__NUXT__={}<div id="app"></div>`
	result = renderer.DetectFramework(html)
	assert.Equal(t, "Nuxt", result)
}

// BenchmarkDetectFramework benchmarks framework detection
func BenchmarkDetectFramework(b *testing.B) {
	html := `<html><body><div id="root">App</div></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.DetectFramework(html)
	}
}

// BenchmarkHasDynamicContent benchmarks dynamic content detection
func BenchmarkHasDynamicContent(b *testing.B) {
	html := `<html><body><div class="loading">Loading...</div></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.HasDynamicContent(html)
	}
}

// BenchmarkNeedsJSRendering benchmarks SPA detection
func BenchmarkNeedsJSRendering(b *testing.B) {
	html := `<html><body><div id="root"></div><script src="/app.js"></script></body></html>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.NeedsJSRendering(html)
	}
}
