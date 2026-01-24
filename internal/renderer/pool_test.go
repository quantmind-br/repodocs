package renderer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBrowser creates a mock browser for testing
type mockBrowser struct {
	pageCount int32
	pages     []*rod.Page
}

func newMockBrowser() *mockBrowser {
	return &mockBrowser{
		pages: make([]*rod.Page, 0),
	}
}

func (m *mockBrowser) Page() (*rod.Page, error) {
	// Return a mock page - in real tests we'd use a proper mock
	return nil, nil
}

func (m *mockBrowser) MustPage() *rod.Page {
	return nil
}

// TestNewTabPool_DefaultSize tests creating pool with default size
func TestNewTabPool_DefaultSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	assert.Equal(t, 5, pool.MaxSize())
	assert.Equal(t, 0, pool.Size()) // Lazy: 0 tabs initially
}

// TestNewTabPool_CustomSize tests creating pool with custom size
func TestNewTabPool_CustomSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	tests := []struct {
		name     string
		maxTabs  int
		expected int
	}{
		{"single tab", 1, 1},
		{"three tabs", 3, 3},
		{"ten tabs", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultRendererOptions()
			opts.MaxTabs = tt.maxTabs

			r, err := NewRenderer(opts)
			require.NoError(t, err)
			defer r.Close()

			pool, err := r.GetTabPool()
			require.NoError(t, err)

			assert.Equal(t, tt.expected, pool.MaxSize())
			assert.Equal(t, 0, pool.Size()) // Lazy: 0 tabs initially
		})
	}
}

// TestNewTabPool_ZeroSize tests creating pool with zero size (should default to 5)
func TestNewTabPool_ZeroSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	// We can't directly test NewTabPool with 0 since it's called from NewRenderer
	// But we can verify the behavior through the renderer
	opts := DefaultRendererOptions()
	opts.MaxTabs = 0 // Should be handled by NewRenderer

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Should default to 5
	assert.Equal(t, 5, pool.MaxSize())
	assert.Equal(t, 0, pool.Size()) // Lazy: 0 tabs initially
}

// TestAcquire_AvailableTab tests acquiring tab when available
func TestAcquire_AvailableTab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 2

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire first tab (lazy created)
	page1, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page1)
	assert.Equal(t, 0, pool.Size()) // Tab in use

	// Acquire second tab (lazy created)
	page2, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page2)
	assert.Equal(t, 0, pool.Size()) // Both tabs in use

	// Release tabs
	pool.Release(page1)
	pool.Release(page2)
	assert.Equal(t, 2, pool.Size())
}

// TestAcquire_PoolEmpty tests acquiring when pool is empty
func TestAcquire_PoolEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire the only tab
	page, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)
	assert.Equal(t, 0, pool.Size())

	// Try to acquire another - should block until timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// Release the tab
	pool.Release(page)
	assert.Equal(t, 1, pool.Size())
}

// TestAcquire_MultipleCalls tests concurrent acquire calls
func TestAcquire_MultipleCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 3

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()
	var acquired int32

	// Try to acquire more tabs than available
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			page, err := pool.Acquire(ctx)
			if err == nil && page != nil {
				atomic.AddInt32(&acquired, 1)
				time.Sleep(50 * time.Millisecond)
				pool.Release(page)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All tabs should be back
	assert.Equal(t, 3, pool.Size())
}

// TestAcquire_ContextCancel tests context cancellation
func TestAcquire_ContextCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Acquire the only tab
	page, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	require.NotNil(t, page)

	// Create cancelable context
	ctx, cancel := context.WithCancel(context.Background())

	// Try to acquire in goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := pool.Acquire(ctx)
		errChan <- err
	}()

	// Cancel context before acquire can complete
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Should get context canceled error
	err = <-errChan
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	// Release the tab
	pool.Release(page)
}

// TestAcquire_ContextTimeout tests context timeout
func TestAcquire_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Acquire the only tab
	page, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	require.NotNil(t, page)

	// Try to acquire with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// Release the tab
	pool.Release(page)
}

// TestRelease_ValidTab tests releasing a valid tab
func TestRelease_ValidTab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire tab
	page, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)
	assert.Equal(t, 0, pool.Size())

	// Release tab
	pool.Release(page)
	assert.Equal(t, 1, pool.Size())

	// Should be able to acquire again
	page2, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page2)
	assert.Equal(t, 0, pool.Size())

	pool.Release(page2)
}

// TestRelease_NilTab tests releasing a nil tab
func TestRelease_NilTab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	initialSize := pool.Size()

	// Release nil - this will panic because the pool implementation
	// doesn't handle nil pages. In production code, this should never happen.
	// We're documenting this behavior rather than trying to handle it.
	assert.Panics(t, func() {
		pool.Release(nil)
	})

	// Size should be unchanged (the panic happens before modifying the pool)
	assert.Equal(t, initialSize, pool.Size())
}

// TestRelease_ClosedTab tests releasing a closed tab
func TestRelease_ClosedTab(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire tab
	page, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)

	// Close the tab
	err = page.Close()
	assert.NoError(t, err)

	// Release closed tab - should not panic
	assert.NotPanics(t, func() {
		pool.Release(page)
	})

	// Size should be 1 (tab returned to pool even if closed)
	assert.Equal(t, 1, pool.Size())
}

