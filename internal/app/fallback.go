package app

import (
	"context"
	"fmt"
	"time"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/recovery"
	"github.com/quantmind-br/repodocs/internal/strategies"
)

// maxFallbackAttempts bounds how many alternative attempts may run beyond the
// initial one, keeping recovery to at most three executions per run (initial +
// two fallbacks) as required by the design budget.
const maxFallbackAttempts = 2

// runWithFallback executes the initial attempt and, when its outcome is judged
// VerdictRetryAlternative, retries with alternative strategies. It proceeds in
// two tiers: first the static planner candidates (Phase 2 — R1/R3), then, if the
// attempt budget remains, probe-informed candidates (Phase 3 — Plan C) derived
// from cheap diagnostic probes. It returns the result and verdict of the attempt
// that satisfied the criteria, or the original attempt's result and verdict when
// no alternative succeeds. Fallback is suppressed when the user disabled it
// (--no-fallback) or explicitly forced a strategy (--strategy / manifest).
func (o *Orchestrator) runWithFallback(
	ctx context.Context,
	initial recovery.Attempt,
	opts OrchestratorOptions,
) (*domain.StrategyResult, recovery.Verdict, error) {
	result, execErr := o.execAttempt(ctx, initial, opts)
	verdict := o.validator.Validate(result, execErr, o.validationOpts(initial, opts))

	retry, ok := verdict.(recovery.VerdictRetryAlternative)
	if !ok || opts.NoFallback || opts.StrategyOverride != "" {
		return result, verdict, execErr
	}

	snap := domain.StrategyResultSnapshot{}
	if result != nil {
		snap = result.Snapshot()
	}

	tried := map[string]bool{attemptKey(initial): true}
	budget := maxFallbackAttempts

	// Tier 1: static planner candidates (Phase 2 — R1/R3).
	for _, alt := range o.planner.Plan(initial, retry, snap) {
		if budget <= 0 || ctx.Err() != nil {
			break
		}
		if tried[attemptKey(alt)] {
			continue
		}
		tried[attemptKey(alt)] = true
		budget--
		if r, v, done := o.tryFallback(ctx, initial, alt, opts); done {
			return r, v, nil
		}
	}

	// Tier 2: probe-informed candidates (Phase 3 — Plan C). Only worth probing
	// while attempts remain in the budget.
	if budget > 0 && ctx.Err() == nil {
		probeResults, elapsed := o.probeRunner.Run(ctx, initial)
		o.logProbes(initial, probeResults, elapsed)
		for _, alt := range o.planner.RefineWith(initial, retry, snap, probeResults) {
			if budget <= 0 || ctx.Err() != nil {
				break
			}
			if tried[attemptKey(alt)] {
				continue
			}
			tried[attemptKey(alt)] = true
			budget--
			if r, v, done := o.tryFallback(ctx, initial, alt, opts); done {
				return r, v, nil
			}
		}
	}

	// No alternative satisfied the criteria: surface the original outcome.
	return result, verdict, execErr
}

// tryFallback executes one alternative attempt and reports whether the caller
// should stop. It returns done=true when the attempt satisfied the criteria
// (VerdictOK) or the context was cancelled mid-flight; otherwise done=false and
// the caller advances to the next candidate.
func (o *Orchestrator) tryFallback(
	ctx context.Context,
	from recovery.Attempt,
	alt recovery.Attempt,
	opts OrchestratorOptions,
) (*domain.StrategyResult, recovery.Verdict, bool) {
	o.logger.Info().
		Str("from", from.Strategy).
		Str("to", alt.Strategy).
		Str("url", alt.URL).
		Str("reason", alt.Reason).
		Msg("Primary strategy yielded no usable documents; attempting fallback")

	altResult, altErr := o.execAttempt(ctx, alt, opts)
	if ctx.Err() != nil {
		return altResult, recovery.VerdictPropagate{Cause: ctx.Err()}, true
	}
	altVerdict := o.validator.Validate(altResult, altErr, o.validationOpts(alt, opts))
	if _, ok := altVerdict.(recovery.VerdictOK); ok {
		o.logger.Info().
			Str("strategy", alt.Strategy).
			Int("written", altResult.Snapshot().DocsWritten).
			Msg("Fallback strategy recovered documents")
		return altResult, altVerdict, true
	}
	return altResult, altVerdict, false
}

// validationOpts builds the per-attempt validation options shared by the initial
// run and every fallback candidate.
func (o *Orchestrator) validationOpts(a recovery.Attempt, opts OrchestratorOptions) recovery.ValidationOptions {
	return recovery.ValidationOptions{
		FilterURL: a.FilterURL,
		DryRun:    opts.DryRun,
		MinDocs:   opts.MinDocs,
	}
}

// logProbes emits a single structured line summarizing probe outcomes and the
// total probing budget consumed, keeping recovery loud by default.
func (o *Orchestrator) logProbes(initial recovery.Attempt, results []recovery.ProbeResult, elapsed time.Duration) {
	if len(results) == 0 {
		return
	}
	event := o.logger.Info().
		Str("entry", initial.URL).
		Dur("probe_budget", elapsed)
	for _, r := range results {
		event = event.Str("probe_"+r.Probe, string(r.Outcome))
	}
	event.Msg("Diagnostic probes completed")
}

// attemptKey is the deduplication key for an attempt: the strategy, entry URL,
// and filter together. It prevents re-running an identical attempt across tiers.
func attemptKey(a recovery.Attempt) string {
	return a.Strategy + "\x00" + a.URL + "\x00" + a.FilterURL
}

// execAttempt runs a single attempt: it resolves the concrete strategy, records
// the active source/strategy for metadata attribution, builds strategy options
// scoped by the attempt's filter, and executes. It is the single execution path
// shared by the initial run and every fallback candidate.
func (o *Orchestrator) execAttempt(
	ctx context.Context,
	a recovery.Attempt,
	opts OrchestratorOptions,
) (*domain.StrategyResult, error) {
	strategyType := StrategyType(a.Strategy)
	if !IsValidStrategy(strategyType) {
		return nil, fmt.Errorf("invalid strategy for attempt: %s", a.Strategy)
	}

	strategy := o.strategyFactory(strategyType, o.deps)
	if strategy == nil {
		return nil, fmt.Errorf("failed to create strategy for URL: %s", a.URL)
	}

	o.logger.Info().
		Str("strategy", strategy.Name()).
		Str("url", a.URL).
		Msg("Using extraction strategy")

	o.deps.SetSourceURL(a.URL)
	o.deps.SetStrategy(strategy.Name())

	strategyOpts := strategies.Options{
		CommonOptions: domain.CommonOptions{
			Verbose:  opts.Verbose,
			DryRun:   opts.DryRun,
			Force:    opts.Force || o.config.Output.Overwrite,
			RenderJS: opts.RenderJS || o.config.Rendering.ForceJS,
			Limit:    opts.Limit,
		},
		Output:          o.config.Output.Directory,
		Concurrency:     o.config.Concurrency.Workers,
		MaxDepth:        o.config.Concurrency.MaxDepth,
		Exclude:         append(o.config.Exclude, opts.ExcludePatterns...),
		NoFolders:       o.config.Output.Flat,
		Split:           opts.Split,
		IncludeAssets:   opts.IncludeAssets,
		ContentSelector: opts.ContentSelector,
		ExcludeSelector: opts.ExcludeSelector,
		FilterURL:       a.FilterURL,
	}

	return strategy.Execute(ctx, a.URL, strategyOpts)
}
