package core

import (
	"context"
	"testing"
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
