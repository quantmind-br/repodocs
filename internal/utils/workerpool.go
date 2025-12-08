package utils

import (
	"context"
	"sync"
)

// Task represents a unit of work
type Task[T any] struct {
	Data   T
	Result any
	Err    error
}

// Worker is a function that processes a task
type Worker[T any] func(ctx context.Context, data T) (any, error)

// Pool is a worker pool for concurrent task processing
type Pool[T any] struct {
	workers    int
	taskQueue  chan *Task[T]
	resultChan chan *Task[T]
	wg         sync.WaitGroup
	worker     Worker[T]
	stopOnce   sync.Once
}

// NewPool creates a new worker pool
func NewPool[T any](workers int, worker Worker[T]) *Pool[T] {
	return &Pool[T]{
		workers:    workers,
		taskQueue:  make(chan *Task[T], workers*2),
		resultChan: make(chan *Task[T], workers*2),
		worker:     worker,
	}
}

// Start starts the worker pool
func (p *Pool[T]) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.runWorker(ctx)
	}
}

// runWorker runs a single worker
func (p *Pool[T]) runWorker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			result, err := p.worker(ctx, task.Data)
			task.Result = result
			task.Err = err

			select {
			case p.resultChan <- task:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Submit submits a task to the pool
func (p *Pool[T]) Submit(data T) {
	p.taskQueue <- &Task[T]{Data: data}
}

// Results returns the results channel
func (p *Pool[T]) Results() <-chan *Task[T] {
	return p.resultChan
}

// Stop stops the pool and waits for workers to finish
func (p *Pool[T]) Stop() {
	p.stopOnce.Do(func() {
		close(p.taskQueue)
		p.wg.Wait()
		close(p.resultChan)
	})
}

// Process processes a slice of data items concurrently
func (p *Pool[T]) Process(ctx context.Context, items []T) ([]*Task[T], error) {
	// Handle empty slice case
	if len(items) == 0 {
		return []*Task[T]{}, nil
	}

	p.Start(ctx)

	// Submit all items
	go func() {
		for _, item := range items {
			select {
			case <-ctx.Done():
				return
			default:
				p.Submit(item)
			}
		}
		close(p.taskQueue)
	}()

	// Collect results with context awareness
	results := make([]*Task[T], 0, len(items))
	collectDone := false
	for !collectDone {
		select {
		case <-ctx.Done():
			collectDone = true
		case task, ok := <-p.resultChan:
			if !ok {
				collectDone = true
			} else {
				results = append(results, task)
				if len(results) == len(items) {
					collectDone = true
				}
			}
		}
	}

	p.wg.Wait()

	// Drain remaining results to avoid goroutine leak
	go func() {
		for range p.resultChan {
		}
	}()
	close(p.resultChan)

	// Check for context error
	if ctx.Err() != nil {
		return results, ctx.Err()
	}

	return results, nil
}

// SimplePool is a simpler worker pool without generics for basic use cases
type SimplePool struct {
	workers int
	wg      sync.WaitGroup
}

// NewSimplePool creates a new simple worker pool
func NewSimplePool(workers int) *SimplePool {
	return &SimplePool{workers: workers}
}

// Run runs tasks concurrently with the given function
func (p *SimplePool) Run(ctx context.Context, tasks []func(context.Context) error) []error {
	errors := make([]error, len(tasks))
	taskChan := make(chan int, len(tasks))
	var mu sync.Mutex

	// Start workers
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-taskChan:
					if !ok {
						return
					}
					err := tasks[idx](ctx)
					mu.Lock()
					errors[idx] = err
					mu.Unlock()
				}
			}
		}()
	}

	// Submit tasks
	for i := range tasks {
		select {
		case <-ctx.Done():
			close(taskChan)
			p.wg.Wait()
			return errors
		case taskChan <- i:
		}
	}

	close(taskChan)
	p.wg.Wait()

	return errors
}

// ParallelForEach executes a function for each item in parallel
func ParallelForEach[T any](ctx context.Context, items []T, workers int, fn func(context.Context, T) error) []error {
	if workers <= 0 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	errors := make([]error, len(items))
	taskChan := make(chan int, len(items))
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-taskChan:
					if !ok {
						return
					}
					err := fn(ctx, items[idx])
					mu.Lock()
					errors[idx] = err
					mu.Unlock()
				}
			}
		}()
	}

	// Submit tasks
	for i := range items {
		select {
		case <-ctx.Done():
			close(taskChan)
			wg.Wait()
			return errors
		case taskChan <- i:
		}
	}

	close(taskChan)
	wg.Wait()

	return errors
}

// FirstError returns the first non-nil error from a slice of errors
func FirstError(errors []error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// CollectErrors collects all non-nil errors from a slice
func CollectErrors(errors []error) []error {
	var result []error
	for _, err := range errors {
		if err != nil {
			result = append(result, err)
		}
	}
	return result
}
