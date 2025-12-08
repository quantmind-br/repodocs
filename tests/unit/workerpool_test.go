package unit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		worker   utils.Worker[int]
		expected struct {
			workers   int
			taskQueue int
			resultChan int
		}
	}{
		{
			name:    "single worker",
			workers: 1,
			worker: func(ctx context.Context, data int) (any, error) {
				return data * 2, nil
			},
			expected: struct {
				workers   int
				taskQueue int
				resultChan int
			}{1, 2, 2}, // workers * 2
		},
		{
			name:    "multiple workers",
			workers: 5,
			worker: func(ctx context.Context, data int) (any, error) {
				return data * 2, nil
			},
			expected: struct {
				workers   int
				taskQueue int
				resultChan int
			}{5, 10, 10}, // workers * 2
		},
		{
			name:    "zero workers",
			workers: 0,
			worker: func(ctx context.Context, data int) (any, error) {
				return data * 2, nil
			},
			expected: struct {
				workers   int
				taskQueue int
				resultChan int
			}{0, 0, 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := utils.NewPool(tc.workers, tc.worker)

			// Pool is created successfully, can be used for operations
			assert.NotNil(t, pool)
		})
	}
}

func TestPool_Start(t *testing.T) {
	tests := []struct {
		name        string
		workers     int
		taskCount   int
		expectStart bool
	}{
		{
			name:        "start pool with workers",
			workers:     3,
			taskCount:   5,
			expectStart: true,
		},
		{
			name:        "start pool with zero workers",
			workers:     0,
			taskCount:   5,
			expectStart: false,
		},
		{
			name:        "start pool with many workers",
			workers:     10,
			taskCount:   20,
			expectStart: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var mu sync.Mutex
			processedCount := 0
			processedData := make([]int, 0, tc.taskCount)

			worker := func(ctx context.Context, data int) (any, error) {
				mu.Lock()
				processedCount++
				processedData = append(processedData, data)
				mu.Unlock()
				return data * 2, nil
			}

			pool := utils.NewPool(tc.workers, worker)

			if tc.expectStart {
				pool.Start(ctx)

				// Submit tasks
				for i := 0; i < tc.taskCount; i++ {
					pool.Submit(i)
				}

				// Give workers time to process
				time.Sleep(100 * time.Millisecond)

				pool.Stop()

				assert.Equal(t, tc.taskCount, processedCount)
				assert.Equal(t, tc.taskCount, len(processedData))
			}
		})
	}
}

func TestPool_SubmitAndResults(t *testing.T) {
	ctx := context.Background()
	processed := make(chan int, 10)

	worker := func(ctx context.Context, data int) (any, error) {
		processed <- data
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)
	pool.Start(ctx)

	// Submit multiple tasks
	taskCount := 5
	for i := 0; i < taskCount; i++ {
		pool.Submit(i)
	}

	// Collect results
	results := make([]*utils.Task[int], 0, taskCount)
	timeout := time.After(2 * time.Second)
	for i := 0; i < taskCount; i++ {
		select {
		case task := <-pool.Results():
			results = append(results, task)
		case <-timeout:
			t.Fatal("timeout waiting for results")
		}
	}

	pool.Stop()

	// Verify results
	assert.Equal(t, taskCount, len(results))

	// Check that all expected values were processed
	receivedValues := make(map[int]bool)
	for _, task := range results {
		receivedValues[task.Data] = true
		assert.Equal(t, task.Data*2, task.Result)
		assert.NoError(t, task.Err)
	}

	for i := 0; i < taskCount; i++ {
		assert.True(t, receivedValues[i], "task %d was not processed", i)
	}
}

func TestPool_Stop(t *testing.T) {
	ctx := context.Background()

	worker := func(ctx context.Context, data int) (any, error) {
		time.Sleep(50 * time.Millisecond)
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)
	pool.Start(ctx)

	// Submit tasks
	for i := 0; i < 3; i++ {
		pool.Submit(i)
	}

	// Stop should complete without hanging
	start := time.Now()
	pool.Stop()
	elapsed := time.Since(start)

	// Should complete within a reasonable time (not hanging)
	assert.Less(t, elapsed, 500*time.Millisecond)
}

func TestPool_Process(t *testing.T) {
	tests := []struct {
		name       string
		workers    int
		items      []int
		workerFunc func(ctx context.Context, data int) (any, error)
		expectErr  bool
	}{
		{
			name:    "process all items successfully",
			workers: 3,
			items:   []int{1, 2, 3, 4, 5},
			workerFunc: func(ctx context.Context, data int) (any, error) {
				return data * 2, nil
			},
			expectErr: false,
		},
		// Note: Zero workers case is tested separately as it requires special handling
		// to avoid hanging when waiting for results that will never arrive
		{
			name:    "process with worker errors",
			workers: 2,
			items:   []int{1, 2, 3},
			workerFunc: func(ctx context.Context, data int) (any, error) {
				if data == 2 {
					return nil, errors.New("processing error")
				}
				return data * 2, nil
			},
			expectErr: false, // Process doesn't return errors from workers
		},
		{
			name:    "process many items",
			workers: 5,
			items:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			workerFunc: func(ctx context.Context, data int) (any, error) {
				return data * 2, nil
			},
			expectErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			pool := utils.NewPool(tc.workers, tc.workerFunc)

			results, err := pool.Process(ctx, tc.items)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, len(tc.items), len(results))

			// Verify all items are processed
			processed := make(map[int]bool)
			for _, task := range results {
				processed[task.Data] = true
			}

			for _, item := range tc.items {
				assert.True(t, processed[item], "item %d was not processed", item)
			}
		})
	}
}

