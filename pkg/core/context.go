package core

import (
	"context"
)

// FluxorContext represents the execution context for a verticle or handler.
//
// This is distinct from context.Context (Go's standard context):
//   - context.Context: Go's cancellation/deadline/value propagation
//   - FluxorContext: Fluxor's runtime context with access to GoCMD, EventBus, Config
//
// FluxorContext wraps a context.Context and provides additional Fluxor-specific
// functionality. Use Context() to get the underlying context.Context when needed
// for cancellation or passing to Go standard library functions.
type FluxorContext interface {
	// Context returns the underlying context.Context (Go's standard context)
	Context() context.Context

	// EventBus returns the event bus instance
	EventBus() EventBus

	// GoCMD returns the GoCMD instance (kept as GoCMD for backward compatibility)
	GoCMD() GoCMD

	// Config returns the configuration map
	Config() map[string]interface{}

	// SetConfig sets a configuration value
	SetConfig(key string, value interface{})

	// Deploy deploys a verticle
	Deploy(verticle Verticle) (string, error)

	// Undeploy undeploys a verticle by deployment ID
	Undeploy(deploymentID string) error
}

// gocmdContext implements FluxorContext
type gocmdContext struct {
	goCtx  context.Context // renamed from 'ctx' for clarity: this is Go's context.Context
	gocmd  GoCMD
	config map[string]interface{}
}

// newFluxorContext creates a new FluxorContext wrapping the given context.Context.
// This is an internal function - renamed from 'newContext' for clarity.
//
// Parameters:
//   - goCtx: the Go context.Context to wrap (typically from GoCMD.rootCtx)
//   - gocmd: the GoCMD instance for accessing EventBus and deploying verticles
func newFluxorContext(goCtx context.Context, gocmd GoCMD) FluxorContext {
	if goCtx == nil {
		// Fail-fast: context cannot be nil
		panic("context cannot be nil")
	}
	return &gocmdContext{
		goCtx:  goCtx,
		gocmd:  gocmd,
		config: make(map[string]interface{}),
	}
}

// Context returns the underlying context.Context (Go's standard context)
func (c *gocmdContext) Context() context.Context {
	return c.goCtx
}

func (c *gocmdContext) EventBus() EventBus {
	if c.gocmd == nil {
		// Fail-fast: gocmd is nil
		panic("gocmd is nil, cannot get EventBus")
	}
	return c.gocmd.EventBus()
}

func (c *gocmdContext) GoCMD() GoCMD {
	return c.gocmd
}

func (c *gocmdContext) Config() map[string]interface{} {
	return c.config
}

func (c *gocmdContext) SetConfig(key string, value interface{}) {
	c.config[key] = value
}

func (c *gocmdContext) Deploy(verticle Verticle) (string, error) {
	return c.gocmd.DeployVerticle(verticle)
}

func (c *gocmdContext) Undeploy(deploymentID string) error {
	return c.gocmd.UndeployVerticle(deploymentID)
}
