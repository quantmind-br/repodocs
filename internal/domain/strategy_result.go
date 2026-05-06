package domain

import (
	"sync"
	"time"
)

// StrategyResult reports the observable outcome of one Strategy.Execute call.
// Counters are populated even when Execute returns an error because partial
// progress is useful for validation and user-facing diagnostics.
type StrategyResult struct {
	mu        sync.Mutex
	startedAt time.Time

	Strategy       string
	EntryURL       string
	URLsDiscovered int
	URLsAttempted  int
	DocsWritten    int
	DocsSkipped    int
	DocsFailed     int
	BytesWritten   int64
	Diagnostics    []Diagnostic
	Duration       time.Duration
}

// Diagnostic is a structured signal emitted by a strategy for the recovery
// validator and later fallback planner phases.
type Diagnostic struct {
	Code    DiagnosticCode
	Message string
	Hint    string
}

// DiagnosticCode identifies a machine-readable extraction outcome signal.
type DiagnosticCode string

const (
	DiagFilterZeroed      DiagnosticCode = "filter_zeroed"
	DiagSitemapShallow    DiagnosticCode = "sitemap_shallow"
	DiagAllFetchesFailed  DiagnosticCode = "all_fetches_failed"
	DiagAllFetchesBlocked DiagnosticCode = "all_fetches_blocked"
	DiagEmptyContent      DiagnosticCode = "empty_content"
	DiagRedirectLoop      DiagnosticCode = "redirect_loop"
	DiagJSRequired        DiagnosticCode = "js_required"
	DiagNoDocuments       DiagnosticCode = "no_documents"
)

// NewStrategyResult creates an empty result for strategy execution.
func NewStrategyResult(strategy, entryURL string) *StrategyResult {
	return &StrategyResult{
		Strategy:  strategy,
		EntryURL:  entryURL,
		startedAt: time.Now(),
	}
}

// NewBasicResult creates a result suitable for mechanical migrations before a
// strategy has detailed counters. Later phases should populate exact counters.
func NewBasicResult(strategy, entryURL string) *StrategyResult {
	return NewStrategyResult(strategy, entryURL)
}

// Finish records elapsed duration once. It is safe to call multiple times.
func (r *StrategyResult) Finish() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Duration == 0 && !r.startedAt.IsZero() {
		r.Duration = time.Since(r.startedAt)
	}
}

func (r *StrategyResult) AddDiscovered(n int) {
	if r == nil || n <= 0 {
		return
	}
	r.mu.Lock()
	r.URLsDiscovered += n
	r.mu.Unlock()
}

func (r *StrategyResult) AddAttempted(n int) {
	if r == nil || n <= 0 {
		return
	}
	r.mu.Lock()
	r.URLsAttempted += n
	r.mu.Unlock()
}

func (r *StrategyResult) IncDiscovered() { r.AddDiscovered(1) }
func (r *StrategyResult) IncAttempted()  { r.AddAttempted(1) }

func (r *StrategyResult) IncWritten() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.DocsWritten++
	r.mu.Unlock()
}

func (r *StrategyResult) IncSkipped() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.DocsSkipped++
	r.mu.Unlock()
}

func (r *StrategyResult) IncFailed() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.DocsFailed++
	r.mu.Unlock()
}

func (r *StrategyResult) AddBytesWritten(n int64) {
	if r == nil || n <= 0 {
		return
	}
	r.mu.Lock()
	r.BytesWritten += n
	r.mu.Unlock()
}

func (r *StrategyResult) AddDiagnostic(code DiagnosticCode, message, hint string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Code:    code,
		Message: message,
		Hint:    hint,
	})
	r.mu.Unlock()
}

func (r *StrategyResult) HasDiagnostic(code DiagnosticCode) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

// CompletedDocs returns documents that are acceptable output for validation:
// freshly written files plus skipped existing/synced files.
func (r *StrategyResult) CompletedDocs() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.DocsWritten + r.DocsSkipped
}

// Snapshot returns a copy without sharing the internal mutex.
func (r *StrategyResult) Snapshot() StrategyResult {
	if r == nil {
		return StrategyResult{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return StrategyResult{
		Strategy:       r.Strategy,
		EntryURL:       r.EntryURL,
		URLsDiscovered: r.URLsDiscovered,
		URLsAttempted:  r.URLsAttempted,
		DocsWritten:    r.DocsWritten,
		DocsSkipped:    r.DocsSkipped,
		DocsFailed:     r.DocsFailed,
		BytesWritten:   r.BytesWritten,
		Diagnostics:    append([]Diagnostic(nil), r.Diagnostics...),
		Duration:       r.Duration,
	}
}
