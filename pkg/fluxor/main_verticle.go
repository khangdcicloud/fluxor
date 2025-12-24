package fluxor

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
)

// MainVerticle is a convenience bootstrapper for "main-like" applications:
// load config -> deploy verticles -> Start() blocks until shutdown signal.
type MainVerticle struct {
	ctx    context.Context
	cancel context.CancelFunc

	vertx core.Vertx

	cfg map[string]any

	mu            sync.Mutex
	deploymentIDs []string
}

// NewMainVerticle loads config from path (json/yaml) and creates an app runtime.
// If configPath is empty, config is an empty map.
func NewMainVerticle(configPath string) (*MainVerticle, error) {
	rootCtx, cancel := context.WithCancel(context.Background())

	cfg := make(map[string]any)
	if configPath != "" {
		if err := config.Load(configPath, &cfg); err != nil {
			cancel()
			return nil, err
		}
	}

	vx := core.NewVertx(rootCtx)

	return &MainVerticle{
		ctx:    rootCtx,
		cancel: cancel,
		vertx:  vx,
		cfg:    cfg,
	}, nil
}

// Vertx returns the underlying Vertx (advanced usage).
func (m *MainVerticle) Vertx() core.Vertx { return m.vertx }

// Config returns the loaded config map (read-only by convention).
func (m *MainVerticle) Config() map[string]any { return m.cfg }

// DeployVerticle deploys a verticle after injecting global config into its FluxorContext.
func (m *MainVerticle) DeployVerticle(v core.Verticle) (string, error) {
	if v == nil {
		return "", &core.Error{Code: "INVALID_INPUT", Message: "verticle cannot be nil"}
	}

	var wrapped core.Verticle
	if av, ok := v.(core.AsyncVerticle); ok {
		wrapped = &configInjectedAsyncVerticle{inner: av, cfg: m.cfg}
	} else {
		wrapped = &configInjectedVerticle{inner: v, cfg: m.cfg}
	}

	id, err := m.vertx.DeployVerticle(wrapped)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	m.deploymentIDs = append(m.deploymentIDs, id)
	m.mu.Unlock()

	return id, nil
}

// Start blocks until SIGINT/SIGTERM then stops the app.
func (m *MainVerticle) Start() error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	<-sig
	return m.Stop()
}

// Stop gracefully shuts down: cancels root context and closes Vertx (undeploys verticles).
func (m *MainVerticle) Stop() error {
	m.cancel()
	return m.vertx.Close()
}

// configInjectedVerticle injects the app config into the FluxorContext before calling Start/Stop.
type configInjectedVerticle struct {
	inner core.Verticle
	cfg   map[string]any
}

func (v *configInjectedVerticle) Start(ctx core.FluxorContext) error {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	return v.inner.Start(ctx)
}

func (v *configInjectedVerticle) Stop(ctx core.FluxorContext) error {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	return v.inner.Stop(ctx)
}

type configInjectedAsyncVerticle struct {
	inner core.AsyncVerticle
	cfg   map[string]any
}

func (v *configInjectedAsyncVerticle) Start(ctx core.FluxorContext) error {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	return v.inner.Start(ctx)
}

func (v *configInjectedAsyncVerticle) Stop(ctx core.FluxorContext) error {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	return v.inner.Stop(ctx)
}

func (v *configInjectedAsyncVerticle) AsyncStart(ctx core.FluxorContext, resultHandler func(error)) {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	v.inner.AsyncStart(ctx, resultHandler)
}

func (v *configInjectedAsyncVerticle) AsyncStop(ctx core.FluxorContext, resultHandler func(error)) {
	for k, val := range v.cfg {
		ctx.SetConfig(k, val)
	}
	v.inner.AsyncStop(ctx, resultHandler)
}
