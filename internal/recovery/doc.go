// Package recovery validates extraction outcomes and formats actionable recovery
// errors.
//
// Phase 0+1 is validation-only: the package does not execute fallback
// strategies, probes, or recipe cache lookups. Later phases can reuse Verdict
// values from Validator to decide whether an alternative strategy should run.
package recovery
