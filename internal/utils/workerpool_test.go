package utils

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPool(t *testing.T) {
	t.Parallel()

	worker := func(ctx context.Context, data int) (any, error) {
		return data * 2, nil
	}

	pool := NewPool(5, worker)
	require.NotNil(t, pool)
	assert.Equal(t, 5, pool.workers)
}

func TestPoolProcess(t *testing.T) {
	t.Parallel()

	t.Run("process items successfully", func(t *testing.T) {
		worker := func(ctx context.Context, data int) (any, error) {
			return data * 2, nil
		}

		pool := NewPool(3, worker)
		items := []int{1, 2, 3, 4, 5}

		ctx := context.Background()
		results, err := pool.Process(ctx, items)

		require.NoError(t, err)
		assert.Len(t, results, 5)

		// Check results
		for _, task := range results {
			assert.NoError(t, task.Err)
			expected := task.Data * 2
			assert.Equal(t, expected, task.Result)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		worker := func(ctx context.Context, data int) (any, error) {
			return data * 2, nil
		}

		pool := NewPool(3, worker)
		ctx := context.Background()
		results, err := pool.Process(ctx, []int{})

		require.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("worker returns error", func(t *testing.T) {
		worker := func(ctx context.Context, data int) (any, error) {
			if data == 2 {
				return nil, errors.New("error processing 2")
			}
			return data * 2, nil
		}

		pool := NewPool(3, worker)
		items := []int{1, 2, 3}

		ctx := context.Background()
		results, err := pool.Process(ctx, items)

		require.NoError(t, err)
		assert.Len(t, results, 3)

		// Find the error task
		for _, task := range results {
			if task.Data == 2 {
				assert.Error(t, task.Err)
			} else {
				assert.NoError(t, task.Err)
			}
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		worker := func(ctx context.Context, data int) (any, error) {
			time.Sleep(100 * time.Millisecond)
			return data * 2, nil
		}

		pool := NewPool(2, worker)
		items := []int{1, 2, 3, 4, 5}

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		results, err := pool.Process(ctx, items)

		// Should return context error
		assert.Error(t, err)
		// Results may be partial
		assert.LessOrEqual(t, len(results), 5)
	})
}

func TestPoolStartStop(t *testing.T) {
	t.Parallel()

	t.Run("manual start and stop", func(t *testing.T) {
		worker := func(ctx context.Context, data int) (any, error) {
			return data * 2, nil
		}

		pool := NewPool(2, worker)
		ctx := context.Background()

		pool.Start(ctx)

		// Submit tasks manually
		pool.Submit(1)
		pool.Submit(2)
		pool.Submit(3)

		pool.Stop()

		// Collect results
		results := make([]*Task[int], 0)
		for task := range pool.Results() {
			results = append(results, task)
		}

		assert.Len(t, results, 3)
	})
}

func TestSimplePool(t *testing.T) {
	t.Parallel()

	t.Run("run tasks successfully", func(t *testing.T) {
		pool := NewSimplePool(3)
		ctx := context.Background()

		tasks := make([]func(context.Context) error, 5)
		results := make([]int, 5)
		var mu sync.Mutex

		for i := 0; i < 5; i++ {
			idx := i
			tasks[i] = func(ctx context.Context) error {
				mu.Lock()
				results[idx] = idx * 2
				mu.Unlock()
				return nil
			}
		}

		errors := pool.Run(ctx, tasks)

		assert.Len(t, errors, 5)
		for _, err := range errors {
			assert.NoError(t, err)
		}

		// Check results
		for i, val := range results {
			assert.Equal(t, i*2, val)
		}
	})

	t.Run("tasks with errors", func(t *testing.T) {
		pool := NewSimplePool(2)
		ctx := context.Background()

		tasks := []func(context.Context) error{
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return errors.New("task 1 failed") },
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return errors.New("task 3 failed") },
		}

		errors := pool.Run(ctx, tasks)

		assert.Len(t, errors, 4)
		assert.NoError(t, errors[0])
		assert.Error(t, errors[1])
		assert.NoError(t, errors[2])
		assert.Error(t, errors[3])
	})

	t.Run("context cancellation", func(t *testing.T) {
		pool := NewSimplePool(2)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		tasks := make([]func(context.Context) error, 5)
		for i := 0; i < 5; i++ {
			tasks[i] = func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}
		}

		errors := pool.Run(ctx, tasks)

		// Should have some errors due to cancellation
		assert.Len(t, errors, 5)
	})
}

func TestParallelForEach(t *testing.T) {
	t.Parallel()

	t.Run("process all items", func(t *testing.T) {
		ctx := context.Background()
		items := []int{1, 2, 3, 4, 5}
		results := make([]int, 5)
		var mu sync.Mutex

		errors := ParallelForEach(ctx, items, 3, func(ctx context.Context, item int) error {
			mu.Lock()
			results[item-1] = item * 2
			mu.Unlock()
			return nil
		})

		assert.Len(t, errors, 5)
		for _, err := range errors {
			assert.NoError(t, err)
		}

		for i, val := range results {
			assert.Equal(t, (i+1)*2, val)
		}
	})

	t.Run("with errors", func(t *testing.T) {
		ctx := context.Background()
		items := []int{1, 2, 3}

		errors := ParallelForEach(ctx, items, 2, func(ctx context.Context, item int) error {
			if item == 2 {
				return errors.New("error on 2")
			}
			return nil
		})

		assert.Len(t, errors, 3)
		assert.NoError(t, errors[0])
		assert.Error(t, errors[1])
		assert.NoError(t, errors[2])
	})

	t.Run("workers count adjustment", func(t *testing.T) {
		ctx := context.Background()
		items := []int{1, 2, 3}
		results := make([]int, 3)
		var mu sync.Mutex

		// More workers than items
		errors := ParallelForEach(ctx, items, 10, func(ctx context.Context, item int) error {
			mu.Lock()
			results[item-1] = item
			mu.Unlock()
			return nil
		})

		assert.Len(t, errors, 3)
		assert.NoError(t, errors[0])
		assert.NoError(t, errors[1])
		assert.NoError(t, errors[2])
	})

	t.Run("zero workers defaults to 1", func(t *testing.T) {
		ctx := context.Background()
		items := []int{1, 2}
		results := make([]int, 2)
		var mu sync.Mutex

		errors := ParallelForEach(ctx, items, 0, func(ctx context.Context, item int) error {
			mu.Lock()
			results[item-1] = item
			mu.Unlock()
			return nil
		})

		assert.Len(t, errors, 2)
		assert.NoError(t, errors[0])
		assert.NoError(t, errors[1])
	})
}

func TestFirstError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errors   []error
		expected error
	}{
		{
			name:     "no errors",
			errors:   []error{nil, nil, nil},
			expected: nil,
		},
		{
			name:     "first error",
			errors:   []error{nil, errors.New("error"), nil},
			expected: errors.New("error"),
		},
		{
			name:     "all errors",
			errors:   []error{errors.New("error1"), errors.New("error2")},
			expected: errors.New("error1"),
		},
		{
			name:     "empty slice",
			errors:   []error{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FirstError(tt.errors)
			if tt.expected == nil {
				assert.NoError(t, result)
			} else {
				assert.Error(t, result)
			}
		})
	}
}

func TestCollectErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errors   []error
		expected int
	}{
		{
			name:     "no errors",
			errors:   []error{nil, nil, nil},
			expected: 0,
		},
		{
			name:     "some errors",
			errors:   []error{nil, errors.New("error1"), nil, errors.New("error2")},
			expected: 2,
		},
		{
			name:     "all errors",
			errors:   []error{errors.New("e1"), errors.New("e2"), errors.New("e3")},
			expected: 3,
		},
		{
			name:     "empty slice",
			errors:   []error{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CollectErrors(tt.errors)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestPoolResults(t *testing.T) {
	t.Parallel()

	worker := func(ctx context.Context, data int) (any, error) {
		return data * 2, nil
	}

	pool := NewPool(2, worker)
	ctx := context.Background()

	pool.Start(ctx)

	// Submit tasks
	pool.Submit(1)
	pool.Submit(2)

	pool.Stop()

	// Verify results channel is closed
	count := 0
	for range pool.Results() {
		count++
	}

	assert.Equal(t, 2, count)
}
