package inspector

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/runtime"
)

// Inspector is a component that provides an HTTP endpoint for inspecting the runtime.
type Inspector struct {
	component.Base
	runtime *runtime.Runtime
	addr    string
	server  *http.Server
}

// NewInspector creates a new Inspector.
func NewInspector(addr string, rt *runtime.Runtime) *Inspector {
	return &Inspector{
		addr:    addr,
		runtime: rt,
	}
}

// Start starts the inspector's HTTP server.
func (i *Inspector) Start(ctx context.Context, b bus.Bus) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", i.handleStatus)

	i.server = &http.Server{
		Addr:    i.addr,
		Handler: mux,
	}

	go func() {
		if err := i.server.ListenAndServe(); err != http.ErrServerClosed {
			// log error
		}
	}()
	return nil
}

// Stop gracefully shuts down the inspector's HTTP server.
func (i *Inspector) Stop(ctx context.Context) error {
	if i.server != nil {
		return i.server.Shutdown(ctx)
	}
	return nil
}

// handleStatus returns the runtime's status as JSON.
func (i *Inspector) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := i.runtime.Status()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
