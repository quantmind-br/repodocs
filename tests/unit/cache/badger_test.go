package cache_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBadgerCache_Success(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	opts := cache.Options{
		Directory: tmpDir,
		InMemory:  false,
		Logger:    false,
	}

	c, err := cache.NewBadgerCache(opts)
	require.NoError(t, err)
	defer c.Close()

	assert.NotNil(t, c)
}

func TestNewBadgerCache_WithOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    cache.Options
		wantErr bool
	}{
		{
			name: "InMemory",
			opts: cache.Options{
				InMemory: true,
			},
			wantErr: false,
		},
		{
			name: "WithDirectory",
			opts: cache.Options{
				Directory: t.TempDir(),
				InMemory:  false,
			},
			wantErr: false,
		},
		{
			name: "WithLogger",
			opts: cache.Options{
				InMemory: true,
				Logger:   true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := cache.NewBadgerCache(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, c)
				defer c.Close()
			}
		})
	}
}

func TestGet_Found(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/test"
	value := []byte("test value")

	// Set a value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Get the value
	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestGet_NotFound(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Try to get a non-existent key
	_, err = c.Get(ctx, "https://example.com/nonexistent")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrCacheMiss, err)
}

func TestGet_Expired(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/expire"
	value := []byte("test value")

	// Set a value with very short TTL
	err = c.Set(ctx, key, value, 10*time.Millisecond)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Try to get the expired value
	_, err = c.Get(ctx, key)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrCacheMiss, err)
}

func TestSet_Success(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/set"
	value := []byte("test value")

	// Set a value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, c.Has(ctx, key))
}

func TestSet_Update(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/update"

	// Set initial value
	value1 := []byte("value1")
	err = c.Set(ctx, key, value1, time.Hour)
	require.NoError(t, err)

	// Update with new value
	value2 := []byte("value2")
	err = c.Set(ctx, key, value2, time.Hour)
	require.NoError(t, err)

	// Verify the updated value
	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value2, retrieved)
}

func TestHas_Exists(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/has"
	value := []byte("test value")

	// Set a value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Check if it exists
	assert.True(t, c.Has(ctx, key))
}

func TestHas_NotExists(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Check non-existent key
	assert.False(t, c.Has(ctx, "https://example.com/nonexistent"))
}

func TestDelete_Success(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/delete"
	value := []byte("test value")

	// Set a value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, c.Has(ctx, key))

	// Delete it
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Verify it's gone
	assert.False(t, c.Has(ctx, key))
}

func TestDelete_NotExists(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Delete non-existent key should not error
	err = c.Delete(ctx, "https://example.com/nonexistent")
	require.NoError(t, err)
}

func TestClear_Success(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 10; i++ {
		key := "https://example.com/key" + string(rune(i))
		value := []byte("value" + string(rune(i)))
		err = c.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	// Verify size
	size := c.Size()
	assert.Equal(t, int64(10), size)

	// Clear all
	err = c.Clear()
	require.NoError(t, err)

	// Verify size is 0
	size = c.Size()
	assert.Equal(t, int64(0), size)
}

func TestSize_ReturnsCount(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Initially empty
	assert.Equal(t, int64(0), c.Size())

	// Add one item
	err = c.Set(ctx, "https://example.com/1", []byte("value1"), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), c.Size())

	// Add another item
	err = c.Set(ctx, "https://example.com/2", []byte("value2"), time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(2), c.Size())

	// Delete one item
	err = c.Delete(ctx, "https://example.com/1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), c.Size())
}

func TestStats_ReturnsStatistics(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Add some values
	err = c.Set(ctx, "https://example.com/1", []byte("value1"), time.Hour)
	require.NoError(t, err)
	err = c.Set(ctx, "https://example.com/2", []byte("value2"), time.Hour)
	require.NoError(t, err)

	// Get stats
	stats := c.Stats()

	// Verify stats structure
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "entries")
	assert.Contains(t, stats, "lsm_size")
	assert.Contains(t, stats, "vlog_size")

	// Verify entries count
	entries := stats["entries"].(int64)
	assert.Equal(t, int64(2), entries)

	// Verify size values are non-negative
	lsmSize := stats["lsm_size"].(int64)
	vlogSize := stats["vlog_size"].(int64)
	assert.GreaterOrEqual(t, lsmSize, int64(0))
	assert.GreaterOrEqual(t, vlogSize, int64(0))
}

func TestClose_Success(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)

	// Close should not error
	err = c.Close()
	require.NoError(t, err)

	// Close again should also not error (idempotent)
	err = c.Close()
	require.NoError(t, err)
}

func TestNewBadgerCache_HomeDirFallback(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	opts := cache.Options{
		Directory: "",
		InMemory:  false,
	}

	c, err := cache.NewBadgerCache(opts)
	require.NoError(t, err)
	defer c.Close()

	expectedDir := filepath.Join(tmpHome, ".repodocs", "cache")
	assert.DirExists(t, expectedDir)
}

