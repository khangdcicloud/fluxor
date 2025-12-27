package web

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestFastHTTPServer_NewServer(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	if server == nil {
		t.Error("NewFastHTTPServer() should not return nil")
	}

	if server.FastRouter() == nil {
		t.Error("FastRouter() should not return nil")
	}
}

func TestFastRequestContext_JSON(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a test context (we can't easily test fasthttp.RequestCtx without actual request)
	// This test verifies the validation logic

	// Test fail-fast: invalid status code
	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := reqCtx.JSON(999, "test") // Invalid status code
	if err == nil {
		t.Error("JSON() with invalid status code should fail")
	}

	err = reqCtx.JSON(0, "test") // Invalid status code
	if err == nil {
		t.Error("JSON() with zero status code should fail")
	}
}

func TestFastRequestContext_BindJSON(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Test fail-fast: nil target
	err := reqCtx.BindJSON(nil)
	if err == nil {
		t.Error("BindJSON() with nil target should fail")
	}
}

func TestDefaultFastHTTPServerConfig(t *testing.T) {
	config := DefaultFastHTTPServerConfig(":8080")

	if config.Addr != ":8080" {
		t.Errorf("Addr = %v, want :8080", config.Addr)
	}

	if config.MaxQueue <= 0 {
		t.Error("MaxQueue should be positive")
	}

	if config.Workers <= 0 {
		t.Error("Workers should be positive")
	}
}
