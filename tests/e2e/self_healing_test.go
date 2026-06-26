package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs/internal/app"
	"github.com/quantmind-br/repodocs/internal/config"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRustBookLikeServer serves a sitemap whose URLs all fall OUTSIDE /book/
// (so a --filter on /book/ zeroes the sitemap), while also serving a crawlable
// /book/ subtree. This mirrors the real doc.rust-lang.org scenario where the
// sitemap omits the book but the book is reachable by crawling.
func newRustBookLikeServer() *httptest.Server {
	mux := http.NewServeMux()

	// Sitemap lists only non-/book/ URLs.
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		baseURL := "http://" + r.Host
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
    <url><loc>` + baseURL + `/reference/foo</loc></url>
    <url><loc>` + baseURL + `/std/bar</loc></url>
</urlset>`))
	})

	// Crawlable /book/ subtree: an index linking to two chapter pages.
	mux.HandleFunc("/book/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>The Book</title></head>
<body><main><h1>The Book</h1><p>Welcome to the book.</p>
<a href="/book/page1">Chapter 1</a>
<a href="/book/page2">Chapter 2</a>
</main></body></html>`))
	})
	mux.HandleFunc("/book/page1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Chapter 1</title></head>
<body><main><h1>Chapter 1</h1><p>Content of chapter one.</p></main></body></html>`))
	})
	mux.HandleFunc("/book/page2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Chapter 2</title></head>
<body><main><h1>Chapter 2</h1><p>Content of chapter two.</p></main></body></html>`))
	})

	return httptest.NewServer(mux)
}

func countMarkdownFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(path) == ".md" {
			count++
		}
		return nil
	})
	return count
}

func newFallbackOrchestrator(t *testing.T, tmpDir string) *app.Orchestrator {
	t.Helper()
	cfg := config.Default()
	cfg.Output.Directory = tmpDir
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 2

	orch, err := app.NewOrchestrator(app.OrchestratorOptions{Config: cfg})
	require.NoError(t, err)
	return orch
}

// TestE2E_SelfHealingFallback_RustBook verifies that entering by a sitemap
// whose URLs are excluded by --filter auto-recovers: the orchestrator falls
// back to crawling the filtered subtree and writes documents without manual
// intervention.
func TestE2E_SelfHealingFallback_RustBook(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	server := newRustBookLikeServer()
	defer server.Close()

	t.Run("auto-recovers via crawler fallback", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "repodocs-e2e-selfheal-ok-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		orch := newFallbackOrchestrator(t, tmpDir)
		defer orch.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = orch.Run(ctx, server.URL+"/sitemap.xml", app.OrchestratorOptions{
			FilterURL: server.URL + "/book/",
			CommonOptions: domain.CommonOptions{
				Limit: 10,
			},
		})
		require.NoError(t, err, "filter-zeroed sitemap should auto-recover via crawler fallback")
		assert.GreaterOrEqual(t, countMarkdownFiles(t, tmpDir), 1,
			"fallback crawler should have written at least one doc under /book/")
	})

	t.Run("--no-fallback fails loudly", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "repodocs-e2e-selfheal-nofb-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		orch := newFallbackOrchestrator(t, tmpDir)
		defer orch.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = orch.Run(ctx, server.URL+"/sitemap.xml", app.OrchestratorOptions{
			FilterURL:  server.URL + "/book/",
			NoFallback: true,
			CommonOptions: domain.CommonOptions{
				Limit: 10,
			},
		})
		require.Error(t, err, "with --no-fallback a filter-zeroed sitemap must error")
		assert.Equal(t, 0, countMarkdownFiles(t, tmpDir), "no docs should be written")
	})

	t.Run("forced --strategy suppresses fallback", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "repodocs-e2e-selfheal-override-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		orch := newFallbackOrchestrator(t, tmpDir)
		defer orch.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = orch.Run(ctx, server.URL+"/sitemap.xml", app.OrchestratorOptions{
			FilterURL:        server.URL + "/book/",
			StrategyOverride: "sitemap",
			CommonOptions: domain.CommonOptions{
				Limit: 10,
			},
		})
		require.Error(t, err, "an explicitly forced strategy must not trigger fallback")
		assert.Equal(t, 0, countMarkdownFiles(t, tmpDir), "no docs should be written")
	})
}
