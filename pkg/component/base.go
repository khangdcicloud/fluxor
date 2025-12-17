package component

import (
	"context"

	"github.com/example/goflux/pkg/bus"
)

// Base is a default implementation of the Component interface.
// It can be embedded in other components to provide default behavior.
type Base struct{}

// Start is a no-op for the base component.
func (b *Base) Start(ctx context.Context, bus bus.Bus) error {
	return nil
}

// Stop is a no-op for the base component.
func (b *Base) Stop(ctx context.Context) error {
	return nil
}
