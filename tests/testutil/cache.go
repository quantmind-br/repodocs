package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/stretchr/testify/require"
)

// NewBadgerCache creates an in-memory BadgerDB cache for testing
func NewBadgerCache(t *testing.T) domain.Cache {
	t.Helper()

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

// CreateTestCacheEntry creates a test cache entry with expiration
func CreateTestCacheEntry(t *testing.T, url, content string, ttlSeconds int) *domain.CacheEntry {
	t.Helper()

	return &domain.CacheEntry{
		URL:         url,
		Content:     []byte(content),
		ContentType: "text/html",
		FetchedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(ttlSeconds) * time.Second),
	}
}

// CreateTestCacheEntryExpired creates an expired cache entry
func CreateTestCacheEntryExpired(t *testing.T, url, content string) *domain.CacheEntry {
	t.Helper()

	return &domain.CacheEntry{
		URL:         url,
		Content:     []byte(content),
		ContentType: "text/html",
		FetchedAt:   time.Now().Add(-24 * time.Hour),
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	}
}

// VerifyCacheEntry verifies a cache entry was stored correctly
func VerifyCacheEntry(t *testing.T, cache domain.Cache, key, expectedValue string) {
	t.Helper()

	result, err := cache.Get(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, expectedValue, string(result))
}
