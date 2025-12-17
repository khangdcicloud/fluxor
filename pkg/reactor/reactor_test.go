package reactor_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/goflux/pkg/reactor"
)

func TestReactor_SequentialExecution(t *testing.T) {
	r := reactor.NewReactor(10)
	r.Start()
	defer r.Stop(context.Background())

	var counter int32
	var wg sync.WaitGroup
	wg.Add(2)

	// Submit two events that will run concurrently if not for the reactor.
	r.Submit(func() {
		defer wg.Done()
		atomic.AddInt32(&counter, 1)
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&counter, 1)
	})
	r.Submit(func() {
		defer wg.Done()
		// If the reactor is working, the counter should be 2 when this function is called.
		if atomic.LoadInt32(&counter) != 2 {
			t.Errorf("Expected counter to be 2, but got %d", atomic.LoadInt32(&counter))
		}
	})

	wg.Wait()
}

func TestReactor_Backpressure(t *testing.T) {
	r := reactor.NewReactor(1)
	r.Start()
	defer r.Stop(context.Background())

	// Block the reactor.
	var wg sync.WaitGroup
	wg.Add(1)
	r.Submit(func() {
		wg.Wait()
	})

	// The mailbox is now full. The next submit should fail.
	if err := r.Submit(func() {}); err != reactor.ErrBackpressure {
		t.Errorf("Expected ErrBackpressure, but got %v", err)
	}

	wg.Done()
}

func TestReactor_Stop(t *testing.T) {
	r := reactor.NewReactor(10)
	r.Start()

	var counter int32
	for i := 0; i < 5; i++ {
		r.Submit(func() {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond)
		})
	}

	// Stop the reactor and wait for all events to be processed.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r.Stop(ctx)

	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("Expected counter to be 5, but got %d", atomic.LoadInt32(&counter))
	}

	// Submitting to a stopped reactor should still work, but the event will not be processed.
	if err := r.Submit(func() {
		atomic.AddInt32(&counter, 1)
	}); err != nil {
		t.Errorf("Submit to a stopped reactor should not return an error, but got %v", err)
	}

	// The counter should not have been incremented.
	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("Expected counter to remain 5, but got %d", atomic.LoadInt32(&counter))
	}
}
