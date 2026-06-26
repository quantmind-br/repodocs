package recovery

import (
	"bytes"
	"context"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// Probe budgets keep diagnostic probing cheap and bounded. Each probe runs
// under its own timeout; the runner caps total wall-clock with a wider budget.
// These ceilings honor the design constraint of "≤ 2s per probe, ≤ 6s total".
const (
	probeTimeout      = 2 * time.Second
	probeTotalBudget  = 6 * time.Second
	maxAncestorProbes = 4
)

// indexPageLinkThreshold is the minimum number of anchors a page must expose to
// be considered a crawlable index worth a crawler fallback.
const indexPageLinkThreshold = 20

// Probe names. Planner.RefineWith matches ProbeResult.Probe against these to map
// a successful probe to a concrete fallback Attempt.
const (
	probeLLMSTxtOnAncestor = "llms_txt_on_ancestor"
	probeHasOwnSitemap     = "has_own_sitemap"
	probeLooksLikeIndex    = "looks_like_index_page"
	probeGitHubPagesBacked = "github_pages_backed"
)

// ProbeOutcome classifies a probe's finding.
type ProbeOutcome string

const (
	ProbeSuccess      ProbeOutcome = "success"
	ProbeFailure      ProbeOutcome = "failure"
	ProbeInconclusive ProbeOutcome = "inconclusive"
)

// ProbeResult is the structured outcome of a single diagnostic probe. Data
// carries probe-specific signals consumed by Planner.RefineWith, for example
// the resolved llms.txt URL, sub-sitemap URL, crawl URL, or backing git repo.
type ProbeResult struct {
	Probe   string
	Target  string
	Outcome ProbeOutcome
	Data    map[string]string
}

// ProbeRunner executes cheap, bounded diagnostic probes that differentiate
// recovery causes after the static fallback plan is exhausted. Its only
// dependency is a domain.Fetcher, so the recovery package keeps importing
// nothing beyond domain and the standard library.
type ProbeRunner struct {
	fetcher domain.Fetcher
}

// NewProbeRunner creates a ProbeRunner backed by the given fetcher.
func NewProbeRunner(fetcher domain.Fetcher) *ProbeRunner {
	return &ProbeRunner{fetcher: fetcher}
}

// probeFn analyzes a failed attempt against shared fetched resources and returns
// a structured result. It performs only bounded, read-only HTTP.
type probeFn func(ctx context.Context, cache *fetchCache, failed Attempt) ProbeResult

// Run executes the diagnostic probes relevant to a failed attempt and returns
// their results together with the total elapsed probing time (for budget
// logging). It never returns an error: a probe that cannot reach its target
// yields a failure/inconclusive ProbeResult rather than a hard failure. Probes
// run concurrently and share a per-run fetch cache so overlapping targets (such
// as the origin HTML used by both the index and github-pages probes) are
// fetched at most once.
func (pr *ProbeRunner) Run(ctx context.Context, failed Attempt) ([]ProbeResult, time.Duration) {
	start := time.Now()
	if pr == nil || pr.fetcher == nil {
		return nil, 0
	}

	budgetCtx, cancel := context.WithTimeout(ctx, probeTotalBudget)
	defer cancel()

	cache := newFetchCache(pr.fetcher)

	probes := []struct {
		name string
		run  probeFn
	}{
		{probeLLMSTxtOnAncestor, probeLLMSTxt},
		{probeHasOwnSitemap, probeOwnSitemap},
		{probeLooksLikeIndex, probeIndexPage},
		{probeGitHubPagesBacked, probeGitHubPages},
	}

	results := make([]ProbeResult, len(probes))
	var wg sync.WaitGroup
	for i, p := range probes {
		wg.Add(1)
		go func(i int, name string, run probeFn) {
			defer wg.Done()
			probeCtx, probeCancel := context.WithTimeout(budgetCtx, probeTimeout)
			defer probeCancel()
			res := run(probeCtx, cache, failed)
			res.Probe = name
			results[i] = res
		}(i, p.name, p.run)
	}
	wg.Wait()

	return results, time.Since(start)
}

// probeLLMSTxt looks for an llms.txt at the entry path and each ancestor up to
// the site origin. The nearest one that responds 200 with a non-empty body wins.
func probeLLMSTxt(ctx context.Context, cache *fetchCache, failed Attempt) ProbeResult {
	base := failed.URL
	if failed.FilterURL != "" {
		base = failed.FilterURL
	}
	res := ProbeResult{Target: base, Outcome: ProbeFailure}

	for _, candidate := range llmsCandidates(base) {
		if ctx.Err() != nil {
			return res
		}
		resp, err := cache.get(ctx, candidate)
		if err != nil || resp == nil || resp.StatusCode != 200 {
			continue
		}
		if len(bytes.TrimSpace(resp.Body)) == 0 {
			continue
		}
		return ProbeResult{
			Target:  base,
			Outcome: ProbeSuccess,
			Data:    map[string]string{"llms_url": candidate},
		}
	}
	return res
}

// probeOwnSitemap checks whether the entry subtree exposes its own sitemap.xml,
// which the origin-scoped discovery in the main flow would have missed.
func probeOwnSitemap(ctx context.Context, cache *fetchCache, failed Attempt) ProbeResult {
	base := failed.FilterURL
	if base == "" {
		base = failed.URL
	}
	res := ProbeResult{Target: base, Outcome: ProbeFailure}

	candidate := joinPath(base, "sitemap.xml")
	if candidate == "" {
		return res
	}
	resp, err := cache.get(ctx, candidate)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return res
	}
	if !looksLikeSitemap(resp.Body) {
		return res
	}
	return ProbeResult{
		Target:  base,
		Outcome: ProbeSuccess,
		Data:    map[string]string{"sitemap_url": candidate},
	}
}

