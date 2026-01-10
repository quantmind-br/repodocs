package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/manifest"
)

func TestLoader_Load_YAML_Valid(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(*testing.T, *manifest.Config)
	}{
		{
			name: "full manifest with all fields",
			content: `
sources:
  - url: https://docs.example.com
    strategy: crawler
    content_selector: "article.main"
    exclude_selector: ".sidebar"
    exclude:
      - "*/changelog/*"
      - "*/archive/*"
    max_depth: 3
    render_js: true
    limit: 100
  - url: https://github.com/org/repo
    strategy: git
    include:
      - "docs/**/*.md"
options:
  continue_on_error: true
  output: ./knowledge-base
  concurrency: 10
`,
			check: func(t *testing.T, cfg *manifest.Config) {
				require.Len(t, cfg.Sources, 2)

				s1 := cfg.Sources[0]
				assert.Equal(t, "https://docs.example.com", s1.URL)
				assert.Equal(t, "crawler", s1.Strategy)
				assert.Equal(t, "article.main", s1.ContentSelector)
				assert.Equal(t, ".sidebar", s1.ExcludeSelector)
				assert.Equal(t, []string{"*/changelog/*", "*/archive/*"}, s1.Exclude)
				assert.Equal(t, 3, s1.MaxDepth)
				require.NotNil(t, s1.RenderJS)
				assert.True(t, *s1.RenderJS)
				assert.Equal(t, 100, s1.Limit)

				s2 := cfg.Sources[1]
				assert.Equal(t, "https://github.com/org/repo", s2.URL)
				assert.Equal(t, "git", s2.Strategy)
				assert.Equal(t, []string{"docs/**/*.md"}, s2.Include)

				assert.True(t, cfg.Options.ContinueOnError)
				assert.Equal(t, "./knowledge-base", cfg.Options.Output)
				assert.Equal(t, 10, cfg.Options.Concurrency)
			},
		},
		{
			name: "minimal manifest",
			content: `
sources:
  - url: https://example.com
`,
			check: func(t *testing.T, cfg *manifest.Config) {
				require.Len(t, cfg.Sources, 1)
				assert.Equal(t, "https://example.com", cfg.Sources[0].URL)

				assert.False(t, cfg.Options.ContinueOnError)
				assert.Equal(t, "./docs", cfg.Options.Output)
				assert.Equal(t, 5, cfg.Options.Concurrency)
			},
		},
		{
			name: "render_js explicitly false",
			content: `
sources:
  - url: https://example.com
    render_js: false
`,
			check: func(t *testing.T, cfg *manifest.Config) {
				require.Len(t, cfg.Sources, 1)
				require.NotNil(t, cfg.Sources[0].RenderJS)
				assert.False(t, *cfg.Sources[0].RenderJS)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, "manifest.yaml")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0644))

			loader := manifest.NewLoader()
			cfg, err := loader.Load(path)

			require.NoError(t, err)
			require.NotNil(t, cfg)
			tt.check(t, cfg)
		})
	}
}

func TestLoader_Load_YML_Extension(t *testing.T) {
	content := `
sources:
  - url: https://example.com
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "manifest.yml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	loader := manifest.NewLoader()
	cfg, err := loader.Load(path)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)
}

func TestLoader_Load_JSON(t *testing.T) {
	content := `{
		"sources": [
			{
				"url": "https://example.com",
				"strategy": "crawler",
				"max_depth": 5
			}
		],
		"options": {
			"output": "./output",
			"continue_on_error": true
		}
	}`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	loader := manifest.NewLoader()
	cfg, err := loader.Load(path)

	require.NoError(t, err)
	assert.Len(t, cfg.Sources, 1)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)
	assert.Equal(t, "crawler", cfg.Sources[0].Strategy)
	assert.Equal(t, 5, cfg.Sources[0].MaxDepth)
	assert.True(t, cfg.Options.ContinueOnError)
	assert.Equal(t, "./output", cfg.Options.Output)
}

func TestLoader_Load_Errors(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError error
	}{
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/manifest.yaml"
			},
			wantError: manifest.ErrFileNotFound,
		},
		{
			name: "unsupported extension",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "manifest.txt")
				require.NoError(t, os.WriteFile(path, []byte("content"), 0644))
				return path
			},
			wantError: manifest.ErrUnsupportedExt,
		},
		{
			name: "invalid YAML",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "manifest.yaml")
				require.NoError(t, os.WriteFile(path, []byte("not: valid: yaml: ["), 0644))
				return path
			},
			wantError: manifest.ErrInvalidFormat,
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "manifest.json")
				require.NoError(t, os.WriteFile(path, []byte("{invalid json}"), 0644))
				return path
			},
			wantError: manifest.ErrInvalidFormat,
		},
		{
			name: "empty sources",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "manifest.yaml")
				require.NoError(t, os.WriteFile(path, []byte("sources: []"), 0644))
				return path
			},
			wantError: manifest.ErrNoSources,
		},
		{
			name: "source without URL",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "manifest.yaml")
				content := "sources:\n  - strategy: crawler"
				require.NoError(t, os.WriteFile(path, []byte(content), 0644))
				return path
			},
			wantError: manifest.ErrEmptyURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			loader := manifest.NewLoader()
			_, err := loader.Load(path)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}

func TestLoader_LoadFromBytes(t *testing.T) {
	yaml := []byte(`
sources:
  - url: https://example.com
`)
	json := []byte(`{"sources": [{"url": "https://example.com"}]}`)

	loader := manifest.NewLoader()

	cfg, err := loader.LoadFromBytes(yaml, ".yaml")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)

	cfg, err = loader.LoadFromBytes(yaml, ".yml")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)

	cfg, err = loader.LoadFromBytes(json, ".json")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)

	_, err = loader.LoadFromBytes(yaml, ".xml")
	assert.ErrorIs(t, err, manifest.ErrUnsupportedExt)
}

func TestLoader_LoadFromBytes_CaseInsensitive(t *testing.T) {
	yaml := []byte(`
sources:
  - url: https://example.com
`)

	loader := manifest.NewLoader()

	cfg, err := loader.LoadFromBytes(yaml, ".YAML")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)

	cfg, err = loader.LoadFromBytes(yaml, ".YML")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)
}

func TestLoader_DefaultsApplied(t *testing.T) {
	content := `
sources:
  - url: https://example.com
`
	loader := manifest.NewLoader()
	cfg, err := loader.LoadFromBytes([]byte(content), ".yaml")

	require.NoError(t, err)
	assert.Equal(t, "./docs", cfg.Options.Output)
	assert.Equal(t, 5, cfg.Options.Concurrency)
	assert.NotZero(t, cfg.Options.CacheTTL)
}

func TestLoader_PartialOptions(t *testing.T) {
	content := `
sources:
  - url: https://example.com
options:
  output: ./custom
`
	loader := manifest.NewLoader()
	cfg, err := loader.LoadFromBytes([]byte(content), ".yaml")

	require.NoError(t, err)
	assert.Equal(t, "./custom", cfg.Options.Output)
	assert.Equal(t, 5, cfg.Options.Concurrency)
}
