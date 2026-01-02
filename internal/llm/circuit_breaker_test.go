package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitState_String tests state string representation
func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// TestDefaultCircuitBreakerConfig tests default config
func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	assert.Equal(t, 5, cfg.FailureThreshold)
	assert.Equal(t, 1, cfg.SuccessThresholdHalfOpen)
	assert.Equal(t, 30*time.Second, cfg.ResetTimeout)
}

// TestNewCircuitBreaker tests creating a circuit breaker
func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name  string
		cfg   CircuitBreakerConfig
		state CircuitState
	}{
		{
			name: "standard config",
			cfg: CircuitBreakerConfig{
				FailureThreshold:         5,
				SuccessThresholdHalfOpen: 1,
				ResetTimeout:             30 * time.Second,
			},
			state: StateClosed,
		},
		{
			name: "zero failure threshold defaults to 5",
			cfg: CircuitBreakerConfig{
				FailureThreshold:         0,
				SuccessThresholdHalfOpen: 1,
				ResetTimeout:             30 * time.Second,
			},
			state: StateClosed,
		},
		{
			name: "zero success threshold defaults to 1",
			cfg: CircuitBreakerConfig{
				FailureThreshold:         5,
				SuccessThresholdHalfOpen: 0,
				ResetTimeout:             30 * time.Second,
			},
			state: StateClosed,
		},
		{
			name: "zero reset timeout defaults to 30s",
			cfg: CircuitBreakerConfig{
				FailureThreshold:         5,
				SuccessThresholdHalfOpen: 1,
				ResetTimeout:             0,
			},
			state: StateClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := NewCircuitBreaker(tt.cfg)
			require.NotNil(t, cb)
			assert.Equal(t, tt.state, cb.State())
		})
	}
}

// TestCircuitBreaker_Allow tests allowing requests
func TestCircuitBreaker_Allow(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:         3,
		SuccessThresholdHalfOpen: 2,
		ResetTimeout:             100 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Initially closed, should allow
	assert.True(t, cb.Allow())
	assert.Equal(t, StateClosed, cb.State())

	// Record failures to trip the breaker
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Should not allow when open
	assert.False(t, cb.Allow())

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should now allow and transition to half-open
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())
}

// TestCircuitBreaker_RecordSuccess tests recording successes
func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	t.Run("success in closed state", func(t *testing.T) {
		cfg := CircuitBreakerConfig{
			FailureThreshold:         3,
			SuccessThresholdHalfOpen: 2,
			ResetTimeout:             30 * time.Second,
		}
		cb := NewCircuitBreaker(cfg)
		assert.Equal(t, StateClosed, cb.State())

		// Record success in closed state - stays closed
		cb.RecordSuccess()
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("success in half-open transitions to closed", func(t *testing.T) {
		cfg := CircuitBreakerConfig{
			FailureThreshold:         2,
			SuccessThresholdHalfOpen: 1,
			ResetTimeout:             50 * time.Millisecond,
		}
		cb := NewCircuitBreaker(cfg)

		// Trip the breaker
		cb.RecordFailure()
		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())

		// Wait for reset timeout
		time.Sleep(75 * time.Millisecond)

		// Move to half-open
		assert.True(t, cb.Allow())
		assert.Equal(t, StateHalfOpen, cb.State())

		// Success closes the circuit
		cb.RecordSuccess()
		assert.Equal(t, StateClosed, cb.State())
	})

	t.Run("partial success in half-open stays half-open", func(t *testing.T) {
		cfg := CircuitBreakerConfig{
			FailureThreshold:         2,
			SuccessThresholdHalfOpen: 2,
			ResetTimeout:             50 * time.Millisecond,
		}
		cb := NewCircuitBreaker(cfg)

		// Trip the breaker
		cb.RecordFailure()
		cb.RecordFailure()
		assert.Equal(t, StateOpen, cb.State())

		// Wait for reset timeout
		time.Sleep(75 * time.Millisecond)

		// Move to half-open
		assert.True(t, cb.Allow())
		assert.Equal(t, StateHalfOpen, cb.State())

		// Partial success stays half-open
		cb.RecordSuccess()
		assert.Equal(t, StateHalfOpen, cb.State())
	})
}

// TestCircuitBreaker_RecordFailure tests recording failures
func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:         3,
		SuccessThresholdHalfOpen: 2,
		ResetTimeout:             30 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	// Record failures up to threshold
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

// TestCircuitBreaker_HalfOpenFailure tests that failure in half-open opens circuit
func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:         2,
		SuccessThresholdHalfOpen: 2,
		ResetTimeout:             50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Wait for reset timeout
	time.Sleep(75 * time.Millisecond)

	// Move to half-open
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())

	// Record failure in half-open
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())
}

// TestCircuitBreaker_StateTransition tests state transitions
func TestCircuitBreaker_StateTransition(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:         2,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             50 * time.Millisecond,
	}
	cb := NewCircuitBreaker(cfg)

	// Start in closed state
	assert.Equal(t, StateClosed, cb.State())

	// Trip to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, StateOpen, cb.State())

	// Wait for half-open
	time.Sleep(75 * time.Millisecond)
	assert.True(t, cb.Allow())
	assert.Equal(t, StateHalfOpen, cb.State())

	// Success closes the circuit
	cb.RecordSuccess()
	assert.Equal(t, StateClosed, cb.State())
}

// TestCircuitBreaker_ConcurrentAccess tests concurrent access safety
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cfg := CircuitBreakerConfig{
		FailureThreshold:         100,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             1 * time.Second,
	}
	cb := NewCircuitBreaker(cfg)

	// Concurrent operations
	done := make(chan bool)
	for i := 0; i < 50; i++ {
		go func() {
			cb.Allow()
			cb.RecordSuccess()
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		go func() {
			cb.Allow()
			cb.RecordFailure()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not panic and state should be valid
	state := cb.State()
	assert.True(t, state == StateClosed || state == StateOpen || state == StateHalfOpen)
}

// TestNoOpCircuitBreaker tests the no-op circuit breaker
func TestNoOpCircuitBreaker(t *testing.T) {
	cb := &NoOpCircuitBreaker{}

	// Should always allow
	assert.True(t, cb.Allow())

	// Should not panic on recording - call multiple times to ensure coverage
	for i := 0; i < 5; i++ {
		cb.RecordSuccess()
		cb.RecordFailure()
	}

	// Should always return closed
	assert.Equal(t, StateClosed, cb.State())
}

// TestCircuitBreakerInterface tests that both implement the interface
func TestCircuitBreakerInterface(t *testing.T) {
	var cb CircuitBreaker

	// Real circuit breaker
	cb = NewCircuitBreaker(DefaultCircuitBreakerConfig())
	assert.NotNil(t, cb)
	assert.True(t, cb.Allow())
	cb.RecordSuccess()
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())

	// No-op circuit breaker
	cb = &NoOpCircuitBreaker{}
	assert.NotNil(t, cb)
	assert.True(t, cb.Allow())
	cb.RecordSuccess()
	cb.RecordFailure()
	assert.Equal(t, StateClosed, cb.State())
}
