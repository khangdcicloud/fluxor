package core

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GoCMD is the main entry point for the Fluxor runtime
type GoCMD interface {
	// EventBus returns the event bus
	EventBus() EventBus

	// DeployVerticle deploys a verticle
	DeployVerticle(verticle Verticle) (string, error)

	// UndeployVerticle undeploys a verticle
	UndeployVerticle(deploymentID string) error

	// DeploymentCount returns the number of deployed verticles
	DeploymentCount() int

	// Close closes the GoCMD instance
	Close() error

	// Context returns the root context
	Context() context.Context
}

// gocmd implements GoCMD
//
// Ownership and lifecycle:
//   - gocmd owns the EventBus instance (created in constructor, closed in Close())
//   - gocmd owns all deployment records
//   - gocmd owns the root context (rootCtx) and its cancel function
//
// Note: EventBus has a back-reference to GoCMD (circular dependency) to create
// FluxorContext for message handlers. This is intentional and doesn't cause
// memory leaks since both are cleaned up together in Close().
type gocmd struct {
	eventBus    EventBus
	deployments map[string]*deployment
	mu          sync.RWMutex
	rootCtx     context.Context    // renamed from 'ctx' for clarity: this is the root context.Context
	rootCancel  context.CancelFunc // renamed from 'cancel' for clarity
	logger      Logger
}

// GoCMDOptions configures GoCMD construction.
type GoCMDOptions struct {
	// EventBusFactory allows swapping the default in-memory EventBus with a custom implementation
	// (e.g., a clustered EventBus backed by NATS).
	//
	// The factory is called after the GoCMD struct is created so implementations can reference GoCMD.
	EventBusFactory func(ctx context.Context, gocmd GoCMD) (EventBus, error)
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

// NewGoCMD creates a new GoCMD instance
func NewGoCMD(ctx context.Context) GoCMD {
	gx, err := NewGoCMDWithOptions(ctx, GoCMDOptions{})
	if err != nil {
		// Fail-fast: default construction should not fail.
		panic(err)
	}
	return gx
}

// NewGoCMDWithOptions creates a new GoCMD instance with configurable EventBus.
//
// The provided ctx becomes the parent of the root context. When the parent is
// cancelled, the GoCMD instance will also be cancelled.
func NewGoCMDWithOptions(ctx context.Context, opts GoCMDOptions) (GoCMD, error) {
	rootCtx, rootCancel := context.WithCancel(ctx)
	g := &gocmd{
		deployments: make(map[string]*deployment),
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
		logger:      NewDefaultLogger(),
	}

	if opts.EventBusFactory != nil {
		bus, err := opts.EventBusFactory(rootCtx, g)
		if err != nil {
			rootCancel()
			return nil, err
		}
		g.eventBus = bus
		return g, nil
	}

	// Default: in-memory EventBus.
	g.eventBus = NewEventBus(rootCtx, g)
	return g, nil
}

func (g *gocmd) EventBus() EventBus {
	return g.eventBus
}

func (g *gocmd) DeployVerticle(verticle Verticle) (string, error) {
	// Fail-fast: validate verticle immediately
	if err := ValidateVerticle(verticle); err != nil {
		return "", err
	}

	deploymentID := generateDeploymentID()
	fluxorCtx := newFluxorContext(g.rootCtx, g)

	dep := &deployment{
		id:        deploymentID,
		verticle:  verticle,
		fluxorCtx: fluxorCtx,
		state:     DeploymentStatePending,
	}

	// All verticles are started in goroutine - single Start() method
	// Framework handles blocking operations automatically
	// Add to map in PENDING state before starting
	g.mu.Lock()
	g.deployments[deploymentID] = dep
	g.mu.Unlock()

	// Start verticle in goroutine - framework handles blocking operations
	// Single Start() method - no need for AsyncStart
	go func() {
		if err := verticle.Start(fluxorCtx); err != nil {
			// Remove from map on failure
			g.mu.Lock()
			dep.state = DeploymentStateFailed
			delete(g.deployments, deploymentID)
			g.mu.Unlock()
			g.logger.Error(fmt.Sprintf("verticle start failed for deployment %s: %v", deploymentID, err))
			return
		}

		// Mark as started on success
		g.mu.Lock()
		dep.state = DeploymentStateStarted
		g.mu.Unlock()
	}()

	return deploymentID, nil
}

func (g *gocmd) UndeployVerticle(deploymentID string) error {
	// Fail-fast: validate deployment ID
	if deploymentID == "" {
		return &EventBusError{Code: "INVALID_DEPLOYMENT_ID", Message: "deployment ID cannot be empty"}
	}

	g.mu.Lock()
	dep, exists := g.deployments[deploymentID]
	if !exists {
		g.mu.Unlock()
		return &EventBusError{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}

	// Check if deployment is in a valid state for undeploy
	// Allow undeploying PENDING deployments during shutdown (Close())
	isShuttingDown := false
	select {
	case <-g.rootCtx.Done():
		isShuttingDown = true
	default:
	}

	if dep.state == DeploymentStatePending && !isShuttingDown {
		g.mu.Unlock()
		return &EventBusError{Code: "DEPLOYMENT_PENDING", Message: "Cannot undeploy pending deployment: " + deploymentID}
	}
	if dep.state == DeploymentStateStopping || dep.state == DeploymentStateStopped {
		g.mu.Unlock()
		return &EventBusError{Code: "DEPLOYMENT_ALREADY_STOPPING", Message: "Deployment already stopping/stopped: " + deploymentID}
	}

	dep.state = DeploymentStateStopping
	delete(g.deployments, deploymentID)
	g.mu.Unlock()

	// Stop verticle - framework handles blocking operations
	// Single Stop() method - no need for AsyncStop
	go func() {
		if err := dep.verticle.Stop(dep.fluxorCtx); err != nil {
			g.logger.Error(fmt.Sprintf("verticle stop failed for deployment %s: %v", deploymentID, err))
		}
		dep.state = DeploymentStateStopped
	}()

	return nil
}

// DeploymentCount returns the number of deployed verticles
func (g *gocmd) DeploymentCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.deployments)
}

