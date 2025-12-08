package fetcher

import (
	"math/rand"
	"time"
)

// UserAgents is a pool of real Chrome/Firefox/Safari user agents
var UserAgents = []string{
	// Chrome on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	// Chrome on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	// Chrome on Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	// Firefox on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",
	// Firefox on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:131.0) Gecko/20100101 Firefox/131.0",
	// Firefox on Linux
	"Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0",
	// Safari on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	// Edge on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
}

// AcceptLanguages are common Accept-Language header values
var AcceptLanguages = []string{
	"en-US,en;q=0.9",
	"en-GB,en;q=0.9,en-US;q=0.8",
	"en-US,en;q=0.9,es;q=0.8",
	"en-US,en;q=0.9,de;q=0.8",
	"en-US,en;q=0.9,fr;q=0.8",
	"en,en-US;q=0.9",
}

// SecChUaPlatforms are Sec-CH-UA-Platform header values
var SecChUaPlatforms = []string{
	`"Windows"`,
	`"macOS"`,
	`"Linux"`,
}

// init seeds the random number generator
func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomUserAgent returns a random user agent from the pool
func RandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

// RandomAcceptLanguage returns a random Accept-Language header value
func RandomAcceptLanguage() string {
	return AcceptLanguages[rand.Intn(len(AcceptLanguages))]
}

// RandomSecChUaPlatform returns a random Sec-CH-UA-Platform header value
func RandomSecChUaPlatform() string {
	return SecChUaPlatforms[rand.Intn(len(SecChUaPlatforms))]
}

// StealthHeaders returns a map of stealth headers for HTTP requests
func StealthHeaders(userAgent string) map[string]string {
	if userAgent == "" {
		userAgent = RandomUserAgent()
	}

	headers := map[string]string{
		"User-Agent":                userAgent,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language":           RandomAcceptLanguage(),
		"Accept-Encoding":           "gzip, deflate, br",
		"Cache-Control":             "no-cache",
		"Pragma":                    "no-cache",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}

	// Add Chrome-specific headers if using Chrome UA
	if isChrome(userAgent) {
		headers["Sec-CH-UA"] = `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
		headers["Sec-CH-UA-Mobile"] = "?0"
		headers["Sec-CH-UA-Platform"] = RandomSecChUaPlatform()
	}

	return headers
}

// isChrome checks if the user agent is Chrome
func isChrome(userAgent string) bool {
	return len(userAgent) > 0 && (contains(userAgent, "Chrome") || contains(userAgent, "Chromium"))
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RandomDelay returns a random delay between min and max duration
func RandomDelay(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	delta := max - min
	return min + time.Duration(rand.Int63n(int64(delta)))
}
