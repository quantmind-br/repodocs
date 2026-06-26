package recovery

import (
	"net/url"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// Attempt describes a single extraction attempt: a concrete strategy to run
// against an entry URL, optionally scoped by a path filter. The Reason is a
// human-readable note (logged by the caller) explaining why the attempt was
// scheduled.
type Attempt struct {
	Strategy  string // concrete strategy name: "crawler", "sitemap", ...
	URL       string // entry URL to execute against
	FilterURL string // path filter; empty means no scoping
	Reason    string // why this attempt was planned (for logging)
}

// Planner proposes alternative attempts after a strategy outcome is judged
// VerdictRetryAlternative. It is intentionally pure: it depends only on the
// failed attempt, the verdict, and the result snapshot, so it can be unit
// tested without any strategy or network machinery.
type Planner struct{}

// NewPlanner creates a stateless fallback Planner.
func NewPlanner() *Planner { return &Planner{} }

// Plan returns ordered fallback candidates for a failed attempt. An empty
// slice means no viable alternative exists and the caller should surface the
// original outcome error.
//
// Only zero-output triggers produce candidates, which keeps fallback safe:
// the failed attempt wrote no files, so re-running cannot conflict. The
// high_failure_ratio trigger (which may have written partial output) is
// deliberately terminal and yields no candidates.
func (Planner) Plan(failed Attempt, v VerdictRetryAlternative, snap domain.StrategyResultSnapshot) []Attempt {
	switch v.Reason {
	case string(domain.DiagFilterZeroed), "no_urls_attempted":
		// R1: a path filter excluded every candidate URL. Crawl the filtered
		// subtree directly, keeping the filter so the crawler stays scoped to
		// that path (an empty filter would crawl the whole host).
		if failed.FilterURL != "" && failed.Strategy != "crawler" {
			return []Attempt{{
				Strategy:  "crawler",
				URL:       failed.FilterURL,
				FilterURL: failed.FilterURL,
				Reason:    "filter zeroed; crawling filtered subtree",
			}}
		}
		// R3: a shallow/empty sitemap discovered URLs but attempted none, with
		// no filter in play. Fall back to crawling the site origin.
		if failed.FilterURL == "" && failed.Strategy == "sitemap" {
			if origin := originOf(snap.EntryURL); origin != "" {
				return []Attempt{{
					Strategy:  "crawler",
					URL:       origin,
					FilterURL: "",
					Reason:    "sitemap shallow; crawling site origin",
				}}
			}
		}
	}
	return nil
}

// RefineWith proposes probe-informed fallback candidates ("Plan C") after the
// static plan is exhausted. It consumes the diagnostic signals gathered by a
// ProbeRunner and maps each successful probe to a concrete attempt, ordered
// cheapest first: an llms.txt index, a sub-tree sitemap, a crawl of a rich
// index page, and finally a git clone (the most expensive recovery). It is
// pure — it performs no I/O — and never re-proposes the failed strategy at the
// same URL. The caller deduplicates candidates already attempted statically.
func (Planner) RefineWith(failed Attempt, _ VerdictRetryAlternative, _ domain.StrategyResultSnapshot, probes []ProbeResult) []Attempt {
	succeeded := make(map[string]ProbeResult, len(probes))
	for _, p := range probes {
		if p.Outcome == ProbeSuccess {
			succeeded[p.Probe] = p
		}
	}

	var attempts []Attempt

	// 1. llms.txt discovered on the entry path or an ancestor -> LLMS strategy.
	if p, ok := succeeded[probeLLMSTxtOnAncestor]; ok && failed.Strategy != "llms" {
		if u := p.Data["llms_url"]; u != "" {
			attempts = append(attempts, Attempt{
				Strategy: "llms",
				URL:      u,
				Reason:   "llms.txt discovered on an ancestor path",
			})
		}
	}

	// 2. The entry subtree exposes its own sitemap -> sitemap strategy.
	if p, ok := succeeded[probeHasOwnSitemap]; ok {
		if u := p.Data["sitemap_url"]; u != "" && u != failed.URL {
			attempts = append(attempts, Attempt{
				Strategy: "sitemap",
				URL:      u,
				Reason:   "entry subtree exposes its own sitemap",
			})
		}
	}

	// 3. The entry/origin looks like a link-rich index -> crawl it.
	if p, ok := succeeded[probeLooksLikeIndex]; ok && failed.Strategy != "crawler" {
		if u := p.Data["crawl_url"]; u != "" {
			attempts = append(attempts, Attempt{
				Strategy: "crawler",
				URL:      u,
				Reason:   "entry looks like a crawlable index page",
			})
		}
	}

	// 4. A GitHub Pages site is backed by a git repo -> clone it (last resort).
	if p, ok := succeeded[probeGitHubPagesBacked]; ok && failed.Strategy != "git" {
		if u := p.Data["repo_url"]; u != "" {
			attempts = append(attempts, Attempt{
				Strategy: "git",
				URL:      u,
				Reason:   "github pages site backed by a git repository",
			})
		}
	}

	return attempts
}

// originOf returns the scheme://host origin of a URL, or "" when the URL does
// not parse into an absolute http(s) origin.
func originOf(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Scheme == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
