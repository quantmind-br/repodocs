package llm_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_InitialState(t *testing.T) {
	cb := llm.NewCircuitBreaker(llm.DefaultCircuitBreakerConfig())
	assert.Equal(t, llm.StateClosed, cb.State())
	assert.True(t, cb.Allow())
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         3,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Second,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	assert.Equal(t, llm.StateClosed, cb.State())
	cb.RecordFailure()
	assert.Equal(t, llm.StateClosed, cb.State())
	cb.RecordFailure()
	assert.Equal(t, llm.StateOpen, cb.State())
}

func TestCircuitBreaker_RejectsWhenOpen(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         1,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Hour,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	assert.Equal(t, llm.StateOpen, cb.State())
	assert.False(t, cb.Allow())
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         1,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             50 * time.Millisecond,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	assert.Equal(t, llm.StateOpen, cb.State())

	time.Sleep(100 * time.Millisecond)
	assert.True(t, cb.Allow())
	assert.Equal(t, llm.StateHalfOpen, cb.State())
}

func TestCircuitBreaker_ClosesOnSuccessInHalfOpen(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         1,
		SuccessThresholdHalfOpen: 2,
		ResetTimeout:             50 * time.Millisecond,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	time.Sleep(100 * time.Millisecond)
	cb.Allow()

	assert.Equal(t, llm.StateHalfOpen, cb.State())
	cb.RecordSuccess()
	assert.Equal(t, llm.StateHalfOpen, cb.State())
	cb.RecordSuccess()
	assert.Equal(t, llm.StateClosed, cb.State())
}

func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         1,
		SuccessThresholdHalfOpen: 2,
		ResetTimeout:             50 * time.Millisecond,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	time.Sleep(100 * time.Millisecond)
	cb.Allow()

	assert.Equal(t, llm.StateHalfOpen, cb.State())
	cb.RecordFailure()
	assert.Equal(t, llm.StateOpen, cb.State())
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         3,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Second,
	}
	cb := llm.NewCircuitBreaker(config)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()

	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, llm.StateClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, llm.StateOpen, cb.State())
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         100,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Hour,
	}
	cb := llm.NewCircuitBreaker(config)

	var wg sync.WaitGroup
	var allowed int64
	goroutines := 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if cb.Allow() {
					atomic.AddInt64(&allowed, 1)
				}
				if idx%2 == 0 {
					cb.RecordSuccess()
				} else {
					cb.RecordFailure()
				}
			}
		}(i)
	}

	wg.Wait()

	state := cb.State()
	assert.True(t, state == llm.StateClosed || state == llm.StateOpen)
}

func TestCircuitBreaker_DefaultsInvalidConfig(t *testing.T) {
	config := llm.CircuitBreakerConfig{
		FailureThreshold:         -1,
		SuccessThresholdHalfOpen: -1,
		ResetTimeout:             -1,
	}
	cb := llm.NewCircuitBreaker(config)

	require.NotNil(t, cb)
	assert.True(t, cb.Allow())

	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	assert.Equal(t, llm.StateOpen, cb.State())
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    llm.CircuitState
		expected string
	}{
		{llm.StateClosed, "closed"},
		{llm.StateOpen, "open"},
		{llm.StateHalfOpen, "half-open"},
		{llm.CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNoOpCircuitBreaker(t *testing.T) {
	cb := &llm.NoOpCircuitBreaker{}

	assert.True(t, cb.Allow())
	assert.Equal(t, llm.StateClosed, cb.State())

	cb.RecordFailure()
	assert.True(t, cb.Allow())
	assert.Equal(t, llm.StateClosed, cb.State())

	cb.RecordSuccess()
	assert.Equal(t, llm.StateClosed, cb.State())
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := llm.DefaultCircuitBreakerConfig()

	assert.Equal(t, 5, config.FailureThreshold)
	assert.Equal(t, 1, config.SuccessThresholdHalfOpen)
	assert.Equal(t, 30*time.Second, config.ResetTimeout)
}
