package recovery

import (
	"context"
	"sync"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// fetchCache memoizes Fetcher.Get results for the lifetime of a single probe
// run so that probes sharing a target URL fetch it at most once. It also
// collapses concurrent in-flight fetches for the same URL into one request,
// behaving like a minimal single-flight cache.
type fetchCache struct {
	fetcher domain.Fetcher

	mu    sync.Mutex
	calls map[string]*fetchCall
}

type fetchCall struct {
	done chan struct{}
	resp *domain.Response
	err  error
}

// newFetchCache creates an empty cache backed by the given fetcher.
func newFetchCache(fetcher domain.Fetcher) *fetchCache {
	return &fetchCache{
		fetcher: fetcher,
		calls:   make(map[string]*fetchCall),
	}
}

// get returns the (possibly cached) response for url. Concurrent callers for the
// same url share a single Fetcher.Get; later callers for a completed url receive
// the memoized result. A caller whose own context is cancelled while waiting on
// an in-flight fetch returns promptly with that context's error.
func (c *fetchCache) get(ctx context.Context, url string) (*domain.Response, error) {
	c.mu.Lock()
	if call, ok := c.calls[url]; ok {
		c.mu.Unlock()
		select {
		case <-call.done:
			return call.resp, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &fetchCall{done: make(chan struct{})}
	c.calls[url] = call
	c.mu.Unlock()

	// Close on return rather than only after a clean Get, so a panicking or
	// runtime-exiting fetcher can never leave waiters blocked on call.done.
	defer close(call.done)
	call.resp, call.err = c.fetcher.Get(ctx, url)
	return call.resp, call.err
}
