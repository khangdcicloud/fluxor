package component

import (
	"context"

	"github.com/example/goflux/pkg/bus"
)

// Component is the basic building block of a goflux application.
// Components are started by the runtime and interact with each other through the event bus.
type Component interface {
	// Start is called by the runtime to start the component.
	// The component should register its consumers and start any background goroutines.
	Start(ctx context.Context, bus bus.Bus) error
	// Stop is called by the runtime to stop the component.
	// The component should stop any background goroutines and release any resources.
	Stop(ctx context.Context) error

	// OnStart is called when the component is started.
	// It is a good place to register consumers.
	OnStart(ctx context.Context, bus bus.Bus)
	// OnStop is called when the component is stopped.
	// It is a good place to release resources.
	OnStop(ctx context.Context)
}
