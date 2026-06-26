package recovery

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingFetcher records how many times Get is invoked per URL and returns a
// canned response, so tests can assert the cache collapses duplicate fetches.
type countingFetcher struct {
	mu     sync.Mutex
	counts map[string]int
	err    error
}

func (f *countingFetcher) Get(_ context.Context, url string) (*domain.Response, error) {
	f.mu.Lock()
	f.counts[url]++
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return &domain.Response{StatusCode: 200, Body: []byte("ok"), URL: url}, nil
}
func (f *countingFetcher) GetWithHeaders(ctx context.Context, url string, _ map[string]string) (*domain.Response, error) {
	return f.Get(ctx, url)
}
func (f *countingFetcher) GetCookies(string) []*http.Cookie { return nil }
func (f *countingFetcher) Transport() http.RoundTripper     { return nil }
func (f *countingFetcher) Close() error                     { return nil }

func (f *countingFetcher) count(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.counts[url]
}

func TestFetchCache_DeduplicatesSequential(t *testing.T) {
	fetcher := &countingFetcher{counts: map[string]int{}}
	cache := newFetchCache(fetcher)

	for range 3 {
		resp, err := cache.get(context.Background(), "https://x.dev/a")
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	}
	assert.Equal(t, 1, fetcher.count("https://x.dev/a"), "same URL should be fetched once")
}

func TestFetchCache_DeduplicatesConcurrent(t *testing.T) {
	fetcher := &countingFetcher{counts: map[string]int{}}
	cache := newFetchCache(fetcher)

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = cache.get(context.Background(), "https://x.dev/shared")
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, fetcher.count("https://x.dev/shared"),
		"concurrent fetches of the same URL should collapse to one")
}

func TestFetchCache_PropagatesError(t *testing.T) {
	wantErr := errors.New("boom")
	fetcher := &countingFetcher{counts: map[string]int{}, err: wantErr}
	cache := newFetchCache(fetcher)

	_, err := cache.get(context.Background(), "https://x.dev/err")
	assert.ErrorIs(t, err, wantErr)
}
