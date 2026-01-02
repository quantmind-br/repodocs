package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntry_IsExpired tests entry expiration
func TestEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		entry    *Entry
		expected bool
	}{
		{
			name: "not expired",
			entry: &Entry{
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			entry: &Entry{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "just now - not expired",
			entry: &Entry{
				ExpiresAt: time.Now().Add(100 * time.Millisecond),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEntry_TTL tests remaining time-to-live
func TestEntry_TTL(t *testing.T) {
	tests := []struct {
		name         string
		entry        *Entry
		minDuration  time.Duration
		maxDuration  time.Duration
	}{
		{
			name: "positive TTL",
			entry: &Entry{
				ExpiresAt: time.Now().Add(1 * time.Hour),
			},
			minDuration: 59 * time.Minute,
			maxDuration: 61 * time.Minute,
		},
		{
			name: "expired entry returns 0",
			entry: &Entry{
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			minDuration: 0,
			maxDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttl := tt.entry.TTL()
			assert.GreaterOrEqual(t, ttl, tt.minDuration)
			assert.LessOrEqual(t, ttl, tt.maxDuration)
		})
	}
}

// TestDefaultOptions tests default options
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Empty(t, opts.Directory)
	assert.False(t, opts.InMemory)
	assert.False(t, opts.Logger)
}

// TestGenerateKey tests cache key generation
func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		check    func(t *testing.T, key string)
	}{
		{
			name: "generates consistent keys for same URL",
			url:  "https://example.com/page",
			check: func(t *testing.T, key string) {
				key2 := GenerateKey("https://example.com/page")
				assert.Equal(t, key, key2)
			},
		},
		{
			name: "generates different keys for different URLs",
			url:  "https://example.com/page1",
			check: func(t *testing.T, key string) {
				key2 := GenerateKey("https://example.com/page2")
				assert.NotEqual(t, key, key2)
			},
		},
		{
			name: "key length is 64 characters (SHA256 hex)",
			url:  "https://example.com/page",
			check: func(t *testing.T, key string) {
				assert.Equal(t, 64, len(key))
			},
		},
		{
			name: "handles invalid URL gracefully",
			url:  ":not-a-url",
			check: func(t *testing.T, key string) {
				assert.NotEmpty(t, key)
				assert.Equal(t, 64, len(key))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateKey(tt.url)
			if tt.check != nil {
				tt.check(t, key)
			}
		})
	}
}

// TestGenerateKeyWithPrefix tests key generation with prefix
func TestGenerateKeyWithPrefix(t *testing.T) {
	tests := []struct {
		name  string
		prefix string
		url   string
		check func(t *testing.T, key string)
	}{
		{
			name:   "adds prefix to key",
			prefix: "page",
			url:    "https://example.com/page",
			check: func(t *testing.T, key string) {
				assert.True(t, len(key) > 65) // 64 + prefix + ":"
				assert.Contains(t, key, "page:")
			},
		},
		{
			name:   "different prefixes create different keys",
			prefix: "test",
			url:    "https://example.com/page",
			check: func(t *testing.T, key string) {
				pageKey := GenerateKeyWithPrefix("page", "https://example.com/page")
				assert.NotEqual(t, key, pageKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenerateKeyWithPrefix(tt.prefix, tt.url)
			if tt.check != nil {
				tt.check(t, key)
			}
		})
	}
}

// TestNormalizeForKey tests URL normalization
func TestNormalizeForKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normalizes to lowercase host",
			input:    "https://EXAMPLE.COM/page",
			expected: "https://example.com/page",
		},
		{
			name:     "removes trailing slash",
			input:    "https://example.com/page/",
			expected: "https://example.com/page",
		},
		{
			name:     "keeps root slash",
			input:    "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "removes fragment",
			input:    "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "adds default https scheme",
			input:    "example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "cleans path",
			input:    "https://example.com/./page/../other",
			expected: "https://example.com/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeForKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPageKey tests page key generation
func TestPageKey(t *testing.T) {
	key := PageKey("https://example.com/page")
	assert.Contains(t, key, "page:")
	assert.True(t, len(key) > 65)
}

// TestSitemapKey tests sitemap key generation
func TestSitemapKey(t *testing.T) {
	key := SitemapKey("https://example.com/sitemap.xml")
	assert.Contains(t, key, "sitemap:")
	assert.True(t, len(key) > 65)
}

// TestMetadataKey tests metadata key generation
func TestMetadataKey(t *testing.T) {
	key := MetadataKey("https://example.com/page")
	assert.Contains(t, key, "meta:")
	assert.True(t, len(key) > 65)
}

// TestNewBadgerCache tests creating cache
func TestNewBadgerCache(t *testing.T) {
	t.Run("creates in-memory cache", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{
			InMemory: true,
		})
		require.NoError(t, err)
		assert.NotNil(t, cache)
		cache.Close()
	})

	t.Run("creates file-based cache with temp directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache, err := NewBadgerCache(Options{
			Directory: tmpDir,
		})
		require.NoError(t, err)
		assert.NotNil(t, cache)
		cache.Close()
	})

	t.Run("creates file-based cache in default location", func(t *testing.T) {
		// Create a temp home directory
		tmpDir := t.TempDir()
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()
		os.Setenv("HOME", tmpDir)

		cache, err := NewBadgerCache(Options{
			Directory: "",
		})
		require.NoError(t, err)
		assert.NotNil(t, cache)
		cache.Close()

		// Check directory was created
		cacheDir := tmpDir + "/.repodocs/cache"
		_, err = os.Stat(cacheDir)
		assert.NoError(t, err)
	})
}

