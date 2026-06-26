package recovery

import (
	"testing"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPlanner_Plan(t *testing.T) {
	tests := []struct {
		name    string
		failed  Attempt
		verdict VerdictRetryAlternative
		snap    domain.StrategyResultSnapshot
		want    []Attempt
	}{
		{
			name:    "R1 filter_zeroed on sitemap -> scoped crawler keeping filter",
			failed:  Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml", FilterURL: "https://x.dev/book/"},
			verdict: VerdictRetryAlternative{Reason: string(domain.DiagFilterZeroed)},
			want: []Attempt{{
				Strategy:  "crawler",
				URL:       "https://x.dev/book/",
				FilterURL: "https://x.dev/book/",
				Reason:    "filter zeroed; crawling filtered subtree",
			}},
		},
		{
			name:    "R1 via no_urls_attempted with filter -> scoped crawler",
			failed:  Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml", FilterURL: "https://x.dev/book/"},
			verdict: VerdictRetryAlternative{Reason: "no_urls_attempted"},
			want: []Attempt{{
				Strategy:  "crawler",
				URL:       "https://x.dev/book/",
				FilterURL: "https://x.dev/book/",
				Reason:    "filter zeroed; crawling filtered subtree",
			}},
		},
		{
			name:    "R1 guard: already crawler -> no self-loop",
			failed:  Attempt{Strategy: "crawler", URL: "https://x.dev/book/", FilterURL: "https://x.dev/book/"},
			verdict: VerdictRetryAlternative{Reason: string(domain.DiagFilterZeroed)},
			want:    nil,
		},
		{
			name:    "R3 sitemap shallow no filter -> crawl origin",
			failed:  Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml", FilterURL: ""},
			verdict: VerdictRetryAlternative{Reason: "no_urls_attempted"},
			snap:    domain.StrategyResultSnapshot{EntryURL: "https://x.dev/sitemap.xml"},
			want: []Attempt{{
				Strategy:  "crawler",
				URL:       "https://x.dev",
				FilterURL: "",
				Reason:    "sitemap shallow; crawling site origin",
			}},
		},
		{
			name:    "R3 omitted when origin not derivable",
			failed:  Attempt{Strategy: "sitemap", URL: "not-a-url", FilterURL: ""},
			verdict: VerdictRetryAlternative{Reason: "no_urls_attempted"},
			snap:    domain.StrategyResultSnapshot{EntryURL: "not-a-url"},
			want:    nil,
		},
		{
			name:    "no_urls_attempted on crawler with no filter -> nil",
			failed:  Attempt{Strategy: "crawler", URL: "https://x.dev", FilterURL: ""},
			verdict: VerdictRetryAlternative{Reason: "no_urls_attempted"},
			want:    nil,
		},
		{
			name:    "high_failure_ratio is terminal -> nil",
			failed:  Attempt{Strategy: "crawler", URL: "https://x.dev"},
			verdict: VerdictRetryAlternative{Reason: "high_failure_ratio: 0.05"},
			want:    nil,
		},
		{
			name:    "unknown reason -> nil",
			failed:  Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml", FilterURL: "https://x.dev/book/"},
			verdict: VerdictRetryAlternative{Reason: "something_else"},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Planner{}.Plan(tt.failed, tt.verdict, tt.snap)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPlanner_RefineWith(t *testing.T) {
	llms := ProbeResult{Probe: probeLLMSTxtOnAncestor, Outcome: ProbeSuccess, Data: map[string]string{"llms_url": "https://x.dev/llms.txt"}}
	sitemap := ProbeResult{Probe: probeHasOwnSitemap, Outcome: ProbeSuccess, Data: map[string]string{"sitemap_url": "https://x.dev/book/sitemap.xml"}}
	index := ProbeResult{Probe: probeLooksLikeIndex, Outcome: ProbeSuccess, Data: map[string]string{"crawl_url": "https://x.dev"}}
	git := ProbeResult{Probe: probeGitHubPagesBacked, Outcome: ProbeSuccess, Data: map[string]string{"repo_url": "https://github.com/o/r"}}

	tests := []struct {
		name   string
		failed Attempt
		probes []ProbeResult
		want   []Attempt
	}{
		{
			name:   "all probes succeed -> ordered cheapest first",
			failed: Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml"},
			probes: []ProbeResult{git, index, sitemap, llms},
			want: []Attempt{
				{Strategy: "llms", URL: "https://x.dev/llms.txt", Reason: "llms.txt discovered on an ancestor path"},
				{Strategy: "sitemap", URL: "https://x.dev/book/sitemap.xml", Reason: "entry subtree exposes its own sitemap"},
				{Strategy: "crawler", URL: "https://x.dev", Reason: "entry looks like a crawlable index page"},
				{Strategy: "git", URL: "https://github.com/o/r", Reason: "github pages site backed by a git repository"},
			},
		},
		{
			name:   "failed strategy is not re-proposed",
			failed: Attempt{Strategy: "llms", URL: "https://x.dev/llms.txt"},
			probes: []ProbeResult{llms},
			want:   nil,
		},
		{
			name:   "sub-sitemap equal to failed URL is skipped",
			failed: Attempt{Strategy: "crawler", URL: "https://x.dev/book/sitemap.xml"},
			probes: []ProbeResult{sitemap},
			want:   nil,
		},
		{
			name:   "index probe skipped when failed strategy is crawler",
			failed: Attempt{Strategy: "crawler", URL: "https://x.dev"},
			probes: []ProbeResult{index},
			want:   nil,
		},
		{
			name:   "failed/inconclusive probes yield nothing",
			failed: Attempt{Strategy: "sitemap", URL: "https://x.dev/sitemap.xml"},
			probes: []ProbeResult{
				{Probe: probeLLMSTxtOnAncestor, Outcome: ProbeFailure},
				{Probe: probeLooksLikeIndex, Outcome: ProbeInconclusive},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Planner{}.RefineWith(tt.failed, VerdictRetryAlternative{}, domain.StrategyResultSnapshot{}, tt.probes)
			assert.Equal(t, tt.want, got)
		})
	}
}