// probeIndexPage fetches the most specific entry (filter subtree or origin) and
// reports whether it looks like a link-rich index page worth crawling.
func probeIndexPage(ctx context.Context, cache *fetchCache, failed Attempt) ProbeResult {
	target := originOf(failed.URL)
	if failed.FilterURL != "" {
		target = failed.FilterURL
	}
	res := ProbeResult{Target: target, Outcome: ProbeFailure}
	if target == "" {
		return res
	}
	resp, err := cache.get(ctx, target)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		return res
	}
	if countAnchors(resp.Body) < indexPageLinkThreshold {
		return ProbeResult{Target: target, Outcome: ProbeInconclusive}
	}
	return ProbeResult{
		Target:  target,
		Outcome: ProbeSuccess,
		Data:    map[string]string{"crawl_url": target},
	}
}

// probeGitHubPages detects whether the entry is a GitHub Pages site backed by a
// git repository, first via the *.github.io host convention, then by scraping
// the page for a github.com repository link (covering custom domains).
func probeGitHubPages(ctx context.Context, cache *fetchCache, failed Attempt) ProbeResult {
	parsed, err := url.Parse(failed.URL)
	if err != nil || parsed.Host == "" {
		return ProbeResult{Target: failed.URL, Outcome: ProbeFailure}
	}
	res := ProbeResult{Target: failed.URL, Outcome: ProbeFailure}

	if repo := githubRepoFromPagesHost(parsed); repo != "" {
		return ProbeResult{
			Target:  failed.URL,
			Outcome: ProbeSuccess,
			Data:    map[string]string{"repo_url": repo},
		}
	}

	origin := originOf(failed.URL)
	if origin == "" {
		return res
	}
	resp, err := cache.get(ctx, origin)
	if err == nil && resp != nil && resp.StatusCode == 200 {
		if repo := githubRepoFromHTML(resp.Body); repo != "" {
			return ProbeResult{
				Target:  failed.URL,
				Outcome: ProbeSuccess,
				Data:    map[string]string{"repo_url": repo},
			}
		}
	}
	return res
}

