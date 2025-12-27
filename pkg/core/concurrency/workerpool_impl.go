package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// defaultWorkerPool implements WorkerPool
// Hides goroutine creation and management from public API
type defaultWorkerPool struct {
	workers  int
	taskChan chan Task // Hidden: internal channel
	wg       sync.WaitGroup
	mu       sync.RWMutex
	running  int32 // Atomic flag
	ctx      context.Context
	cancel   context.CancelFunc
	logger   simpleLogger // Logger for error messages
}

// WorkerPoolConfig configures a WorkerPool
type WorkerPoolConfig struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Task queue size
}

// DefaultWorkerPoolConfig returns default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Workers:   10,
		QueueSize: 1000,
	}
}

// NewWorkerPool creates a new WorkerPool
// Hides goroutine and channel creation from callers
func NewWorkerPool(ctx context.Context, config WorkerPoolConfig) WorkerPool {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.QueueSize < 1 {
		config.QueueSize = 100
	}

	ctx, cancel := context.WithCancel(ctx)

	return &defaultWorkerPool{
		workers:  config.Workers,
		taskChan: make(chan Task, config.QueueSize), // Hidden: channel creation
		ctx:      ctx,
		cancel:   cancel,
		logger:   newDefaultSimpleLogger(),
	}
}

// Start implements WorkerPool interface
// Hides goroutine creation
func (wp *defaultWorkerPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if atomic.LoadInt32(&wp.running) == 1 {
		return fmt.Errorf("worker pool is already running")
	}

	atomic.StoreInt32(&wp.running, 1)
	wp.wg.Add(wp.workers)

	// Start worker goroutines (hidden: go func() calls)
	for i := 0; i < wp.workers; i++ {
		go wp.worker(i) // Hidden: goroutine creation
	}

	return nil
}

// worker processes tasks (hides channel operations)
func (wp *defaultWorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskChan: // Hidden: channel receive
			if !ok {
				return // Channel closed
			}

			// Execute task
			if err := task.Execute(wp.ctx); err != nil {
				// Don't log context.Canceled or ErrMailboxClosed - they're expected during shutdown
				if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrMailboxClosed) {
					wp.logger.Errorf("worker %d: task %s failed: %v", id, task.Name(), err)
				}
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// Stop implements WorkerPool interface
func (wp *defaultWorkerPool) Stop(ctx context.Context) error {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if atomic.LoadInt32(&wp.running) == 0 {
		return nil
	}

	atomic.StoreInt32(&wp.running, 0)
	wp.cancel()

	// Close task channel (hidden: channel close)
	close(wp.taskChan)

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop timeout: %w", ctx.Err())
	}
}

// Submit implements WorkerPool interface
// Hides channel send operations
func (wp *defaultWorkerPool) Submit(task Task) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if atomic.LoadInt32(&wp.running) == 0 {
		return fmt.Errorf("worker pool is not running")
	}

	// Try to send (non-blocking for backpressure)
	select {
	case wp.taskChan <- task: // Hidden: channel send
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrMailboxFull
	}
}

// Workers implements WorkerPool interface
func (wp *defaultWorkerPool) Workers() int {
	return wp.workers
}

// IsRunning implements WorkerPool interface
func (wp *defaultWorkerPool) IsRunning() bool {
	return atomic.LoadInt32(&wp.running) == 1
}