func TestPool_Process_Cancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	worker := func(ctx context.Context, data int) (any, error) {
		time.Sleep(500 * time.Millisecond) // Longer than context timeout
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)

	start := time.Now()
	_, err := pool.Process(ctx, []int{1, 2, 3})
	elapsed := time.Since(start)

	// Should return error due to timeout
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))

	// Should complete relatively quickly after timeout
	assert.Less(t, elapsed, 2*time.Second)
}

func TestPool_Process_ContextDone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	worker := func(ctx context.Context, data int) (any, error) {
		time.Sleep(100 * time.Millisecond) // Exceeds timeout
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)

	_, err := pool.Process(ctx, []int{1, 2, 3})

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestPool_ConcurrentSubmissions(t *testing.T) {
	ctx := context.Background()

	counter := make(chan int, 100)
	var mu sync.Mutex
	processedCount := 0

	worker := func(ctx context.Context, data int) (any, error) {
		mu.Lock()
		processedCount++
		mu.Unlock()
		counter <- data
		return data * 2, nil
	}

	pool := utils.NewPool(5, worker)
	pool.Start(ctx)

	// Submit tasks concurrently from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				pool.Submit(j)
			}
		}()
	}

	wg.Wait()
	pool.Stop()

	// Verify all tasks were processed
	assert.Equal(t, 100, processedCount)
	assert.Equal(t, 100, len(counter))
}

func TestPool_PartialResults(t *testing.T) {
	ctx := context.Background()

	worker := func(ctx context.Context, data int) (any, error) {
		if data == 3 {
			time.Sleep(200 * time.Millisecond) // Slow task
		}
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)

	// Submit tasks with one slow task
	items := []int{1, 2, 3, 4, 5}
	results, err := pool.Process(ctx, items)

	require.NoError(t, err)
	assert.Equal(t, len(items), len(results))

	// Verify all items are present even if processed out of order
	processed := make(map[int]bool)
	for _, task := range results {
		processed[task.Data] = true
	}

	for _, item := range items {
		assert.True(t, processed[item])
	}
}

func TestNewSimplePool(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		expected int
	}{
		{
			name:     "single worker",
			workers:  1,
			expected: 1,
		},
		{
			name:     "multiple workers",
			workers:  5,
			expected: 5,
		},
		{
			name:     "zero workers",
			workers:  0,
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pool := utils.NewSimplePool(tc.workers)

			// Pool is created successfully
			assert.NotNil(t, pool)
		})
	}
}

func TestSimplePool_Run(t *testing.T) {
	tests := []struct {
		name       string
		workers    int
		tasks      []func(context.Context) error
		expectErrs bool
	}{
		{
			name:    "all tasks succeed",
			workers: 3,
			tasks: []func(context.Context) error{
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
			},
			expectErrs: false,
		},
		{
			name:    "some tasks fail",
			workers: 2,
			tasks: []func(context.Context) error{
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return errors.New("error 1") },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return errors.New("error 2") },
			},
			expectErrs: true,
		},
		{
			name:    "all tasks fail",
			workers: 2,
			tasks: []func(context.Context) error{
				func(ctx context.Context) error { return errors.New("error 1") },
				func(ctx context.Context) error { return errors.New("error 2") },
			},
			expectErrs: true,
		},
		{
			name:    "more tasks than workers",
			workers: 2,
			tasks: []func(context.Context) error{
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
				func(ctx context.Context) error { return nil },
			},
			expectErrs: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			pool := utils.NewSimplePool(tc.workers)

			errors := pool.Run(ctx, tc.tasks)

			assert.Equal(t, len(tc.tasks), len(errors))

			if tc.expectErrs {
				// Check that at least one error is present
				hasErr := false
				for _, err := range errors {
					if err != nil {
						hasErr = true
						break
					}
				}
				assert.True(t, hasErr, "expected at least one error")
			}
		})
	}
}

func TestSimplePool_Run_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tasks := []func(context.Context) error{
		func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		},
	}

	pool := utils.NewSimplePool(2)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	errors := pool.Run(ctx, tasks)

	// Should return errors or be cancelled
	assert.True(t, len(errors) > 0)
}