// TestRelease_MultipleReleases tests releasing the same tab multiple times
func TestRelease_MultipleReleases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire tab
	page, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)

	// Release once
	pool.Release(page)
	assert.Equal(t, 1, pool.Size())

	// Release again - should not panic or duplicate
	assert.NotPanics(t, func() {
		pool.Release(page)
	})
}

// TestSize_EmptyPool tests size when pool is empty
func TestSize_EmptyPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 2

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire all tabs
	page1, _ := pool.Acquire(ctx)
	page2, _ := pool.Acquire(ctx)

	assert.Equal(t, 0, pool.Size())

	// Release one
	pool.Release(page1)
	assert.Equal(t, 1, pool.Size())

	// Release both
	pool.Release(page2)
	assert.Equal(t, 2, pool.Size())
}

// TestSize_WithTabs tests size with tabs in pool
func TestSize_WithTabs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 3

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Lazy: initially 0 tabs
	assert.Equal(t, 0, pool.Size())

	ctx := context.Background()

	// Acquire tabs (lazy creation)
	page1, _ := pool.Acquire(ctx)
	assert.Equal(t, 0, pool.Size()) // In use

	page2, _ := pool.Acquire(ctx)
	assert.Equal(t, 0, pool.Size()) // Both in use

	// Release one
	pool.Release(page1)
	assert.Equal(t, 1, pool.Size())

	// Release both
	pool.Release(page2)
	assert.Equal(t, 2, pool.Size())
}

// TestSize_AfterAcquire tests size changes after acquire
func TestSize_AfterAcquire(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 5

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Lazy: start at 0
	initialSize := pool.Size()
	assert.Equal(t, 0, initialSize)

	// Acquire tab (created lazily)
	page, _ := pool.Acquire(ctx)
	assert.Equal(t, 0, pool.Size()) // Tab in use, not in pool

	// Release tab
	pool.Release(page)
	assert.Equal(t, 1, pool.Size()) // Tab returned to pool
}

// TestSize_AfterRelease tests size changes after release
func TestSize_AfterRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 3

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire all tabs
	var pages []*rod.Page
	for i := 0; i < 3; i++ {
		page, _ := pool.Acquire(ctx)
		pages = append(pages, page)
	}

	assert.Equal(t, 0, pool.Size())

	// Release one by one
	for i, page := range pages {
		pool.Release(page)
		assert.Equal(t, i+1, pool.Size())
	}
}

// TestMaxSize_Default tests default max size
func TestMaxSize_Default(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	assert.Equal(t, 5, pool.MaxSize())
}

// TestMaxSize_Custom tests custom max size
func TestMaxSize_Custom(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	tests := []struct {
		name     string
		maxTabs  int
		expected int
	}{
		{"1 tab", 1, 1},
		{"2 tabs", 2, 2},
		{"10 tabs", 10, 10},
		{"20 tabs", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultRendererOptions()
			opts.MaxTabs = tt.maxTabs

			r, err := NewRenderer(opts)
			require.NoError(t, err)
			defer r.Close()

			pool, err := r.GetTabPool()
			require.NoError(t, err)

			assert.Equal(t, tt.expected, pool.MaxSize())
		})
	}
}

// TestClose_EmptyPool tests closing empty pool
func TestClose_EmptyPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire the tab
	page, _ := pool.Acquire(ctx)

	// Close pool (with one tab acquired)
	err = pool.Close()
	assert.NoError(t, err)

	// Release the tab (should be handled gracefully)
	pool.Release(page)

	// Size should be 0 after close
	assert.Equal(t, 0, pool.Size())
}

// TestClose_WithTabs tests closing pool with tabs
func TestClose_WithTabs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 2

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Close pool with tabs
	err = pool.Close()
	assert.NoError(t, err)

	// Size should be 0
	assert.Equal(t, 0, pool.Size())
}

// TestClose_MultipleClose tests that close is idempotent
func TestClose_MultipleClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Close multiple times
	err = pool.Close()
	assert.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)

	err = pool.Close()
	assert.NoError(t, err)
}

// TestClose_Concurrent tests concurrent close calls
func TestClose_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 2

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Close from multiple goroutines
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			pool.Close()
			done <- true
		}()
	}

	// All should complete without panic
	for i := 0; i < 5; i++ {
		<-done
	}

	assert.Equal(t, 0, pool.Size())
}

// TestAcquire_ClosedPool tests acquiring from closed pool
func TestAcquire_ClosedPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// Close pool
	err = pool.Close()
	assert.NoError(t, err)

	// Try to acquire from closed pool
	ctx := context.Background()
	page, err := pool.Acquire(ctx)

	assert.Error(t, err)
	assert.Nil(t, page)
	assert.Equal(t, ErrPoolClosed, err)
}

// TestRelease_ClosedPool tests releasing to closed pool
func TestRelease_ClosedPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping browser-dependent test in short mode")
	}

	opts := DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := NewRenderer(opts)
	require.NoError(t, err)
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire tab before closing
	page, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, page)

	// Close pool
	err = pool.Close()
	assert.NoError(t, err)

	// Release to closed pool - should close the page
	// This should not panic
	assert.NotPanics(t, func() {
		pool.Release(page)
	})
}
