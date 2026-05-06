package recovery_test

import (
	"testing"

	"github.com/quantmind-br/repodocs/internal/domain"
	"github.com/quantmind-br/repodocs/internal/recovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_Validate(t *testing.T) {
	validator := recovery.NewValidator(nil)

	tests := []struct {
		name      string
		result    func() *domain.StrategyResult
		err       error
		opts      recovery.ValidationOptions
		wantType  any
		wantCause error
	}{
		{
			name: "written docs ok",
			result: func() *domain.StrategyResult {
				r := domain.NewStrategyResult("crawler", "https://example.com")
				r.IncAttempted()
				r.IncWritten()
				r.Finish()
				return r
			},
			wantType: recovery.VerdictOK{},
		},
		{
			name: "skipped docs ok",
			result: func() *domain.StrategyResult {
				r := domain.NewStrategyResult("sitemap", "https://example.com/sitemap.xml")
				r.IncAttempted()
				r.IncSkipped()
				r.Finish()
				return r
			},
			wantType: recovery.VerdictOK{},
		},
		{
			name: "dry run attempted ok",
			result: func() *domain.StrategyResult {
				r := domain.NewStrategyResult("crawler", "https://example.com")
				r.IncAttempted()
				r.Finish()
				return r
			},
			opts:     recovery.ValidationOptions{DryRun: true},
			wantType: recovery.VerdictOK{},
		},
		{
			name: "filter zeroed retries alternative",
			result: func() *domain.StrategyResult {
				r := domain.NewStrategyResult("sitemap", "https://example.com/sitemap.txt")
				r.AddDiscovered(3)
				r.AddDiagnostic(domain.DiagFilterZeroed, "excluded", "try crawler")
				r.Finish()
				return r
			},
			opts:     recovery.ValidationOptions{FilterURL: "https://example.com/book/"},
			wantType: recovery.VerdictRetryAlternative{},
		},
		{
			name:      "transient propagates",
			result:    func() *domain.StrategyResult { return domain.NewStrategyResult("crawler", "https://example.com") },
			err:       domain.ErrTimeout,
			wantType:  recovery.VerdictPropagate{},
			wantCause: domain.ErrTimeout,
		},
		{
			name: "zero docs hard fail",
			result: func() *domain.StrategyResult {
				r := domain.NewStrategyResult("crawler", "https://example.com")
				r.Finish()
				return r
			},
			wantType:  recovery.VerdictHardFail{},
			wantCause: domain.ErrInsufficientOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := validator.Validate(tt.result(), tt.err, tt.opts)
			assert.IsType(t, tt.wantType, verdict)
			if tt.wantCause != nil {
				switch v := verdict.(type) {
				case recovery.VerdictPropagate:
					assert.ErrorIs(t, v.Cause, tt.wantCause)
				case recovery.VerdictHardFail:
					assert.ErrorIs(t, v.Cause, tt.wantCause)
				default:
					require.Failf(t, "unexpected verdict", "%T", verdict)
				}
			}
		})
	}
}