// llmsCandidates returns <dir>llms.txt for the URL's directory and each parent
// directory up to the origin, nearest first, capped at maxAncestorProbes.
func llmsCandidates(raw string) []string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil
	}
	origin := parsed.Scheme + "://" + parsed.Host

	dir := dirOf(parsed.Path)

	var out []string
	seen := make(map[string]bool)
	for {
		candidate := origin + dir + "llms.txt"
		if !seen[candidate] {
			out = append(out, candidate)
			seen[candidate] = true
		}
		if dir == "/" || len(out) >= maxAncestorProbes {
			break
		}
		dir = parentDir(dir)
	}
	return out
}

// dirOf normalizes a URL path to its containing directory (with trailing slash).
func dirOf(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasSuffix(path, "/") {
		return path
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx+1]
	}
	return "/"
}

// parentDir returns the parent directory of a trailing-slash directory path.
func parentDir(dir string) string {
	trimmed := strings.TrimSuffix(dir, "/")
	if idx := strings.LastIndex(trimmed, "/"); idx >= 0 {
		return trimmed[:idx+1]
	}
	return "/"
}

// joinPath appends leaf to the directory of raw, returning an absolute URL, or
// "" when raw is not an absolute http(s) URL.
func joinPath(raw, leaf string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Path = dirOf(parsed.Path) + leaf
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// looksLikeSitemap reports whether a body's leading bytes are sitemap XML. It is
// a local, dependency-free reimplementation of the strategies-package check so
// the recovery package stays pure.
func looksLikeSitemap(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	segment := body
	if len(segment) > 1024 {
		segment = segment[:1024]
	}
	segment = bytes.TrimPrefix(segment, []byte{0xEF, 0xBB, 0xBF})
	segment = bytes.TrimLeft(segment, " \t\r\n")
	lower := bytes.ToLower(segment)
	return bytes.Contains(lower, []byte("<urlset")) || bytes.Contains(lower, []byte("<sitemapindex"))
}

// countAnchors counts opening anchor tags in an HTML body. It is a cheap
// heuristic (no DOM parsing) used only to gauge whether a page is an index.
func countAnchors(body []byte) int {
	return bytes.Count(bytes.ToLower(body), []byte("<a "))
}

// githubRepoFromPagesHost derives the backing repository URL from a *.github.io
// host. Project pages map to github.com/<user>/<firstPathSegment>; user and
// organization pages map to github.com/<user>/<user>.github.io.
func githubRepoFromPagesHost(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	const suffix = ".github.io"
	if !strings.HasSuffix(host, suffix) {
		return ""
	}
	user := strings.TrimSuffix(host, suffix)
	if user == "" {
		return ""
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) > 0 && segments[0] != "" {
		return "https://github.com/" + user + "/" + segments[0]
	}
	return "https://github.com/" + user + "/" + user + ".github.io"
}

// githubRepoLinkRegex matches a github.com/<owner>/<repo> reference in HTML.
var githubRepoLinkRegex = regexp.MustCompile(`github\.com/([A-Za-z0-9](?:[A-Za-z0-9-]{0,38})?)/([A-Za-z0-9_.-]+)`)

// reservedGitHubOwners are github.com path roots that are never user repos.
var reservedGitHubOwners = map[string]bool{
	"sponsors": true, "login": true, "about": true, "features": true,
	"marketplace": true, "topics": true, "collections": true, "trending": true,
	"settings": true, "notifications": true, "explore": true, "apps": true,
}

// githubRepoFromHTML extracts the first plausible github.com/<owner>/<repo>
// reference from an HTML body, normalizing the repo segment.
func githubRepoFromHTML(body []byte) string {
	matches := githubRepoLinkRegex.FindAllSubmatch(body, -1)
	for _, m := range matches {
		owner := string(m[1])
		repo := strings.TrimSuffix(string(m[2]), ".git")
		if owner == "" || repo == "" || reservedGitHubOwners[strings.ToLower(owner)] {
			continue
		}
		return "https://github.com/" + owner + "/" + repo
	}
	return ""
}
