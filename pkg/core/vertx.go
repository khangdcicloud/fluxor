package core

import (
	"context"
	"fmt"
	"sync"
)

// Vertx is the main entry point for the Fluxor runtime
type Vertx interface {
	// EventBus returns the event bus
	EventBus() EventBus

	// DeployVerticle deploys a verticle
	DeployVerticle(verticle Verticle) (string, error)

	// UndeployVerticle undeploys a verticle
	UndeployVerticle(deploymentID string) error

	// DeploymentCount returns the number of deployed verticles
	DeploymentCount() int

	// Close closes the Vertx instance
	Close() error

	// Context returns the root context
	Context() context.Context
}

// vertx implements Vertx
type vertx struct {
	eventBus    EventBus
	deployments map[string]*deployment
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      Logger // Added logger
}

// VertxOptions configures Vertx construction.
type VertxOptions struct {
	// EventBusFactory allows swapping the default in-memory EventBus with a custom implementation
	// (e.g., a clustered EventBus backed by NATS).
	//
	// The factory is called after the Vertx struct is created so implementations can reference Vertx.
	EventBusFactory func(ctx context.Context, vertx Vertx) (EventBus, error)
}

// NewVertx creates a new Vertx instance
func NewVertx(ctx context.Context) Vertx {
	vx, err := NewVertxWithOptions(ctx, VertxOptions{})
	if err != nil {
		// Fail-fast: default construction should not fail.
		panic(err)
	}
	return vx
}

// NewVertxWithOptions creates a new Vertx instance with configurable EventBus.
func NewVertxWithOptions(ctx context.Context, opts VertxOptions) (Vertx, error) {
	ctx, cancel := context.WithCancel(ctx)
	v := &vertx{
		deployments: make(map[string]*deployment),
		ctx:         ctx,
		cancel:      cancel,
		logger:      NewDefaultLogger(), // Initialize logger
	}

	if opts.EventBusFactory != nil {
		bus, err := opts.EventBusFactory(ctx, v)
		if err != nil {
			cancel()
			return nil, err
		}
		v.eventBus = bus
		return v, nil
	}

	// Default: in-memory EventBus.
	v.eventBus = NewEventBus(ctx, v)
	return v, nil
}

func (v *vertx) EventBus() EventBus {
	return v.eventBus
}

func (v *vertx) DeployVerticle(verticle Verticle) (string, error) {
	// Fail-fast: validate verticle immediately
	if err := ValidateVerticle(verticle); err != nil {
		return "", err
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	deploymentID := generateDeploymentID()
	ctx := newContext(v.ctx, v)

	dep := &deployment{
		id:       deploymentID,
		verticle: verticle,
		ctx:      ctx,
	}

	// Handle async verticles
	if asyncVerticle, ok := verticle.(AsyncVerticle); ok {
		// Asynchronous start with error handling
		asyncVerticle.AsyncStart(ctx, func(err error) {
			if err != nil {
				// Log the async start failure
				v.logger.Errorf("async verticle start failed for deployment %s: %v", deploymentID, err)
				// Remove from deployments on failure
				v.mu.Lock()
				delete(v.deployments, deploymentID)
				v.mu.Unlock()
				return
			}
		})
	} else {
		// Fail-fast: start errors are propagated immediately
		if err := verticle.Start(ctx); err != nil {
			return "", fmt.Errorf("verticle start failed: %w", err)
		}
	}

	v.deployments[deploymentID] = dep
	return deploymentID, nil
}

func (v *vertx) UndeployVerticle(deploymentID string) error {
	// Fail-fast: validate deployment ID
	if deploymentID == "" {
		return &Error{Code: "INVALID_DEPLOYMENT_ID", Message: "deployment ID cannot be empty"}
	}

	v.mu.Lock()
	dep, exists := v.deployments[deploymentID]
	if !exists {
		v.mu.Unlock()
		// Use a more specific error code
		return &Error{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}
	delete(v.deployments, deploymentID)
	v.mu.Unlock()

	// Handle async verticles
	if asyncVerticle, ok := dep.verticle.(AsyncVerticle); ok {
		asyncVerticle.AsyncStop(dep.ctx, func(err error) {
			// Fail-fast: log async stop errors instead of panicking
			if err != nil {
				v.logger.Errorf("async verticle stop failed for deployment %s: %v", deploymentID, err)
			}
		})
	} else {
		// Fail-fast: stop errors are propagated immediately
		if err := dep.verticle.Stop(dep.ctx); err != nil {
			return fmt.Errorf("verticle stop failed: %w", err)
		}
	}

	return nil
}

// DeploymentCount returns the number of deployed verticles
func (v *vertx) DeploymentCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.deployments)
}

func (v *vertx) Close() error {
	v.mu.Lock()
	deployments := make([]*deployment, 0, len(v.deployments))
	for _, dep := range v.deployments {
		deployments = append(deployments, dep)
	}
	v.mu.Unlock()

	// Undeploy all verticles
	for _, dep := range deployments {
		if err := v.UndeployVerticle(dep.id); err != nil {
			// Log error during mass undeploy but continue
			v.logger.Warnf("Failed to undeploy verticle %s during close: %v", dep.id, err)
		}
	}

	v.cancel()
	return v.eventBus.Close()
}

func (v *vertx) Context() context.Context {
	return v.ctx
}

type deployment struct {
	id       string
	verticle Verticle
	ctx      FluxorContext
}

func generateDeploymentID() string {
	return fmt.Sprintf("deployment.%s", generateUUID())
}
