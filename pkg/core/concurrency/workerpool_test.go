package concurrency

import (
	"context"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	ctx := context.Background()
	config := DefaultWorkerPoolConfig()

	pool := NewWorkerPool(ctx, config)

	if pool == nil {
		t.Error("NewWorkerPool() should not return nil")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   2,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)

	// Test start
	err := pool.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !pool.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	// Test double start
	err = pool.Start()
	if err == nil {
		t.Error("Start() when already running should fail")
	}

	// Test stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = pool.Stop(stopCtx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if pool.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   2,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	// Test nil task
	err := pool.Submit(nil)
	if err == nil {
		t.Error("Submit() with nil task should fail")
	}

	// Test submit when not running
	pool2 := NewWorkerPool(ctx, config)
	err = pool2.Submit(NewNamedTask("test", func(ctx context.Context) error {
		return nil
	}))
	if err == nil {
		t.Error("Submit() when not running should fail")
	}

	// Test valid submit
	task := NewNamedTask("test-task", func(ctx context.Context) error {
		return nil
	})

	err = pool.Submit(task)
	if err != nil {
		t.Errorf("Submit() error = %v", err)
	}

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)
}

func TestWorkerPool_Workers(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   5,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)

	if pool.Workers() != 5 {
		t.Errorf("Workers() = %d, want 5", pool.Workers())
	}
}
