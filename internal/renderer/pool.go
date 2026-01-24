package renderer

import (
	"context"
	"sync"

	"github.com/go-rod/rod"
)

// TabPool manages a pool of browser tabs for concurrent rendering
type TabPool struct {
	browser    *rod.Browser
	maxTabs    int
	activeTabs chan *rod.Page
	mu         sync.Mutex
	closed     bool
	created    int
}

// NewTabPool creates a new tab pool with lazy tab initialization
func NewTabPool(browser *rod.Browser, maxTabs int) (*TabPool, error) {
	if maxTabs <= 0 {
		maxTabs = 5
	}

	pool := &TabPool{
		browser:    browser,
		maxTabs:    maxTabs,
		activeTabs: make(chan *rod.Page, maxTabs),
		created:    0,
	}

	return pool, nil
}

// Acquire gets a page from the pool, blocking if none available
func (p *TabPool) Acquire(ctx context.Context) (*rod.Page, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	select {
	case page := <-p.activeTabs:
		p.mu.Unlock()
		return page, nil
	default:
		if p.created < p.maxTabs {
			page, err := StealthPage(p.browser)
			if err != nil {
				p.mu.Unlock()
				return nil, err
			}
			p.created++
			p.mu.Unlock()
			return page, nil
		}
	}
	p.mu.Unlock()

	select {
	case page := <-p.activeTabs:
		return page, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns a page to the pool after cleaning up
func (p *TabPool) Release(page *rod.Page) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		page.Close()
		return
	}
	p.mu.Unlock()

	// Clean up the page before returning to pool
	_ = page.Navigate("about:blank")

	select {
	case p.activeTabs <- page:
		// Successfully returned to pool
	default:
		// Pool is full (shouldn't happen normally)
		page.Close()
	}
}

// Close closes all tabs and the pool
func (p *TabPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.activeTabs)

	// Close remaining pages
	for page := range p.activeTabs {
		page.Close()
	}

	return nil
}

// Size returns the current number of available tabs
func (p *TabPool) Size() int {
	return len(p.activeTabs)
}

// MaxSize returns the maximum pool size
func (p *TabPool) MaxSize() int {
	return p.maxTabs
}

func (p *TabPool) Created() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.created
}

// ErrPoolClosed is returned when trying to acquire from a closed pool
var ErrPoolClosed = &poolError{message: "pool is closed"}

type poolError struct {
	message string
}

func (e *poolError) Error() string {
	return e.message
}
