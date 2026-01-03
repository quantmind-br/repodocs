package strategies_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
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

func TestDocsRSStrategy_ParseURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantCrate    string
		wantVersion  string
		wantIsCrate  bool
		wantIsSource bool
		wantErr      bool
	}{
		{
			name:        "crate root",
			url:         "https://docs.rs/serde",
			wantCrate:   "serde",
			wantVersion: "latest",
		},
		{
			name:        "crate with version",
			url:         "https://docs.rs/serde/1.0.0",
			wantCrate:   "serde",
			wantVersion: "1.0.0",
		},
		{
			name:        "crate info page",
			url:         "https://docs.rs/crate/serde/1.0.0",
			wantCrate:   "serde",
			wantVersion: "1.0.0",
			wantIsCrate: true,
		},
		{
			name:        "crate info page latest",
			url:         "https://docs.rs/crate/tokio/latest",
			wantCrate:   "tokio",
			wantVersion: "latest",
			wantIsCrate: true,
		},
		{
			name:         "source view",
			url:          "https://docs.rs/serde/1.0.0/src/serde/lib.rs.html",
			wantCrate:    "serde",
			wantVersion:  "1.0.0",
			wantIsSource: true,
		},
		{
			name:         "crate source view",
			url:          "https://docs.rs/crate/serde/1.0.0/source/",
			wantCrate:    "serde",
			wantVersion:  "1.0.0",
			wantIsCrate:  true,
			wantIsSource: true,
		},
		{
			name:    "not docs.rs",
			url:     "https://github.com/serde-rs/serde",
			wantErr: true,
		},
		{
			name:    "empty path",
			url:     "https://docs.rs/",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := strategies.ParseDocsRSPathForTest(tc.url)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.wantCrate, result.CrateName, "CrateName mismatch")
			assert.Equal(t, tc.wantVersion, result.Version, "Version mismatch")
			assert.Equal(t, tc.wantIsCrate, result.IsCratePage, "IsCratePage mismatch")
			assert.Equal(t, tc.wantIsSource, result.IsSourceView, "IsSourceView mismatch")
		})
	}
}

func TestDocsRSJSONEndpoint(t *testing.T) {
	tests := []struct {
		crate   string
		version string
		want    string
	}{
		{"serde", "1.0.0", "https://docs.rs/crate/serde/1.0.0/json"},
		{"tokio", "latest", "https://docs.rs/crate/tokio/latest/json"},
		{"ratatui", "0.30.0", "https://docs.rs/crate/ratatui/0.30.0/json"},
	}

	for _, tc := range tests {
		t.Run(tc.crate+"_"+tc.version, func(t *testing.T) {
			result := strategies.DocsRSJSONEndpoint(tc.crate, tc.version)
			assert.Equal(t, tc.want, result)
		})
	}
}
