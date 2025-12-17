package component

import (
	"context"

	"github.com/fluxor-io/fluxor/pkg/bus"
)

// Component is the basic building block of a fluxor application.
// Components are started and stopped by the runtime.
type Component interface {
	// Start is called by the runtime to start the component.
	Start(ctx context.Context, bus bus.Bus) error
	// Stop is called by the runtime to stop the component.
	Stop(ctx context.Context) error
}
