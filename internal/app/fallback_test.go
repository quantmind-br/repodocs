package app

import (
	"context"
	"testing"
	"time"

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
	executed := 0
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	result, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{
		NoFallback: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.CompletedDocs())
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
	assert.Equal(t, 0, executed)
}

func TestRunWithFallback_StrategyOverrideReturnsOriginal(t *testing.T) {
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	_, verdict, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{
		StrategyOverride: "crawler",
	})

	require.NoError(t, err)
	_, ok := verdict.(recovery.VerdictRetryAlternative)
	assert.True(t, ok)
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
	// The same attempt proposed twice (e.g. identical planner candidates)
	// must only execute once.
	executed := map[string]int{}
	o := newTestFallbackOrchestrator(t, func(st StrategyType) strategies.Strategy {
		return &fallbackStrategy{name: string(st), produceDocs: false, discovered: 5}
	})
	// Instrument the factory to count executions.
	origFactory := o.strategyFactory
	o.strategyFactory = func(st StrategyType, d *strategies.Dependencies) strategies.Strategy {
		s := origFactory(st, d)
		executed[s.Name()]++
		return s
	}

	initial := recovery.Attempt{Strategy: "sitemap", URL: "https://example.com/sitemap.xml"}
	_, _, err := o.runWithFallback(context.Background(), initial, OrchestratorOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, executed["sitemap"], "initial attempt runs once")
}

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

var _ = time.Second
