package strategies_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
)

func TestParseRobotsTxt(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		baseURL string
		want    []string
	}{
		{
			name:    "single sitemap directive",
			content: []byte("Sitemap: https://example.com/sitemap.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/sitemap.xml"},
		},
		{
			name:    "multiple sitemap lines",
			content: []byte("Sitemap: https://example.com/sitemap.xml\nSitemap: https://example.com/sitemap-news.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/sitemap.xml", "https://example.com/sitemap-news.xml"},
		},
		{
			name:    "case insensitive sitemap key",
			content: []byte("SITEMAP: https://example.com/a.xml\nsitemap: https://example.com/b.xml\nSiteMap: https://example.com/c.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/a.xml", "https://example.com/b.xml", "https://example.com/c.xml"},
		},
		{
			name:    "comment-only lines are skipped",
			content: []byte("# comment\n# Sitemap: https://example.com/ignored.xml\nSitemap: https://example.com/live.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/live.xml"},
		},
		{
			name:    "whitespace trimmed",
			content: []byte("  Sitemap:   https://example.com/sitemap.xml  "),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/sitemap.xml"},
		},
		{
			name:    "mixed with user agent directives",
			content: []byte("User-agent: *\nDisallow: /admin\nSitemap: https://example.com/sitemap.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/sitemap.xml"},
		},
		{
			name:    "no sitemap directives",
			content: []byte("User-agent: *\nDisallow: /"),
			baseURL: "https://example.com",
			want:    []string{},
		},
		{
			name:    "empty content",
			content: []byte(""),
			baseURL: "https://example.com",
			want:    []string{},
		},
		{
			name:    "relative sitemap url resolved against base",
			content: []byte("Sitemap: /sitemap.xml"),
			baseURL: "https://example.com/docs",
			want:    []string{"https://example.com/sitemap.xml"},
		},
		{
			name:    "inline comment stripped",
			content: []byte("Sitemap: https://example.com/sitemap.xml # main sitemap"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/sitemap.xml"},
		},
		{
			name:    "windows line endings",
			content: []byte("Sitemap: https://example.com/a.xml\r\nSitemap: https://example.com/b.xml\r\n"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/a.xml", "https://example.com/b.xml"},
		},
		{
			name:    "blank lines between directives",
			content: []byte("Sitemap: https://example.com/a.xml\n\n\nSitemap: https://example.com/b.xml"),
			baseURL: "https://example.com",
			want:    []string{"https://example.com/a.xml", "https://example.com/b.xml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategies.ParseRobotsTxt(tt.content, tt.baseURL)
			assert.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSitemapContent(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want bool
	}{
		{name: "urlset xml", body: []byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`), want: true},
		{name: "sitemapindex xml", body: []byte(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></sitemapindex>`), want: true},
		{name: "html body", body: []byte(`<html><body>hello</body></html>`), want: false},
		{name: "json", body: []byte(`{"urls": []}`), want: false},
		{name: "empty", body: []byte{}, want: false},
		{name: "bom with urlset", body: append([]byte{0xEF, 0xBB, 0xBF}, []byte(`<urlset></urlset>`)...), want: true},
		{name: "leading whitespace with urlset", body: []byte("   \n\t\r<urlset></urlset>"), want: true},
		{name: "plain text", body: []byte(`hello sitemap maybe`), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, strategies.IsSitemapContent(tt.body))
		})
	}
}

func TestGetSitemapProbes(t *testing.T) {
	probes := strategies.GetSitemapProbes()
	require.Len(t, probes, 6)
	assert.Equal(t, "/robots.txt", probes[0].Path)
	assert.Equal(t, "robots.txt", probes[0].Name)
}

func TestDiscoverSitemap_RobotsTxt(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Sitemap: " + server.URL + "/sitemap.xml"))
		case "/sitemap.xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	result, err := strategies.DiscoverSitemap(context.Background(), deps.Fetcher, server.URL, deps.Logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, server.URL+"/sitemap.xml", result.SitemapURL)
	assert.Equal(t, "robots.txt", result.Method)
}

func TestDiscoverSitemap_DirectProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/sitemap.xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<urlset></urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	result, err := strategies.DiscoverSitemap(context.Background(), deps.Fetcher, server.URL, deps.Logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, server.URL+"/sitemap.xml", result.SitemapURL)
	assert.Equal(t, "probe:/sitemap.xml", result.Method)
}

func TestDiscoverSitemap_RobotsTxtPriority(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Sitemap: " + server.URL + "/server-sitemap.xml"))
		case "/sitemap.xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<urlset></urlset>`))
		case "/server-sitemap.xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<urlset></urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	result, err := strategies.DiscoverSitemap(context.Background(), deps.Fetcher, server.URL, deps.Logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, server.URL+"/server-sitemap.xml", result.SitemapURL)
	assert.Equal(t, "robots.txt", result.Method)
}

func TestDiscoverSitemap_NoSitemap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	result, err := strategies.DiscoverSitemap(context.Background(), deps.Fetcher, server.URL, deps.Logger)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDiscoverSitemap_NextJS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap-0.xml":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	result, err := strategies.DiscoverSitemap(context.Background(), deps.Fetcher, server.URL, deps.Logger)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, server.URL+"/sitemap-0.xml", result.SitemapURL)
	assert.Equal(t, "probe:/sitemap-0.xml", result.Method)
}

func TestDiscoverSitemap_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<urlset></urlset>`))
	}))
	defer server.Close()

	deps, err := strategies.NewDependencies(strategies.DependencyOptions{
		Timeout:     5 * time.Second,
		EnableCache: false,
		OutputDir:   t.TempDir(),
		CommonOptions: domain.CommonOptions{
			DryRun: true,
		},
	})
	require.NoError(t, err)
	defer deps.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := strategies.DiscoverSitemap(ctx, deps.Fetcher, server.URL, deps.Logger)
	if err != nil {
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
	}
	if err == nil {
		assert.Nil(t, result)
	}
}
