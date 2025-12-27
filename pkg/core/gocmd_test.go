package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type testVerticle struct {
	mu      sync.RWMutex
	started bool
	stopped bool
}

func (v *testVerticle) Start(ctx FluxorContext) error {
	v.mu.Lock()
	v.started = true
	v.mu.Unlock()
	return nil
}

func (v *testVerticle) Stop(ctx FluxorContext) error {
	v.mu.Lock()
	v.stopped = true
	v.mu.Unlock()
	return nil
}

func (v *testVerticle) isStarted() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.started
}

func (v *testVerticle) isStopped() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.stopped
}

func TestGoCMD_DeployVerticle(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Test fail-fast: nil verticle
	_, err := gocmd.DeployVerticle(nil)
	if err == nil {
		t.Error("DeployVerticle() with nil verticle should fail")
	}

	// Test valid deployment
	verticle := &testVerticle{}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Errorf("DeployVerticle() error = %v", err)
	}
	if deploymentID == "" {
		t.Error("DeployVerticle() returned empty deployment ID")
	}

	// Wait for async start to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !verticle.isStarted() {
		t.Error("Verticle should be started")
	}
}

func TestGoCMD_UndeployVerticle(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Test fail-fast: empty deployment ID
	err := gocmd.UndeployVerticle("")
	if err == nil {
		t.Error("UndeployVerticle() with empty ID should fail")
	}

	// Test fail-fast: non-existent deployment
	err = gocmd.UndeployVerticle("non-existent")
	if err == nil {
		t.Error("UndeployVerticle() with non-existent ID should fail")
	}

	// Deploy and undeploy
	verticle := &testVerticle{}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start to complete before undeploying
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = gocmd.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Wait for async stop to complete
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStopped() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !verticle.isStopped() {
		t.Error("Verticle should be stopped")
	}
}

func TestGoCMD_EventBus(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	if eb == nil {
		t.Error("EventBus() should not return nil")
	}
}

func TestNewGoCMDWithOptions_FailFast_EventBusFactoryErrorCancelsContext(t *testing.T) {
	parent := context.Background()

	wantErr := errors.New("factory failed")
	var factoryCtx context.Context

	vx, err := NewGoCMDWithOptions(parent, GoCMDOptions{
		EventBusFactory: func(ctx context.Context, _ GoCMD) (EventBus, error) {
			factoryCtx = ctx
			return nil, wantErr
		},
	})
	if err == nil {
		t.Fatalf("NewGoCMDWithOptions() expected error, got nil (vx=%v)", vx)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewGoCMDWithOptions() error = %v, want %v", err, wantErr)
	}
	if factoryCtx == nil {
		t.Fatalf("expected factory to be invoked and capture ctx")
	}

	select {
	case <-factoryCtx.Done():
		// ok: fail-fast cleanup should cancel internal ctx
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("expected internal context to be cancelled on factory error")
	}
}

type failingStartVerticle struct{}

func (v *failingStartVerticle) Start(ctx FluxorContext) error { return errors.New("start failed") }
func (v *failingStartVerticle) Stop(ctx FluxorContext) error  { return nil }

func TestGoCMD_DeployVerticle_FailFast_StartError(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Since Start() is now async, DeployVerticle() always succeeds
	// The error is handled asynchronously and deployment is removed from map
	id, err := gocmd.DeployVerticle(&failingStartVerticle{})
	if err != nil {
		t.Fatalf("DeployVerticle() should not return error (start is async), got %v", err)
	}
	if id == "" {
		t.Fatalf("DeployVerticle() should return deployment ID, got empty")
	}

	// Wait for async start to complete and fail
	// The goroutine will remove the deployment from map on failure
	// Use DeploymentCount() to verify deployment was removed
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gocmd.DeploymentCount() == 0 {
			// Deployment removed - failure handled correctly
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify deployment was removed
	if gocmd.DeploymentCount() != 0 {
		t.Errorf("expected 0 deployments after start failure, got %d", gocmd.DeploymentCount())
	}
}
