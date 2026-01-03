package strategies_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocsRSStrategy(t *testing.T) {
	t.Run("with nil deps", func(t *testing.T) {
		strategy := strategies.NewDocsRSStrategy(nil)
		assert.NotNil(t, strategy)
		assert.Equal(t, "docsrs", strategy.Name())
	})

	t.Run("with valid deps", func(t *testing.T) {
		logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})
		tmpDir := t.TempDir()
		writer := output.NewWriter(output.WriterOptions{
			BaseDir: tmpDir,
			Force:   true,
		})

		deps := &strategies.Dependencies{
			Logger: logger,
			Writer: writer,
		}

		strategy := strategies.NewDocsRSStrategy(deps)
		assert.NotNil(t, strategy)
		assert.Equal(t, "docsrs", strategy.Name())
	})
}

func TestDocsRSStrategy_Name(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	assert.Equal(t, "docsrs", strategy.Name())
}

func TestDocsRSStrategy_CanHandle(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"crate root", "https://docs.rs/serde", true},
		{"crate with version", "https://docs.rs/serde/1.0.0", true},
		{"crate latest", "https://docs.rs/serde/latest/serde/", true},
		{"module path", "https://docs.rs/serde/1.0.0/serde/de", true},
		{"struct page", "https://docs.rs/tokio/1.0.0/tokio/net/struct.TcpStream.html", true},
		{"trait page", "https://docs.rs/serde/1.0.0/serde/trait.Serialize.html", true},
		{"crate info page", "https://docs.rs/crate/serde/1.0.0", true},
		{"with trailing slash", "https://docs.rs/tokio/latest/tokio/", true},
		{"nested module", "https://docs.rs/serde/1.0.0/serde/de/value/index.html", true},

		{"source view", "https://docs.rs/serde/1.0.0/src/serde/lib.rs.html", false},
		{"source view alt", "https://docs.rs/crate/serde/1.0.0/source/", false},
		{"github", "https://github.com/serde-rs/serde", false},
		{"crates.io", "https://crates.io/crates/serde", false},
		{"pkg.go.dev", "https://pkg.go.dev/encoding/json", false},
		{"empty", "", false},
		{"malformed", "not-a-url", false},
		{"other domain", "https://example.com/docs.rs/serde", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategy.CanHandle(tc.url)
			assert.Equal(t, tc.expected, result, "URL: %s", tc.url)
		})
	}
}

func TestDocsRSStrategy_ExtractMetadata(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "fixtures", "docsrs")

	tests := []struct {
		name          string
		fixtureFile   string
		wantItemType  string
		wantStability string
		wantTitle     string
	}{
		{
			name:          "crate root page",
			fixtureFile:   "serde_crate_root.html",
			wantItemType:  "module",
			wantStability: "stable",
			wantTitle:     "Crate serde",
		},
		{
			name:          "module page",
			fixtureFile:   "serde_module.html",
			wantItemType:  "module",
			wantStability: "stable",
			wantTitle:     "Module serde::de",
		},
		{
			name:          "struct page",
			fixtureFile:   "serde_struct.html",
			wantItemType:  "struct",
			wantStability: "stable",
			wantTitle:     "Struct serde::de::Deserializer",
		},
		{
			name:          "trait page",
			fixtureFile:   "tokio_async_trait.html",
			wantItemType:  "trait",
			wantStability: "stable",
			wantTitle:     "Trait tokio::io::AsyncRead",
		},
		{
			name:          "deprecated item",
			fixtureFile:   "with_deprecated.html",
			wantItemType:  "function",
			wantStability: "deprecated",
			wantTitle:     "Function example::old_function",
		},
		{
			name:          "nightly only item",
			fixtureFile:   "nightly_only.html",
			wantItemType:  "trait",
			wantStability: "nightly",
			wantTitle:     "Trait example::NightlyFeature",
		},
		{
			name:          "minimal page",
			fixtureFile:   "minimal.html",
			wantItemType:  "module",
			wantStability: "stable",
			wantTitle:     "Crate minimal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixturePath := filepath.Join(fixturesDir, tc.fixtureFile)
			content, err := os.ReadFile(fixturePath)
			require.NoError(t, err, "Failed to read fixture file: %s", tc.fixtureFile)

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
			require.NoError(t, err)

			strategy := strategies.NewDocsRSStrategy(nil)
			baseInfo := &strategies.DocsRSURL{
				CrateName: "test",
				Version:   "1.0.0",
			}

			meta := strategy.ExtractMetadataForTest(doc, baseInfo)

			assert.Equal(t, tc.wantItemType, meta.ItemType, "ItemType mismatch")
			assert.Equal(t, tc.wantStability, meta.Stability, "Stability mismatch")
			assert.Contains(t, meta.Title, tc.wantTitle, "Title mismatch")
		})
	}
}

func TestDocsRSStrategy_ShouldCrawl(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)
	baseInfo := &strategies.DocsRSURL{
		CrateName: "serde",
		Version:   "1.0.0",
	}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"same crate module", "https://docs.rs/serde/1.0.0/serde/de/", true},
		{"same crate struct", "https://docs.rs/serde/1.0.0/serde/struct.Serialize.html", true},
		{"same crate trait", "https://docs.rs/serde/1.0.0/serde/trait.Deserialize.html", true},
		{"same crate nested", "https://docs.rs/serde/1.0.0/serde/de/value/index.html", true},
		{"with latest version", "https://docs.rs/serde/latest/serde/de/", true},

		{"different crate", "https://docs.rs/tokio/1.0.0/tokio/", false},
		{"std library", "https://docs.rs/std/latest/std/", false},
		{"core library", "https://docs.rs/core/latest/core/", false},
		{"source view", "https://docs.rs/serde/1.0.0/src/serde/lib.rs.html", false},
		{"js file", "https://docs.rs/serde/1.0.0/search-index.js", false},
		{"css file", "https://docs.rs/serde/1.0.0/rustdoc.css", false},
		{"svg file", "https://docs.rs/serde/1.0.0/rust-logo.svg", false},
		{"different host", "https://github.com/serde-rs/serde", false},
		{"all.html", "https://docs.rs/serde/1.0.0/serde/all.html", false},
		{"static assets", "https://docs.rs/-/rustdoc.static/main.js", false},
		{"different version", "https://docs.rs/serde/2.0.0/serde/de/", false},
		{"sidebar-items.js", "https://docs.rs/serde/1.0.0/serde/sidebar-items.js", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategy.ShouldCrawlForTest(tc.url, baseInfo)
			assert.Equal(t, tc.expected, result, "URL: %s", tc.url)
		})
	}
}

func TestDocsRSStrategy_BuildStartURL(t *testing.T) {
	strategy := strategies.NewDocsRSStrategy(nil)

	tests := []struct {
		name     string
		info     *strategies.DocsRSURL
		expected string
	}{
		{
			name: "standard crate",
			info: &strategies.DocsRSURL{
				CrateName: "serde",
				Version:   "1.0.0",
			},
			expected: "https://docs.rs/serde/1.0.0/serde/",
		},
		{
			name: "latest version",
			info: &strategies.DocsRSURL{
				CrateName: "tokio",
				Version:   "latest",
			},
			expected: "https://docs.rs/tokio/latest/tokio/",
		},
		{
			name: "crate info page",
			info: &strategies.DocsRSURL{
				CrateName:   "serde",
				Version:     "1.0.0",
				IsCratePage: true,
			},
			expected: "https://docs.rs/crate/serde/1.0.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := strategy.BuildStartURLForTest(tc.info)
			assert.Equal(t, tc.expected, result)
		})
	}
}
