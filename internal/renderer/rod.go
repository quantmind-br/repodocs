package renderer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Renderer provides JavaScript rendering using headless Chrome
type Renderer struct {
	browser  *rod.Browser
	pool     *TabPool
	timeout  time.Duration
	stealth  bool
	headless bool
}

// RendererOptions contains options for creating a Renderer
type RendererOptions struct {
	Timeout     time.Duration
	MaxTabs     int
	Stealth     bool
	Headless    bool
	BrowserPath string
	NoSandbox   bool // Required for running in CI/Docker environments
}

// DefaultRendererOptions returns default renderer options
func DefaultRendererOptions() RendererOptions {
	return RendererOptions{
		Timeout:     60 * time.Second,
		MaxTabs:     5,
		Stealth:     true,
		Headless:    true,
		BrowserPath: "",
		NoSandbox:   isCI(), // Auto-detect CI environment
	}
}

// isCI returns true if running in a CI environment
func isCI() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != ""
}

// NewRenderer creates a new headless browser renderer
func NewRenderer(opts RendererOptions) (*Renderer, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.MaxTabs <= 0 {
		opts.MaxTabs = 5
	}

	// Create launcher
	l := launcher.New()

	if opts.BrowserPath != "" {
		l = l.Bin(opts.BrowserPath)
	}

	if opts.Headless {
		l = l.Headless(true)
	}

	// Additional flags for stealth
	if opts.Stealth {
		l = l.Set("disable-blink-features", "AutomationControlled")
	}

	// NoSandbox is required for running in CI/Docker environments
	if opts.NoSandbox {
		l = l.NoSandbox(true)
	}

	// Launch browser
	controlURL, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// Create tab pool
	pool, err := NewTabPool(browser, opts.MaxTabs)
	if err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to create tab pool: %w", err)
	}

	return &Renderer{
		browser:  browser,
		pool:     pool,
		timeout:  opts.Timeout,
		stealth:  opts.Stealth,
		headless: opts.Headless,
	}, nil
}

// Render fetches and renders a page with JavaScript
func (r *Renderer) Render(ctx context.Context, url string, opts domain.RenderOptions) (string, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = r.timeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Acquire a page from the pool
	page, err := r.pool.Acquire(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to acquire page: %w", err)
	}
	defer r.pool.Release(page)

	// Apply context to page so all operations respect the timeout
	page = page.Context(ctx)

	// Apply stealth mode
	if r.stealth {
		if err := ApplyStealthMode(page); err != nil {
			return "", fmt.Errorf("failed to apply stealth mode: %w", err)
		}
	}

	// Set cookies if provided
	if len(opts.Cookies) > 0 {
		if err := r.setCookies(page, url, opts.Cookies); err != nil {
			return "", fmt.Errorf("failed to set cookies: %w", err)
		}
	}

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return "", domain.NewFetchError(url, 0, fmt.Errorf("navigation failed: %w", err))
	}

	// Wait for page to load
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("failed waiting for load: %w", err)
	}

	// Wait for specific selector if provided
	if opts.WaitFor != "" {
		if err := page.Timeout(opts.Timeout).MustElement(opts.WaitFor).WaitVisible(); err != nil {
			// Don't fail, just continue
		}
	}

	// Wait for network to be idle
	if opts.WaitStable > 0 {
		if err := page.WaitRequestIdle(opts.WaitStable, nil, nil, nil); err != nil {
			// Don't fail, just continue
		}
	}

	// Scroll to bottom to load lazy content
	if opts.ScrollToEnd {
		if err := r.scrollToEnd(page); err != nil {
			// Don't fail, just continue
		}
	}

	// Get rendered HTML
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("failed to get HTML: %w", err)
	}

	return html, nil
}

// setCookies sets cookies on a page
func (r *Renderer) setCookies(page *rod.Page, pageURL string, cookies []*http.Cookie) error {
	// Parse URL to extract domain if cookie domain is empty
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL for cookies: %w", err)
	}

	for _, cookie := range cookies {
		// Use cookie domain if set, otherwise extract from URL
		domain := cookie.Domain
		if domain == "" {
			domain = parsedURL.Hostname()
		}

		// Use cookie path if set, otherwise default to "/"
		path := cookie.Path
		if path == "" {
			path = "/"
		}

		err := page.SetCookies([]*proto.NetworkCookieParam{
			{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   domain,
				Path:     path,
				Secure:   cookie.Secure,
				HTTPOnly: cookie.HttpOnly,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// scrollToEnd scrolls to the bottom of the page to trigger lazy loading
func (r *Renderer) scrollToEnd(page *rod.Page) error {
	// Get initial scroll height
	result, err := page.Eval(`() => document.body.scrollHeight`)
	if err != nil {
		return err
	}
	lastHeight := result.Value.Int()

	for i := 0; i < 10; i++ { // Max 10 scroll iterations
		// Scroll to bottom
		_, err := page.Eval(`() => window.scrollTo(0, document.body.scrollHeight)`)
		if err != nil {
			return err
		}

		// Wait for content to load
		time.Sleep(500 * time.Millisecond)

		// Check new scroll height
		result, err := page.Eval(`() => document.body.scrollHeight`)
		if err != nil {
			return err
		}
		newHeight := result.Value.Int()

		// If height hasn't changed, we've reached the bottom
		if newHeight == lastHeight {
			break
		}
		lastHeight = newHeight
	}

	// Scroll back to top
	_, _ = page.Eval(`() => window.scrollTo(0, 0)`)

	return nil
}

// DefaultRenderOptions returns default render options
func DefaultRenderOptions() domain.RenderOptions {
	return domain.RenderOptions{
		Timeout:     60 * time.Second,
		WaitStable:  2 * time.Second,
		ScrollToEnd: true,
	}
}

// Close releases browser resources
func (r *Renderer) Close() error {
	if r.pool != nil {
		r.pool.Close()
		r.pool = nil
	}
	if r.browser != nil {
		browser := r.browser
		r.browser = nil
		return browser.Close()
	}
	return nil
}

// IsAvailable checks if the browser is available
func IsAvailable() bool {
	path, exists := launcher.LookPath()
	return exists && path != ""
}

// GetBrowserPath returns the detected browser path
func GetBrowserPath() (string, bool) {
	return launcher.LookPath()
}

// GetTabPool returns the tab pool for testing purposes
func (r *Renderer) GetTabPool() (*TabPool, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("pool not initialized")
	}
	return r.pool, nil
}