func TestNewBadgerCache_DirectoryCreation(t *testing.T) {
	// Test with directory that needs to be created
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "cache", "nested", "directory")

	opts := cache.Options{
		Directory: newDir,
		InMemory:  false,
	}

	// Should create the directory
	c, err := cache.NewBadgerCache(opts)
	require.NoError(t, err)
	assert.NotNil(t, c)
	defer c.Close()

	// Verify the directory was created
	assert.DirExists(t, newDir)
}

func TestGet_InvalidKeyHandling(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Test with various key formats
	testKeys := []string{
		"",
		"simple-key",
		"https://example.com/path",
		"file:///local/path",
	}

	for _, key := range testKeys {
		t.Run("key:"+key, func(t *testing.T) {
			_, err := c.Get(ctx, key)
			// Should return ErrCacheMiss for non-existent keys
			assert.Error(t, err)
		})
	}
}

func TestSet_WithZeroTTL(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/zero-ttl"
	value := []byte("test value")

	// Set with zero TTL
	err = c.Set(ctx, key, value, 0)
	require.NoError(t, err)

	// Should still be retrievable immediately
	retrieved, err := c.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, value, retrieved)
}

func TestEntry_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		entry    *cache.Entry
		expected bool
	}{
		{
			name: "Not expired",
			entry: &cache.Entry{
				ExpiresAt: now.Add(time.Hour),
			},
			expected: false,
		},
		{
			name: "Already expired",
			entry: &cache.Entry{
				ExpiresAt: now.Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "Exactly now",
			entry: &cache.Entry{
				ExpiresAt: now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.entry.IsExpired())
		})
	}
}

func TestEntry_TTL(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		entry    *cache.Entry
		expected time.Duration
	}{
		{
			name: "Future expiration",
			entry: &cache.Entry{
				ExpiresAt: now.Add(time.Hour),
			},
			expected: time.Hour,
		},
		{
			name: "Past expiration",
			entry: &cache.Entry{
				ExpiresAt: now.Add(-time.Hour),
			},
			expected: 0,
		},
		{
			name: "Exactly now",
			entry: &cache.Entry{
				ExpiresAt: now,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := tt.entry.TTL()
			// Allow small variance for test execution time
			assert.GreaterOrEqual(t, ttl, tt.expected-time.Millisecond*10)
			assert.LessOrEqual(t, ttl, tt.expected+time.Millisecond*10)
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := cache.DefaultOptions()

	// Verify default values
	assert.Equal(t, "", opts.Directory)
	assert.Equal(t, false, opts.InMemory)
	assert.Equal(t, false, opts.Logger)
}

// TestSet_WithVariousTTLs tests different TTL values
func TestSet_WithVariousTTLs(t *testing.T) {
	tests := []struct {
		name        string
		ttl         time.Duration
		shouldExist bool
		waitTime    time.Duration
	}{
		{
			name:        "Long TTL - 1 hour",
			ttl:         time.Hour,
			shouldExist: true,
			waitTime:    0,
		},
		{
			name:        "Short TTL - 10ms",
			ttl:         10 * time.Millisecond,
			shouldExist: false,
			waitTime:    20 * time.Millisecond,
		},
		{
			name:        "Zero TTL",
			ttl:         0,
			shouldExist: true,
			waitTime:    0,
		},
		{
			name:        "Negative TTL (treated as zero)",
			ttl:         -time.Hour,
			shouldExist: true,
			waitTime:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
			require.NoError(t, err)
			defer c.Close()

			ctx := context.Background()
			key := "https://example.com/ttl-test"
			value := []byte("value")

			err = c.Set(ctx, key, value, tt.ttl)
			require.NoError(t, err)

			if tt.waitTime > 0 {
				time.Sleep(tt.waitTime)
			}

			exists := c.Has(ctx, key)
			if tt.shouldExist {
				assert.True(t, exists, "key should exist with TTL: %v", tt.ttl)
			} else {
				assert.False(t, exists, "key should not exist with TTL: %v", tt.ttl)
			}
		})
	}
}

// TestGet_WithCancelledContext tests context cancellation
func TestGet_WithCancelledContext(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to get with cancelled context
	_, err = c.Get(ctx, "https://example.com/test")
	// BadgerDB may or may not return error for cancelled context
	// The important thing is it doesn't panic
	_ = err
}

// TestSet_WithCancelledContext tests set with cancelled context
func TestSet_WithCancelledContext(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	key := "https://example.com/test"
	value := []byte("value")

	// Try to set with cancelled context
	err = c.Set(ctx, key, value, time.Hour)
	// BadgerDB may or may not return error for cancelled context
	// The important thing is it doesn't panic
	_ = err
}

// TestHas_AfterDeletion tests Has after deletion
func TestHas_AfterDeletion(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/delete-has"
	value := []byte("value")

	// Set value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Verify exists
	assert.True(t, c.Has(ctx, key))

	// Delete
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Verify doesn't exist
	assert.False(t, c.Has(ctx, key))
}

// TestSize_AfterClear tests Size after Clear operation
func TestSize_AfterClear(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		key := "https://example.com/size" + string(rune('0'+i))
		value := []byte("value")
		err = c.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	// Verify size
	assert.Equal(t, int64(5), c.Size())

	// Clear all
	err = c.Clear()
	require.NoError(t, err)

	// Verify size is zero
	assert.Equal(t, int64(0), c.Size())
}

// TestStats_WithEmptyCache tests Stats with empty cache
func TestStats_WithEmptyCache(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	// Get stats for empty cache
	stats := c.Stats()

	// Verify stats structure
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "entries")
	assert.Contains(t, stats, "lsm_size")
	assert.Contains(t, stats, "vlog_size")

	// Verify empty cache has 0 entries
	entries := stats["entries"].(int64)
	assert.Equal(t, int64(0), entries)
}

