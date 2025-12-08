package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"path"
	"strings"
)

// GenerateKey generates a cache key from a URL
// The key is a SHA256 hash of the normalized URL
func GenerateKey(rawURL string) string {
	normalized := normalizeForKey(rawURL)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// GenerateKeyWithPrefix generates a cache key with a prefix
func GenerateKeyWithPrefix(prefix, rawURL string) string {
	key := GenerateKey(rawURL)
	return prefix + ":" + key
}

// normalizeForKey normalizes a URL for consistent key generation
func normalizeForKey(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Normalize scheme
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	// Normalize host
	u.Host = strings.ToLower(u.Host)

	// Remove default ports
	if (u.Scheme == "http" && u.Port() == "80") ||
		(u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}

	// Clean path
	if u.Path == "" {
		u.Path = "/"
	} else {
		u.Path = path.Clean(u.Path)
	}

	// Remove trailing slash except for root
	if u.Path != "/" && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	// Remove fragment
	u.Fragment = ""

	return u.String()
}

// KeyPrefix constants for different cache types
const (
	PrefixPage     = "page"
	PrefixSitemap  = "sitemap"
	PrefixGit      = "git"
	PrefixMetadata = "meta"
)

// PageKey generates a cache key for a page
func PageKey(url string) string {
	return GenerateKeyWithPrefix(PrefixPage, url)
}

// SitemapKey generates a cache key for a sitemap
func SitemapKey(url string) string {
	return GenerateKeyWithPrefix(PrefixSitemap, url)
}

// MetadataKey generates a cache key for metadata
func MetadataKey(url string) string {
	return GenerateKeyWithPrefix(PrefixMetadata, url)
}