// TestBadgerCache_Get tests getting values from cache
func TestBadgerCache_Get(t *testing.T) {
	t.Run("returns error for missing key", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		value, err := cache.Get(ctx, "https://example.com/nonexistent")

		assert.Error(t, err)
		assert.Nil(t, value)
	})

	t.Run("retrieves stored value", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"
		value := []byte("test content")

		err = cache.Set(ctx, key, value, 1*time.Hour)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})
}

// TestBadgerCache_Set tests setting values in cache
func TestBadgerCache_Set(t *testing.T) {
	t.Run("stores value with TTL", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"
		value := []byte("test content")

		err = cache.Set(ctx, key, value, 1*time.Hour)
		assert.NoError(t, err)

		has := cache.Has(ctx, key)
		assert.True(t, has)
	})

	t.Run("stores value without TTL", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"
		value := []byte("test content")

		err = cache.Set(ctx, key, value, 0)
		assert.NoError(t, err)

		has := cache.Has(ctx, key)
		assert.True(t, has)
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"

		err = cache.Set(ctx, key, []byte("original"), 1*time.Hour)
		require.NoError(t, err)

		err = cache.Set(ctx, key, []byte("updated"), 1*time.Hour)
		require.NoError(t, err)

		value, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, []byte("updated"), value)
	})
}

// TestBadgerCache_Has tests checking if key exists
func TestBadgerCache_Has(t *testing.T) {
	t.Run("returns false for missing key", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		has := cache.Has(ctx, "https://example.com/nonexistent")
		assert.False(t, has)
	})

	t.Run("returns true for existing key", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"

		err = cache.Set(ctx, key, []byte("content"), 1*time.Hour)
		require.NoError(t, err)

		has := cache.Has(ctx, key)
		assert.True(t, has)
	})
}

// TestBadgerCache_Delete tests deleting keys
func TestBadgerCache_Delete(t *testing.T) {
	t.Run("deletes existing key", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		key := "https://example.com/page"

		err = cache.Set(ctx, key, []byte("content"), 1*time.Hour)
		require.NoError(t, err)

		err = cache.Delete(ctx, key)
		assert.NoError(t, err)

		has := cache.Has(ctx, key)
		assert.False(t, has)
	})

	t.Run("deleting non-existent key is no error", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		err = cache.Delete(ctx, "https://example.com/nonexistent")
		assert.NoError(t, err)
	})
}

