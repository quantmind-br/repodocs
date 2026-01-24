package renderer_test

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTabPool_LazyCreation verifies that no tabs are created at pool creation time
func TestNewTabPool_LazyCreation(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 5

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	// With lazy initialization, Size() should be 0 immediately after creation
	assert.Equal(t, 0, pool.Size(), "Pool should have 0 available tabs initially (lazy creation)")

	// Created() should also be 0
	assert.Equal(t, 0, pool.Created(), "No tabs should be created initially")

	// MaxSize should still reflect the configured maximum
	assert.Equal(t, 5, pool.MaxSize(), "MaxSize should reflect configured maximum")
}

// TestTabPool_CreatesOnAcquire verifies that tabs are created on-demand when acquired
func TestTabPool_CreatesOnAcquire(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 3

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Initially no tabs created
	assert.Equal(t, 0, pool.Created(), "No tabs should be created initially")

	// Acquire first tab - should create one
	tab1, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, tab1)
	assert.Equal(t, 1, pool.Created(), "One tab should be created after first Acquire")

	// Acquire second tab - should create another
	tab2, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, tab2)
	assert.Equal(t, 2, pool.Created(), "Two tabs should be created after second Acquire")

	// Release first tab - it goes back to pool
	pool.Release(tab1)
	assert.Equal(t, 2, pool.Created(), "Created count should not change on Release")
	assert.Equal(t, 1, pool.Size(), "One tab should be available in pool")

	// Acquire again - should reuse the released tab, not create new
	tab3, err := pool.Acquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, tab3)
	assert.Equal(t, 2, pool.Created(), "Should reuse existing tab, not create new one")

	// Cleanup
	pool.Release(tab2)
	pool.Release(tab3)
}

// TestTabPool_DoesNotExceedMaxTabs verifies that created count never exceeds maxTabs
func TestTabPool_DoesNotExceedMaxTabs(t *testing.T) {
	if !renderer.IsAvailable() {
		t.Skip("Browser not available, skipping test")
	}

	opts := renderer.DefaultRendererOptions()
	opts.MaxTabs = 2

	r, err := renderer.NewRenderer(opts)
	if err != nil {
		t.Skip("Failed to create renderer:", err)
	}
	defer r.Close()

	pool, err := r.GetTabPool()
	require.NoError(t, err)

	ctx := context.Background()

	// Acquire all tabs up to max
	tab1, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, pool.Created())

	tab2, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, pool.Created())

	// Now pool is at capacity - trying to acquire should block until timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(ctxTimeout)
	assert.Error(t, err, "Should timeout when all tabs are in use")
	assert.Equal(t, context.DeadlineExceeded, err)

	// Created count should still be at max (2), not higher
	assert.Equal(t, 2, pool.Created(), "Created count should not exceed maxTabs")
	assert.LessOrEqual(t, pool.Created(), pool.MaxSize(), "Created should never exceed MaxSize")

	// Cleanup
	pool.Release(tab1)
	pool.Release(tab2)
}

// TestTabPool_LazyWithSingleTab tests lazy creation with maxTabs=1
func TestTabPool_LazyWithSingleTab(t *testing.T) {
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
	require.NoError(t, err)

	// Initially empty
	assert.Equal(t, 0, pool.Size())
	assert.Equal(t, 0, pool.Created())

	ctx := context.Background()

	// Acquire the only tab
	tab, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, pool.Created())
	assert.Equal(t, 0, pool.Size())

	// Release it
	pool.Release(tab)
	assert.Equal(t, 1, pool.Created(), "Created count unchanged after release")
	assert.Equal(t, 1, pool.Size(), "Tab available in pool after release")

	// Acquire again - should reuse
	tab2, err := pool.Acquire(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, pool.Created(), "Should reuse, not create new")

	pool.Release(tab2)
}