// TestDelete_MultipleTimes tests deleting same key multiple times
func TestDelete_MultipleTimes(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/multi-delete"
	value := []byte("value")

	// Set value
	err = c.Set(ctx, key, value, time.Hour)
	require.NoError(t, err)

	// Delete first time
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Delete second time - should not error
	err = c.Delete(ctx, key)
	require.NoError(t, err)

	// Delete third time - should not error
	err = c.Delete(ctx, key)
	require.NoError(t, err)
}

// TestSet_UpdateWithDifferentTTL tests updating with different TTL
func TestSet_UpdateWithDifferentTTL(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()
	key := "https://example.com/update-ttl"

	// Set with long TTL
	err = c.Set(ctx, key, []byte("value1"), time.Hour)
	require.NoError(t, err)

	// Update with short TTL
	err = c.Set(ctx, key, []byte("value2"), 50*time.Millisecond)
	require.NoError(t, err)

	// Wait for short TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	exists := c.Has(ctx, key)
	assert.False(t, exists, "updated entry should expire with new TTL")
}

// TestGet_EmptyKey tests getting with empty key
func TestGet_EmptyKey(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Try to get with empty key
	_, err = c.Get(ctx, "")
	assert.Error(t, err)
}

// TestSet_EmptyKey tests setting with empty key
func TestSet_EmptyKey(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Set with empty key
	err = c.Set(ctx, "", []byte("value"), time.Hour)
	// Should either succeed or error, but not panic
	_ = err
}

// TestHas_EmptyKey tests Has with empty key
func TestHas_EmptyKey(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Check empty key
	exists := c.Has(ctx, "")
	// Should return false or handle gracefully
	assert.False(t, exists)
}

// TestStats_LSMAndVLogSizes tests that LSM and VLog sizes are reported
func TestStats_LSMAndVLogSizes(t *testing.T) {
	c, err := cache.NewBadgerCache(cache.Options{InMemory: true})
	require.NoError(t, err)
	defer c.Close()

	ctx := context.Background()

	// Add some data to populate LSM/VLog
	for i := 0; i < 10; i++ {
		key := "https://example.com/sizes" + string(rune('0'+i))
		value := []byte("some value for size testing")
		err = c.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)
	}

	stats := c.Stats()

	// Verify size fields exist and are non-negative
	lsmSize := stats["lsm_size"].(int64)
	vlogSize := stats["vlog_size"].(int64)

	assert.GreaterOrEqual(t, lsmSize, int64(0), "LSM size should be non-negative")
	assert.GreaterOrEqual(t, vlogSize, int64(0), "VLog size should be non-negative")
}

// TestNewBadgerCache_Constructor tests cache construction
func TestNewBadgerCache_Constructor(t *testing.T) {
	tests := []struct {
		name   string
		opts   cache.Options
		verify func(t *testing.T, c *cache.BadgerCache, err error)
	}{
		{
			name: "In-memory cache",
			opts: cache.Options{
				InMemory: true,
			},
			verify: func(t *testing.T, c *cache.BadgerCache, err error) {
				require.NoError(t, err)
				assert.NotNil(t, c)
			},
		},
		{
			name: "File-based cache with temp dir",
			opts: cache.Options{
				Directory: t.TempDir(),
				InMemory:  false,
			},
			verify: func(t *testing.T, c *cache.BadgerCache, err error) {
				require.NoError(t, err)
				assert.NotNil(t, c)
			},
		},
		{
			name: "Cache with logger enabled",
			opts: cache.Options{
				InMemory: true,
				Logger:   true,
			},
			verify: func(t *testing.T, c *cache.BadgerCache, err error) {
				require.NoError(t, err)
				assert.NotNil(t, c)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := cache.NewBadgerCache(tt.opts)
			if c != nil {
				defer c.Close()
			}
			tt.verify(t, c, err)
		})
	}
}

// TestBadgerCache_ImplementsCacheInterface verifies BadgerCache implements domain.Cache
func TestBadgerCache_ImplementsCacheInterface(t *testing.T) {
	// This is a compile-time check
	var _ interface {
		Get(ctx context.Context, key string) ([]byte, error)
		Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
		Has(ctx context.Context, key string) bool
		Delete(ctx context.Context, key string) error
		Close() error
		Clear() error
		Size() int64
		Stats() map[string]interface{}
	} = (*cache.BadgerCache)(nil)
}
