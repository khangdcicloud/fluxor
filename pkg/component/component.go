package component

import (
	"context"

	"github.com/example/goflux/pkg/bus"
)

// Component is the interface for all components in the system.
type Component interface {
	// Start is called when the component is started. It is passed a context and an event bus proxy.
	Start(ctx context.Context, bus bus.Bus) error
	// Stop is called when the component is stopped. It is passed a context.
	Stop(ctx context.Context) error
}
