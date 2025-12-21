package core

import (
	"context"
)

// FluxorContext represents the execution context for a verticle or handler
type FluxorContext interface {
	// Context returns the underlying context.Context
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
	ctx    context.Context
	vertx  Vertx
	config map[string]interface{}
}

func newContext(ctx context.Context, vertx Vertx) FluxorContext {
	if ctx == nil {
		// Fail-fast: context cannot be nil
		panic("context cannot be nil")
	}
	return &vertxContext{
		ctx:    ctx,
		vertx:  vertx,
		config: make(map[string]interface{}),
	}
}

func (c *vertxContext) Context() context.Context {
	return c.ctx
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
