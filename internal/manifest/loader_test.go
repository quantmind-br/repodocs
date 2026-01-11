package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
}

func TestLoader_Load_FileNotFound(t *testing.T) {
	loader := NewLoader()

	cfg, err := loader.Load("/nonexistent/path/manifest.yaml")

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestLoader_Load_ValidYAML(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
    strategy: crawler
    max_depth: 3
  - url: https://github.com/org/repo
    strategy: git
    include:
      - "docs/**/*.md"
options:
  output: ./output
  continue_on_error: true
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 2)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)
	assert.Equal(t, "crawler", cfg.Sources[0].Strategy)
	assert.Equal(t, 3, cfg.Sources[0].MaxDepth)
	assert.Equal(t, "https://github.com/org/repo", cfg.Sources[1].URL)
	assert.Equal(t, "git", cfg.Sources[1].Strategy)
	assert.Equal(t, []string{"docs/**/*.md"}, cfg.Sources[1].Include)
	assert.True(t, cfg.Options.ContinueOnError)
	assert.Equal(t, "./output", cfg.Options.Output)
}

func TestLoader_Load_ValidJSON(t *testing.T) {
	loader := NewLoader()

	jsonContent := `{
		"sources": [
			{"url": "https://example.com", "strategy": "crawler"},
			{"url": "https://github.com/org/repo"}
		],
		"options": {
			"output": "./output",
			"concurrency": 10
		}
	}`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(manifestPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 2)
	assert.Equal(t, "https://example.com", cfg.Sources[0].URL)
	assert.Equal(t, "crawler", cfg.Sources[0].Strategy)
	assert.Equal(t, "https://github.com/org/repo", cfg.Sources[1].URL)
	assert.Equal(t, "./output", cfg.Options.Output)
	assert.Equal(t, 10, cfg.Options.Concurrency)
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
    strategy: crawler
invalid_yaml: [unclosed
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestLoader_Load_InvalidJSON(t *testing.T) {
	loader := NewLoader()

	jsonContent := `{invalid json content}`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(manifestPath, []byte(jsonContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestLoader_Load_UnsupportedExtension(t *testing.T) {
	loader := NewLoader()

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(manifestPath, []byte("content"), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrUnsupportedExt)
}

func TestLoader_Load_YMLExtension(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 1)
}

func TestLoader_Load_ReadError(t *testing.T) {
	loader := NewLoader()

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.Mkdir(manifestPath, 0755)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read manifest file")
}

func TestLoadFromBytes_YAML(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
    max_depth: 2
options:
  output: ./custom
`

	cfg, err := loader.LoadFromBytes([]byte(yamlContent), ".yaml")

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 1)
	assert.Equal(t, 2, cfg.Sources[0].MaxDepth)
	assert.Equal(t, "./custom", cfg.Options.Output)
}

func TestLoadFromBytes_JSON(t *testing.T) {
	loader := NewLoader()

	jsonContent := `{"sources": [{"url": "https://example.com"}]}`

	cfg, err := loader.LoadFromBytes([]byte(jsonContent), ".json")

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 1)
}

func TestLoadFromBytes_InvalidExt(t *testing.T) {
	loader := NewLoader()

	cfg, err := loader.LoadFromBytes([]byte("content"), ".txt")

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrUnsupportedExt)
}

func TestLoadFromBytes_CaseInsensitiveExt(t *testing.T) {
	loader := NewLoader()

	yamlContent := `sources: [{"url": "https://example.com"}]`
	jsonContent := `{"sources": [{"url": "https://example.com"}]}`

	cfg, err := loader.LoadFromBytes([]byte(yamlContent), ".YAML")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	cfg, err = loader.LoadFromBytes([]byte(yamlContent), ".Yml")
	assert.NoError(t, err)

	cfg, err = loader.LoadFromBytes([]byte(jsonContent), ".JSON")
	assert.NoError(t, err)
}

func TestLoader_applyDefaults_Output(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.Equal(t, "./docs", cfg.Options.Output)
}

func TestLoader_applyDefaults_Concurrency(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
options:
  output: ./custom
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.Equal(t, 5, cfg.Options.Concurrency)
}

func TestLoader_applyDefaults_CacheTTL(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
options:
  output: ./custom
  concurrency: 10
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.Equal(t, 24*3600*1000000000, int(cfg.Options.CacheTTL))
}

func TestLoader_PreservesCustomDefaults(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://example.com
options:
  output: /custom/path
  concurrency: 15
  cache_ttl: 48h
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.Equal(t, "/custom/path", cfg.Options.Output)
	assert.Equal(t, 15, cfg.Options.Concurrency)
	assert.Equal(t, 48*3600*1000000000, int(cfg.Options.CacheTTL))
}

func TestLoader_Load_ComplexManifest(t *testing.T) {
	loader := NewLoader()

	yamlContent := `
sources:
  - url: https://docs.example.com
    strategy: crawler
    content_selector: "article.main"
    exclude_selector: "nav,footer"
    exclude:
      - /admin
      - /api
    include:
      - /docs/*
    max_depth: 4
    render_js: true
    limit: 100
  - url: https://github.com/org/repo
    strategy: git
    max_depth: 1
    limit: 50
  - url: https://example.com/sitemap.xml
    strategy: sitemap
    limit: 200
options:
  output: ./knowledge-base
  continue_on_error: true
  concurrency: 3
  cache_ttl: 12h
`

	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "complex.yaml")
	err := os.WriteFile(manifestPath, []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := loader.Load(manifestPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Sources, 3)

	assert.Equal(t, "https://docs.example.com", cfg.Sources[0].URL)
	assert.Equal(t, "crawler", cfg.Sources[0].Strategy)
	assert.Equal(t, "article.main", cfg.Sources[0].ContentSelector)
	assert.Equal(t, "nav,footer", cfg.Sources[0].ExcludeSelector)
	assert.Equal(t, []string{"/admin", "/api"}, cfg.Sources[0].Exclude)
	assert.Equal(t, []string{"/docs/*"}, cfg.Sources[0].Include)
	assert.Equal(t, 4, cfg.Sources[0].MaxDepth)
	assert.NotNil(t, cfg.Sources[0].RenderJS)
	assert.Equal(t, true, *cfg.Sources[0].RenderJS)
	assert.Equal(t, 100, cfg.Sources[0].Limit)

	assert.Equal(t, "https://github.com/org/repo", cfg.Sources[1].URL)
	assert.Equal(t, "git", cfg.Sources[1].Strategy)
	assert.Equal(t, 50, cfg.Sources[1].Limit)

	assert.Equal(t, "https://example.com/sitemap.xml", cfg.Sources[2].URL)
	assert.Equal(t, "sitemap", cfg.Sources[2].Strategy)
	assert.Equal(t, 200, cfg.Sources[2].Limit)

	assert.Equal(t, "./knowledge-base", cfg.Options.Output)
	assert.True(t, cfg.Options.ContinueOnError)
	assert.Equal(t, 3, cfg.Options.Concurrency)
	assert.Equal(t, 12*3600*1000000000, int(cfg.Options.CacheTTL))
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNoSources", ErrNoSources},
		{"ErrEmptyURL", ErrEmptyURL},
		{"ErrInvalidFormat", ErrInvalidFormat},
		{"ErrFileNotFound", ErrFileNotFound},
		{"ErrUnsupportedExt", ErrUnsupportedExt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}
