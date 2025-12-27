package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testVerticle struct {
	started bool
	stopped bool
}

func (v *testVerticle) Start(ctx FluxorContext) error {
	v.started = true
	return nil
}

func (v *testVerticle) Stop(ctx FluxorContext) error {
	v.stopped = true
	return nil
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
	if !verticle.started {
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

	err = gocmd.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}
	if !verticle.stopped {
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

	id, err := gocmd.DeployVerticle(&failingStartVerticle{})
	if err == nil {
		t.Fatalf("DeployVerticle() expected error when Start() fails")
	}
	if id != "" {
		t.Fatalf("DeployVerticle() id = %q, want empty on start failure", id)
	}
}
