package app

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/quantmind-br/repodocs/internal/config"
	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/recovery"
	"github.com/quantmind-br/repodocs/internal/strategies"
	"github.com/quantmind-br/repodocs/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fallbackStrategy is a controllable strategy: it produces either an empty
// outcome (VerdictRetryAlternative) or a one-document outcome (VerdictOK)
// depending on the produceDocs flag.
type fallbackStrategy struct {
	name         string
	produceDocs  bool
	discovered   int
	cancelDuring func()
}

func (m *fallbackStrategy) Name() string { return m.name }
func (m *fallbackStrategy) CanHandle(url string) bool {
	return true
}
func (m *fallbackStrategy) Execute(ctx context.Context, url string, _ strategies.Options) (*domain.StrategyResult, error) {
	result := domain.NewStrategyResult(m.name, url)
	if m.discovered > 0 {
		result.AddDiscovered(m.discovered)
	}
	if m.produceDocs {
		result.IncAttempted()
		result.IncWritten()
	}
	result.Finish()
	if m.cancelDuring != nil {
		m.cancelDuring()
	}
	return result, nil
}

// newTestFallbackOrchestrator builds an Orchestrator with a controllable
// strategy factory and a probe runner that performs no I/O (nil fetcher), so
// the probe tier yields no candidates unless a test injects its own.
func newTestFallbackOrchestrator(t *testing.T, factory func(StrategyType) strategies.Strategy) *Orchestrator {
	t.Helper()
	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Logging.Level = "error"

	return &Orchestrator{
		config: cfg,
		deps:   &strategies.Dependencies{},
		logger: utils.NewLogger(utils.LoggerOptions{Level: "error"}),
		strategyFactory: func(st StrategyType, _ *strategies.Dependencies) strategies.Strategy {
			return factory(st)
		},
		validator:   recovery.NewValidator(nil),
		planner:     recovery.NewPlanner(),
		probeRunner: recovery.NewProbeRunner(nil), // nil fetcher: no probes run
	}
}

func TestRunWithFallback_VerdictOKReturnsImmediately(t *testing.T) {
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: true}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.CompletedDocs())
	_, ok := verdict.(recovery.VerdictOK)
	assert.True(t, ok)
}

func TestRunWithFallback_NoFallbackReturnsOriginal(t *testing.T) {
	executed := map[string]int{}
	var mu sync.Mutex
	base := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})
	origFactory := base.strategyFactory
	base.strategyFactory = func(st StrategyType, d *strategies.Dependencies) strategies.Strategy {
		s := origFactory(st, d)
		mu.Lock()
		executed[s.Name()]++
		mu.Unlock()
		return s
	}

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := base.runWithFallback(context.Background(), initial, OrchestratorOptions{
		NoFallback: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.CompletedDocs())
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, executed["sitemap"], "initial attempt runs once")
	assert.Equal(t, 0, executed["crawler"], "fallback must be suppressed")
}

func TestRunWithFallback_StrategyOverrideReturnsOriginal(t *testing.T) {
	executed := map[string]int{}
	var mu sync.Mutex
	base := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})
	origFactory := base.strategyFactory
	base.strategyFactory = func(st StrategyType, d *strategies.Dependencies) strategies.Strategy {
		s := origFactory(st, d)
		mu.Lock()
		executed[s.Name()]++
		mu.Unlock()
		return s
	}

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	_, verdict, err := base.runWithFallback(context.Background(), initial, OrchestratorOptions{
		StrategyOverride: "crawler",
	})

	require.NoError(t, err)
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, executed["sitemap"], "initial attempt runs once")
	assert.Equal(t, 0, executed["crawler"], "override must suppress fallback")
}

func TestRunWithFallback_PlannedCandidateRecovers(t *testing.T) {
	// The planner's R3 rule (sitemap that discovered URLs but attempted none)
	// proposes crawling the site origin; the crawler produces documents.
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		switch st {
		case "sitemap":
			return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
		default:
			return &fallbackStrategy{name: string(st), produceDocs: true}
		}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "crawler", result.Strategy, "fallback candidate should be the crawler")
	assert.Equal(t, 1, result.CompletedDocs())
	_, ok := verdict.(recovery.VerdictOK)
	assert.True(t, ok)
}

func TestRunWithFallback_NoCandidateSatisfiesReturnsOriginal(t *testing.T) {
	// Every strategy yields an empty outcome; the planner proposes the crawler,
	// it also fails, and the original verdict is surfaced.
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "sitemap", result.Strategy, "original strategy result must be surfaced")
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
}

func TestRunWithFallback_ContextCancelledBeforeRunReturnsOriginal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := o.runWithFallback(ctx, initial, OrchestratorOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	// A pre-cancelled context skips fallback tiers and surfaces the original
	// outcome; VerdictPropagate is only produced when cancellation happens
	// mid-fallback.
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
}

