package reactor_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fluxor-io/fluxor/pkg/reactor"
)

func TestReactor_SequentialExecution(t *testing.T) {
	r := reactor.NewReactor("test", 10)
	r.Start(context.Background(), nil)
	defer r.Stop(context.Background())

	var counter int32
	var wg sync.WaitGroup
	wg.Add(2)

	// Submit two events that will run concurrently if not for the reactor.
	r.Execute(func() {
		defer wg.Done()
		atomic.AddInt32(&counter, 1)
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&counter, 1)
	})
	r.Execute(func() {
		defer wg.Done()
		// If the reactor is working, the counter should be 2 when this function is called.
		if atomic.LoadInt32(&counter) != 2 {
			t.Errorf("Expected counter to be 2, but got %d", atomic.LoadInt32(&counter))
		}
	})

	wg.Wait()
}

func TestReactor_Stop(t *testing.T) {
	r := reactor.NewReactor("test", 10)
	r.Start(context.Background(), nil)

	var counter int32
	for i := 0; i < 5; i++ {
		r.Execute(func() {
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
	r.Execute(func() {
		atomic.AddInt32(&counter, 1)
	})

	// The counter should not have been incremented.
	if atomic.LoadInt32(&counter) != 5 {
			t.Errorf("Expected counter to remain 5, but got %d", atomic.LoadInt32(&counter))
	}
}
