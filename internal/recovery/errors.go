package recovery

import (
	"fmt"
	"strings"

	"github.com/quantmind-br/repodocs/internal/domain"
)

// OutcomeError wraps a strategy result and verdict into a user-facing,
// actionable error message.
type OutcomeError struct {
	Verdict     Verdict
	Result      *domain.StrategyResult
	Suggestions []string
}

// NewOutcomeError creates an OutcomeError from a verdict and result.
func NewOutcomeError(verdict Verdict, result *domain.StrategyResult) *OutcomeError {
	return &OutcomeError{
		Verdict:     verdict,
		Result:      result,
		Suggestions: suggestionsFor(verdict, result),
	}
}

// Error formats the outcome for human consumption.
func (e *OutcomeError) Error() string {
	var b strings.Builder
	reason := verdictReason(e.Verdict)
	if reason == "" {
		reason = "extraction outcome was unsatisfactory"
	}
	fmt.Fprintf(&b, "extraction outcome unsatisfactory: %s", reason)

	if e.Result != nil {
		snapshot := e.Result.Snapshot()
		fmt.Fprintf(&b, "\n\nStrategy:    %s", snapshot.Strategy)
		fmt.Fprintf(&b, "\nEntry URL:   %s", snapshot.EntryURL)
		fmt.Fprintf(&b, "\nDiscovered:  %d URLs", snapshot.URLsDiscovered)
		fmt.Fprintf(&b, "\nAttempted:   %d URLs", snapshot.URLsAttempted)
		fmt.Fprintf(&b, "\nWritten:     %d docs", snapshot.DocsWritten)
		fmt.Fprintf(&b, "\nSkipped:     %d docs", snapshot.DocsSkipped)
		fmt.Fprintf(&b, "\nFailed:      %d docs", snapshot.DocsFailed)
		if len(snapshot.Diagnostics) > 0 {
			b.WriteString("\nDiagnostics:")
			for _, d := range snapshot.Diagnostics {
				fmt.Fprintf(&b, "\n  - %s: %s", d.Code, d.Message)
				if d.Hint != "" {
					fmt.Fprintf(&b, " (%s)", d.Hint)
				}
			}
		}
	}

	if len(e.Suggestions) > 0 {
		b.WriteString("\n\nSuggestions:")
		for _, s := range e.Suggestions {
			fmt.Fprintf(&b, "\n  - %s", s)
		}
	}
	return b.String()
}

// Unwrap returns the underlying error for HardFail and Propagate verdicts,
// supporting errors.Is / errors.As chains.
func (e *OutcomeError) Unwrap() error {
	switch v := e.Verdict.(type) {
	case VerdictHardFail:
		return v.Cause
	case VerdictPropagate:
		return v.Cause
	default:
		return nil
	}
}

func verdictReason(verdict Verdict) string {
	switch v := verdict.(type) {
	case VerdictRetryAlternative:
		return v.Reason
	case VerdictHardFail:
		return v.Reason
	case VerdictPropagate:
		if v.Cause != nil {
			return v.Cause.Error()
		}
	}
	return ""
}

func suggestionsFor(verdict Verdict, result *domain.StrategyResult) []string {
	var suggestions []string
	if retry, ok := verdict.(VerdictRetryAlternative); ok {
		switch retry.Reason {
		case string(domain.DiagFilterZeroed), "no_urls_attempted":
			suggestions = append(suggestions,
				"Try the filtered URL as the entry point instead of only --filter",
				"Check filter URL spelling and trailing slash",
				"Force a different strategy after Phase 2 adds --strategy")
		}
	}
	if result != nil {
		snapshot := result.Snapshot()
		if snapshot.Strategy == "sitemap" && snapshot.URLsDiscovered > 0 && snapshot.URLsAttempted == 0 {
			suggestions = append(suggestions, "The sitemap may be too shallow for this path; try crawler on the filtered URL")
		}
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Run again with --verbose to inspect strategy diagnostics")
	}
	return suggestions
}
