package fluxor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

type cfgCheckVerticle struct{}

func (v *cfgCheckVerticle) Start(ctx core.FluxorContext) error {
	if got := ctx.Config()["foo"]; got != "bar" {
		return &core.Error{Code: "CONFIG_MISSING", Message: "expected foo=bar in context config"}
	}
	return nil
}

func (v *cfgCheckVerticle) Stop(ctx core.FluxorContext) error { return nil }

func TestMainVerticle_LoadConfig_AndInjectOnDeploy(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"foo":"bar"}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	app, err := NewMainVerticle(cfgPath)
	if err != nil {
		t.Fatalf("NewMainVerticle: %v", err)
	}
	defer app.Stop()

	if _, err := app.DeployVerticle(&cfgCheckVerticle{}); err != nil {
		t.Fatalf("DeployVerticle: %v", err)
	}
}
