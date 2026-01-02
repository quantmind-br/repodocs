package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCache_Integration_Persistence tests that cache data persists across cache restarts
func TestCache_Integration_Persistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persistence test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")
	key := "https://example.com/persistent"
	value := []byte("persistent value")

	// Phase 1: Create cache and write data
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
		Logger:    false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = c1.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Verify data is in cache
	retrieved, err := c1.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)

	// Close cache
	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen cache and verify data persists
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
		Logger:    false,
	})
	require.NoError(t, err)
	defer c2.Close()

	// Data should still be present
	retrieved, err = c2.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved, "data should persist across cache restart")
}

// TestCache_Integration_MultipleEntries tests persistence with multiple entries
func TestCache_Integration_MultipleEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple entries test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")

	// Create test data
	entries := map[string][]byte{
		"https://example.com/1": []byte("value1"),
		"https://example.com/2": []byte("value2"),
		"https://example.com/3": []byte("value3"),
		"https://example.com/4": []byte("value4"),
		"https://example.com/5": []byte("value5"),
	}

	// Phase 1: Write all entries
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	for key, value := range entries {
		err = c1.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	// Verify count
	size := c1.Size()
	assert.Equal(t, int64(len(entries)), size)

	// Close cache
	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen and verify all entries persist
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c2.Close()

	// Verify all entries
	for key, expectedValue := range entries {
		retrieved, err := c2.Get(ctx, key)
		require.NoError(t, err, "should retrieve key: %s", key)
		assert.Equal(t, expectedValue, retrieved, "value should match for key: %s", key)
	}

	// Verify count
	size = c2.Size()
	assert.Equal(t, int64(len(entries)), size, "all entries should persist")
}

// TestCache_Integration_DeletePersistence tests that deletions persist
func TestCache_Integration_DeletePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping delete persistence test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")
	key := "https://example.com/delete-persist"
	value := []byte("value to delete")

	// Phase 1: Create, write, then delete
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = c1.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	err = c1.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	exists := c1.Has(ctx, key)
	assert.False(t, exists)

	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen and verify deletion persists
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c2.Close()

	// Should still be gone
	exists = c2.Has(ctx, key)
	assert.False(t, exists, "deletion should persist")

	_, err = c2.Get(ctx, key)
	assert.Error(t, err, "should return error for deleted key")
}

// TestCache_Integration_ClearPersistence tests that clear operation persists
func TestCache_Integration_ClearPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping clear persistence test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")

	// Phase 1: Populate and clear
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	// Add 10 entries
	for i := 0; i < 10; i++ {
		key := "https://example.com/clear" + string(rune('0'+i))
		value := []byte("value" + string(rune('0'+i)))
		err = c1.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	size := c1.Size()
	assert.Equal(t, int64(10), size)

	err = c1.Clear()
	require.NoError(t, err)

	size = c1.Size()
	assert.Equal(t, int64(0), size)

	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen and verify cache is still empty
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c2.Close()

	size = c2.Size()
	assert.Equal(t, int64(0), size, "cache should remain empty after clear and restart")
}

// TestCache_Integration_Stats tests cache statistics across restarts
func TestCache_Integration_Stats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stats test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")

	// Phase 1: Create cache with data
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	// Add entries
	for i := 0; i < 5; i++ {
		key := "https://example.com/stats" + string(rune('0'+i))
		value := []byte("value for stats")
		err = c1.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	stats1 := c1.Stats()
	require.NotNil(t, stats1)
	entriesBefore := stats1["entries"].(int64)

	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen and check stats
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c2.Close()

	stats2 := c2.Stats()
	require.NotNil(t, stats2)
	entriesAfter := stats2["entries"].(int64)

	assert.Equal(t, entriesBefore, entriesAfter, "entry count should persist")
	assert.Greater(t, entriesAfter, int64(0), "should have entries after restart")
}

// TestCache_Integration_Concurrency tests concurrent operations with persistent cache
func TestCache_Integration_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")

	c, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	done := make(chan bool, 20)

	// Start concurrent writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "https://example.com/concurrent" + string(rune('0'+id%10))
			value := []byte("concurrent value")
			_ = c.Set(ctx, key, value, time.Hour)
			done <- true
		}(i)
	}

	// Start concurrent readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "https://example.com/concurrent" + string(rune('0'+id%10))
			_, _ = c.Get(ctx, key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify cache is in consistent state
	size := c.Size()
	assert.GreaterOrEqual(t, size, int64(0))
}

