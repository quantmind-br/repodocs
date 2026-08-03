package domain

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStrategyResult_InitializesFields(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")
	require.NotNil(t, r)
	assert.Equal(t, "crawler", r.Strategy)
	assert.Equal(t, "https://example.com", r.EntryURL)
	assert.Equal(t, 0, r.URLsDiscovered)
	assert.NotZero(t, r.startedAt)
}

func TestNewBasicResult_DelegatesToNewStrategyResult(t *testing.T) {
	r := NewBasicResult("git", "https://github.com/org/repo")
	require.NotNil(t, r)
	assert.Equal(t, "git", r.Strategy)
	assert.Equal(t, "https://github.com/org/repo", r.EntryURL)
}

func TestStrategyResult_CounterSemantics(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")

	r.AddDiscovered(3)
	r.AddAttempted(2)
	assert.Equal(t, 3, r.URLsDiscovered)
	assert.Equal(t, 2, r.URLsAttempted)

	r.IncDiscovered()
	r.IncAttempted()
	assert.Equal(t, 4, r.URLsDiscovered)
	assert.Equal(t, 3, r.URLsAttempted)

	r.IncWritten()
	r.IncSkipped()
	r.IncFailed()
	assert.Equal(t, 1, r.DocsWritten)
	assert.Equal(t, 1, r.DocsSkipped)
	assert.Equal(t, 1, r.DocsFailed)

	r.AddBytesWritten(1024)
	assert.Equal(t, int64(1024), r.BytesWritten)
}

func TestStrategyResult_NonPositiveAddsAreIgnored(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")

	r.AddDiscovered(-5)
	r.AddDiscovered(0)
	r.AddAttempted(-1)
	r.AddBytesWritten(-10)
	r.AddBytesWritten(0)

	assert.Equal(t, 0, r.URLsDiscovered)
	assert.Equal(t, 0, r.URLsAttempted)
	assert.Equal(t, int64(0), r.BytesWritten)
}

func TestStrategyResult_FinishRecordsDurationOnce(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")
	// Backdate startedAt so the elapsed duration is non-zero even on
	// platforms with coarse clocks (e.g. Windows 15ms timers).
	r.startedAt = time.Now().Add(-50 * time.Millisecond)

	r.Finish()
	d1 := r.Duration
	assert.NotZero(t, d1)

	// Second call must not overwrite the recorded duration
	time.Sleep(5 * time.Millisecond)
	r.Finish()
	assert.Equal(t, d1, r.Duration)
}

func TestStrategyResult_AddDiagnosticAndHasDiagnostic(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")

	r.AddDiagnostic(DiagEmptyContent, "page was empty", "enable JS rendering")
	r.AddDiagnostic(DiagRedirectLoop, "redirect loop detected", "")

	assert.True(t, r.HasDiagnostic(DiagEmptyContent))
	assert.True(t, r.HasDiagnostic(DiagRedirectLoop))
	assert.False(t, r.HasDiagnostic(DiagAllFetchesFailed))
	assert.Equal(t, 2, len(r.Diagnostics))
	assert.Equal(t, "page was empty", r.Diagnostics[0].Message)
	assert.Equal(t, "enable JS rendering", r.Diagnostics[0].Hint)
}

func TestStrategyResult_CompletedDocsSumsWrittenAndSkipped(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")
	r.IncWritten()
	r.IncWritten()
	r.IncSkipped()
	assert.Equal(t, 3, r.CompletedDocs())
}

func TestStrategyResult_SnapshotCopiesState(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")
	r.AddDiscovered(7)
	r.AddAttempted(4)
	r.IncWritten()
	r.IncSkipped()
	r.IncFailed()
	r.AddBytesWritten(2048)
	r.AddDiagnostic(DiagNoDocuments, "nothing found", "")
	r.Finish()

	snap := r.Snapshot()
	assert.Equal(t, "crawler", snap.Strategy)
	assert.Equal(t, "https://example.com", snap.EntryURL)
	assert.Equal(t, 7, snap.URLsDiscovered)
	assert.Equal(t, 4, snap.URLsAttempted)
	assert.Equal(t, 1, snap.DocsWritten)
	assert.Equal(t, 1, snap.DocsSkipped)
	assert.Equal(t, 1, snap.DocsFailed)
	assert.Equal(t, int64(2048), snap.BytesWritten)
	assert.Equal(t, 1, len(snap.Diagnostics))
	assert.Equal(t, r.Duration, snap.Duration)

	// The snapshot must be isolated from subsequent mutations
	r.IncWritten()
	r.AddDiagnostic(DiagJSRequired, "needs JS", "")
	assert.Equal(t, 1, snap.DocsWritten, "snapshot must not reflect later writes")
	assert.Equal(t, 1, len(snap.Diagnostics), "snapshot diagnostics must be copied")

	// Mutating the snapshot's diagnostics slice must not affect the source
	snap.Diagnostics[0].Message = "tampered"
	assert.Equal(t, "nothing found", r.Diagnostics[0].Message)
}

func TestStrategyResult_ConcurrentMutation(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")

	var wg sync.WaitGroup
	const goroutines = 8
	const opsPerGoroutine = 1000

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range opsPerGoroutine {
				r.IncWritten()
				r.AddDiscovered(1)
				r.AddBytesWritten(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, goroutines*opsPerGoroutine, r.DocsWritten)
	assert.Equal(t, goroutines*opsPerGoroutine, r.URLsDiscovered)
	assert.Equal(t, int64(goroutines*opsPerGoroutine), r.BytesWritten)
}

func TestStrategyResult_SnapshotIsLockFree(t *testing.T) {
	r := NewStrategyResult("crawler", "https://example.com")

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Mutate concurrently while taking snapshots
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				r.IncWritten()
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = r.Snapshot()
			}
		}
	}()

	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestStrategyResult_NilReceiverSafety(t *testing.T) {
	var r *StrategyResult

	r.Finish()
	r.AddDiscovered(1)
	r.AddAttempted(1)
	r.IncDiscovered()
	r.IncAttempted()
	r.IncWritten()
	r.IncSkipped()
	r.IncFailed()
	r.AddBytesWritten(1)
	r.AddDiagnostic(DiagEmptyContent, "", "")

	assert.False(t, r.HasDiagnostic(DiagEmptyContent))
	assert.Equal(t, 0, r.CompletedDocs())
	assert.Equal(t, StrategyResultSnapshot{}, r.Snapshot())
}
