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

func TestVertx_DeployVerticle(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	// Test fail-fast: nil verticle
	_, err := vertx.DeployVerticle(nil)
	if err == nil {
		t.Error("DeployVerticle() with nil verticle should fail")
	}

	// Test valid deployment
	verticle := &testVerticle{}
	deploymentID, err := vertx.DeployVerticle(verticle)
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

func TestVertx_UndeployVerticle(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	// Test fail-fast: empty deployment ID
	err := vertx.UndeployVerticle("")
	if err == nil {
		t.Error("UndeployVerticle() with empty ID should fail")
	}

	// Test fail-fast: non-existent deployment
	err = vertx.UndeployVerticle("non-existent")
	if err == nil {
		t.Error("UndeployVerticle() with non-existent ID should fail")
	}

	// Deploy and undeploy
	verticle := &testVerticle{}
	deploymentID, err := vertx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	err = vertx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}
	if !verticle.stopped {
		t.Error("Verticle should be stopped")
	}
}

func TestVertx_EventBus(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	eb := vertx.EventBus()
	if eb == nil {
		t.Error("EventBus() should not return nil")
	}
}

func TestNewVertxWithOptions_FailFast_EventBusFactoryErrorCancelsContext(t *testing.T) {
	parent := context.Background()

	wantErr := errors.New("factory failed")
	var factoryCtx context.Context

	vx, err := NewVertxWithOptions(parent, VertxOptions{
		EventBusFactory: func(ctx context.Context, _ Vertx) (EventBus, error) {
			factoryCtx = ctx
			return nil, wantErr
		},
	})
	if err == nil {
		t.Fatalf("NewVertxWithOptions() expected error, got nil (vx=%v)", vx)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewVertxWithOptions() error = %v, want %v", err, wantErr)
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
