package manifest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.False(t, opts.ContinueOnError, "ContinueOnError should default to false")
	assert.Equal(t, "./docs", opts.Output, "Output should default to ./docs")
	assert.Equal(t, 5, opts.Concurrency, "Concurrency should default to 5")
	assert.Equal(t, 24*time.Hour, opts.CacheTTL, "CacheTTL should default to 24 hours")
}

func TestConfig_Validate_NoSources(t *testing.T) {
	cfg := &Config{
		Sources: []Source{},
		Options: DefaultOptions(),
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNoSources)
}

func TestConfig_Validate_EmptyURL(t *testing.T) {
	cfg := &Config{
		Sources: []Source{
			{URL: "https://example.com"},
			{URL: ""}, // Empty URL
		},
		Options: DefaultOptions(),
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source 1")
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestConfig_Validate_EmptyURLFirstSource(t *testing.T) {
	cfg := &Config{
		Sources: []Source{
			{URL: ""}, // Empty URL first source
			{URL: "https://example.com"},
		},
		Options: DefaultOptions(),
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source 0")
	assert.ErrorIs(t, err, ErrEmptyURL)
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Sources: []Source{
			{URL: "https://example.com"},
			{URL: "https://github.com/org/repo"},
		},
		Options: DefaultOptions(),
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_SingleSource(t *testing.T) {
	cfg := &Config{
		Sources: []Source{
			{URL: "https://example.com"},
		},
		Options: DefaultOptions(),
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Fields(t *testing.T) {
	// Test all Source fields
	src := Source{
		URL:             "https://example.com",
		Strategy:        "crawler",
		ContentSelector: "article.main",
		ExcludeSelector: "nav,footer",
		Exclude:         []string{"/admin", "/api"},
		Include:         []string{"/docs/**", "*.md"},
		MaxDepth:        3,
		RenderJS:        boolPtr(true),
		Limit:           10,
	}

	assert.Equal(t, "https://example.com", src.URL)
	assert.Equal(t, "crawler", src.Strategy)
	assert.Equal(t, "article.main", src.ContentSelector)
	assert.Equal(t, "nav,footer", src.ExcludeSelector)
	assert.Equal(t, []string{"/admin", "/api"}, src.Exclude)
	assert.Equal(t, []string{"/docs/**", "*.md"}, src.Include)
	assert.Equal(t, 3, src.MaxDepth)
	assert.NotNil(t, src.RenderJS)
	assert.Equal(t, true, *src.RenderJS)
	assert.Equal(t, 10, src.Limit)
}

func TestConfig_FieldsDefaultValues(t *testing.T) {
	src := Source{
		URL: "https://example.com",
	}

	assert.Equal(t, "", src.Strategy)
	assert.Equal(t, "", src.ContentSelector)
	assert.Equal(t, "", src.ExcludeSelector)
	assert.Empty(t, src.Exclude)
	assert.Empty(t, src.Include)
	assert.Equal(t, 0, src.MaxDepth)
	assert.Nil(t, src.RenderJS)
	assert.Equal(t, 0, src.Limit)
}

func TestOptions_Fields(t *testing.T) {
	opts := Options{
		ContinueOnError: true,
		Output:          "/custom/output",
		Concurrency:     10,
		CacheTTL:        12 * time.Hour,
	}

	assert.True(t, opts.ContinueOnError)
	assert.Equal(t, "/custom/output", opts.Output)
	assert.Equal(t, 10, opts.Concurrency)
	assert.Equal(t, 12*time.Hour, opts.CacheTTL)
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
