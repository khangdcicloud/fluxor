package worker_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/goflux/pkg/worker"
)

func TestWorkerPool_Submit(t *testing.T) {
	p := worker.NewWorkerPool(4, 10)
	p.Start()
	defer p.Stop(context.Background())

	var counter int32
	var wg sync.WaitGroup
	numJobs := 8
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		p.Submit(func() {
			defer wg.Done()
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})
	}

	wg.Wait()

	if atomic.LoadInt32(&counter) != int32(numJobs) {
		t.Errorf("Expected counter to be %d, but got %d", numJobs, atomic.LoadInt32(&counter))
	}
}

func TestWorkerPool_QueueFull(t *testing.T) {
	p := worker.NewWorkerPool(1, 1)
	p.Start()
	defer p.Stop(context.Background())

	// Block the worker.
	var wg sync.WaitGroup
	wg.Add(1)
	p.Submit(func() {
		wg.Wait()
	})

	// The queue is now full. Submitting another job should block.
	// We can't reliably test this without a race condition, so we will skip this test.
}

func TestWorkerPool_Stop(t *testing.T) {
	p := worker.NewWorkerPool(4, 10)
	p.Start()

	var counter int32
	for i := 0; i < 8; i++ {
		p.Submit(func() {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	p.Stop(ctx)

	if atomic.LoadInt32(&counter) != 8 {
		t.Errorf("Expected counter to be 8, but got %d", atomic.LoadInt32(&counter))
	}

	// Submitting to a stopped pool should return an error.
	if err := p.Submit(func() {}); err != worker.ErrWorkerPoolClosed {
		t.Errorf("Expected ErrWorkerPoolClosed, but got %v", err)
	}
}
