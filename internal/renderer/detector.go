package renderer

import (
	"regexp"
	"strings"
)

// SPA detection patterns
var (
	// React patterns
	reactPatterns = []string{
		`<div id="root"></div>`,
		`<div id="root"/>`,
		`<div id="app"></div>`,
		`<div id="app"/>`,
		`data-reactroot`,
		`__REACT_DEVTOOLS_GLOBAL_HOOK__`,
	}

	// Vue patterns
	vuePatterns = []string{
		`<div id="app"></div>`,
		`<div id="app"/>`,
		`__VUE__`,
		`v-cloak`,
		`Vue.createApp`,
	}

	// Next.js patterns
	nextPatterns = []string{
		`<div id="__next"></div>`,
		`<div id="__next"/>`,
		`__NEXT_DATA__`,
		`_next/static`,
	}

	// Nuxt patterns
	nuxtPatterns = []string{
		`__NUXT__`,
		`window.__NUXT__`,
		`<div id="__nuxt">`,
	}

	// Angular patterns
	angularPatterns = []string{
		`ng-version`,
		`ng-app`,
		`ng-controller`,
		`<app-root>`,
	}

	// Svelte patterns
	sveltePatterns = []string{
		`__svelte`,
		`svelte-`,
	}

	// Generic SPA indicators
	spaIndicators = []string{
		`window.__INITIAL_STATE__`,
		`window.__STATE__`,
		`window.__PRELOADED_STATE__`,
	}
)

// contentMinLength is the minimum content length to consider a page as rendered
const contentMinLength = 500

// scriptTagRegex matches script tags
var scriptTagRegex = regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)

// htmlTagRegex matches HTML tags
var htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

// NeedsJSRendering detects if a page needs JavaScript rendering
func NeedsJSRendering(html string) bool {
	// Check for SPA framework patterns
	if hasSPAPattern(html) {
		return true
	}

	// Check content length without scripts
	contentWithoutScripts := scriptTagRegex.ReplaceAllString(html, "")
	textContent := htmlTagRegex.ReplaceAllString(contentWithoutScripts, "")
	textContent = strings.TrimSpace(textContent)

	// If there's very little content but many scripts, likely a SPA
	if len(textContent) < contentMinLength {
		scriptCount := strings.Count(strings.ToLower(html), "<script")
		if scriptCount > 3 {
			return true
		}
	}

	return false
}

// hasSPAPattern checks if the HTML contains any SPA framework patterns
func hasSPAPattern(html string) bool {
	htmlLower := strings.ToLower(html)

	allPatterns := append([]string{}, reactPatterns...)
	allPatterns = append(allPatterns, vuePatterns...)
	allPatterns = append(allPatterns, nextPatterns...)
	allPatterns = append(allPatterns, nuxtPatterns...)
	allPatterns = append(allPatterns, angularPatterns...)
	allPatterns = append(allPatterns, sveltePatterns...)
	allPatterns = append(allPatterns, spaIndicators...)

	for _, pattern := range allPatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// DetectFramework attempts to detect which SPA framework is being used
func DetectFramework(html string) string {
	htmlLower := strings.ToLower(html)

	for _, pattern := range nextPatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "Next.js"
		}
	}

	for _, pattern := range nuxtPatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "Nuxt"
		}
	}

	for _, pattern := range reactPatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "React"
		}
	}

	for _, pattern := range vuePatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "Vue"
		}
	}

	for _, pattern := range angularPatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "Angular"
		}
	}

	for _, pattern := range sveltePatterns {
		if strings.Contains(htmlLower, strings.ToLower(pattern)) {
			return "Svelte"
		}
	}

	return "Unknown"
}

// HasDynamicContent checks for indicators of dynamic content loading
func HasDynamicContent(html string) bool {
	indicators := []string{
		"loading...",
		"loading…",
		"please wait",
		"spinner",
		"skeleton",
		"lazy-load",
		"lazyload",
		"infinite-scroll",
	}

	htmlLower := strings.ToLower(html)
	for _, indicator := range indicators {
		if strings.Contains(htmlLower, indicator) {
			return true
		}
	}

	return false
}
