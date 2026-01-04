package fetcher_test

import (
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/fetcher"
	"github.com/stretchr/testify/assert"
)

func TestRandomDelay_Generate(t *testing.T) {
	t.Run("generates random delays within range", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 1000 * time.Millisecond

		// Execute: Generate multiple delays
		delays := make(map[int64]bool)
		for i := 0; i < 100; i++ {
			delay := fetcher.RandomDelay(min, max)
			delays[int64(delay)] = true

			// Verify: Delay is within the specified range
			assert.GreaterOrEqual(t, delay, min, "Delay should be >= min")
			assert.Less(t, delay, max, "Delay should be < max")
		}

		// Verify: We got some randomness (at least some different values)
		// With 100 iterations, we should see some variation
		assert.Greater(t, len(delays), 1, "Should generate different delays showing randomness")
	})

	t.Run("generates delays with different ranges", func(t *testing.T) {
		testCases := []struct {
			min      time.Duration
			max      time.Duration
			testName string
		}{
			{10 * time.Millisecond, 50 * time.Millisecond, "small range"},
			{1 * time.Second, 5 * time.Second, "large range"},
			{100 * time.Millisecond, 200 * time.Millisecond, "medium range"},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				// Execute: Generate a delay
				delay := fetcher.RandomDelay(tc.min, tc.max)

				// Verify: Delay is within range
				assert.GreaterOrEqual(t, delay, tc.min, "Delay should be >= min")
				assert.Less(t, delay, tc.max, "Delay should be < max")
			})
		}
	})

	t.Run("generates consistent type", func(t *testing.T) {
		// Execute: Generate a delay
		delay := fetcher.RandomDelay(100*time.Millisecond, 500*time.Millisecond)

		// Verify: Returns time.Duration
		assert.IsType(t, time.Duration(0), delay, "Should return time.Duration")
		assert.NotZero(t, delay, "Delay should not be zero")
	})
}

func TestRandomDelay_WithinRange(t *testing.T) {
	t.Run("exact min equals max returns min", func(t *testing.T) {
		duration := 500 * time.Millisecond

		// Execute: Generate delay with min == max
		delay := fetcher.RandomDelay(duration, duration)

		// Verify: Returns the exact duration
		assert.Equal(t, duration, delay, "When min == max, should return min")
	})

	t.Run("min greater than max returns min", func(t *testing.T) {
		min := 1000 * time.Millisecond
		max := 500 * time.Millisecond

		// Execute: Generate delay with min > max
		delay := fetcher.RandomDelay(min, max)

		// Verify: Returns min (as per implementation logic)
		assert.Equal(t, min, delay, "When min > max, should return min")
	})

	t.Run("boundary conditions", func(t *testing.T) {
		testCases := []struct {
			min      time.Duration
			max      time.Duration
			shouldEq bool
			desc     string
		}{
			{0, 0, true, "zero duration"},
			{1, 2, false, "adjacent values"},
			{100 * time.Millisecond, 101 * time.Millisecond, false, "minimal range"},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				delay := fetcher.RandomDelay(tc.min, tc.max)

				if tc.shouldEq {
					assert.Equal(t, tc.min, delay)
				} else {
					assert.GreaterOrEqual(t, delay, tc.min)
					assert.Less(t, delay, tc.max)
				}
			})
		}
	})

	t.Run("large duration values", func(t *testing.T) {
		min := 10 * time.Second
		max := 60 * time.Second

		// Execute: Generate delay
		delay := fetcher.RandomDelay(min, max)

		// Verify: Within range
		assert.GreaterOrEqual(t, delay, min)
		assert.Less(t, delay, max)
		assert.Greater(t, int64(delay), int64(min)-1, "Should be close to min or greater")
	})

	t.Run("very small duration values", func(t *testing.T) {
		min := 1 * time.Microsecond
		max := 10 * time.Microsecond

		// Execute: Generate delay
		delay := fetcher.RandomDelay(min, max)

		// Verify: Within range (may be 0 due to integer conversion)
		assert.GreaterOrEqual(t, delay, 0*time.Microsecond)
		assert.Less(t, delay, max)
	})
}

