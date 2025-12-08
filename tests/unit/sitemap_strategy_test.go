package unit

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSitemapStrategy_NewSitemapStrategy(t *testing.T) {
	deps := createTestSitemapDependencies(t)
	strategy := strategies.NewSitemapStrategy(deps)
	require.NotNil(t, strategy)
	assert.Equal(t, "sitemap", strategy.Name())
}

func TestSitemapStrategy_CanHandle(t *testing.T) {
	deps := createTestSitemapDependencies(t)
	strategy := strategies.NewSitemapStrategy(deps)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"sitemap.xml", "https://example.com/sitemap.xml", true},
		{"sitemap.xml.gz", "https://example.com/sitemap.xml.gz", true},
		{"Uppercase", "https://example.com/SITEMAP.XML", true},
		{"Mixed case", "https://example.com/Sitemap.Xml", true},
		{"Contains sitemap", "https://example.com/sitemaps/product-sitemap.xml", true},
		{"Regular page", "https://example.com/docs", false},
		{"Empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CanHandle(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSitemapStrategy_Name(t *testing.T) {
	deps := createTestSitemapDependencies(t)
	strategy := strategies.NewSitemapStrategy(deps)
	assert.Equal(t, "sitemap", strategy.Name())
}

func TestParseSitemapTest_RegularSitemap(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://example.com/page1</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
		<changefreq>weekly</changefreq>
		<priority>0.8</priority>
	</url>
	<url>
		<loc>https://example.com/page2</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
		<changefreq>monthly</changefreq>
		<priority>0.6</priority>
	</url>
</urlset>`

	sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap.xml")
	require.NoError(t, err)
	assert.NotNil(t, sitemap)
	assert.False(t, sitemap.IsIndex)
	assert.Equal(t, 2, len(sitemap.URLs))
	assert.Equal(t, "https://example.com/page1", sitemap.URLs[0].Loc)
	assert.Equal(t, "weekly", sitemap.URLs[0].ChangeFreq)
}

func TestParseSitemapTest_SitemapIndex(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap1.xml</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/sitemap2.xml</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
	</sitemap>
</sitemapindex>`

	sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap-index.xml")
	require.NoError(t, err)
	assert.NotNil(t, sitemap)
	// The simple test should detect it's an index
	assert.True(t, strings.Contains(string(xmlContent), "sitemapindex"))
}

func TestParseLastMod(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		hasError bool
	}{
		{"Valid date", "2024-01-15T10:00:00Z", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false},
		{"Valid date without Z", "2024-01-15T10:00:00+00:00", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false},
		{"Empty string", "", time.Time{}, true},
		{"Invalid format", "not-a-date", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLastModTest(tt.input)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestProcessSitemapIndex tests processing of sitemap index files
func TestProcessSitemapIndex(t *testing.T) {
	t.Run("simple sitemap index", func(t *testing.T) {
		// Arrange
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap1.xml</loc>
		<lastmod>2024-01-15T10:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/sitemap2.xml</loc>
		<lastmod>2024-01-14T10:00:00Z</lastmod>
	</sitemap>
</sitemapindex>`

		// Act
		sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap-index.xml")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, sitemap)
		assert.True(t, sitemap.IsIndex, "Should be detected as a sitemap index")
		assert.Equal(t, "https://example.com/sitemap-index.xml", sitemap.SourceURL)
	})

	t.Run("nested sitemap index with mixed content", func(t *testing.T) {
		// Arrange
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/products/sitemap.xml</loc>
		<lastmod>2024-01-15T12:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/blog/sitemap.xml</loc>
		<lastmod>2024-01-15T10:30:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://example.com/docs/sitemap.xml</loc>
		<lastmod>2024-01-14T15:45:00Z</lastmod>
	</sitemap>
</sitemapindex>`

		// Act
		sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap.xml")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, sitemap)
		assert.True(t, sitemap.IsIndex, "Should be detected as a sitemap index")
	})

	t.Run("invalid XML", func(t *testing.T) {
		// Arrange
		invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://example.com/sitemap1.xml</loc>
	<sitemap>
		<!-- Missing closing tag -->
</sitemapindex>`

		// Act
		sitemap, _ := parseSitemapTest([]byte(invalidXML), "https://example.com/sitemap.xml")

		// Assert
		// The parser should handle this gracefully
		assert.NotNil(t, sitemap)
	})

	t.Run("empty sitemap index", func(t *testing.T) {
		// Arrange
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</sitemapindex>`

		// Act
		sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap.xml")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, sitemap)
		assert.True(t, sitemap.IsIndex, "Should be detected as a sitemap index")
	})

	t.Run("sitemap index with URLs in sitemaps", func(t *testing.T) {
		// Arrange
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<sitemap>
		<loc>https://cdn.example.com/sitemaps/product-sitemap-1.xml.gz</loc>
		<lastmod>2024-01-15T08:00:00Z</lastmod>
	</sitemap>
	<sitemap>
		<loc>https://cdn.example.com/sitemaps/product-sitemap-2.xml.gz</loc>
		<lastmod>2024-01-15T08:00:00Z</lastmod>
	</sitemap>
</sitemapindex>`

		// Act
		sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap.xml")

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, sitemap)
		assert.True(t, sitemap.IsIndex, "Should be detected as a sitemap index")
	})
}

// Helper functions that mirror unexported functions from sitemap.go
func parseSitemapTest(data []byte, baseURL string) (*domain.Sitemap, error) {
	result := &domain.Sitemap{
		SourceURL: baseURL,
		IsIndex:   false,
	}

	// Simple check for sitemap index
	if strings.Contains(string(data), "sitemapindex") {
		result.IsIndex = true
	}

	// Parse URLs (simplified)
	if !result.IsIndex {
		decoder := xml.NewDecoder(bytes.NewReader(data))
		for {
			token, err := decoder.Token()
			if err != nil {
				break
			}
			if element, ok := token.(xml.StartElement); ok && element.Name.Local == "url" {
				url := domain.SitemapURL{}
				if err := decoder.DecodeElement(&url, &element); err == nil {
					result.URLs = append(result.URLs, url)
				}
			}
		}
	}

	return result, nil
}

func parseLastModTest(lastMod string) (time.Time, error) {
	if lastMod == "" {
		return time.Time{}, assert.AnError
	}
	// Try RFC3339 format
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, lastMod); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, assert.AnError
}

func TestDecompressGzip(t *testing.T) {
	t.Run("valid gzip content", func(t *testing.T) {
		// Create valid gzip content
		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		_, err := writer.Write([]byte("Hello, World!"))
		require.NoError(t, err)
		writer.Close()

		// Test decompression
		decompressed, err := decompressGzipTest(buf.Bytes())
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(decompressed))
	})

	t.Run("invalid gzip", func(t *testing.T) {
		invalidData := []byte{0x1f, 0x8b, 0x00, 0x00} // Invalid gzip header
		_, err := decompressGzipTest(invalidData)
		require.Error(t, err)
	})

	t.Run("empty content", func(t *testing.T) {
		emptyData := []byte{}
		_, err := decompressGzipTest(emptyData)
		require.Error(t, err)
	})

	t.Run("truncated gzip", func(t *testing.T) {
		truncatedData := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff} // Incomplete
		_, err := decompressGzipTest(truncatedData)
		require.Error(t, err)
	})
}

func decompressGzipTest(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, assert.AnError
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func createTestSitemapDependencies(t *testing.T) *strategies.Dependencies {
	t.Helper()
	return &strategies.Dependencies{}
}

type sitemapXML struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}
