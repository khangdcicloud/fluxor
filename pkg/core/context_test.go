package core

import (
	"context"
	"testing"
)

func TestNewContext(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	fluxorCtx := newContext(ctx, vertx)

	if fluxorCtx == nil {
		t.Error("newContext() should not return nil")
	}

	if fluxorCtx.Context() == nil {
		t.Error("Context() should not return nil")
	}

	if fluxorCtx.Vertx() == nil {
		t.Error("Vertx() should not return nil")
	}

	if fluxorCtx.EventBus() == nil {
		t.Error("EventBus() should not return nil")
	}
}

func TestFluxorContext_Config(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	fluxorCtx := newContext(ctx, vertx)

	// Test setting config
	fluxorCtx.SetConfig("key1", "value1")
	fluxorCtx.SetConfig("key2", 42)

	config := fluxorCtx.Config()
	if config["key1"] != "value1" {
		t.Errorf("Config() key1 = %v, want value1", config["key1"])
	}

	if config["key2"] != 42 {
		t.Errorf("Config() key2 = %v, want 42", config["key2"])
	}
}

func TestFluxorContext_Deploy(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	fluxorCtx := newContext(ctx, vertx)

	verticle := &testVerticle{}
	deploymentID, err := fluxorCtx.Deploy(verticle)

	if err != nil {
		t.Errorf("Deploy() error = %v", err)
	}

	if deploymentID == "" {
		t.Error("Deploy() returned empty deployment ID")
	}

	if !verticle.started {
		t.Error("Verticle should be started")
	}
}

func TestFluxorContext_Undeploy(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	defer vertx.Close()

	fluxorCtx := newContext(ctx, vertx)

	verticle := &testVerticle{}
	deploymentID, err := fluxorCtx.Deploy(verticle)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	err = fluxorCtx.Undeploy(deploymentID)
	if err != nil {
		t.Errorf("Undeploy() error = %v", err)
	}

	if !verticle.stopped {
		t.Error("Verticle should be stopped")
	}
}
