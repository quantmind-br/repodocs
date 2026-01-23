package utils

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// NormalizeURL normalizes a URL for consistent handling
func NormalizeURL(rawURL string) (string, error) {
	// If no scheme is present, prepend https:// before parsing
	// This ensures the host is correctly identified
	if !strings.Contains(rawURL, "://") && !strings.HasPrefix(rawURL, "//") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Ensure scheme
	if u.Scheme == "" {
		u.Scheme = "https"
	}

	// Normalize host to lowercase
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

	// Remove trailing slash (except for root)
	if u.Path != "/" && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	// Remove fragment
	u.Fragment = ""

	// Build the result manually to ensure trailing slash for root path
	result := u.String()

	// Ensure root path has trailing slash
	if u.Path == "/" && u.RawQuery == "" && !strings.HasSuffix(result, "/") {
		result += "/"
	}

	return result, nil
}

// NormalizeURLWithoutQuery normalizes a URL and removes query parameters
func NormalizeURLWithoutQuery(rawURL string) (string, error) {
	normalized, err := NormalizeURL(rawURL)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}

	u.RawQuery = ""
	return u.String(), nil
}

// ResolveURL resolves a relative URL against a base URL
func ResolveURL(base, ref string) (string, error) {
	// If the base doesn't end with / and doesn't have a file extension,
	// treat it as a directory by adding a trailing slash
	// This ensures that "../page" from "/docs/api" resolves to "/docs/page" not "/page"
	if !strings.HasSuffix(base, "/") && !strings.Contains(path.Base(base), ".") {
		base += "/"
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	resolved := baseURL.ResolveReference(refURL)
	return resolved.String(), nil
}

// GetDomain extracts the domain from a URL
func GetDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// GetBaseDomain extracts the base domain from a URL, removing only the "www" prefix
// For example: "www.example.com" -> "example.com", "docs.example.com" -> "docs.example.com"
func GetBaseDomain(rawURL string) string {
	host := GetDomain(rawURL)
	if host == "" {
		return ""
	}

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Only strip "www." prefix, keep other subdomains
	if strings.HasPrefix(strings.ToLower(host), "www.") {
		return host[4:]
	}

	return host
}

// extractRootDomain extracts the root domain (domain + TLD) without any subdomains
// For example: "docs.example.com" -> "example.com", "www.example.com" -> "example.com"
func extractRootDomain(rawURL string) string {
	host := GetDomain(rawURL)
	if host == "" {
		return ""
	}

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Split by dots
	parts := strings.Split(strings.ToLower(host), ".")

	// Need at least 2 parts (domain + TLD)
	if len(parts) < 2 {
		return host
	}

	// Extract the last 2 parts (domain + TLD)
	// This works for most TLDs like .com, .org, .net
	// Note: This is a simplified approach and won't work perfectly for
	// compound TLDs like .co.uk, but covers the common cases
	return strings.Join(parts[len(parts)-2:], ".")
}

// IsSameDomain checks if two URLs have the same domain
func IsSameDomain(url1, url2 string) bool {
	return strings.EqualFold(GetDomain(url1), GetDomain(url2))
}

// IsSameBaseDomain checks if two URLs have the same base domain (ignoring subdomains)
// For example: "docs.example.com" and "api.example.com" return true
func IsSameBaseDomain(url1, url2 string) bool {
	return extractRootDomain(url1) == extractRootDomain(url2)
}

// IsAbsoluteURL checks if a URL is absolute
func IsAbsoluteURL(rawURL string) bool {
	// Protocol-relative URLs (starting with //) are considered absolute
	if strings.HasPrefix(rawURL, "//") {
		return true
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.IsAbs()
}

// IsHTTPURL checks if a URL uses HTTP or HTTPS scheme
func IsHTTPURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// IsGitURL checks if a URL is a git repository URL
func IsGitURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "git@") ||
		strings.HasSuffix(rawURL, ".git") ||
		strings.Contains(rawURL, "github.com") ||
		strings.Contains(rawURL, "gitlab.com") ||
		strings.Contains(rawURL, "bitbucket.org")
}

// IsSitemapURL checks if a URL points to a sitemap
func IsSitemapURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.HasSuffix(lower, "sitemap.xml") ||
		strings.HasSuffix(lower, "sitemap.xml.gz") ||
		strings.Contains(lower, "sitemap")
}

// IsLLMSURL checks if a URL points to an llms.txt file
func IsLLMSURL(rawURL string) bool {
	lower := strings.ToLower(rawURL)
	return strings.HasSuffix(lower, "/llms.txt") ||
		strings.HasSuffix(lower, "llms.txt")
}

// IsPkgGoDevURL checks if a URL is a pkg.go.dev URL
func IsPkgGoDevURL(rawURL string) bool {
	return strings.Contains(rawURL, "pkg.go.dev")
}

