package worker

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrWorkerPoolClosed = errors.New("worker pool is closed")
	ErrNoWorkers        = errors.New("worker pool has no workers")
)

// Job represents a task to be executed by a worker.
type Job func()

// WorkerPool is a fixed-size pool of goroutines for executing blocking or CPU-heavy tasks.
type WorkerPool struct {
	jobs    chan Job
	stop    chan struct{}
	workers int
	wg      sync.WaitGroup
}

// NewWorkerPool creates a new WorkerPool with a given number of workers and job queue size.
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	if workers <= 0 {
		panic("number of workers must be positive")
	}
	return &WorkerPool{
		jobs:    make(chan Job, queueSize),
		stop:    make(chan struct{}),
		workers: workers,
	}
}

// Start initializes the workers in the pool.
func (p *WorkerPool) Start() {
	p.wg.Add(p.workers)
	for i := 0; i < p.workers; i++ {
		go p.run()
	}
}

// Stop gracefully stops the worker pool, waiting for all active jobs to complete.
func (p *WorkerPool) Stop(ctx context.Context) {
	close(p.jobs) // Stop accepting new jobs
	close(p.stop)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// all workers have stopped
	case <-ctx.Done():
		// timeout waiting for workers to stop
	}
}

// run is the worker's execution loop.
func (p *WorkerPool) run() {
	defer p.wg.Done()
	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				return // jobs channel closed
			}
			job()
		case <-p.stop:
			return
		}
	}
}

// Submit sends a job to the worker pool for execution.
// It returns ErrWorkerPoolClosed if the pool is closed.
func (p *WorkerPool) Submit(job Job) error {
	select {
	case p.jobs <- job:
		return nil
	case <-p.stop:
		return ErrWorkerPoolClosed
	}
}
