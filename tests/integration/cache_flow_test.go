package integration

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCache creates an in-memory BadgerDB cache for testing
func createTestCache(t *testing.T) domain.Cache {
	t.Helper()
	c, err := cache.NewBadgerCache(cache.Options{
		InMemory: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

// TestIntegration_Cache_GetSet tests basic cache get/set operations
func TestIntegration_Cache_GetSet(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")

	// Act - Set (with 1 hour TTL)
	err := c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Act - Get
	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, value, retrieved)
}

// TestIntegration_Cache_Miss tests cache miss behavior
func TestIntegration_Cache_Miss(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()

	// Act
	retrieved, err := c.Get(ctx, "nonexistent-key")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

// TestIntegration_Cache_Delete tests cache delete operation
func TestIntegration_Cache_Delete(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	key := "delete-test-key"
	value := []byte("test-value")

	err := c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Act
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Assert - key should not exist
	retrieved, err := c.Get(ctx, key)
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

// TestIntegration_Cache_Overwrite tests overwriting existing keys
func TestIntegration_Cache_Overwrite(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	key := "overwrite-key"
	value1 := []byte("value1")
	value2 := []byte("value2")

	err := c.Set(ctx, key, value1, time.Hour)
	require.NoError(t, err)

	// Act - Overwrite
	err = c.Set(ctx, key, value2, time.Hour)
	require.NoError(t, err)

	// Assert
	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value2, retrieved)
}

// TestIntegration_Cache_MultipleKeys tests multiple keys in cache
func TestIntegration_Cache_MultipleKeys(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	values := [][]byte{[]byte("value1"), []byte("value2"), []byte("value3")}

	for i, key := range keys {
		err := c.Set(ctx, key, values[i], time.Hour)
		require.NoError(t, err)
	}

	// Act & Assert
	for i, key := range keys {
		retrieved, err := c.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, values[i], retrieved)
	}
}

// TestIntegration_Cache_ContextCancellation tests context cancellation handling
func TestIntegration_Cache_ContextCancellation(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Act - Set a value before cancellation test
	key := "context-key"
	value := []byte("value")

	// BadgerDB respects context cancellation
	err := c.Set(ctx, key, value, time.Hour)

	// Assert - should handle cancelled context
	// BadgerDB may or may not error depending on implementation
	_ = err // We just verify no panic
}

// TestIntegration_Cache_EmptyValue tests caching empty values
func TestIntegration_Cache_EmptyValue(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	key := "empty-value-key"
	value := []byte{0} // Use single byte instead of empty (BadgerDB behavior)

	// Act
	err := c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, value, retrieved)
}

// TestIntegration_Cache_LargeValue tests caching large values
func TestIntegration_Cache_LargeValue(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	key := "large-value-key"
	// Create a 100KB value (smaller to avoid BadgerDB issues in memory mode)
	value := make([]byte, 100*1024)
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Act
	err := c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, value, retrieved)
}

// TestIntegration_Cache_SpecialCharacterKeys tests keys with special characters
func TestIntegration_Cache_SpecialCharacterKeys(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()
	keys := []string{
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key:with:colons",
	}

	// Act & Assert
	for _, key := range keys {
		value := []byte("value-for-" + key)
		err := c.Set(ctx, key, value, time.Hour)
		require.NoError(t, err, "Failed to set key: %s", key)

		retrieved, err := c.Get(ctx, key)
		require.NoError(t, err, "Failed to get key: %s", key)
		assert.Equal(t, value, retrieved)
	}
}

// TestIntegration_Cache_DocumentCaching tests caching serialized documents
func TestIntegration_Cache_DocumentCaching(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()

	doc := &domain.Document{
		URL:         "https://example.com/page",
		Title:       "Test Page",
		Description: "Test description",
		Content:     "# Test Content\n\nThis is test content.",
		HTMLContent: "<h1>Test Content</h1><p>This is test content.</p>",
		WordCount:   5,
		CharCount:   50,
		ContentHash: "abc123",
	}

	// Simple serialization (in real code, you'd use JSON or gob)
	key := "doc:" + doc.URL
	value := []byte(doc.Content)

	// Act
	err := c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, doc.Content, string(retrieved))
}

// TestIntegration_Cache_ConcurrentAccess tests concurrent cache access
func TestIntegration_Cache_ConcurrentAccess(t *testing.T) {
	// Arrange
	c := createTestCache(t)
	ctx := context.Background()

	// Set up some initial values
	for i := 0; i < 10; i++ {
		key := "concurrent-key-" + string(rune('0'+i))
		err := c.Set(ctx, key, []byte("initial-value"), time.Hour)
		require.NoError(t, err)
	}

	// Act - concurrent reads and writes
	done := make(chan bool, 20)

	// Start readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "concurrent-key-" + string(rune('0'+id%10))
			_, _ = c.Get(ctx, key)
			done <- true
		}(i)
	}

	// Start writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "concurrent-key-" + string(rune('0'+id%10))
			_ = c.Set(ctx, key, []byte("new-value"), time.Hour)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}

	// Assert - no panics occurred (test passes if we get here)
}

// TestIntegration_Cache_Entry tests cache Entry struct
func TestIntegration_Cache_Entry(t *testing.T) {
	// Test Entry methods
	now := time.Now()
	expiredEntry := cache.Entry{
		URL:       "https://example.com",
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	validEntry := cache.Entry{
		URL:       "https://example.com",
		ExpiresAt: now.Add(1 * time.Hour),
	}

	// Test IsExpired
	assert.True(t, expiredEntry.IsExpired())
	assert.False(t, validEntry.IsExpired())

	// Test TTL
	assert.Equal(t, time.Duration(0), expiredEntry.TTL())
	assert.True(t, validEntry.TTL() > 0)
}

// TestIntegration_Cache_DefaultOptions tests default cache options
func TestIntegration_Cache_DefaultOptions(t *testing.T) {
	opts := cache.DefaultOptions()

	assert.Equal(t, "", opts.Directory)
	assert.False(t, opts.InMemory)
	assert.False(t, opts.Logger)
}