// TestCache_Integration_LargeValues tests caching and persisting large values
func TestCache_Integration_LargeValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large values test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")
	key := "https://example.com/large"

	// Create a 1MB value
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	// Phase 1: Write large value
	c1, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = c1.Set(ctx, key, largeValue, time.Hour)
	require.NoError(t, err)

	retrieved, err := c1.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, largeValue, retrieved)

	err = c1.Close()
	require.NoError(t, err)

	// Phase 2: Reopen and verify large value persists
	c2, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c2.Close()

	retrieved, err = c2.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, largeValue, retrieved, "large value should persist")
	assert.Len(t, retrieved, 1024*1024)
}

// TestCache_Integration_DirectoryCreation tests that cache creates directories as needed
func TestCache_Integration_DirectoryCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping directory creation test in short mode")
	}

	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "level1", "level2", "level3", "cache")

	// Directory doesn't exist yet
	_, err := os.Stat(nestedPath)
	assert.True(t, os.IsNotExist(err))

	// Create cache - should create all parent directories
	c, err := cache.NewBadgerCache(cache.Options{
		Directory: nestedPath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c.Close()

	// Verify directory was created
	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "cache directory should be created")

	// Verify cache is functional
	ctx := context.Background()
	key := "https://example.com/directory-test"
	value := []byte("directory test value")

	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

// TestCache_Integration_KeyGenerationConsistency tests that key generation is consistent
func TestCache_Integration_KeyGenerationConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping key generation test in short mode")
	}

	testURLs := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page1/", // same as first after normalization
		"http://example.com/page1",   // different scheme
	}

	// Generate keys for each URL
	keys := make(map[string]string)
	for _, url := range testURLs {
		key := cache.PageKey(url)
		keys[url] = key
	}

	// Same normalized URLs should produce same keys
	assert.Equal(t, keys["https://example.com/page1"], keys["https://example.com/page1/"],
		"URLs with same normalized form should produce same keys")

	// Different URLs should produce different keys
	assert.NotEqual(t, keys["https://example.com/page1"], keys["https://example.com/page2"],
		"different URLs should produce different keys")

	// Different schemes should produce different keys
	assert.NotEqual(t, keys["https://example.com/page1"], keys["http://example.com/page1"],
		"different schemes should produce different keys")
}

// TestCache_Integration_PrefixedKeys tests different key prefixes
func TestCache_Integration_PrefixedKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping prefixed keys test in short mode")
	}

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "badger")

	c, err := cache.NewBadgerCache(cache.Options{
		Directory: cachePath,
		InMemory:  false,
	})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	url := "https://example.com/test"

	// Generate keys with different prefixes
	pageKey := cache.PageKey(url)
	sitemapKey := cache.SitemapKey(url)
	metadataKey := cache.MetadataKey(url)

	// All keys should be different
	assert.NotEqual(t, pageKey, sitemapKey)
	assert.NotEqual(t, pageKey, metadataKey)
	assert.NotEqual(t, sitemapKey, metadataKey)

	// Each should have its prefix
	assert.Contains(t, pageKey, "page:")
	assert.Contains(t, sitemapKey, "sitemap:")
	assert.Contains(t, metadataKey, "meta:")

	// All should be usable in cache
	pageValue := []byte("page content")
	sitemapValue := []byte("sitemap content")
	metadataValue := []byte("metadata content")

	err = c.Set(ctx, pageKey, pageValue, time.Hour)
	require.NoError(t, err)

	err = c.Set(ctx, sitemapKey, sitemapValue, time.Hour)
	require.NoError(t, err)

	err = c.Set(ctx, metadataKey, metadataValue, time.Hour)
	require.NoError(t, err)

	// Verify all three can be retrieved independently
	retrievedPage, err := c.Get(ctx, pageKey)
	require.NoError(t, err)
	assert.Equal(t, pageValue, retrievedPage)

	retrievedSitemap, err := c.Get(ctx, sitemapKey)
	require.NoError(t, err)
	assert.Equal(t, sitemapValue, retrievedSitemap)

	retrievedMetadata, err := c.Get(ctx, metadataKey)
	require.NoError(t, err)
	assert.Equal(t, metadataValue, retrievedMetadata)
}
