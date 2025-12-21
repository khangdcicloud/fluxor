package concurrency

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// defaultExecutor implements Executor using channels and goroutines internally
// Hides all Go concurrency primitives from public API
type defaultExecutor struct {
	taskChan      chan Task // Hidden: internal channel
	workers       int
	queueSize     int
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	closed        bool
	logger        simpleLogger // Logger for error messages
	
	// Metrics (atomic for thread-safety)
	queuedTasks    int64
	completedTasks int64
	rejectedTasks  int64
}

// ExecutorConfig configures an Executor
type ExecutorConfig struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Maximum queue size (bounded for backpressure)
}

// DefaultExecutorConfig returns default executor configuration
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Workers:   10,
		QueueSize: 1000,
	}
}

// NewExecutor creates a new Executor with the given configuration
// Hides goroutine and channel creation from callers
func NewExecutor(ctx context.Context, config ExecutorConfig) Executor {
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.QueueSize < 1 {
		config.QueueSize = 100
	}

	ctx, cancel := context.WithCancel(ctx)
	
	exec := &defaultExecutor{
		taskChan:  make(chan Task, config.QueueSize), // Hidden channel
		workers:   config.Workers,
		queueSize: config.QueueSize,
		ctx:       ctx,
		cancel:    cancel,
		logger:    newDefaultSimpleLogger(),
	}

	// Start worker goroutines (hidden from public API)
	exec.startWorkers()

	return exec
}

// startWorkers starts worker goroutines (hides go func() calls)
func (e *defaultExecutor) startWorkers() {
	e.wg.Add(e.workers)
	for i := 0; i < e.workers; i++ {
		go e.worker(i) // Hidden: goroutine creation
	}
}

// worker processes tasks from the queue (hides channel operations)
func (e *defaultExecutor) worker(id int) {
	defer e.wg.Done()

	for {
		select {
		case task, ok := <-e.taskChan: // Hidden: channel receive
			if !ok {
				return // Channel closed
			}
			atomic.AddInt64(&e.queuedTasks, -1)
			
			// Execute task
			if err := task.Execute(e.ctx); err != nil {
				// Log error but continue processing
				e.logger.Errorf("task %s failed: %v", task.Name(), err)
			}
			atomic.AddInt64(&e.completedTasks, 1)

		case <-e.ctx.Done():
			return
		}
	}
}

// Submit implements Executor interface
// Hides channel send operations and select statements
func (e *defaultExecutor) Submit(task Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	e.mu.RLock()
	closed := e.closed
	e.mu.RUnlock()

	if closed {
		return fmt.Errorf("executor is closed")
	}

	// Try to send to channel (non-blocking for backpressure)
	select {
	case e.taskChan <- task: // Hidden: channel send
		atomic.AddInt64(&e.queuedTasks, 1)
		return nil
	case <-e.ctx.Done():
		return e.ctx.Err()
	default:
		// Queue full - backpressure
		atomic.AddInt64(&e.rejectedTasks, 1)
		return ErrMailboxFull
	}
}

// SubmitWithTimeout implements Executor interface
func (e *defaultExecutor) SubmitWithTimeout(task Task, timeout time.Duration) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	e.mu.RLock()
	closed := e.closed
	e.mu.RUnlock()

	if closed {
		return fmt.Errorf("executor is closed")
	}

	// Try to send with timeout
	select {
	case e.taskChan <- task: // Hidden: channel send
		atomic.AddInt64(&e.queuedTasks, 1)
		return nil
	case <-time.After(timeout):
		atomic.AddInt64(&e.rejectedTasks, 1)
		return fmt.Errorf("submit timeout after %v", timeout)
	case <-e.ctx.Done():
		return e.ctx.Err()
	}
}

// Shutdown implements Executor interface
func (e *defaultExecutor) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	e.mu.Unlock()

	// Cancel context to stop workers
	e.cancel()

	// Close task channel (hidden: channel close)
	close(e.taskChan)

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// Stats implements Executor interface
func (e *defaultExecutor) Stats() ExecutorStats {
	queued := atomic.LoadInt64(&e.queuedTasks)
	queueUtilization := float64(queued) / float64(e.queueSize) * 100.0
	if queueUtilization > 100.0 {
		queueUtilization = 100.0
	}

	return ExecutorStats{
		QueuedTasks:      queued,
		ActiveWorkers:    e.workers,
		CompletedTasks:   atomic.LoadInt64(&e.completedTasks),
		RejectedTasks:    atomic.LoadInt64(&e.rejectedTasks),
		QueueCapacity:    e.queueSize,
		QueueUtilization: queueUtilization,
	}
}

