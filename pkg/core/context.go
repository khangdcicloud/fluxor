package core

import (
	"context"
)

// FluxorContext represents the execution context for a verticle or handler.
//
// This is distinct from context.Context (Go's standard context):
//   - context.Context: Go's cancellation/deadline/value propagation
//   - FluxorContext: Fluxor's runtime context with access to Vertx, EventBus, Config
//
// FluxorContext wraps a context.Context and provides additional Fluxor-specific
// functionality. Use Context() to get the underlying context.Context when needed
// for cancellation or passing to Go standard library functions.
type FluxorContext interface {
	// Context returns the underlying context.Context (Go's standard context)
	Context() context.Context

	// EventBus returns the event bus instance
	EventBus() EventBus

	// Vertx returns the Vertx instance
	Vertx() Vertx

	// Config returns the configuration map
	Config() map[string]interface{}

	// SetConfig sets a configuration value
	SetConfig(key string, value interface{})

	// Deploy deploys a verticle
	Deploy(verticle Verticle) (string, error)

	// Undeploy undeploys a verticle by deployment ID
	Undeploy(deploymentID string) error
}

// vertxContext implements FluxorContext
type vertxContext struct {
	goCtx  context.Context // renamed from 'ctx' for clarity: this is Go's context.Context
	vertx  Vertx
	config map[string]interface{}
}

// newFluxorContext creates a new FluxorContext wrapping the given context.Context.
// This is an internal function - renamed from 'newContext' for clarity.
//
// Parameters:
//   - goCtx: the Go context.Context to wrap (typically from Vertx.rootCtx)
//   - vertx: the Vertx instance for accessing EventBus and deploying verticles
func newFluxorContext(goCtx context.Context, vertx Vertx) FluxorContext {
	if goCtx == nil {
		// Fail-fast: context cannot be nil
		panic("context cannot be nil")
	}
	return &vertxContext{
		goCtx:  goCtx,
		vertx:  vertx,
		config: make(map[string]interface{}),
	}
}

// Context returns the underlying context.Context (Go's standard context)
func (c *vertxContext) Context() context.Context {
	return c.goCtx
}

func (c *vertxContext) EventBus() EventBus {
	if c.vertx == nil {
		// Fail-fast: vertx is nil
		panic("vertx is nil, cannot get EventBus")
	}
	return c.vertx.EventBus()
}

func (c *vertxContext) Vertx() Vertx {
	return c.vertx
}

func (c *vertxContext) Config() map[string]interface{} {
	return c.config
}

func (c *vertxContext) SetConfig(key string, value interface{}) {
	c.config[key] = value
}

func (c *vertxContext) Deploy(verticle Verticle) (string, error) {
	return c.vertx.DeployVerticle(verticle)
}

func (c *vertxContext) Undeploy(deploymentID string) error {
	return c.vertx.UndeployVerticle(deploymentID)
}
