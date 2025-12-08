package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createInMemoryCache(t *testing.T) *cache.BadgerCache {
	c, err := cache.NewBadgerCache(cache.Options{
		InMemory: true,
		Logger:   false,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		c.Close()
	})
	return c
}

func TestBadgerCache_SetGet(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/test"
	value := []byte("test content")
	ttl := 1 * time.Hour

	// Set value
	err := c.Set(ctx, key, value, ttl)
	require.NoError(t, err)

	// Get value
	result, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestBadgerCache_GetNonExistent(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	_, err := c.Get(ctx, "https://example.com/nonexistent")
	assert.ErrorIs(t, err, domain.ErrCacheMiss)
}

func TestBadgerCache_Has(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/test"
	value := []byte("test content")

	// Initially should not exist
	assert.False(t, c.Has(ctx, key))

	// Set value
	err := c.Set(ctx, key, value, 1*time.Hour)
	require.NoError(t, err)

	// Now should exist
	assert.True(t, c.Has(ctx, key))
}

func TestBadgerCache_Delete(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/test"
	value := []byte("test content")

	// Set value
	err := c.Set(ctx, key, value, 1*time.Hour)
	require.NoError(t, err)

	// Delete value
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Should no longer exist
	assert.False(t, c.Has(ctx, key))
}

func TestBadgerCache_Expiration(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/test"
	value := []byte("test content")

	// Set with very short TTL (BadgerDB minimum TTL is 1 second)
	err := c.Set(ctx, key, value, 1*time.Second)
	require.NoError(t, err)

	// Should exist initially
	result, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)

	// Wait for expiration
	time.Sleep(1500 * time.Millisecond)

	// Should be expired
	_, err = c.Get(ctx, key)
	assert.ErrorIs(t, err, domain.ErrCacheMiss)
}

func TestBadgerCache_Concurrent(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "https://example.com/concurrent/" + string(rune(id)) + "/" + string(rune(j))
				value := []byte("value-" + string(rune(id)) + "-" + string(rune(j)))

				// Set
				err := c.Set(ctx, key, value, 1*time.Hour)
				if err != nil {
					t.Errorf("Set failed: %v", err)
					return
				}

				// Get
				_, err = c.Get(ctx, key)
				if err != nil {
					t.Errorf("Get failed: %v", err)
					return
				}

				// Has
				_ = c.Has(ctx, key)
			}
		}(i)
	}

	wg.Wait()
}

func TestBadgerCache_Clear(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 10; i++ {
		key := "https://example.com/clear/" + string(rune(i))
		err := c.Set(ctx, key, []byte("value"), 1*time.Hour)
		require.NoError(t, err)
	}

	// Verify size
	assert.Greater(t, c.Size(), int64(0))

	// Clear cache
	err := c.Clear()
	require.NoError(t, err)

	// Size should be 0
	assert.Equal(t, int64(0), c.Size())
}

func TestBadgerCache_Size(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	// Initially empty
	assert.Equal(t, int64(0), c.Size())

	// Add entries
	for i := 0; i < 5; i++ {
		key := "https://example.com/size/" + string(rune('a'+i))
		err := c.Set(ctx, key, []byte("value"), 1*time.Hour)
		require.NoError(t, err)
	}

	// Size should be 5
	assert.Equal(t, int64(5), c.Size())
}

func TestBadgerCache_Stats(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	// Add some entries
	for i := 0; i < 3; i++ {
		key := "https://example.com/stats/" + string(rune('a'+i))
		err := c.Set(ctx, key, []byte("value data here"), 1*time.Hour)
		require.NoError(t, err)
	}

	stats := c.Stats()

	assert.Contains(t, stats, "entries")
	assert.Contains(t, stats, "lsm_size")
	assert.Contains(t, stats, "vlog_size")
	assert.Equal(t, int64(3), stats["entries"])
}

func TestBadgerCache_UpdateExisting(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/update"

	// Set initial value
	err := c.Set(ctx, key, []byte("initial"), 1*time.Hour)
	require.NoError(t, err)

	// Update value
	err = c.Set(ctx, key, []byte("updated"), 1*time.Hour)
	require.NoError(t, err)

	// Get should return updated value
	result, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, []byte("updated"), result)
}

func TestBadgerCache_LargeValue(t *testing.T) {
	c := createInMemoryCache(t)
	ctx := context.Background()

	key := "https://example.com/large"
	// Create a moderately large value (64KB - safe for in-memory BadgerDB)
	value := make([]byte, 64*1024)
	for i := range value {
		value[i] = byte(i % 256)
	}

	// Set large value
	err := c.Set(ctx, key, value, 1*time.Hour)
	require.NoError(t, err)

	// Get large value
	result, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestCacheEntry_IsExpired(t *testing.T) {
	// Not expired
	entry := cache.Entry{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	assert.False(t, entry.IsExpired())

	// Expired
	entry = cache.Entry{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	assert.True(t, entry.IsExpired())
}

func TestCacheEntry_TTL(t *testing.T) {
	// Has TTL remaining
	entry := cache.Entry{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	ttl := entry.TTL()
	assert.Greater(t, ttl, time.Duration(0))
	assert.Less(t, ttl, 2*time.Hour)

	// Expired (TTL should be 0)
	entry = cache.Entry{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	assert.Equal(t, time.Duration(0), entry.TTL())
}

func TestCacheKeyGeneration(t *testing.T) {
	// Same URLs should generate same keys
	key1 := cache.GenerateKey("https://example.com/test")
	key2 := cache.GenerateKey("https://example.com/test")
	assert.Equal(t, key1, key2)

	// Different URLs should generate different keys
	key3 := cache.GenerateKey("https://example.com/other")
	assert.NotEqual(t, key1, key3)

	// Keys should be consistent length (SHA256 = 64 hex chars)
	assert.Len(t, key1, 64)
}
