package recovery

import (
	"fmt"

	"github.com/quantmind-br/repodocs/internal/domain"
)

const (
	DefaultMinDocsWritten  = 1
	DefaultMinSuccessRatio = 0.10
)

// Verdict is the recovery validator's decision about a strategy outcome.
type Verdict interface{ verdict() }

// VerdictOK means the outcome satisfies the configured criteria.
type VerdictOK struct{}

// VerdictRetryAlternative means the outcome was insufficient but another
// strategy (or the same strategy with different parameters) may succeed.
type VerdictRetryAlternative struct {
	Reason      string
	Diagnostics []domain.Diagnostic
}

// VerdictHardFail means the strategy failed with a non-transient error
// or produced 0 completed documents and no alternative is viable.
type VerdictHardFail struct {
	Reason string
	Cause  error
}

// VerdictPropagate means the error is transient (retry/backoff managed by
// the fetch layer) and should be propagated without triggering recovery.
type VerdictPropagate struct {
	Cause error
}

func (VerdictOK) verdict()               {}
func (VerdictRetryAlternative) verdict() {}
func (VerdictHardFail) verdict()         {}
func (VerdictPropagate) verdict()        {}

// Criteria are the thresholds that define a successful outcome.
type Criteria struct {
	MinDocsWritten  int
	MinSuccessRatio float64
}

// ValidationOptions carry per-run overrides for validation behavior.
type ValidationOptions struct {
	FilterURL       string
	DryRun          bool
	MinDocs         int
	MinSuccessRatio float64
}

// Validator checks strategy outcomes against configured criteria.
type Validator struct {
	criteria Criteria
}

// NewValidator creates a Validator with the given criteria, falling back
// to sensible defaults for zero-value fields.
func NewValidator(criteria *Criteria) *Validator {
	resolved := Criteria{
		MinDocsWritten:  DefaultMinDocsWritten,
		MinSuccessRatio: DefaultMinSuccessRatio,
	}
	if criteria != nil {
		if criteria.MinDocsWritten > 0 {
			resolved.MinDocsWritten = criteria.MinDocsWritten
		}
		if criteria.MinSuccessRatio > 0 {
			resolved.MinSuccessRatio = criteria.MinSuccessRatio
		}
	}
	return &Validator{criteria: resolved}
}

// Validate inspects a strategy result and error and returns a Verdict.
func (v *Validator) Validate(r *domain.StrategyResult, err error, opts ValidationOptions) Verdict {
	criteria := v.criteriaFor(opts)

	if err != nil && domain.IsTransient(err) {
		return VerdictPropagate{Cause: err}
	}
	if err != nil {
		return VerdictHardFail{Reason: "strategy execution failed", Cause: err}
	}
	if r == nil {
		return VerdictHardFail{Reason: "strategy returned no outcome telemetry", Cause: domain.ErrInsufficientOutput}
	}

	r.Finish()
	snapshot := r.Snapshot()
	completedDocs := snapshot.DocsWritten + snapshot.DocsSkipped

	if completedDocs >= criteria.MinDocsWritten {
		return VerdictOK{}
	}
	if opts.DryRun && snapshot.URLsAttempted >= criteria.MinDocsWritten && snapshot.DocsFailed == 0 {
		return VerdictOK{}
	}
	if snapshot.URLsAttempted == 0 && opts.FilterURL != "" {
		return VerdictRetryAlternative{
			Reason:      string(domain.DiagFilterZeroed),
			Diagnostics: diagnosticsOrDefault(snapshot, domain.DiagFilterZeroed, "URL filter excluded all candidate URLs"),
		}
	}
	if snapshot.URLsDiscovered > 0 && snapshot.URLsAttempted == 0 {
		return VerdictRetryAlternative{
			Reason:      "no_urls_attempted",
			Diagnostics: diagnosticsOrDefault(snapshot, domain.DiagNoDocuments, "No discovered URLs survived filtering or limits"),
		}
	}
	if snapshot.URLsAttempted > 20 {
		ratio := float64(completedDocs) / float64(snapshot.URLsAttempted)
		if ratio < criteria.MinSuccessRatio {
			return VerdictRetryAlternative{
				Reason:      fmt.Sprintf("high_failure_ratio: %.2f", ratio),
				Diagnostics: snapshot.Diagnostics,
			}
		}
	}
	if completedDocs == 0 {
		return VerdictHardFail{Reason: "extraction produced 0 documents", Cause: domain.ErrInsufficientOutput}
	}
	return VerdictHardFail{Reason: "extraction below minimum output threshold", Cause: domain.ErrInsufficientOutput}
}

func (v *Validator) criteriaFor(opts ValidationOptions) Criteria {
	criteria := v.criteria
	if opts.MinDocs > 0 {
		criteria.MinDocsWritten = opts.MinDocs
	}
	if opts.MinSuccessRatio > 0 {
		criteria.MinSuccessRatio = opts.MinSuccessRatio
	}
	return criteria
}

func diagnosticsOrDefault(snapshot domain.StrategyResultSnapshot, code domain.DiagnosticCode, message string) []domain.Diagnostic {
	if len(snapshot.Diagnostics) > 0 {
		return snapshot.Diagnostics
	}
	return []domain.Diagnostic{{Code: code, Message: message}}
}
