package recovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpStubFetcher is a minimal domain.Fetcher that performs real HTTP GETs
// against an httptest.Server, used to drive probes end to end.
type httpStubFetcher struct{}

func newHTTPStubFetcher() *httpStubFetcher {
	return &httpStubFetcher{}
}

func (f *httpStubFetcher) Get(ctx context.Context, url string) (*domain.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &domain.Response{
		StatusCode:  resp.StatusCode,
		Body:        body,
		Headers:     resp.Header,
		ContentType: resp.Header.Get("Content-Type"),
		URL:         url,
	}, nil
}

func (f *httpStubFetcher) GetWithHeaders(ctx context.Context, url string, _ map[string]string) (*domain.Response, error) {
	return f.Get(ctx, url)
}
func (f *httpStubFetcher) GetCookies(string) []*http.Cookie { return nil }
func (f *httpStubFetcher) Transport() http.RoundTripper     { return http.DefaultTransport }
func (f *httpStubFetcher) Close() error                     { return nil }

// newProbeServer serves the resources the diagnostic probes look for: an
// llms.txt at the origin (but not under /book/), a sub-tree sitemap under
// /book/, and a link-rich index page under /index/.
func newProbeServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "# Docs\n\n- [Intro](/guide/intro)\n")
	})

	mux.HandleFunc("/book/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		base := "http://" + r.Host
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>`+base+`/book/page1</loc></url>
</urlset>`)
	})

	mux.HandleFunc("/index/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		var b strings.Builder
		b.WriteString("<html><body>")
		for i := range 25 {
			fmt.Fprintf(&b, `<a href="/p%d">link %d</a>`, i, i)
		}
		b.WriteString("</body></html>")
		_, _ = io.WriteString(w, b.String())
	})

	return httptest.NewServer(mux)
}

func TestProbeLLMSTxt_FindsAncestor(t *testing.T) {
	server := newProbeServer()
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	// Filter under /book/ has no llms.txt; the probe must climb to the origin.
	failed := Attempt{Strategy: "sitemap", URL: server.URL + "/sitemap.xml", FilterURL: server.URL + "/book/"}
	res := probeLLMSTxt(context.Background(), cache, failed)

	assert.Equal(t, ProbeSuccess, res.Outcome)
	assert.Equal(t, server.URL+"/llms.txt", res.Data["llms_url"])
}

func TestProbeLLMSTxt_NotFound(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	res := probeLLMSTxt(context.Background(), cache, Attempt{URL: server.URL + "/docs/"})
	assert.Equal(t, ProbeFailure, res.Outcome)
	assert.Empty(t, res.Data["llms_url"])
}

func TestProbeOwnSitemap_FindsSubtreeSitemap(t *testing.T) {
	server := newProbeServer()
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	failed := Attempt{Strategy: "crawler", URL: server.URL, FilterURL: server.URL + "/book/"}
	res := probeOwnSitemap(context.Background(), cache, failed)

	assert.Equal(t, ProbeSuccess, res.Outcome)
	assert.Equal(t, server.URL+"/book/sitemap.xml", res.Data["sitemap_url"])
}

func TestProbeIndexPage_RichIndex(t *testing.T) {
	server := newProbeServer()
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	failed := Attempt{Strategy: "llms", URL: server.URL + "/index/", FilterURL: server.URL + "/index/"}
	res := probeIndexPage(context.Background(), cache, failed)

	assert.Equal(t, ProbeSuccess, res.Outcome)
	assert.Equal(t, server.URL+"/index/", res.Data["crawl_url"])
}

func TestProbeIndexPage_SparsePageInconclusive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><body><a href="/one">only link</a></body></html>`)
	}))
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	res := probeIndexPage(context.Background(), cache, Attempt{URL: server.URL, FilterURL: server.URL})
	assert.Equal(t, ProbeInconclusive, res.Outcome)
}

func TestProbeGitHubPages_HostHeuristic(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantRepo string
	}{
		{"project pages", "https://owner.github.io/myproject/guide", "https://github.com/owner/myproject"},
		{"user pages root", "https://owner.github.io/", "https://github.com/owner/owner.github.io"},
		{"not github pages", "https://example.com/docs", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No fetch needed for the host heuristic; a nil-backed cache is fine
			// because the *.github.io cases never reach the HTML scrape.
			cache := newFetchCache(newHTTPStubFetcher())
			res := probeGitHubPages(context.Background(), cache, Attempt{URL: tt.url})
			if tt.wantRepo == "" {
				assert.NotEqual(t, ProbeSuccess, res.Outcome)
				return
			}
			assert.Equal(t, ProbeSuccess, res.Outcome)
			assert.Equal(t, tt.wantRepo, res.Data["repo_url"])
		})
	}
}

