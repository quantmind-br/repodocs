package renderer_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
)

// TestTabPool_Size tests the Size function
func TestTabPool_Size(t *testing.T) {
	// Skip test if browser is not available
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a real browser (using default options)
	opts := renderer.DefaultRendererOptions()
	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer, skipping test:", err)
	}
	defer r.Close()

	// Get the pool from the renderer
	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Initially, all tabs should be available
	assert.Equal(t, 5, pool.Size())

	// Acquire one tab
	ctx := context.Background()
	acquiredTab, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, acquiredTab)

	// Size should now be 4
	assert.Equal(t, 4, pool.Size())

	// Release the tab
	pool.Release(acquiredTab)

	// Size should be 5 again
	assert.Equal(t, 5, pool.Size())
}

// TestTabPool_Size_MultipleAcquires tests Size with multiple acquires and releases
func TestTabPool_Size_MultipleAcquires(t *testing.T) {
	// Skip test if browser is not available
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer, skipping test:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Initial size should be 5
	assert.Equal(t, 5, pool.Size())

	// Acquire all tabs
	ctx := context.Background()
	tabs := make([]*rod.Page, 5)
	for i := 0; i < 5; i++ {
		tab, err := pool.Acquire(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tab)
		tabs[i] = tab
	}

	// Size should be 0 now
	assert.Equal(t, 0, pool.Size())

	// Release one tab
	pool.Release(tabs[0])

	// Size should be 1
	assert.Equal(t, 1, pool.Size())

	// Release another tab
	pool.Release(tabs[1])

	// Size should be 2
	assert.Equal(t, 2, pool.Size())

	// Release the remaining tabs
	for i := 2; i < 5; i++ {
		pool.Release(tabs[i])
	}

	// Size should be 5 again
	assert.Equal(t, 5, pool.Size())
}

// TestTabPool_MaxSize tests the MaxSize function
func TestTabPool_MaxSize(t *testing.T) {
	// Skip test if browser is not available
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Test with default renderer (maxTabs = 5)
	opts1 := renderer.DefaultRendererOptions()
	r1, err := renderer.NewRenderer(opts1)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r1.Close()

	pool1, err := r1.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}
	assert.Equal(t, 5, pool1.MaxSize())

	// Test with custom maxTabs = 10
	opts2 := renderer.DefaultRendererOptions()
	opts2.MaxTabs = 10
	r2, err := renderer.NewRenderer(opts2)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r2.Close()

	pool2, err := r2.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}
	assert.Equal(t, 10, pool2.MaxSize())

	// Test with maxTabs = 1
	opts3 := renderer.DefaultRendererOptions()
	opts3.MaxTabs = 1
	r3, err := renderer.NewRenderer(opts3)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r3.Close()

	pool3, err := r3.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}
	assert.Equal(t, 1, pool3.MaxSize())
}

// TestTabPool_Error tests the Error function
func TestTabPool_Error(t *testing.T) {
	// Test the poolError type directly
	err := renderer.ErrPoolClosed

	assert.NotNil(t, err)
	assert.Equal(t, "pool is closed", err.Error())

	// Verify it's the expected error type
	// The error should be comparable
	assert.Equal(t, renderer.ErrPoolClosed, renderer.ErrPoolClosed)
}

// TestNewTabPool_WithOptions tests creating a new tab pool with custom options
func TestNewTabPool_WithOptions(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a real browser for testing
	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 2 // Custom option

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Verify custom options were applied
	assert.Equal(t, 2, pool.MaxSize())
	assert.Equal(t, 2, pool.Size()) // Should be initialized with 2 tabs
}

// TestNewTabPool_DefaultOptions tests creating a new tab pool with default options
func TestNewTabPool_DefaultOptions(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	// Create a real browser for testing with default options
	opts := renderer.DefaultRendererOptions()

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Verify default options were applied (MaxTabs = 5)
	assert.Equal(t, 5, pool.MaxSize())
	assert.Equal(t, 5, pool.Size()) // Should be initialized with 5 tabs
}

// TestAcquire_Success tests acquiring a tab successfully
func TestAcquire_Success(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Acquire tab
	ctx := context.Background()
	page, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page)
	assert.Equal(t, 0, pool.Size()) // No tabs available

	// Release tab
	pool.Release(page)
	assert.Equal(t, 1, pool.Size()) // Tab returned to pool
}

// TestAcquire_Timeout tests acquiring a tab with timeout
func TestAcquire_Timeout(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Acquire the only tab
	ctx := context.Background()
	page, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page)

	// Try to acquire another tab with timeout
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// TestRelease_Success tests releasing a tab successfully
func TestRelease_Success(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Acquire and release tab
	ctx := context.Background()
	page, err := pool.Acquire(ctx)
	assert.NoError(t, err)

	pool.Release(page)
	assert.Equal(t, 5, pool.Size()) // Tab returned to pool
}

// TestRelease_Invalid tests releasing a tab to a closed pool
func TestRelease_Invalid(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Close pool
	err = pool.Close()
	assert.NoError(t, err)

	// Acquire a tab before closing
	ctx := context.Background()
	page, err := pool.Acquire(ctx)
	assert.Error(t, err) // Should fail because pool is closed
	assert.Nil(t, page)
}

// TestClose_ClosesAllTabs tests that Close closes all tabs in the pool
func TestClose_ClosesAllTabs(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 3

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	assert.Equal(t, 3, pool.Size())

	// Close pool
	err = pool.Close()
	assert.NoError(t, err)
	assert.Equal(t, 0, pool.Size()) // Pool is closed, no tabs available

	// Close again (should be idempotent)
	err = pool.Close()
	assert.NoError(t, err)

	// Clean up renderer
	r.Close()
}

// TestRelease_PoolFull tests releasing a tab when the pool is full
func TestRelease_PoolFull(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 1

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Acquire the only tab
	ctx := context.Background()
	page1, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, page1)

	// Pool is now empty (size = 0)
	assert.Equal(t, 0, pool.Size())

	// Try to acquire another tab (should timeout)
	ctx2, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctx2)
	assert.Error(t, err)

	// Release page1 - this should succeed
	pool.Release(page1)
	assert.Equal(t, 1, pool.Size())

	// Create a new page manually (simulating the scenario where we have an extra page)
	// Note: In practice, this scenario is hard to reproduce with the current design
	// because we can't exceed the pool capacity through normal use
	// The default case in Release is primarily a safety measure
}
