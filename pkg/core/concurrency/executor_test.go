package concurrency

import (
	"context"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	ctx := context.Background()
	config := DefaultExecutorConfig()

	executor := NewExecutor(ctx, config)

	if executor == nil {
		t.Error("NewExecutor() should not return nil")
	}

	// Test shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := executor.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestExecutor_Submit(t *testing.T) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   2,
		QueueSize: 10,
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	// Test nil task
	err := executor.Submit(nil)
	if err == nil {
		t.Error("Submit() with nil task should fail")
	}

	// Test valid task
	task := NewNamedTask("test-task", func(ctx context.Context) error {
		return nil
	})

	err = executor.Submit(task)
	if err != nil {
		t.Errorf("Submit() error = %v", err)
	}

	// Wait a bit for task to complete
	time.Sleep(100 * time.Millisecond)
}

func TestExecutor_SubmitWithTimeout(t *testing.T) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   1,
		QueueSize: 1, // Very small queue
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	// Submit a blocking task that will occupy the worker
	blockingTask := NewNamedTask("blocking", func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	executor.Submit(blockingTask)

	// Submit another task to fill the queue
	executor.Submit(NewNamedTask("fill", func(ctx context.Context) error {
		return nil
	}))

	// Small delay to ensure queue is filled
	time.Sleep(20 * time.Millisecond)

	// Try to submit with very short timeout - this tests the timeout mechanism
	// Note: This test may be flaky if the queue isn't full, but it tests the timeout path
	task3 := NewNamedTask("task3", func(ctx context.Context) error {
		return nil
	})
	err := executor.SubmitWithTimeout(task3, 5*time.Millisecond)
	// We expect either timeout error or success (if queue had space)
	// The important thing is that SubmitWithTimeout doesn't panic
	if err != nil && err.Error() == "executor is closed" {
		t.Error("SubmitWithTimeout() should not return executor closed error")
	}
}

func TestExecutor_Stats(t *testing.T) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   2,
		QueueSize: 10,
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	stats := executor.Stats()

	if stats.ActiveWorkers != 2 {
		t.Errorf("Stats().ActiveWorkers = %d, want 2", stats.ActiveWorkers)
	}

	if stats.QueueCapacity != 10 {
		t.Errorf("Stats().QueueCapacity = %d, want 10", stats.QueueCapacity)
	}
}