func TestProbeGitHubPages_CustomDomainHTMLScrape(t *testing.T) {
	// A non-*.github.io host falls through to scraping the origin HTML for a
	// github.com repository link (covering custom-domain GitHub Pages sites).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body>
<a href="https://github.com/sponsors/acme">Sponsor</a>
<a href="https://github.com/acme/docs">Edit this page on GitHub</a>
</body></html>`)
	}))
	defer server.Close()
	cache := newFetchCache(newHTTPStubFetcher())

	res := probeGitHubPages(context.Background(), cache, Attempt{URL: server.URL + "/guide/intro"})
	assert.Equal(t, ProbeSuccess, res.Outcome)
	assert.Equal(t, "https://github.com/acme/docs", res.Data["repo_url"])
}

func TestProbeRunner_Run_CollectsSignals(t *testing.T) {
	server := newProbeServer()
	defer server.Close()

	runner := NewProbeRunner(newHTTPStubFetcher())
	failed := Attempt{Strategy: "sitemap", URL: server.URL + "/sitemap.xml", FilterURL: server.URL + "/book/"}

	results, elapsed := runner.Run(context.Background(), failed)
	require.Len(t, results, 4)
	assert.GreaterOrEqual(t, elapsed.Nanoseconds(), int64(0))

	byProbe := map[string]ProbeResult{}
	for _, r := range results {
		byProbe[r.Probe] = r
	}
	assert.Equal(t, ProbeSuccess, byProbe[probeLLMSTxtOnAncestor].Outcome)
	assert.Equal(t, ProbeSuccess, byProbe[probeHasOwnSitemap].Outcome)

	// RefineWith should turn the successful probes into ordered attempts.
	attempts := Planner{}.RefineWith(failed, VerdictRetryAlternative{}, domain.StrategyResultSnapshot{}, results)
	require.GreaterOrEqual(t, len(attempts), 2)
	assert.Equal(t, "llms", attempts[0].Strategy)
	assert.Equal(t, server.URL+"/llms.txt", attempts[0].URL)
	assert.Equal(t, "sitemap", attempts[1].Strategy)
	assert.Equal(t, server.URL+"/book/sitemap.xml", attempts[1].URL)
}

func TestProbeRunner_NilFetcher(t *testing.T) {
	var runner *ProbeRunner
	results, elapsed := runner.Run(context.Background(), Attempt{URL: "https://x.dev"})
	assert.Nil(t, results)
	assert.Zero(t, elapsed)

	empty := NewProbeRunner(nil)
	results, _ = empty.Run(context.Background(), Attempt{URL: "https://x.dev"})
	assert.Nil(t, results)
}

func TestLLMSCandidates_OrderAndCap(t *testing.T) {
	got := llmsCandidates("https://x.dev/a/b/c/page.html")
	want := []string{
		"https://x.dev/a/b/c/llms.txt",
		"https://x.dev/a/b/llms.txt",
		"https://x.dev/a/llms.txt",
		"https://x.dev/llms.txt",
	}
	assert.Equal(t, want, got)

	// Deeper paths are capped at maxAncestorProbes entries.
	deep := llmsCandidates("https://x.dev/a/b/c/d/e/f/page")
	assert.Len(t, deep, maxAncestorProbes)

	assert.Nil(t, llmsCandidates("not-a-url"))
}

func TestJoinPath(t *testing.T) {
	assert.Equal(t, "https://x.dev/book/sitemap.xml", joinPath("https://x.dev/book/", "sitemap.xml"))
	assert.Equal(t, "https://x.dev/book/sitemap.xml", joinPath("https://x.dev/book/index.html", "sitemap.xml"))
	assert.Equal(t, "https://x.dev/sitemap.xml", joinPath("https://x.dev", "sitemap.xml"))
	assert.Equal(t, "", joinPath("not-a-url", "sitemap.xml"))
}

func TestLooksLikeSitemap(t *testing.T) {
	assert.True(t, looksLikeSitemap([]byte(`<?xml version="1.0"?><urlset></urlset>`)))
	assert.True(t, looksLikeSitemap([]byte("\xEF\xBB\xBF\n  <sitemapindex>")))
	assert.False(t, looksLikeSitemap([]byte("<html><body>not a sitemap</body></html>")))
	assert.False(t, looksLikeSitemap(nil))
}

func TestCountAnchors(t *testing.T) {
	assert.Equal(t, 2, countAnchors([]byte(`<A href="x">1</a><a class="y">2</a><area>`)))
	assert.Equal(t, 0, countAnchors([]byte(`<span>no anchors</span>`)))
}

func TestGithubRepoFromHTML(t *testing.T) {
	html := []byte(`<a href="https://github.com/sponsors/foo">sponsor</a>
	<a href="https://github.com/rust-lang/book">source</a>`)
	assert.Equal(t, "https://github.com/rust-lang/book", githubRepoFromHTML(html))

	assert.Equal(t, "", githubRepoFromHTML([]byte(`<a href="https://example.com">x</a>`)))
	assert.Equal(t, "https://github.com/owner/repo", githubRepoFromHTML([]byte(`see github.com/owner/repo.git`)))
}
