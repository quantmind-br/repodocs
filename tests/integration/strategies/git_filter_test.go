package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/output"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFetcher implements domain.Fetcher for testing
type MockFetcher struct {
	responses map[string]*domain.Response
}

func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		responses: make(map[string]*domain.Response),
	}
}

func (m *MockFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("mock 404: %s", url)
}

func (m *MockFetcher) GetWithHeaders(ctx context.Context, url string, headers map[string]string) (*domain.Response, error) {
	return m.Get(ctx, url)
}

func (m *MockFetcher) GetCookies(url string) []*http.Cookie {
	return nil
}

func (m *MockFetcher) Transport() http.RoundTripper {
	return nil
}

func (m *MockFetcher) Close() error {
	return nil
}

func (m *MockFetcher) Register(url string, body []byte) {
	m.responses[url] = &domain.Response{
		StatusCode:  200,
		Body:        body,
		ContentType: "application/zip",
	}
}

func createZipArchive(t *testing.T, files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestGitStrategy_Execute_PathFiltering(t *testing.T) {
	outputDir := t.TempDir()

	logger := utils.NewLogger(utils.LoggerOptions{Level: "debug"})
	writer := output.NewWriter(output.WriterOptions{
		BaseDir: outputDir,
		Force:   true,
	})

	fetcher := NewMockFetcher()

	files := map[string]string{
		"README.md":         "# Root Readme",
		"docs/intro.md":     "# Introduction",
		"docs/api.md":       "# API",
		"src/main.go":       "package main",
		"other/notes.md":    "some notes",
		"other/skipped.txt": "skipped",
	}
	zipData := createZipArchive(t, files)

	repoURL := "https://github.com/test/repo"
	archiveURL := repoURL + "/archive/refs/heads/main.zip"
	fetcher.Register(archiveURL, zipData)

	deps := &strategies.Dependencies{
		Fetcher: fetcher,
		Writer:  writer,
		Logger:  logger,
	}

	strategy := strategies.NewGitStrategy(deps)

	t.Run("FilterDocsDir", func(t *testing.T) {
		opts := strategies.DefaultOptions()
		opts.Output = filepath.Join(outputDir, "docs-run")

		targetURL := "https://github.com/test/repo/tree/main/docs"
		err := strategy.Execute(context.Background(), targetURL, opts)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(opts.Output, "docs", "intro.md"))
		assert.FileExists(t, filepath.Join(opts.Output, "docs", "api.md"))

		assert.NoFileExists(t, filepath.Join(opts.Output, "README.md"))
		assert.NoFileExists(t, filepath.Join(opts.Output, "other", "notes.md"))
	})

	t.Run("FilterFlag", func(t *testing.T) {
		opts := strategies.DefaultOptions()
		opts.Output = filepath.Join(outputDir, "flag-run")
		opts.FilterURL = "other"

		targetURL := "https://github.com/test/repo"
		err := strategy.Execute(context.Background(), targetURL, opts)
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(opts.Output, "other", "notes.md"))

		assert.NoFileExists(t, filepath.Join(opts.Output, "README.md"))
		assert.NoFileExists(t, filepath.Join(opts.Output, "docs", "intro.md"))
	})
}