// Close gracefully shuts down the GoCMD instance.
//
// Shutdown order:
//  1. Undeploy all verticles (calls Stop on each)
//  2. Cancel the root context (signals all children to stop)
//  3. Close the EventBus (which also cancels its internal context - intentionally
//     redundant as defense-in-depth since EventBus.ctx is a child of rootCtx)
func (g *gocmd) Close() error {
	g.mu.Lock()
	deployments := make([]*deployment, 0, len(g.deployments))
	for _, dep := range g.deployments {
		deployments = append(deployments, dep)
	}
	g.mu.Unlock()

	// Cancel root context first to signal all goroutines to stop
	// This will cause pending Start() calls to fail gracefully
	g.rootCancel()

	// Wait a bit for pending Start() calls to complete or fail
	// This gives goroutines time to finish their Start() and update state
	time.Sleep(100 * time.Millisecond)

	// Undeploy all verticles (including any that are still PENDING)
	for _, dep := range deployments {
		// Force undeploy even if PENDING (during shutdown)
		g.mu.Lock()
		if dep.state == DeploymentStatePending {
			// Mark as stopping and remove from map
			dep.state = DeploymentStateStopping
			delete(g.deployments, dep.id)
			g.mu.Unlock()

			// Stop verticle in goroutine
			go func(d *deployment, id string) {
				if err := d.verticle.Stop(d.fluxorCtx); err != nil {
					g.logger.Error(fmt.Sprintf("verticle stop failed for deployment %s: %v", id, err))
				}
			}(dep, dep.id)
		} else {
			g.mu.Unlock()
			if err := g.UndeployVerticle(dep.id); err != nil {
				// Log error during mass undeploy but continue
				g.logger.Info(fmt.Sprintf("Failed to undeploy verticle %s during close: %v", dep.id, err))
			}
		}
	}

	// Close EventBus (its internal cancel is redundant but kept for defense-in-depth)
	return g.eventBus.Close()
}

// Context returns the root context.Context for this GoCMD instance.
// This context is cancelled when Close() is called.
func (g *gocmd) Context() context.Context {
	return g.rootCtx
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
//   - The underlying context.Context is cancelled when GoCMD.Close() is called
type deployment struct {
	id        string
	verticle  Verticle
	fluxorCtx FluxorContext   // renamed from 'ctx' for clarity: this is FluxorContext, not context.Context
	state     DeploymentState // tracks lifecycle state
}

func generateDeploymentID() string {
	return fmt.Sprintf("deployment.%s", generateUUID())
}
