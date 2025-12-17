package component

import (
	"context"

	"github.com/gemini-testing/go-react-vertx/pkg/bus"
)

// Component is the base interface for all application components. It is similar to a Vert.x Verticle.
type Component interface {
	// Start is called when the component is deployed. This is where the component
	// should register its event bus consumers and perform any other initialization.
	Start(ctx context.Context, bus bus.Bus) error

	// Stop is called when the component is undeployed. This is where the component
	// should perform any cleanup, such as closing connections.
	Stop(ctx context.Context) error
}
