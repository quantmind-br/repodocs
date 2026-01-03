package strategies

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

// SitemapXMLForDiscovery represents the XML structure of a sitemap (for discovery)
type SitemapXMLForDiscovery struct {
	XMLName xml.Name                 `xml:"urlset"`
	URLs    []SitemapURLForDiscovery `xml:"url"`
}

// SitemapURLForDiscovery represents a URL entry in a sitemap
type SitemapURLForDiscovery struct {
	Loc string `xml:"loc"`
}

// SitemapIndexXMLForDiscovery represents the XML structure of a sitemap index
type SitemapIndexXMLForDiscovery struct {
	XMLName  xml.Name                      `xml:"sitemapindex"`
	Sitemaps []SitemapLocationForDiscovery `xml:"sitemap"`
}

// SitemapLocationForDiscovery represents a sitemap location in an index
type SitemapLocationForDiscovery struct {
	Loc string `xml:"loc"`
}

// DiscoveryProbe defines a URL discovery mechanism for GitHub Pages sites
type DiscoveryProbe struct {
	Path   string
	Parser func(content []byte, baseURL string) ([]string, error)
	Name   string
}

// GetDiscoveryProbes returns all discovery probes in priority order
func GetDiscoveryProbes() []DiscoveryProbe {
	return []DiscoveryProbe{
		// Tier 1: LLM-optimized (highest quality)
		{"/llms.txt", ParseLLMsTxt, "llms.txt"},

		// Tier 2: Sitemaps (most common)
		{"/sitemap.xml", ParseSitemapXML, "sitemap.xml"},
		{"/sitemap-0.xml", ParseSitemapXML, "sitemap-0.xml"},
		{"/sitemap_index.xml", ParseSitemapIndexXML, "sitemap_index.xml"},

		// Tier 3: MkDocs (very reliable)
		{"/search/search_index.json", ParseMkDocsIndex, "mkdocs-search"},

		// Tier 4: Docusaurus
		{"/search-index.json", ParseDocusaurusIndex, "docusaurus-search"},

		// Tier 5: Hugo / Generic
		{"/index.json", ParseHugoIndex, "hugo-index"},
		{"/search.json", ParseGenericSearchIndex, "search.json"},

		// Tier 6: Modern SSGs
		{"/hashmap.json", ParseVitePressHashmap, "vitepress"},
	}
}

// ParseLLMsTxt parses llms.txt format (markdown links)
func ParseLLMsTxt(content []byte, baseURL string) ([]string, error) {
	// Use the parseLLMSLinks function from llms.go (same package)
	links := parseLLMSLinks(string(content))
	if len(links) == 0 {
		return nil, fmt.Errorf("no links found in llms.txt")
	}

	urls := make([]string, 0, len(links))
	for _, link := range links {
		urls = append(urls, resolveDiscoveryURL(link.URL, baseURL))
	}
	return urls, nil
}

// FilterAndDeduplicateURLs filters URLs to the same host and deduplicates them
func FilterAndDeduplicateURLs(urls []string, baseURL string) []string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return urls
	}
	baseHost := parsed.Host

	seen := make(map[string]bool)
	var result []string

	for _, u := range urls {
		parsedURL, err := url.Parse(u)
		if err != nil {
			continue
		}

		// Filter to same host
		if parsedURL.Host != "" && parsedURL.Host != baseHost {
			continue
		}

		// Normalize: remove fragment, trailing slash
		parsedURL.Fragment = ""
		normalized := parsedURL.String()
		normalized = strings.TrimSuffix(normalized, "/")

		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}

