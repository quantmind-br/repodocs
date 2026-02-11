package strategies

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCrawlContext_Initialization(t *testing.T) {
	ctx := context.Background()
	opts := DefaultOptions()

	cctx := newCrawlContext(ctx, "https://example.com", opts)

	require.NotNil(t, cctx)
	assert.NotNil(t, cctx.visited)
	assert.NotNil(t, cctx.mu)
	assert.NotNil(t, cctx.barMu)
	assert.NotNil(t, cctx.bar)
	assert.Equal(t, 0, *cctx.processedCount)
	assert.Equal(t, "https://example.com", cctx.baseURL)
	assert.Equal(t, ctx, cctx.ctx)
}

func TestNewCrawlContext_ExcludePatterns(t *testing.T) {
	ctx := context.Background()
	opts := Options{
		Exclude: []string{"/admin.*", "/api/v[0-9]+"},
	}

	cctx := newCrawlContext(ctx, "https://example.com", opts)

	require.Len(t, cctx.excludeRegexps, 2)
	assert.True(t, cctx.excludeRegexps[0].MatchString("/admin/settings"))
	assert.True(t, cctx.excludeRegexps[1].MatchString("/api/v2/users"))
	assert.False(t, cctx.excludeRegexps[0].MatchString("/public/page"))
}

func TestNewCrawlContext_InvalidRegexIgnored(t *testing.T) {
	ctx := context.Background()
	opts := Options{
		Exclude: []string{"[invalid", "valid.*"},
	}

	cctx := newCrawlContext(ctx, "https://example.com", opts)

	assert.Len(t, cctx.excludeRegexps, 1)
	assert.True(t, cctx.excludeRegexps[0].MatchString("valid_pattern"))
}

func TestCrawlContext_ConcurrentVisited(t *testing.T) {
	ctx := context.Background()
	cctx := newCrawlContext(ctx, "https://example.com", DefaultOptions())

	var wg sync.WaitGroup
	urls := make([]string, 100)
	for i := 0; i < 100; i++ {
		urls[i] = fmt.Sprintf("https://example.com/page%d", i)
	}

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			cctx.visited.Store(u, true)
		}(url)
	}
	wg.Wait()

	count := 0
	cctx.visited.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 100, count)
}

func TestCrawlContext_ConcurrentProcessedCount(t *testing.T) {
	ctx := context.Background()
	cctx := newCrawlContext(ctx, "https://example.com", DefaultOptions())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cctx.mu.Lock()
			*cctx.processedCount++
			cctx.mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Equal(t, 100, *cctx.processedCount)
}

func TestNewCrawlContext_EmptyOptions(t *testing.T) {
	ctx := context.Background()
	opts := Options{}

	cctx := newCrawlContext(ctx, "https://example.com", opts)

	require.NotNil(t, cctx)
	assert.Empty(t, cctx.excludeRegexps)
	assert.Equal(t, 0, *cctx.processedCount)
}

func TestNewCrawlContext_CollectorNilByDefault(t *testing.T) {
	ctx := context.Background()
	cctx := newCrawlContext(ctx, "https://example.com", DefaultOptions())

	require.NotNil(t, cctx)
	assert.Nil(t, cctx.collector)
}

func TestNewCrawlContext_OptsPreserved(t *testing.T) {
	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit:  10,
			Force:  true,
			DryRun: true,
		},
		MaxDepth:    5,
		Concurrency: 3,
		FilterURL:   "https://example.com/docs",
	}

	cctx := newCrawlContext(ctx, "https://example.com", opts)

	assert.Equal(t, 10, cctx.opts.Limit)
	assert.Equal(t, 5, cctx.opts.MaxDepth)
	assert.Equal(t, 3, cctx.opts.Concurrency)
	assert.Equal(t, "https://example.com/docs", cctx.opts.FilterURL)
	assert.True(t, cctx.opts.Force)
	assert.True(t, cctx.opts.DryRun)
}

