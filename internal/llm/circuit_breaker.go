package llm

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker protects against cascading failures
type CircuitBreaker interface {
	Allow() bool
	RecordSuccess()
	RecordFailure()
	State() CircuitState
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold         int
	SuccessThresholdHalfOpen int
	ResetTimeout             time.Duration
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:         5,
		SuccessThresholdHalfOpen: 1,
		ResetTimeout:             30 * time.Second,
	}
}

type circuitBreaker struct {
	config          CircuitBreakerConfig
	state           CircuitState
	failures        int
	successes       int
	lastStateChange time.Time
	mu              sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThresholdHalfOpen <= 0 {
		config.SuccessThresholdHalfOpen = 1
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}

	return &circuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Allow checks if a request is allowed to proceed
func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastStateChange) >= cb.config.ResetTimeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThresholdHalfOpen {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		cb.successes++
	}
}

// RecordFailure records a failed operation
func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes = 0

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

// State returns the current state
func (cb *circuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *circuitBreaker) transitionTo(newState CircuitState) {
	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.failures = 0
	cb.successes = 0
}

// NoOpCircuitBreaker always allows requests
type NoOpCircuitBreaker struct{}

// Allow always returns true
func (n *NoOpCircuitBreaker) Allow() bool {
	return true
}

// RecordSuccess does nothing
func (n *NoOpCircuitBreaker) RecordSuccess() {}

// RecordFailure does nothing
func (n *NoOpCircuitBreaker) RecordFailure() {}

// State always returns StateClosed
func (n *NoOpCircuitBreaker) State() CircuitState {
	return StateClosed
}