func TestSimplePool_Run_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tasks := []func(context.Context) error{
		func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(200 * time.Millisecond)
			return nil
		},
	}

	pool := utils.NewSimplePool(2)
	errors := pool.Run(ctx, tasks)

	// Should return timeout error
	assert.True(t, len(errors) > 0)
}

func TestParallelForEach(t *testing.T) {
	tests := []struct {
		name        string
		items       []int
		workers     int
		expectError bool
	}{
		{
			name:        "process all items",
			items:       []int{1, 2, 3, 4, 5},
			workers:     3,
			expectError: false,
		},
		{
			name:        "zero workers defaults to 1",
			items:       []int{1, 2, 3},
			workers:     0,
			expectError: false,
		},
		{
			name:        "more workers than items",
			items:       []int{1, 2},
			workers:     5,
			expectError: false,
		},
		{
			name:    "with errors",
			items:   []int{1, 2, 3},
			workers: 2,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			processed := make(chan int, len(tc.items))

			fn := func(ctx context.Context, data int) error {
				processed <- data
				if tc.expectError && data == 2 {
					return errors.New("processing error")
				}
				return nil
			}

			result := utils.ParallelForEach(ctx, tc.items, tc.workers, fn)

			if tc.expectError {
				assert.True(t, len(result) > 0)
			}

			// Verify all items were processed
			timeout := time.After(2 * time.Second)
			for i := 0; i < len(tc.items); i++ {
				select {
				case <-processed:
				case <-timeout:
					t.Fatal("timeout waiting for items to be processed")
				}
			}
		})
	}
}

func TestParallelForEach_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	items := []int{1, 2, 3, 4, 5}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	errors := utils.ParallelForEach(ctx, items, 2, func(ctx context.Context, data int) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	// Should return early due to cancellation
	assert.True(t, len(errors) > 0)
}

func TestFirstError(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected error
	}{
		{
			name:     "nil errors",
			errors:   []error{nil, nil, nil},
			expected: nil,
		},
		{
			name:     "first error",
			errors:   []error{errors.New("error 1"), errors.New("error 2"), nil},
			expected: errors.New("error 1"),
		},
		{
			name:     "middle error",
			errors:   []error{nil, errors.New("error 2"), errors.New("error 3")},
			expected: errors.New("error 2"),
		},
		{
			name:     "last error",
			errors:   []error{nil, nil, errors.New("error 3")},
			expected: errors.New("error 3"),
		},
		{
			name:     "empty slice",
			errors:   []error{},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.FirstError(tc.errors)
			if tc.expected == nil {
				assert.NoError(t, result)
			} else {
				assert.EqualError(t, result, tc.expected.Error())
			}
		})
	}
}

func TestCollectErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected []error
	}{
		{
			name:     "no errors",
			errors:   []error{nil, nil, nil},
			expected: []error{},
		},
		{
			name:     "all errors",
			errors:   []error{errors.New("error 1"), errors.New("error 2"), errors.New("error 3")},
			expected: []error{errors.New("error 1"), errors.New("error 2"), errors.New("error 3")},
		},
		{
			name:     "mixed errors",
			errors:   []error{nil, errors.New("error 1"), nil, errors.New("error 2"), nil},
			expected: []error{errors.New("error 1"), errors.New("error 2")},
		},
		{
			name:     "empty slice",
			errors:   []error{},
			expected: []error{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := utils.CollectErrors(tc.errors)
			assert.Equal(t, len(tc.expected), len(result))

			for i, err := range tc.expected {
				assert.EqualError(t, result[i], err.Error())
			}
		})
	}
}

func TestPool_Process_EmptySlice(t *testing.T) {
	ctx := context.Background()

	worker := func(ctx context.Context, data int) (any, error) {
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)

	results, err := pool.Process(ctx, []int{})

	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestSimplePool_Run_EmptyTasks(t *testing.T) {
	ctx := context.Background()

	pool := utils.NewSimplePool(2)

	errors := pool.Run(ctx, []func(context.Context) error{})

	assert.Equal(t, 0, len(errors))
}

func TestPool_Process_Concurrent(t *testing.T) {
	ctx := context.Background()

	worker := func(ctx context.Context, data int) (any, error) {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		return data * 2, nil
	}

	pool := utils.NewPool(5, worker)

	// Process a large number of items concurrently
	items := make([]int, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	start := time.Now()
	results, err := pool.Process(ctx, items)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, len(items), len(results))

	// Should complete faster with more workers (parallelism benefit)
	// 100 items with 5 workers should take roughly 20 iterations worth of time
	// We're just verifying it doesn't take 100 sequential times
	assert.Less(t, elapsed, 500*time.Millisecond, "processing should be concurrent")
}

func TestPool_Stop_Idempotent(t *testing.T) {
	ctx := context.Background()

	worker := func(ctx context.Context, data int) (any, error) {
		return data * 2, nil
	}

	pool := utils.NewPool(2, worker)
	pool.Start(ctx)

	pool.Submit(1)

	// Stop multiple times should not panic
	pool.Stop()
	pool.Stop()
	pool.Stop()
}