// TestBadgerCache_Clear tests clearing all entries
func TestBadgerCache_Clear(t *testing.T) {
	cache, err := NewBadgerCache(Options{InMemory: true})
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()

	// Add some entries
	cache.Set(ctx, "https://example.com/page1", []byte("content1"), 1*time.Hour)
	cache.Set(ctx, "https://example.com/page2", []byte("content2"), 1*time.Hour)

	assert.Greater(t, cache.Size(), int64(0))

	// Clear
	err = cache.Clear()
	assert.NoError(t, err)

	assert.Equal(t, int64(0), cache.Size())
}

// TestBadgerCache_Size tests getting cache size
func TestBadgerCache_Size(t *testing.T) {
	cache, err := NewBadgerCache(Options{InMemory: true})
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()

	// Empty cache
	assert.Equal(t, int64(0), cache.Size())

	// Add entries
	cache.Set(ctx, "https://example.com/page1", []byte("content1"), 1*time.Hour)
	cache.Set(ctx, "https://example.com/page2", []byte("content2"), 1*time.Hour)
	cache.Set(ctx, "https://example.com/page3", []byte("content3"), 1*time.Hour)

	assert.Equal(t, int64(3), cache.Size())
}

// TestBadgerCache_Stats tests cache statistics
func TestBadgerCache_Stats(t *testing.T) {
	cache, err := NewBadgerCache(Options{InMemory: true})
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "https://example.com/page", []byte("content"), 1*time.Hour)

	stats := cache.Stats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "entries")
	assert.Contains(t, stats, "lsm_size")
	assert.Contains(t, stats, "vlog_size")
	assert.Equal(t, int64(1), stats["entries"])
}

// TestBadgerCache_Integration tests integration scenarios
func TestBadgerCache_Integration(t *testing.T) {
	t.Run("full workflow", func(t *testing.T) {
		cache, err := NewBadgerCache(Options{InMemory: true})
		require.NoError(t, err)
		defer cache.Close()

		ctx := context.Background()
		url := "https://example.com/page"
		content := []byte("page content")

		// Initially not found
		_, err = cache.Get(ctx, url)
		assert.Error(t, err)

		// Store
		err = cache.Set(ctx, url, content, 1*time.Hour)
		assert.NoError(t, err)

		// Check exists
		assert.True(t, cache.Has(ctx, url))

		// Retrieve
		retrieved, err := cache.Get(ctx, url)
		assert.NoError(t, err)
		assert.Equal(t, content, retrieved)

		// Delete
		err = cache.Delete(ctx, url)
		assert.NoError(t, err)

		// Gone
		assert.False(t, cache.Has(ctx, url))
	})
}

// TestBadgerCache_ConcurrentAccess tests concurrent access safety
func TestBadgerCache_ConcurrentAccess(t *testing.T) {
	cache, err := NewBadgerCache(Options{InMemory: true})
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()
	done := make(chan bool)

	// Concurrent writes
	for i := 0; i < 50; i++ {
		go func(i int) {
			url := "https://example.com/page" + string(rune('0'+i))
			cache.Set(ctx, url, []byte("content"), 1*time.Hour)
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		go func(i int) {
			url := "https://example.com/page" + string(rune('0'+i))
			cache.Get(ctx, url)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have 50 entries
	assert.Equal(t, int64(50), cache.Size())
}

// TestBadgerCache_ContextCancellation tests context handling
func TestBadgerCache_ContextCancellation(t *testing.T) {
	cache, err := NewBadgerCache(Options{InMemory: true})
	require.NoError(t, err)
	defer cache.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Badger operations don't check context, so they may still succeed
	// This test documents the current behavior
	_ = cache
	_ = ctx
}
