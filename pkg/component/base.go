package component

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/example/goflux/pkg/bus"
)

// Base is a helper struct that provides a robust implementation of the Component interface.
// It is intended to be embedded in other components to provide common lifecycle management
// and goroutine supervision.
type Base struct {
	wg      sync.WaitGroup
	running uint32 // atomic
}

// Start is called by the runtime to start the component.
// This implementation calls the OnStart hook.
// It is not intended to be overridden by embedding components.
func (b *Base) Start(ctx context.Context, bus bus.Bus) error {
	if !atomic.CompareAndSwapUint32(&b.running, 0, 1) {
		// Already running
		return nil
	}
	b.OnStart(ctx, bus)
	return nil
}

// Stop is called by the runtime to stop the component.
// This implementation calls the OnStop hook and then waits for all supervised
// goroutines to complete.
// It is not intended to be overridden by embedding components.
func (b *Base) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&b.running, 1, 0) {
		// Already stopped
		return nil
	}
	b.OnStop(ctx)
	b.wg.Wait()
	return nil
}

// OnStart is a lifecycle hook that is called when the component is started.
// Components that embed Base can implement this method to perform their own startup logic.
func (b *Base) OnStart(ctx context.Context, bus bus.Bus) {
	// By default, do nothing.
}

// OnStop is a lifecycle hook that is called when the component is stopped.
// Components that embed Base can implement this method to perform their own shutdown logic.
func (b *Base) OnStop(ctx context.Context) {
	// By default, do nothing.
}

// Go starts a new goroutine that is supervised by the component.
// The component's Stop method will wait for all supervised goroutines to complete.
func (b *Base) Go(f func()) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		f()
	}()
}