func TestRunWithFallback_CancellationDuringFallbackPropagates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		switch st {
		case "sitemap":
			// Initial attempt: empty outcome, triggers retry.
			return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
		default:
			// Fallback candidate cancels the context mid-execution.
			return &fallbackStrategy{name: string(st), cancelDuring: cancel}
		}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	_, verdict, _ := o.runWithFallback(ctx, initial, OrchestratorOptions{})
	propagate, ok := verdict.(recovery.VerdictPropagate)
	assert.True(t, ok, "expected VerdictPropagate, got %T", verdict)
	assert.Equal(t, context.Canceled, propagate.Cause)
}

func TestRunWithFallback_DeduplicatesAttempts(t *testing.T) {
	// Scenario: the initial sitemap attempt discovers URLs but attempts none
	// (VerdictRetryAlternative). Tier 1's planner proposes crawling the site
	// origin (R3). Tier 2's probes then re-propose the same crawler attempt via
	// the index-page probe. The dedup key must prevent a second crawler run.
	originHTML := []byte(`<!DOCTYPE html><html><body>` + indexPageAnchors() + `</body></html>`)
	fetcher := &pathAwareFetcher{
		responses: map[string]*domain.Response{
			// Origin: link-rich index page (20+ anchors, no GitHub meta).
			"https://example.com": {StatusCode: 200, Body: originHTML},
		},
	}

	var mu sync.Mutex
	executed := map[string]int{}
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})
	o.probeRunner = recovery.NewProbeRunner(fetcher)
	origFactory := o.strategyFactory
	o.strategyFactory = func(st StrategyType, d *strategies.Dependencies) strategies.Strategy {
		s := origFactory(st, d)
		mu.Lock()
		executed[s.Name()]++
		mu.Unlock()
		return s
	}

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	_, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, executed["sitemap"], "initial attempt runs once")
	assert.Equal(t, 1, executed["crawler"], "dedup must prevent the re-proposed crawler from running twice")
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok, "no alternative satisfied criteria; original verdict surfaces")
}

// indexPageAnchors returns 25 anchor tags to satisfy the index-page probe.
func indexPageAnchors() string {
	s := ""
	for i := range 25 {
		s += `<a href="https://example.com/page` + string(rune('0'+i)) + `">Page</a>`
	}
	return s
}

// pathAwareFetcher returns per-URL canned responses; URLs not listed return 404.
type pathAwareFetcher struct {
	responses map[string]*domain.Response
}

func (f *pathAwareFetcher) Get(_ context.Context, url string) (*domain.Response, error) {
	if resp, ok := f.responses[url]; ok {
		return resp, nil
	}
	return &domain.Response{StatusCode: 404, Body: []byte("not found")}, nil
}

func (f *pathAwareFetcher) GetWithHeaders(ctx context.Context, url string, _ map[string]string) (*domain.Response, error) {
	return f.Get(ctx, url)
}

func (f *pathAwareFetcher) GetCookies(url string) []*http.Cookie { return nil }
func (f *pathAwareFetcher) Transport() http.RoundTripper         { return nil }
func (f *pathAwareFetcher) Close() error                         { return nil }

func TestAttemptKey_DistinguishesStrategyURLAndFilter(t *testing.T) {
	base := recovery.Attempt{Strategy: "crawler", URL: "https://example.com", FilterURL: "/docs"}
	assert.Equal(t, attemptKey(base), attemptKey(recovery.Attempt{
		Strategy: "crawler", URL: "https://example.com", FilterURL: "/docs",
	}))
	assert.NotEqual(t, attemptKey(base), attemptKey(recovery.Attempt{
		Strategy: "sitemap", URL: "https://example.com", FilterURL: "/docs",
	}))
	assert.NotEqual(t, attemptKey(base), attemptKey(recovery.Attempt{
		Strategy: "crawler", URL: "https://example.com", FilterURL: "/other",
	}))
	assert.NotEqual(t, attemptKey(base), attemptKey(recovery.Attempt{
		Strategy: "crawler", URL: "https://other.example", FilterURL: "/docs",
	}))
}

func TestLogProbes_EmptyResultsIsNoop(t *testing.T) {
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st)}
	})
	// Must not panic and must not emit anything.
	o.logProbes(recovery.Attempt{Strategy: "crawler", URL: "https://example.com"}, nil, 0)
}

func TestValidationOpts_CarriesPerAttemptOverrides(t *testing.T) {
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st)}
	})
	opts := o.validationOpts(recovery.Attempt{FilterURL: "/docs"}, OrchestratorOptions{
		CommonOptions: domain.CommonOptions{DryRun: true},
		MinDocs:       7,
	})
	assert.Equal(t, "/docs", opts.FilterURL)
	assert.True(t, opts.DryRun)
	assert.Equal(t, 7, opts.MinDocs)
}
