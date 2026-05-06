package recovery_test

import (
	"errors"
	"testing"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/recovery"
	"github.com/stretchr/testify/assert"
)

func TestOutcomeError_Error_FilterZeroed(t *testing.T) {
	result := domain.NewStrategyResult("sitemap", "https://example.com/sitemap.txt")
	result.AddDiscovered(3)
	result.AddDiagnostic(domain.DiagFilterZeroed, "URL filter excluded every sitemap URL", "try crawler")
	result.Finish()

	err := recovery.NewOutcomeError(recovery.VerdictRetryAlternative{
		Reason:      string(domain.DiagFilterZeroed),
		Diagnostics: result.Snapshot().Diagnostics,
	}, result)

	msg := err.Error()
	assert.Contains(t, msg, "extraction outcome unsatisfactory")
	assert.Contains(t, msg, "filter_zeroed")
	assert.Contains(t, msg, "Strategy:    sitemap")
	assert.Contains(t, msg, "Discovered:  3 URLs")
	assert.Contains(t, msg, "Attempted:   0 URLs")
	assert.Contains(t, msg, "Suggestions:")
	assert.Contains(t, msg, "filtered URL as the entry point")
}

func TestOutcomeError_Unwrap_HardFail(t *testing.T) {
	base := errors.New("base")
	err := recovery.NewOutcomeError(recovery.VerdictHardFail{Reason: "failed", Cause: base}, nil)
	assert.ErrorIs(t, err, base)
}
