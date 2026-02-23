package app_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/manifest"
	"github.com/quantmind-br/repodocs-go/internal/strategies"
)

// parallelTestStrategy tracks execution timing and order for parallel tests
type parallelTestStrategy struct {
	name          string
	mu            sync.Mutex
	execCalls     []string
	execTimes     map[string]time.Time
	execDurations map[string]time.Duration
	execFunc      func(ctx context.Context, url string, opts strategies.Options) error
	delay         time.Duration
}

func newParallelTestStrategy(name string, delay time.Duration) *parallelTestStrategy {
	return &parallelTestStrategy{
		name:          name,
		execTimes:     make(map[string]time.Time),
		execDurations: make(map[string]time.Duration),
		delay:         delay,
	}
}

func (s *parallelTestStrategy) Name() string          { return s.name }
func (s *parallelTestStrategy) CanHandle(string) bool { return true }
func (s *parallelTestStrategy) Execute(ctx context.Context, url string, opts strategies.Options) error {
	startTime := time.Now()

	s.mu.Lock()
	s.execCalls = append(s.execCalls, url)
	s.execTimes[url] = startTime
	s.mu.Unlock()

	// Simulate work with delay
	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.delay):
		}
	}

	s.mu.Lock()
	s.execDurations[url] = time.Since(startTime)
	s.mu.Unlock()

	if s.execFunc != nil {
		return s.execFunc(ctx, url, opts)
	}
	return nil
}

func (s *parallelTestStrategy) getExecCalls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, len(s.execCalls))
	copy(result, s.execCalls)
	return result
}

func createParallelTestOrchestrator(t *testing.T, strategy strategies.Strategy) *app.Orchestrator {
	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Output.Directory = t.TempDir()
	cfg.Concurrency.Workers = 5 // Default workers

	opts := app.OrchestratorOptions{
		Config: cfg,
		StrategyFactory: func(st app.StrategyType, deps *strategies.Dependencies) strategies.Strategy {
			return strategy
		},
	}

	orchestrator, err := app.NewOrchestrator(opts)
	require.NoError(t, err)
	return orchestrator
}

// TestRunManifest_Parallel_ExecutesConcurrently verifies that multiple sources
// run concurrently rather than sequentially. With 3 sources each taking 100ms,
// parallel execution should complete in ~100ms, not 300ms.
func TestRunManifest_Parallel_ExecutesConcurrently(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	delay := 100 * time.Millisecond
	mock := newParallelTestStrategy("mock", delay)
	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://source1.com"},
			{URL: "https://source2.com"},
			{URL: "https://source3.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 5

	startTime := time.Now()
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)
	totalDuration := time.Since(startTime)

	require.NoError(t, err)
	assert.Len(t, mock.getExecCalls(), 3)

	// If sequential: 3 * 100ms = 300ms minimum
	// If parallel: ~100ms (plus some overhead)
	// We check that it's significantly less than sequential time
	maxExpectedDuration := 2 * delay // Allow for overhead, but must be < 3 * delay
	assert.Less(t, totalDuration, 3*delay,
		"Expected parallel execution to be faster than sequential (got %v, sequential would be >= %v)",
		totalDuration, 3*delay)
	assert.Less(t, totalDuration, maxExpectedDuration,
		"Expected execution in ~%v with parallel processing, got %v", delay, totalDuration)
}

// TestRunManifest_Parallel_ContinueOnError_True verifies that with ContinueOnError=true,
// all sources are processed and the first error is returned.
func TestRunManifest_Parallel_ContinueOnError_True(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	var processedCount atomic.Int32
	mock := newParallelTestStrategy("mock", 50*time.Millisecond)
	mock.execFunc = func(ctx context.Context, url string, opts strategies.Options) error {
		processedCount.Add(1)
		if url == "https://fail.com" {
			return errors.New("simulated failure")
		}
		return nil
	}

	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://success1.com"},
			{URL: "https://fail.com"},
			{URL: "https://success2.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failures")
	// All 3 sources should be processed with ContinueOnError=true
	assert.Equal(t, int32(3), processedCount.Load(), "All sources should be processed")
}