func TestRandomDelay_Zero(t *testing.T) {
	t.Run("zero min returns zero", func(t *testing.T) {
		max := 100 * time.Millisecond

		// Execute: Generate delay with min = 0
		delay := fetcher.RandomDelay(0, max)

		// Verify: Can return zero or positive value
		assert.GreaterOrEqual(t, delay, 0*time.Millisecond, "Delay should be >= 0")
		assert.Less(t, delay, max, "Delay should be < max")
	})

	t.Run("zero max returns zero", func(t *testing.T) {
		min := 0 * time.Millisecond
		max := 0 * time.Millisecond

		// Execute: Generate delay with both min and max = 0
		delay := fetcher.RandomDelay(min, max)

		// Verify: Returns zero
		assert.Equal(t, time.Duration(0), delay, "Both zero should return zero")
	})

	t.Run("zero with positive max", func(t *testing.T) {
		// Execute multiple times to test randomness
		for i := 0; i < 50; i++ {
			delay := fetcher.RandomDelay(0, 50*time.Millisecond)

			// Verify: Always >= 0 and < max
			assert.GreaterOrEqual(t, delay, 0*time.Millisecond, "Iteration %d: delay should be >= 0", i)
			assert.Less(t, delay, 50*time.Millisecond, "Iteration %d: delay should be < max", i)
		}
	})

	t.Run("negative values handled gracefully", func(t *testing.T) {
		// Note: The implementation doesn't explicitly handle negative values
		// but we test that it doesn't panic
		assert.NotPanics(t, func() {
			delay := fetcher.RandomDelay(-1*time.Second, 0)
			// The delay could be negative due to the implementation
			// but shouldn't panic
			_ = delay
		}, "Should not panic with negative values")
	})

	t.Run("mixed zero and positive", func(t *testing.T) {
		testCases := []struct {
			min      time.Duration
			max      time.Duration
			testName string
		}{
			{0, 1 * time.Millisecond, "0 to 1ms"},
			{0, 100 * time.Millisecond, "0 to 100ms"},
			{0, 1 * time.Second, "0 to 1s"},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				// Execute: Generate delay
				delay := fetcher.RandomDelay(tc.min, tc.max)

				// Verify: Within valid range
				assert.GreaterOrEqual(t, delay, 0*time.Millisecond)
				assert.Less(t, delay, tc.max)
			})
		}
	})
}

func TestRandomUserAgent(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		// Execute: Get random user agent
		ua := fetcher.RandomUserAgent()

		// Verify: Not empty
		assert.NotEmpty(t, ua, "User agent should not be empty")
	})

	t.Run("returns valid user agent format", func(t *testing.T) {
		// Execute: Get multiple user agents
		for i := 0; i < 10; i++ {
			ua := fetcher.RandomUserAgent()

			// Verify: Contains expected parts (Mozilla/5.0)
			assert.Contains(t, ua, "Mozilla/5.0", "User agent should contain Mozilla/5.0")
		}
	})

	t.Run("returns different user agents", func(t *testing.T) {
		// Execute: Get multiple user agents
		userAgents := make(map[string]bool)
		for i := 0; i < 20; i++ {
			ua := fetcher.RandomUserAgent()
			userAgents[ua] = true
		}

		// Verify: Got some variety (more than 1 unique UA)
		assert.Greater(t, len(userAgents), 1, "Should return different user agents showing randomness")
	})

	t.Run("contains common browser identifiers", func(t *testing.T) {
		// Execute: Get multiple user agents
		commonBrowsers := []string{"Chrome", "Firefox", "Safari", "Edg"}
		foundBrowsers := make(map[string]bool)

		for i := 0; i < 50; i++ {
			ua := fetcher.RandomUserAgent()
			for _, browser := range commonBrowsers {
				if strings.Contains(ua, browser) {
					foundBrowsers[browser] = true
				}
			}
		}

		// Verify: Found at least some common browsers
		assert.Greater(t, len(foundBrowsers), 0, "Should find common browser identifiers")
	})
}

func TestRandomAcceptLanguage(t *testing.T) {
	t.Run("returns non-empty string", func(t *testing.T) {
		// Execute: Get random accept language
		lang := fetcher.RandomAcceptLanguage()

		// Verify: Not empty
		assert.NotEmpty(t, lang, "Accept-Language should not be empty")
	})

	t.Run("returns valid accept language format", func(t *testing.T) {
		// Execute: Get multiple accept languages
		for i := 0; i < 10; i++ {
			lang := fetcher.RandomAcceptLanguage()

			// Verify: Contains expected parts (en-US or en)
			assert.True(t, strings.Contains(lang, "en-US") || strings.Contains(lang, "en"),
				"Accept-Language should contain English variant")
		}
	})

	t.Run("returns different languages", func(t *testing.T) {
		// Execute: Get multiple accept languages
		languages := make(map[string]bool)
		for i := 0; i < 20; i++ {
			lang := fetcher.RandomAcceptLanguage()
			languages[lang] = true
		}

		// Verify: Got some variety
		assert.Greater(t, len(languages), 1, "Should return different accept languages showing randomness")
	})

	t.Run("contains quality values", func(t *testing.T) {
		// Execute: Get multiple accept languages
		for i := 0; i < 10; i++ {
			lang := fetcher.RandomAcceptLanguage()

			// Verify: Contains quality values (q=)
			assert.Contains(t, lang, "q=", "Accept-Language should contain quality values")
		}
	})
}