func TestCrawlerStrategy_ShouldProcessURL(t *testing.T) {
	strategy := &CrawlerStrategy{}

	tests := []struct {
		name     string
		link     string
		baseURL  string
		opts     Options
		expected bool
	}{
		{
			name:     "empty link returns false",
			link:     "",
			baseURL:  "https://example.com",
			opts:     DefaultOptions(),
			expected: false,
		},
		{
			name:     "different domain returns false",
			link:     "https://other.com/page",
			baseURL:  "https://example.com",
			opts:     DefaultOptions(),
			expected: false,
		},
		{
			name:     "same domain returns true",
			link:     "https://example.com/docs/page",
			baseURL:  "https://example.com",
			opts:     DefaultOptions(),
			expected: true,
		},
		{
			name:     "excluded pattern returns false",
			link:     "https://example.com/admin/settings",
			baseURL:  "https://example.com",
			opts:     Options{Exclude: []string{"/admin"}},
			expected: false,
		},
		{
			name:     "filter URL mismatch returns false",
			link:     "https://example.com/other/page",
			baseURL:  "https://example.com",
			opts:     Options{FilterURL: "https://example.com/docs"},
			expected: false,
		},
		{
			name:     "filter URL match returns true",
			link:     "https://example.com/docs/page",
			baseURL:  "https://example.com",
			opts:     Options{FilterURL: "https://example.com/docs"},
			expected: true,
		},
		{
			name:     "subdomain treated as different domain",
			link:     "https://api.example.com/v1",
			baseURL:  "https://example.com",
			opts:     DefaultOptions(),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cctx := newCrawlContext(ctx, tc.baseURL, tc.opts)
			result := strategy.shouldProcessURL(tc.link, tc.baseURL, cctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCrawlerStrategy_ShouldProcessURL_Limit(t *testing.T) {
	strategy := &CrawlerStrategy{}
	ctx := context.Background()
	opts := Options{
		CommonOptions: domain.CommonOptions{
			Limit: 2,
		},
	}
	cctx := newCrawlContext(ctx, "https://example.com", opts)

	assert.True(t, strategy.shouldProcessURL("https://example.com/page1", "https://example.com", cctx))
	assert.True(t, strategy.shouldProcessURL("https://example.com/page2", "https://example.com", cctx))

	cctx.mu.Lock()
	*cctx.processedCount = 2
	cctx.mu.Unlock()

	assert.False(t, strategy.shouldProcessURL("https://example.com/page3", "https://example.com", cctx))
}

func TestCrawlerStrategy_ShouldProcessURL_AlreadyVisited(t *testing.T) {
	strategy := &CrawlerStrategy{}
	ctx := context.Background()
	cctx := newCrawlContext(ctx, "https://example.com", DefaultOptions())

	url := "https://example.com/page1"

	assert.True(t, strategy.shouldProcessURL(url, "https://example.com", cctx))
	assert.False(t, strategy.shouldProcessURL(url, "https://example.com", cctx))
}

func TestCrawlerStrategy_ShouldProcessURL_MultipleExcludePatterns(t *testing.T) {
	strategy := &CrawlerStrategy{}
	ctx := context.Background()
	opts := Options{
		Exclude: []string{"/admin", "/api/.*", ".*\\.pdf$"},
	}
	cctx := newCrawlContext(ctx, "https://example.com", opts)

	assert.False(t, strategy.shouldProcessURL("https://example.com/admin/page", "https://example.com", cctx))
	assert.False(t, strategy.shouldProcessURL("https://example.com/api/v1/users", "https://example.com", cctx))
	assert.False(t, strategy.shouldProcessURL("https://example.com/docs/file.pdf", "https://example.com", cctx))
	assert.True(t, strategy.shouldProcessURL("https://example.com/docs/page", "https://example.com", cctx))
}

func TestCrawlerStrategy_ShouldProcessURL_ConcurrentAccess(t *testing.T) {
	strategy := &CrawlerStrategy{}
	ctx := context.Background()
	cctx := newCrawlContext(ctx, "https://example.com", DefaultOptions())

	var wg sync.WaitGroup
	results := make([]bool, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			url := fmt.Sprintf("https://example.com/page%d", idx)
			results[idx] = strategy.shouldProcessURL(url, "https://example.com", cctx)
		}(i)
	}
	wg.Wait()

	trueCount := 0
	for _, r := range results {
		if r {
			trueCount++
		}
	}
	assert.Equal(t, 100, trueCount)
}
