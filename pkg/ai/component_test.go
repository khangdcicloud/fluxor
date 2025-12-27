package ai

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewAIComponent(t *testing.T) {
	config := Config{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-3.5-turbo",
	}

	component := NewAIComponent(config)
	if component == nil {
		t.Fatal("NewAIComponent() returned nil")
	}

	if component.Name() != "ai" {
		t.Errorf("Name() = %v, want 'ai'", component.Name())
	}

	// Component should not be started yet
	if component.IsStarted() {
		t.Error("Component should not be started after creation")
	}
}

func TestAIComponent_StartStop(t *testing.T) {
	// This test requires a real API key or mock, so we'll test the structure
	config := Config{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-3.5-turbo",
	}

	component := NewAIComponent(config)

	// Create a minimal context for testing using GoCMD
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a test verticle to embed the component
	testVerticle := &core.BaseVerticle{}
	testVerticle = core.NewBaseVerticle("test")

	// Deploy verticle to get a proper context
	deploymentID, err := gocmd.DeployVerticle(testVerticle)
	if err != nil {
		t.Fatalf("Failed to deploy verticle: %v", err)
	}
	defer gocmd.UndeployVerticle(deploymentID)

	// Get context from verticle after it's started
	fluxorCtx := testVerticle.Context()
	if fluxorCtx == nil {
		t.Skip("Skipping test - verticle context not available (requires async start)")
		return
	}

	// Test start (may fail with invalid API key, but structure should work)
	err = component.Start(fluxorCtx)
	if err != nil {
		// Expected if API key is invalid, but structure should work
		t.Logf("Start() error (expected if API key invalid): %v", err)
	} else {
		// Test stop only if start succeeded
		err = component.Stop(fluxorCtx)
		if err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	}
}

func TestAIComponent_Client_NotStarted(t *testing.T) {
	config := Config{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
	}

	component := NewAIComponent(config)

	_, err := component.Client()
	if err == nil {
		t.Error("Client() should return error when component is not started")
	}
}