// ShouldSkipGitHubPagesURL returns true for URLs that typically don't contain documentation
func ShouldSkipGitHubPagesURL(u string) bool {
	lower := strings.ToLower(u)
	skipPatterns := []string{
		"/assets/", "/static/", "/_next/", "/_nuxt/",
		"/img/", "/images/", "/media/",
		"/css/", "/js/", "/fonts/",
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp",
		".css", ".js", ".woff", ".woff2", ".ttf", ".eot",
		".pdf", ".zip", ".tar", ".gz",
		"/feed.xml", "/rss.xml", "/atom.xml",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ParseSitemapXML parses standard sitemap.xml format
func ParseSitemapXML(content []byte, baseURL string) ([]string, error) {
	var sitemap SitemapXMLForDiscovery
	if err := xml.Unmarshal(content, &sitemap); err != nil {
		return nil, err
	}

	if len(sitemap.URLs) == 0 {
		return nil, fmt.Errorf("empty sitemap")
	}

	urls := make([]string, 0, len(sitemap.URLs))
	for _, u := range sitemap.URLs {
		if u.Loc != "" {
			urls = append(urls, u.Loc)
		}
	}
	return urls, nil
}

// ParseSitemapIndexXML parses sitemap index and fetches nested sitemaps
func ParseSitemapIndexXML(content []byte, baseURL string) ([]string, error) {
	var index SitemapIndexXMLForDiscovery
	if err := xml.Unmarshal(content, &index); err != nil {
		return nil, err
	}

	if len(index.Sitemaps) == 0 {
		return nil, fmt.Errorf("empty sitemap index")
	}

	// Return sitemap URLs for the caller to process
	urls := make([]string, 0, len(index.Sitemaps))
	for _, sm := range index.Sitemaps {
		urls = append(urls, sm.Loc)
	}
	return urls, nil
}

// MkDocsSearchIndex represents MkDocs search_index.json structure
type MkDocsSearchIndex struct {
	Docs []struct {
		Location string `json:"location"`
		Title    string `json:"title"`
		Text     string `json:"text"`
	} `json:"docs"`
}

// ParseMkDocsIndex parses MkDocs /search/search_index.json
func ParseMkDocsIndex(content []byte, baseURL string) ([]string, error) {
	var index MkDocsSearchIndex
	if err := json.Unmarshal(content, &index); err != nil {
		return nil, err
	}

	if len(index.Docs) == 0 {
		return nil, fmt.Errorf("empty MkDocs index")
	}

	seen := make(map[string]bool)
	var urls []string

	for _, doc := range index.Docs {
		loc := strings.Split(doc.Location, "#")[0]
		if loc == "" || loc == "." {
			loc = ""
		}

		fullURL := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(loc, "/")

		if !seen[fullURL] {
			seen[fullURL] = true
			urls = append(urls, fullURL)
		}
	}

	return urls, nil
}

// DocusaurusSearchEntry represents a Docusaurus search index entry
type DocusaurusSearchEntry struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ParseDocusaurusIndex parses Docusaurus /search-index.json
func ParseDocusaurusIndex(content []byte, baseURL string) ([]string, error) {
	var entries []DocusaurusSearchEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("empty Docusaurus index")
	}

	urls := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.URL != "" {
			urls = append(urls, resolveDiscoveryURL(entry.URL, baseURL))
		}
	}
	return urls, nil
}

// HugoSearchEntry represents a Hugo search index entry
type HugoSearchEntry struct {
	Permalink string `json:"permalink"`
	URL       string `json:"url"`
	Title     string `json:"title"`
}

// ParseHugoIndex parses Hugo /index.json
func ParseHugoIndex(content []byte, baseURL string) ([]string, error) {
	var entries []HugoSearchEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("empty Hugo index")
	}

	urls := make([]string, 0, len(entries))
	for _, entry := range entries {
		urlStr := entry.Permalink
		if urlStr == "" {
			urlStr = entry.URL
		}
		if urlStr != "" {
			urls = append(urls, resolveDiscoveryURL(urlStr, baseURL))
		}
	}
	return urls, nil
}

// ParseGenericSearchIndex parses generic search.json format
func ParseGenericSearchIndex(content []byte, baseURL string) ([]string, error) {
	// Try array format first
	var entries []map[string]interface{}
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("empty search index")
	}

	urls := make([]string, 0, len(entries))
	for _, entry := range entries {
		// Try common URL field names
		for _, field := range []string{"url", "permalink", "href", "location", "path"} {
			if val, ok := entry[field].(string); ok && val != "" {
				urls = append(urls, resolveDiscoveryURL(val, baseURL))
				break
			}
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in search index")
	}
	return urls, nil
}

// ParseVitePressHashmap parses VitePress hashmap.json
func ParseVitePressHashmap(content []byte, baseURL string) ([]string, error) {
	var hashmap map[string]string
	if err := json.Unmarshal(content, &hashmap); err != nil {
		return nil, err
	}

	if len(hashmap) == 0 {
		return nil, fmt.Errorf("empty VitePress hashmap")
	}

	urls := make([]string, 0, len(hashmap))
	for path := range hashmap {
		// VitePress hashmap keys are like "guide_getting-started.md"
		// Convert to URL path: /guide/getting-started
		urlPath := strings.ReplaceAll(path, "_", "/")
		urlPath = strings.TrimSuffix(urlPath, ".md")

		fullURL := strings.TrimSuffix(baseURL, "/") + "/" + urlPath
		urls = append(urls, fullURL)
	}
	return urls, nil
}

// resolveDiscoveryURL resolves a potentially relative URL against a base URL
func resolveDiscoveryURL(href, baseURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL + "/" + strings.TrimPrefix(href, "/")
	}

	ref, err := url.Parse(href)
	if err != nil {
		return baseURL + "/" + strings.TrimPrefix(href, "/")
	}

	return parsed.ResolveReference(ref).String()
}
