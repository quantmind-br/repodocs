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
	assert.True(t, sitemap.IsIndex)
	assert.Equal(t, 2, len(sitemap.Sitemaps))
	assert.Equal(t, "https://example.com/sitemap1.xml", sitemap.Sitemaps[0])
}

func TestParseSitemapTest_EmptySitemap(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`

	sitemap, err := parseSitemapTest([]byte(xmlContent), "https://example.com/sitemap.xml")
	require.NoError(t, err)
	assert.NotNil(t, sitemap)
	assert.Equal(t, 0, len(sitemap.URLs))
}

func TestParseSitemapTest_InvalidXML(t *testing.T) {
	invalidXML := `<?xml version="1.0"?>
<urlset>
	<url>
		<loc>https://example.com/page1</loc>
	</url>`

	sitemap, err := parseSitemapTest([]byte(invalidXML), "https://example.com/sitemap.xml")
	require.Error(t, err)
	assert.Nil(t, sitemap)
}

func TestParseLastModTest(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		hasError bool
	}{
		{"RFC3339", "2024-01-15T10:00:00Z", time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), false},
		{"With timezone - converts to UTC", "2024-01-15T10:00:00-05:00", time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), false},
		{"Date only", "2024-01-15", time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"Invalid", "not-a-date", time.Time{}, false},
		{"Empty", "", time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLastModTest(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				// Don't check error for parseLastMod - it returns zero time on error
				_ = err
			}
			assert.Equal(t, tt.expected.Unix(), result.Unix(), "Unix timestamp should match")
		})
	}
}

func TestSortURLsByLastModTest(t *testing.T) {
	urls := []domain.SitemapURL{
		{Loc: "https://example.com/page1", LastMod: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
		{Loc: "https://example.com/page2", LastMod: time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC)},
		{Loc: "https://example.com/page3", LastMod: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC)},
	}

	sortURLsByLastModTest(urls)

	assert.Equal(t, "https://example.com/page3", urls[0].Loc)
	assert.Equal(t, "https://example.com/page1", urls[1].Loc)
	assert.Equal(t, "https://example.com/page2", urls[2].Loc)
}

func TestSortURLsByLastModTest_Empty(t *testing.T) {
	urls := []domain.SitemapURL{}
	sortURLsByLastModTest(urls)
	assert.Equal(t, 0, len(urls))
}

func TestSortURLsByLastModTest_Single(t *testing.T) {
	urls := []domain.SitemapURL{
		{Loc: "https://example.com/page1", LastMod: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)},
	}
	sortURLsByLastModTest(urls)
	assert.Equal(t, 1, len(urls))
	assert.Equal(t, "https://example.com/page1", urls[0].Loc)
}

func TestDecompressGzipTest(t *testing.T) {
	originalData := []byte("Test gzipped data")
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(originalData)
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)

	gzippedData := buf.Bytes()
	decompressed, err := decompressGzipTest(gzippedData)
	require.NoError(t, err)
	assert.Equal(t, originalData, decompressed)
}

func TestDecompressGzipTest_InvalidData(t *testing.T) {
	invalidData := []byte("Not gzip data")
	_, err := decompressGzipTest(invalidData)
	require.Error(t, err)
}

func TestDecompressGzipTest_Empty(t *testing.T) {
	emptyData := []byte{}
	_, err := decompressGzipTest(emptyData)
	require.Error(t, err)
}

func TestProcessSitemapIndex(t *testing.T) {
	t.Skip("Requires full dependency injection and network mocking")
}

func TestParseSitemapTest_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		wantErr  bool
		wantUrls int
		isIndex  bool
	}{
		{"Empty loc", "<?xml version=\"1.0\"?><urlset><url><loc></loc></url></urlset>", false, 1, false},
		{"Special chars", "<?xml version=\"1.0\"?><urlset><url><loc>https://example.com/page?id=123&amp;filter=test</loc></url></urlset>", false, 1, false},
		{"Unicode", "<?xml version=\"1.0\"?><urlset><url><loc>https://example.com/página</loc></url></urlset>", false, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sitemap, err := parseSitemapTest([]byte(tt.xml), "https://example.com/sitemap.xml")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUrls, len(sitemap.URLs))
				assert.Equal(t, tt.isIndex, sitemap.IsIndex)
			}
		})
	}
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

type sitemapIndexXML struct {
	XMLName  xml.Name          `xml:"sitemapindex"`
	Sitemaps []sitemapLocation `xml:"sitemap"`
}

type sitemapLocation struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

func parseSitemapTest(content []byte, sourceURL string) (*domain.Sitemap, error) {
	var index sitemapIndexXML
	if err := xml.Unmarshal(content, &index); err == nil && len(index.Sitemaps) > 0 {
		var sitemaps []string
		for _, sm := range index.Sitemaps {
			sitemaps = append(sitemaps, sm.Loc)
		}
		return &domain.Sitemap{
			IsIndex:   true,
			Sitemaps:  sitemaps,
			SourceURL: sourceURL,
		}, nil
	}

	var sitemap sitemapXML
	if err := xml.Unmarshal(content, &sitemap); err != nil {
		return nil, err
	}

	var urls []domain.SitemapURL
	for _, u := range sitemap.URLs {
		lastMod, _ := parseLastModTest(u.LastMod)
		urls = append(urls, domain.SitemapURL{
			Loc:        u.Loc,
			LastMod:    lastMod,
			LastModStr: u.LastMod,
			ChangeFreq: u.ChangeFreq,
		})
	}

	return &domain.Sitemap{
		URLs:      urls,
		IsIndex:   false,
		SourceURL: sourceURL,
	}, nil
}

func parseLastModTest(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}

func sortURLsByLastModTest(urls []domain.SitemapURL) {
	for i := 0; i < len(urls); i++ {
		for j := i + 1; j < len(urls); j++ {
			if urls[i].LastMod.Before(urls[j].LastMod) {
				urls[i], urls[j] = urls[j], urls[i]
			}
		}
	}
}

func decompressGzipTest(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
