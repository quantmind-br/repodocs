package unit

import (
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/cache"
	"github.com/stretchr/testify/assert"
)

func TestGenerateKeyWithPrefix(t *testing.T) {
	key := cache.GenerateKeyWithPrefix("prefix", "https://example.com/page")
	assert.True(t, strings.HasPrefix(key, "prefix:"))
}

func TestPageKey(t *testing.T) {
	key := cache.PageKey("https://example.com/page")
	assert.Contains(t, key, "page:")
}

func TestSitemapKey(t *testing.T) {
	key := cache.SitemapKey("https://example.com/sitemap.xml")
	assert.Contains(t, key, "sitemap:")
}

func TestMetadataKey(t *testing.T) {
	key := cache.MetadataKey("https://example.com/page")
	assert.Contains(t, key, "meta:")
}

func TestGenerateKeyWithPrefix_Deterministic(t *testing.T) {
	// Same input should produce same output
	key1 := cache.GenerateKeyWithPrefix("test", "https://example.com/page")
	key2 := cache.GenerateKeyWithPrefix("test", "https://example.com/page")
	assert.Equal(t, key1, key2)
}

func TestPageKey_Deterministic(t *testing.T) {
	// Same URL should produce same key
	key1 := cache.PageKey("https://example.com/page")
	key2 := cache.PageKey("https://example.com/page")
	assert.Equal(t, key1, key2)
}

func TestSitemapKey_Deterministic(t *testing.T) {
	// Same URL should produce same key
	key1 := cache.SitemapKey("https://example.com/sitemap.xml")
	key2 := cache.SitemapKey("https://example.com/sitemap.xml")
	assert.Equal(t, key1, key2)
}

func TestMetadataKey_Deterministic(t *testing.T) {
	// Same URL should produce same key
	key1 := cache.MetadataKey("https://example.com/page")
	key2 := cache.MetadataKey("https://example.com/page")
	assert.Equal(t, key1, key2)
}

func TestGenerateKeyWithPrefix_DifferentURLs(t *testing.T) {
	// Different URLs should produce different keys
	key1 := cache.GenerateKeyWithPrefix("page", "https://example.com/page1")
	key2 := cache.GenerateKeyWithPrefix("page", "https://example.com/page2")
	assert.NotEqual(t, key1, key2)
}

func TestPageKey_UrlNormalization(t *testing.T) {
	// URLs with same normalized form should produce same key
	key1 := cache.PageKey("https://example.com/page/")
	key2 := cache.PageKey("https://example.com/page")
	assert.Equal(t, key1, key2)
}

func TestSitemapKey_UrlNormalization(t *testing.T) {
	// URLs with same normalized form should produce same key
	key1 := cache.SitemapKey("https://example.com/sitemap.xml/")
	key2 := cache.SitemapKey("https://example.com/sitemap.xml")
	assert.Equal(t, key1, key2)

	// Different schemes produce different keys
	key3 := cache.SitemapKey("https://example.com/sitemap.xml")
	key4 := cache.SitemapKey("http://example.com/sitemap.xml")
	assert.NotEqual(t, key3, key4)
}

func TestGenerateKeyWithPrefix_Length(t *testing.T) {
	key := cache.GenerateKeyWithPrefix("prefix", "https://example.com/test")
	// prefix + ":" + 64-char SHA256 hash
	expectedMinLength := len("prefix:") + 64
	assert.GreaterOrEqual(t, len(key), expectedMinLength)
}

func TestPageKey_ContainsSHA256(t *testing.T) {
	key := cache.PageKey("https://example.com/test")
	// Extract the hash part (after "page:")
	parts := strings.SplitN(key, ":", 2)
	assert.Len(t, parts, 2)
	assert.Equal(t, "page", parts[0])
	// Hash should be 64 characters (SHA256 hex)
	assert.Len(t, parts[1], 64)
}

func TestSitemapKey_ContainsSHA256(t *testing.T) {
	key := cache.SitemapKey("https://example.com/sitemap.xml")
	// Extract the hash part (after "sitemap:")
	parts := strings.SplitN(key, ":", 2)
	assert.Len(t, parts, 2)
	assert.Equal(t, "sitemap", parts[0])
	// Hash should be 64 characters (SHA256 hex)
	assert.Len(t, parts[1], 64)
}

func TestMetadataKey_ContainsSHA256(t *testing.T) {
	key := cache.MetadataKey("https://example.com/page")
	// Extract the hash part (after "meta:")
	parts := strings.SplitN(key, ":", 2)
	assert.Len(t, parts, 2)
	assert.Equal(t, "meta", parts[0])
	// Hash should be 64 characters (SHA256 hex)
	assert.Len(t, parts[1], 64)
}

func TestGenerateKeyWithPrefix_VariousURLs(t *testing.T) {
	// Test with various URL formats
	testCases := []string{
		"https://example.com",
		"http://example.com",
		"https://example.com/",
		"https://example.com/path",
		"https://example.com/path/",
		"https://example.com/path?query=value",
		"https://example.com/path#fragment",
	}

	for _, url := range testCases {
		key := cache.GenerateKeyWithPrefix("test", url)
		assert.NotEmpty(t, key)
		assert.True(t, strings.HasPrefix(key, "test:"))
	}
}
