package concurrency

import (
	"context"
)

// WorkerPool abstracts worker goroutine management
// Hides go func() calls and goroutine lifecycle from application code
type WorkerPool interface {
	// Start starts the worker pool
	// Initializes worker goroutines and begins processing tasks
	Start() error

	// Stop gracefully stops the worker pool
	// Waits for in-flight tasks to complete (up to ctx timeout)
	// Returns error if stop times out
	Stop(ctx context.Context) error

	// Submit submits a task to the worker pool
	// Returns error if pool is closed or queue is full
	Submit(task Task) error

	// Workers returns the number of worker goroutines
	Workers() int

	// IsRunning returns true if the worker pool is running
	IsRunning() bool
}
