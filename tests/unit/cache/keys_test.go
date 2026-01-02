package cache_test

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

func TestGenerateKey_Simple(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		check func(string)
	}{
		{
			name: "Simple URL",
			url:  "https://example.com",
			check: func(key string) {
				assert.NotEmpty(t, key)
				assert.Len(t, key, 64) // SHA256 hex length
			},
		},
		{
			name: "URL with path",
			url:  "https://example.com/docs/api",
			check: func(key string) {
				assert.NotEmpty(t, key)
				assert.Len(t, key, 64)
			},
		},
		{
			name: "URL with query params",
			url:  "https://example.com/page?param=value",
			check: func(key string) {
				assert.NotEmpty(t, key)
				assert.Len(t, key, 64)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cache.GenerateKey(tt.url)
			tt.check(key)
		})
	}
}

func TestGenerateKey_WithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		url      string
		expected string
	}{
		{
			name:     "With page prefix",
			prefix:   "page",
			url:      "https://example.com",
			expected: "page:",
		},
		{
			name:     "With sitemap prefix",
			prefix:   "sitemap",
			url:      "https://example.com/sitemap.xml",
			expected: "sitemap:",
		},
		{
			name:     "With git prefix",
			prefix:   "git",
			url:      "https://github.com/user/repo",
			expected: "git:",
		},
		{
			name:     "With metadata prefix",
			prefix:   "meta",
			url:      "https://example.com/metadata",
			expected: "meta:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cache.GenerateKeyWithPrefix(tt.prefix, tt.url)
			assert.Contains(t, key, tt.expected)
			// Verify it's a valid SHA256 hash (64 hex chars)
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				assert.Len(t, parts[1], 64)
			}
		})
	}
}

func TestNormalizeForKey_SpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Trailing slash removed",
			input:    "https://example.com/path/",
			expected: "https://example.com/path",
		},
		{
			name:     "Root path normalized",
			input:    "https://example.com",
			expected: "https://example.com/",
		},
		{
			name:     "Empty path normalized to /",
			input:    "https://example.com",
			expected: "https://example.com/",
		},
		{
			name:     "Fragment removed",
			input:    "https://example.com/page#section",
			expected: "https://example.com/page",
		},
		{
			name:     "Default HTTPS port removed",
			input:    "https://example.com:443/page",
			expected: "https://example.com/page",
		},
		{
			name:     "Default HTTP port removed",
			input:    "http://example.com:80/page",
			expected: "http://example.com/page",
		},
		{
			name:     "Host converted to lowercase",
			input:    "https://EXAMPLE.COM/Page",
			expected: "https://example.com/Page",
		},
		{
			name:     "Path cleaned",
			input:    "https://example.com/a/./b/../c",
			expected: "https://example.com/a/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to test the internal normalizeForKey function
			// Since it's not exported, we'll test it indirectly through GenerateKey
			key1 := cache.GenerateKey(tt.input)
			key2 := cache.GenerateKey(tt.expected)
			assert.Equal(t, key1, key2, "normalized URLs should produce same key")
		})
	}
}

func TestPageKey_GeneratesCorrectKey(t *testing.T) {
	url := "https://example.com/docs/api"

	key := cache.PageKey(url)

	// Verify prefix
	assert.Contains(t, key, "page:")

	// Verify it's deterministic
	key2 := cache.PageKey(url)
	assert.Equal(t, key, key2)

	// Verify different URL produces different key
	key3 := cache.PageKey("https://example.com/other")
	assert.NotEqual(t, key, key3)
}

func TestSitemapKey_GeneratesCorrectKey(t *testing.T) {
	url := "https://example.com/sitemap.xml"

	key := cache.SitemapKey(url)

	// Verify prefix
	assert.Contains(t, key, "sitemap:")

	// Verify it's deterministic
	key2 := cache.SitemapKey(url)
	assert.Equal(t, key, key2)

	// Verify different URL produces different key
	key3 := cache.SitemapKey("https://example.com/sitemap2.xml")
	assert.NotEqual(t, key, key3)
}

func TestMetadataKey_GeneratesCorrectKey(t *testing.T) {
	url := "https://example.com/metadata"

	key := cache.MetadataKey(url)

	// Verify prefix
	assert.Contains(t, key, "meta:")

	// Verify it's deterministic
	key2 := cache.MetadataKey(url)
	assert.Equal(t, key, key2)

	// Verify different URL produces different key
	key3 := cache.MetadataKey("https://example.com/other")
	assert.NotEqual(t, key, key3)
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

func TestNormalizeForKey_InvalidURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Invalid URL - returns as-is",
			input: "not a valid url",
		},
		{
			name:  "Empty string",
			input: "",
		},
		{
			name:  "Partial URL",
			input: "example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test indirectly through GenerateKey
			key1 := cache.GenerateKey(tt.input)
			key2 := cache.GenerateKey(tt.input)
			assert.Equal(t, key1, key2, "same input should produce same key")
			assert.NotEmpty(t, key1)
		})
	}
}

func TestGenerateKey_URLWithoutScheme(t *testing.T) {
	// URL without scheme should default to https
	key := cache.GenerateKey("example.com/page")

	// Should still produce a valid hash
	assert.NotEmpty(t, key)
	assert.Len(t, key, 64)

	// Same URL with explicit https should match
	keyWithHTTPS := cache.GenerateKey("https://example.com/page")
	assert.Equal(t, key, keyWithHTTPS)
}

func TestNormalizeForKey_PortHandling(t *testing.T) {
	// Test that default ports are normalized
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTP with default port 80",
			input:    "http://example.com:80/page",
			expected: "http://example.com/page",
		},
		{
			name:     "HTTPS with default port 443",
			input:    "https://example.com:443/page",
			expected: "https://example.com/page",
		},
		{
			name:     "Non-default port preserved",
			input:    "https://example.com:8080/page",
			expected: "https://example.com:8080/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := cache.GenerateKey(tt.input)
			key2 := cache.GenerateKey(tt.expected)
			assert.Equal(t, key1, key2, "normalized URLs should produce same key")
		})
	}
}

func TestNormalizeForKey_PathNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Trailing slash removed",
			input:    "https://example.com/page/",
			expected: "https://example.com/page",
		},
		{
			name:     "Root path preserved",
			input:    "https://example.com/",
			expected: "https://example.com/",
		},
		{
			name:     "Path cleaned",
			input:    "https://example.com/a/b/../c",
			expected: "https://example.com/a/c",
		},
		{
			name:     "Current dir removed",
			input:    "https://example.com/a/./b",
			expected: "https://example.com/a/b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := cache.GenerateKey(tt.input)
			key2 := cache.GenerateKey(tt.expected)
			assert.Equal(t, key1, key2, "normalized URLs should produce same key")
		})
	}
}
