package app

import (
	"context"
	"fmt"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/recovery"
	"github.com/quantmind-br/repodocs/internal/strategies"
)

// runWithFallback executes the initial attempt and, when its outcome is judged
// VerdictRetryAlternative, retries one level deep with planner-proposed
// alternatives. It returns the result and verdict of the attempt that
// satisfied the criteria, or the original attempt's result and verdict when no
// alternative succeeds. Fallback is suppressed when the user disabled it
// (--no-fallback) or explicitly forced a strategy (--strategy / manifest).
func (o *Orchestrator) runWithFallback(
	ctx context.Context,
	initial recovery.Attempt,
	opts OrchestratorOptions,
) (*domain.StrategyResult, recovery.Verdict, error) {
	result, execErr := o.execAttempt(ctx, initial, opts)
	verdict := o.validator.Validate(result, execErr, recovery.ValidationOptions{
		FilterURL: initial.FilterURL,
		DryRun:    opts.DryRun,
		MinDocs:   opts.MinDocs,
	})

	retry, ok := verdict.(recovery.VerdictRetryAlternative)
	if !ok || opts.NoFallback || opts.StrategyOverride != "" {
		return result, verdict, execErr
	}

	snap := domain.StrategyResultSnapshot{}
	if result != nil {
		snap = result.Snapshot()
	}

	for _, alt := range o.planner.Plan(initial, retry, snap) {
		if ctx.Err() != nil {
			break
		}
		o.logger.Info().
			Str("from", initial.Strategy).
			Str("to", alt.Strategy).
			Str("reason", alt.Reason).
			Msg("Primary strategy yielded no usable documents; attempting fallback")

		altResult, altErr := o.execAttempt(ctx, alt, opts)
		if ctx.Err() != nil {
			return altResult, recovery.VerdictPropagate{Cause: ctx.Err()}, altErr
		}
		altVerdict := o.validator.Validate(altResult, altErr, recovery.ValidationOptions{
			FilterURL: alt.FilterURL,
			DryRun:    opts.DryRun,
			MinDocs:   opts.MinDocs,
		})
		if _, ok := altVerdict.(recovery.VerdictOK); ok {
			o.logger.Info().
				Str("strategy", alt.Strategy).
				Int("written", altResult.Snapshot().DocsWritten).
				Msg("Fallback strategy recovered documents")
			return altResult, altVerdict, altErr
		}
	}

	// No alternative satisfied the criteria: surface the original outcome.
	return result, verdict, execErr
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
