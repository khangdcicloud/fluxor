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
//
// Ownership and lifecycle:
//   - vertx owns the EventBus instance (created in constructor, closed in Close())
//   - vertx owns all deployment records
//   - vertx owns the root context (rootCtx) and its cancel function
//
// Note: EventBus has a back-reference to Vertx (circular dependency) to create
// FluxorContext for message handlers. This is intentional and doesn't cause
// memory leaks since both are cleaned up together in Close().
type vertx struct {
	eventBus    EventBus
	deployments map[string]*deployment
	mu          sync.RWMutex
	rootCtx     context.Context    // renamed from 'ctx' for clarity: this is the root context.Context
	rootCancel  context.CancelFunc // renamed from 'cancel' for clarity
	logger      Logger
}

// VertxOptions configures Vertx construction.
type VertxOptions struct {
	// EventBusFactory allows swapping the default in-memory EventBus with a custom implementation
	// (e.g., a clustered EventBus backed by NATS).
	//
	// The factory is called after the Vertx struct is created so implementations can reference Vertx.
	EventBusFactory func(ctx context.Context, vertx Vertx) (EventBus, error)
}

// DeploymentState represents the lifecycle state of a deployed verticle.
type DeploymentState int

const (
	// DeploymentStatePending indicates the verticle is being started (AsyncVerticle only)
	DeploymentStatePending DeploymentState = iota
	// DeploymentStateStarted indicates the verticle has successfully started
	DeploymentStateStarted
	// DeploymentStateFailed indicates the verticle failed to start (AsyncVerticle only)
	DeploymentStateFailed
	// DeploymentStateStopping indicates the verticle is being stopped
	DeploymentStateStopping
	// DeploymentStateStopped indicates the verticle has been stopped
	DeploymentStateStopped
)

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
//
// The provided ctx becomes the parent of the root context. When the parent is
// cancelled, the Vertx instance will also be cancelled.
func NewVertxWithOptions(ctx context.Context, opts VertxOptions) (Vertx, error) {
	rootCtx, rootCancel := context.WithCancel(ctx)
	v := &vertx{
		deployments: make(map[string]*deployment),
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
		logger:      NewDefaultLogger(),
	}

	if opts.EventBusFactory != nil {
		bus, err := opts.EventBusFactory(rootCtx, v)
		if err != nil {
			rootCancel()
			return nil, err
		}
		v.eventBus = bus
		return v, nil
	}

	// Default: in-memory EventBus.
	v.eventBus = NewEventBus(rootCtx, v)
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

	deploymentID := generateDeploymentID()
	fluxorCtx := newFluxorContext(v.rootCtx, v)

	dep := &deployment{
		id:        deploymentID,
		verticle:  verticle,
		fluxorCtx: fluxorCtx,
		state:     DeploymentStatePending,
	}

	// Handle async verticles
	if asyncVerticle, ok := verticle.(AsyncVerticle); ok {
		// Add to map in PENDING state before starting
		v.mu.Lock()
		v.deployments[deploymentID] = dep
		v.mu.Unlock()

		// Asynchronous start with error handling
		// Note: deployment is in PENDING state until callback completes
		asyncVerticle.AsyncStart(fluxorCtx, func(err error) {
			v.mu.Lock()
			defer v.mu.Unlock()

			if err != nil {
				// Log the async start failure and mark as failed
				v.logger.Errorf("async verticle start failed for deployment %s: %v", deploymentID, err)
				if d, exists := v.deployments[deploymentID]; exists {
					d.state = DeploymentStateFailed
				}
				// Remove failed deployment from map
				delete(v.deployments, deploymentID)
				return
			}

			// Mark as started on success
			if d, exists := v.deployments[deploymentID]; exists {
				d.state = DeploymentStateStarted
			}
		})

		return deploymentID, nil
	}

	// Sync verticle: add to map first, then start (lock released during Start)
	v.mu.Lock()
	v.deployments[deploymentID] = dep
	v.mu.Unlock()

	// Start outside the lock to avoid blocking other deployments
	if err := verticle.Start(fluxorCtx); err != nil {
		// Remove from map on failure
		v.mu.Lock()
		dep.state = DeploymentStateFailed
		delete(v.deployments, deploymentID)
		v.mu.Unlock()
		return "", fmt.Errorf("verticle start failed: %w", err)
	}

	// Mark as started
	v.mu.Lock()
	dep.state = DeploymentStateStarted
	v.mu.Unlock()

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
		return &Error{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}

	// Check if deployment is in a valid state for undeploy
	if dep.state == DeploymentStatePending {
		v.mu.Unlock()
		return &Error{Code: "DEPLOYMENT_PENDING", Message: "Cannot undeploy pending deployment: " + deploymentID}
	}
	if dep.state == DeploymentStateStopping || dep.state == DeploymentStateStopped {
		v.mu.Unlock()
		return &Error{Code: "DEPLOYMENT_ALREADY_STOPPING", Message: "Deployment already stopping/stopped: " + deploymentID}
	}

	dep.state = DeploymentStateStopping
	delete(v.deployments, deploymentID)
	v.mu.Unlock()

	// Handle async verticles
	if asyncVerticle, ok := dep.verticle.(AsyncVerticle); ok {
		asyncVerticle.AsyncStop(dep.fluxorCtx, func(err error) {
			// Fail-fast: log async stop errors instead of panicking
			if err != nil {
				v.logger.Errorf("async verticle stop failed for deployment %s: %v", deploymentID, err)
			}
			dep.state = DeploymentStateStopped
		})
	} else {
		// Fail-fast: stop errors are propagated immediately
		if err := dep.verticle.Stop(dep.fluxorCtx); err != nil {
			return fmt.Errorf("verticle stop failed: %w", err)
		}
		dep.state = DeploymentStateStopped
	}

	return nil
}

// DeploymentCount returns the number of deployed verticles
func (v *vertx) DeploymentCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.deployments)
}

// Close gracefully shuts down the Vertx instance.
//
// Shutdown order:
//  1. Undeploy all verticles (calls Stop on each)
//  2. Cancel the root context (signals all children to stop)
//  3. Close the EventBus (which also cancels its internal context - intentionally
//     redundant as defense-in-depth since EventBus.ctx is a child of rootCtx)
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

	// Cancel root context - this signals all child contexts to stop
	v.rootCancel()

	// Close EventBus (its internal cancel is redundant but kept for defense-in-depth)
	return v.eventBus.Close()
}

// Context returns the root context.Context for this Vertx instance.
// This context is cancelled when Close() is called.
func (v *vertx) Context() context.Context {
	return v.rootCtx
}

// deployment represents a deployed verticle instance.
//
// Lifecycle:
//   - Created in DeployVerticle with state PENDING
//   - Transitions to STARTED on successful Start(), or FAILED on error
//   - Transitions to STOPPING when UndeployVerticle is called
//   - Transitions to STOPPED after Stop() completes
//
// Ownership:
//   - fluxorCtx is valid for the lifetime of this deployment
//   - After UndeployVerticle, the fluxorCtx should not be used by the verticle
//   - The underlying context.Context is cancelled when Vertx.Close() is called
type deployment struct {
	id        string
	verticle  Verticle
	fluxorCtx FluxorContext   // renamed from 'ctx' for clarity: this is FluxorContext, not context.Context
	state     DeploymentState // tracks lifecycle state
}

func generateDeploymentID() string {
	return fmt.Sprintf("deployment.%s", generateUUID())
}