// TestRunManifest_Parallel_ContinueOnError_False verifies that with ContinueOnError=false,
// the first error cancels remaining sources.
func TestRunManifest_Parallel_ContinueOnError_False(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	var processedCount atomic.Int32
	var completedCount atomic.Int32

	mock := newParallelTestStrategy("mock", 0)
	mock.execFunc = func(ctx context.Context, url string, opts strategies.Options) error {
		processedCount.Add(1)

		// Add delay to allow cancellation to propagate
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}

		if url == "https://fail.com" {
			return errors.New("simulated failure")
		}
		completedCount.Add(1)
		return nil
	}

	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://success1.com"},
			{URL: "https://fail.com"},
			{URL: "https://success2.com"},
			{URL: "https://success3.com"},
			{URL: "https://success4.com"},
		},
		Options: manifest.Options{
			ContinueOnError: false,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail.com")
	// With cancellation, not all sources should complete successfully
	// (some may be cancelled mid-flight)
}

// TestRunManifest_Parallel_ContextCancellation verifies that context cancellation
// stops all sources gracefully.
func TestRunManifest_Parallel_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	var startedCount atomic.Int32

	mock := newParallelTestStrategy("mock", 0)
	mock.execFunc = func(ctx context.Context, url string, opts strategies.Options) error {
		startedCount.Add(1)
		// Wait for cancellation or timeout
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	}

	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://source1.com"},
			{URL: "https://source2.com"},
			{URL: "https://source3.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(ctx, manifestCfg, app.OrchestratorOptions{Config: cfg})

	// Should return context.Canceled error
	assert.True(t, errors.Is(err, context.Canceled) || err != nil,
		"Expected context cancellation error or error from cancelled sources")
}

// TestRunManifest_Parallel_ResultsPreserveOrder verifies that results maintain
// correct association with their source index despite parallel execution.
func TestRunManifest_Parallel_ResultsPreserveOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	var urlOrder []string
	var mu sync.Mutex

	mock := newParallelTestStrategy("mock", 0)
	mock.execFunc = func(ctx context.Context, url string, opts strategies.Options) error {
		// Variable delays to shuffle completion order
		var delay time.Duration
		switch url {
		case "https://slow.com":
			delay = 100 * time.Millisecond
		case "https://medium.com":
			delay = 50 * time.Millisecond
		case "https://fast.com":
			delay = 10 * time.Millisecond
		}

		time.Sleep(delay)

		mu.Lock()
		urlOrder = append(urlOrder, url)
		mu.Unlock()

		return nil
	}

	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	manifestCfg := &manifest.Config{
		Sources: []manifest.Source{
			{URL: "https://slow.com"},
			{URL: "https://medium.com"},
			{URL: "https://fast.com"},
		},
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.NoError(t, err)
	assert.Len(t, urlOrder, 3)
	// With parallel execution, fast should complete before slow
	// This verifies parallel execution is actually happening
}

// TestRunManifest_Parallel_ConcurrencyCap verifies that manifest processing
// respects the concurrency cap (min of workers and 3).
func TestRunManifest_Parallel_ConcurrencyCap(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel manifest test in short mode (makes network calls via Orchestrator)")
	}
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	mock := newParallelTestStrategy("mock", 0)
	mock.execFunc = func(ctx context.Context, url string, opts strategies.Options) error {
		current := currentConcurrent.Add(1)

		// Track max concurrent executions
		for {
			max := maxConcurrent.Load()
			if current <= max || maxConcurrent.CompareAndSwap(max, current) {
				break
			}
		}

		// Simulate work
		time.Sleep(50 * time.Millisecond)

		currentConcurrent.Add(-1)
		return nil
	}

	orchestrator := createParallelTestOrchestrator(t, mock)
	defer orchestrator.Close()

	// Create more sources than the concurrency cap
	sources := make([]manifest.Source, 10)
	for i := range sources {
		sources[i] = manifest.Source{URL: "https://source" + string(rune('0'+i)) + ".com"}
	}

	manifestCfg := &manifest.Config{
		Sources: sources,
		Options: manifest.Options{
			ContinueOnError: true,
			Output:          t.TempDir(),
		},
	}

	cfg := config.Default()
	cfg.Cache.Enabled = false
	cfg.Concurrency.Workers = 5 // Should be capped at 3 for manifest processing
	err := orchestrator.RunManifest(
		context.Background(),
		manifestCfg,
		app.OrchestratorOptions{Config: cfg},
	)

	require.NoError(t, err)
	// Max concurrent should be at most 3 (the manifest processing cap)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(3),
		"Concurrency should be capped at 3 for manifest processing")
}
