package runtime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/fluxor-io/fluxor/pkg/types"
)

var ErrRuntimeAlreadyStarted = errors.New("runtime has already been started")
var ErrRuntimeNotStarted = errors.New("runtime is not started")

const (
	runtimeStateIdle uint32 = iota
	runtimeStateStarting
	runtimeStateStarted
	runtimeStateStopping
	runtimeStateStopped
)

type NewRuntimeOptions struct {
	NumReactors int
	MailboxSize int
	NumWorkers  int
	QueueSize   int
}

type Runtime struct {
	bus   types.Bus
	state uint32
	comps map[string]types.Component
	mu    sync.RWMutex
}

func NewRuntime(opts NewRuntimeOptions, bus types.Bus) *Runtime {
	return &Runtime{
		bus:   bus,
		comps: make(map[string]types.Component),
	}
}

func (r *Runtime) Deploy(ctx context.Context, comp types.Component) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := comp.Name()
	r.comps[name] = comp
	return comp.OnStart(ctx, r.bus)
}

func (r *Runtime) Start() error {
	if !atomic.CompareAndSwapUint32(&r.state, runtimeStateIdle, runtimeStateStarting) {
		return ErrRuntimeAlreadyStarted
	}
	atomic.StoreUint32(&r.state, runtimeStateStarted)
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.state, runtimeStateStarted, runtimeStateStopping) {
		return ErrRuntimeNotStarted
	}

	r.mu.RLock()
	var wg sync.WaitGroup
	wg.Add(len(r.comps))
	for _, comp := range r.comps {
		go func(comp types.Component) {
			defer wg.Done()
			comp.OnStop(ctx) // Errors are logged by the component
		}(comp)
	}
	wg.Wait()
	r.mu.RUnlock()

	atomic.StoreUint32(&r.state, runtimeStateStopped)
	return nil
}

func (r *Runtime) Status() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	components := make([]string, 0, len(r.comps))
	for name := range r.comps {
		components = append(components, name)
	}

	return map[string]interface{}{
		"state":      r.state,
		"components": components,
	}
}
