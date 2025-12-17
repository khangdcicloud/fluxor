package component

import (
	"context"

	"github.com/gemini-testing/go-react-vertx/pkg/bus"
)

// Base is a default implementation of the Component interface. Components can embed this to avoid having to implement all methods.
type Base struct{}

// Start is a no-op implementation.
func (b *Base) Start(ctx context.Context, bus bus.Bus) error {
	return nil
}

// Stop is a no-op implementation.
func (b *Base) Stop(ctx context.Context) error {
	return nil
}
