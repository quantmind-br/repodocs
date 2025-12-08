package unit

import (
	"context"
	"testing"

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
	// We need to access the pool, but it's not exported
	// So we'll test through the renderer's behavior

	// Create a new tab pool directly
	pool, err := r.GetTabPool()
	if err != nil {
		t.Skip("Failed to get tab pool:", err)
	}

	// Initially, all tabs should be available
	// Size() returns the number of available tabs in the channel
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