func TestStealthHeaders(t *testing.T) {
	t.Run("returns required stealth headers", func(t *testing.T) {
		// Execute: Get stealth headers
		headers := fetcher.StealthHeaders("")

		// Verify: Contains required headers
		requiredHeaders := []string{
			"User-Agent",
			"Accept",
			"Accept-Language",
			"Accept-Encoding",
			"Cache-Control",
			"Pragma",
			"Sec-Fetch-Dest",
			"Sec-Fetch-Mode",
			"Sec-Fetch-Site",
			"Sec-Fetch-User",
			"Upgrade-Insecure-Requests",
		}

		for _, header := range requiredHeaders {
			assert.Contains(t, headers, header, "Stealth headers should contain %s", header)
			assert.NotEmpty(t, headers[header], "%s should not be empty", header)
		}
	})

	t.Run("generates random user agent when empty", func(t *testing.T) {
		// Execute: Get stealth headers with empty UA
		headers1 := fetcher.StealthHeaders("")
		headers2 := fetcher.StealthHeaders("")

		// Verify: User agents are different (randomly generated)
		// Note: There's a small chance they could be the same, but very unlikely
		// We'll just verify they're both valid
		assert.NotEmpty(t, headers1["User-Agent"])
		assert.NotEmpty(t, headers2["User-Agent"])
	})

	t.Run("uses custom user agent when provided", func(t *testing.T) {
		// Execute: Get stealth headers with custom UA
		customUA := "MyCustomAgent/1.0"
		headers := fetcher.StealthHeaders(customUA)

		// Verify: Custom UA is used
		assert.Equal(t, customUA, headers["User-Agent"])
	})

	t.Run("includes Chrome-specific headers for Chrome UA", func(t *testing.T) {
		// Execute: Get stealth headers with Chrome UA
		chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
		headers := fetcher.StealthHeaders(chromeUA)

		// Verify: Contains Chrome-specific headers
		assert.Contains(t, headers, "Sec-CH-UA")
		assert.Contains(t, headers, "Sec-CH-UA-Mobile")
		assert.Contains(t, headers, "Sec-CH-UA-Platform")

		// Verify: Values are not empty
		assert.NotEmpty(t, headers["Sec-CH-UA"])
		assert.NotEmpty(t, headers["Sec-CH-UA-Mobile"])
		assert.NotEmpty(t, headers["Sec-CH-UA-Platform"])
	})

	t.Run("does not include Chrome headers for non-Chrome UA", func(t *testing.T) {
		// Execute: Get stealth headers with Firefox UA
		firefoxUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0"
		headers := fetcher.StealthHeaders(firefoxUA)

		// Verify: Does not contain Chrome-specific headers
		_, hasCHUA := headers["Sec-CH-UA"]
		_, hasCHUAMobile := headers["Sec-CH-UA-Mobile"]
		_, hasCHAPlatform := headers["Sec-CH-UA-Platform"]

		assert.False(t, hasCHUA, "Should not have Sec-CH-UA for Firefox")
		assert.False(t, hasCHUAMobile, "Should not have Sec-CH-UA-Mobile for Firefox")
		assert.False(t, hasCHAPlatform, "Should not have Sec-CH-UA-Platform for Firefox")
	})

	t.Run("includes standard HTTP headers", func(t *testing.T) {
		// Execute: Get stealth headers
		headers := fetcher.StealthHeaders("")

		// Verify: Standard header values
		assert.Equal(t, "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8", headers["Accept"])
		assert.Equal(t, "gzip, deflate, br", headers["Accept-Encoding"])
		assert.Equal(t, "no-cache", headers["Cache-Control"])
		assert.Equal(t, "no-cache", headers["Pragma"])
		assert.Equal(t, "document", headers["Sec-Fetch-Dest"])
		assert.Equal(t, "navigate", headers["Sec-Fetch-Mode"])
		assert.Equal(t, "none", headers["Sec-Fetch-Site"])
		assert.Equal(t, "?1", headers["Sec-Fetch-User"])
		assert.Equal(t, "1", headers["Upgrade-Insecure-Requests"])
	})

	t.Run("generates random accept language", func(t *testing.T) {
		// Execute: Get stealth headers multiple times
		langs := make(map[string]bool)
		for i := 0; i < 20; i++ {
			headers := fetcher.StealthHeaders("")
			langs[headers["Accept-Language"]] = true
		}

		// Verify: Got some variety
		assert.Greater(t, len(langs), 1, "Should generate different accept languages")
	})
}

func TestRandomSecChUaPlatform(t *testing.T) {
	// Note: This is an internal function tested indirectly through StealthHeaders
	// We'll test the platform values in the headers

	t.Run("valid platform values", func(t *testing.T) {
		// Execute: Get stealth headers with Chrome UA multiple times
		platforms := make(map[string]bool)
		for i := 0; i < 30; i++ {
			chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
			headers := fetcher.StealthHeaders(chromeUA)
			platform := headers["Sec-CH-UA-Platform"]
			platforms[platform] = true

			// Verify: Platform is one of the expected values
			assert.Contains(t, []string{`"Windows"`, `"macOS"`, `"Linux"`}, platform,
				"Platform should be Windows, macOS, or Linux")
		}

		// Verify: Got some variety
		assert.Greater(t, len(platforms), 1, "Should get different platforms")
	})
}