// ExtractLinks extracts all href links from HTML content
// This is a simple regex-based extraction, use goquery for more robust parsing
func ExtractLinks(html, baseURL string) []string {
	linkRegex := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := linkRegex.FindAllStringSubmatch(html, -1)

	links := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			link := match[1]
			// Skip anchors, javascript, mailto, etc.
			if strings.HasPrefix(link, "#") ||
				strings.HasPrefix(link, "javascript:") ||
				strings.HasPrefix(link, "mailto:") ||
				strings.HasPrefix(link, "tel:") {
				continue
			}

			// Resolve relative URLs
			if !IsAbsoluteURL(link) {
				resolved, err := ResolveURL(baseURL, link)
				if err != nil {
					continue
				}
				link = resolved
			}

			links = append(links, link)
		}
	}

	return links
}

// GenerateOutputDirFromURL generates an output directory name from a URL
// Examples:
//   - https://github.com/QwenLM/qwen-code -> docs_qwen-code
//   - https://docs.crawl4ai.com/sitemap.xml -> docs_docscrawl4aicom
//   - https://docs.factory.ai/llms.txt -> docs_docsfactoryai
//   - https://pkg.go.dev/github.com/user/package -> docs_package
//   - https://docs.rs/ratatui/latest/ratatui/ -> docs_ratatui
func GenerateOutputDirFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "docs"
	}

	host := strings.ToLower(u.Host)
	pathStr := strings.Trim(u.Path, "/")

	// Remove port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}

	var name string

	// Handle Git repository URLs (GitHub, GitLab, Bitbucket)
	if strings.Contains(host, "github.com") ||
		strings.Contains(host, "gitlab.com") ||
		strings.Contains(host, "bitbucket.org") {
		// Extract repository name from path
		parts := strings.Split(pathStr, "/")
		if len(parts) >= 2 {
			// Use the repo name (second part: owner/repo)
			name = parts[1]
			// Remove .git suffix if present
			name = strings.TrimSuffix(name, ".git")
		} else if len(parts) == 1 && parts[0] != "" {
			name = parts[0]
		}
	}

	// Handle pkg.go.dev URLs
	if name == "" && strings.Contains(host, "pkg.go.dev") {
		// Path is like: /github.com/user/package or /package
		parts := strings.Split(pathStr, "/")
		if len(parts) > 0 {
			// Use the last significant part
			for i := len(parts) - 1; i >= 0; i-- {
				if parts[i] != "" && !strings.Contains(parts[i], ".") {
					name = parts[i]
					break
				}
			}
			// Fallback to last part
			if name == "" && len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
	}

	// Handle docs.rs URLs
	if name == "" && strings.Contains(host, "docs.rs") {
		parts := strings.Split(pathStr, "/")
		if len(parts) >= 1 && parts[0] != "" {
			if parts[0] == "crate" && len(parts) >= 2 {
				name = parts[1]
			} else {
				name = parts[0]
			}
		}
	}

	// For other URLs, use sanitized hostname
	if name == "" {
		// Remove common prefixes
		host = strings.TrimPrefix(host, "www.")

		// Concatenate all parts of the domain (removes dots)
		// example.com -> examplecom
		// docs.crawl4ai.com -> docscrawl4aicom
		// This creates a clean directory name from the full domain
		name = sanitizeForDirName(host)
	}

	// Ensure we have a valid name
	if name == "" {
		return "docs"
	}

	// Sanitize the name for filesystem
	name = sanitizeForDirName(name)

	return "docs_" + name
}

// sanitizeForDirName removes characters that are not safe for directory names
func sanitizeForDirName(s string) string {
	// Remove dots, spaces, and special characters
	var result strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// HasBaseURL checks if a URL starts with the given base URL path
// Example: HasBaseURL("https://example.com/docs/api", "https://example.com/docs") returns true
// Example: HasBaseURL("https://example.com/blog", "https://example.com/docs") returns false
func HasBaseURL(targetURL, baseURL string) bool {
	if baseURL == "" {
		return true
	}

	targetParsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	// Must be same host
	if strings.ToLower(targetParsed.Host) != strings.ToLower(baseParsed.Host) {
		return false
	}

	// Normalize paths
	targetPath := strings.TrimSuffix(targetParsed.Path, "/")
	basePath := strings.TrimSuffix(baseParsed.Path, "/")

	// Target path must start with base path
	if basePath == "" || basePath == "/" {
		return true
	}

	return targetPath == basePath || strings.HasPrefix(targetPath, basePath+"/")
}

// FilterLinks filters links based on patterns
func FilterLinks(links []string, excludePatterns []string) []string {
	var regexps []*regexp.Regexp
	for _, pattern := range excludePatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		regexps = append(regexps, re)
	}

	filtered := make([]string, 0, len(links))
	for _, link := range links {
		excluded := false
		for _, re := range regexps {
			if re.MatchString(link) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, link)
		}
	}

	return filtered
}
