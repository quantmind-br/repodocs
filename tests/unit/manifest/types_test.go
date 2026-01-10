package manifest_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/manifest"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  manifest.Config
		wantErr error
	}{
		{
			name: "valid config with one source",
			config: manifest.Config{
				Sources: []manifest.Source{
					{URL: "https://example.com"},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid config with multiple sources",
			config: manifest.Config{
				Sources: []manifest.Source{
					{URL: "https://example1.com"},
					{URL: "https://example2.com"},
				},
			},
			wantErr: nil,
		},
		{
			name:    "empty sources",
			config:  manifest.Config{Sources: []manifest.Source{}},
			wantErr: manifest.ErrNoSources,
		},
		{
			name:    "nil sources",
			config:  manifest.Config{},
			wantErr: manifest.ErrNoSources,
		},
		{
			name: "source without URL",
			config: manifest.Config{
				Sources: []manifest.Source{
					{Strategy: "crawler"},
				},
			},
			wantErr: manifest.ErrEmptyURL,
		},
		{
			name: "second source without URL",
			config: manifest.Config{
				Sources: []manifest.Source{
					{URL: "https://example.com"},
					{Strategy: "git"},
				},
			},
			wantErr: manifest.ErrEmptyURL,
		},
		{
			name: "valid config with all options",
			config: manifest.Config{
				Sources: []manifest.Source{
					{
						URL:             "https://example.com",
						Strategy:        "crawler",
						ContentSelector: ".content",
						ExcludeSelector: ".nav",
						Exclude:         []string{"/api/*"},
						Include:         []string{"*.md"},
						MaxDepth:        3,
						Limit:           100,
					},
				},
				Options: manifest.Options{
					ContinueOnError: true,
					Output:          "./output",
					Concurrency:     10,
					CacheTTL:        time.Hour,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := manifest.DefaultOptions()

	assert.False(t, opts.ContinueOnError)
	assert.Equal(t, "./docs", opts.Output)
	assert.Equal(t, 5, opts.Concurrency)
	assert.Equal(t, 24*time.Hour, opts.CacheTTL)
}

func TestSource_OptionalFields(t *testing.T) {
	source := manifest.Source{
		URL: "https://example.com",
	}

	assert.Empty(t, source.Strategy)
	assert.Empty(t, source.ContentSelector)
	assert.Nil(t, source.Exclude)
	assert.Nil(t, source.RenderJS)
	assert.Zero(t, source.MaxDepth)
}

func TestSource_RenderJSPointer(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name     string
		source   manifest.Source
		expected *bool
	}{
		{
			name:     "render_js not set",
			source:   manifest.Source{URL: "https://example.com"},
			expected: nil,
		},
		{
			name:     "render_js set to true",
			source:   manifest.Source{URL: "https://example.com", RenderJS: &trueVal},
			expected: &trueVal,
		},
		{
			name:     "render_js set to false",
			source:   manifest.Source{URL: "https://example.com", RenderJS: &falseVal},
			expected: &falseVal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.source.RenderJS)
		})
	}
}

func TestConfig_ValidateErrorMessage(t *testing.T) {
	config := manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://example.com"},
			{Strategy: "git"},
		},
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source 1")
	assert.ErrorIs(t, err, manifest.ErrEmptyURL)
}
