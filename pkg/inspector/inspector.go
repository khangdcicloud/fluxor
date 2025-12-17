package inspector

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/example/fluxor/pkg/bus"
	"github.com/example/fluxor/pkg/component"
	"github.com/example/fluxor/pkg/runtime"
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

// OnStart starts the inspector's HTTP server.
func (i *Inspector) OnStart(ctx context.Context, b bus.Bus) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", i.handleStatus)

	i.server = &http.Server{
		Addr:    i.addr,
		Handler: mux,
	}

	i.Go(func() {
		if err := i.server.ListenAndServe(); err != http.ErrServerClosed {
			// log error
		}
	})
}

// OnStop gracefully shuts down the inspector's HTTP server.
func (i *Inspector) OnStop(ctx context.Context) {
	if i.server != nil {
		i.server.Shutdown(ctx)
	}
}

// handleStatus returns the runtime's status as JSON.
func (i *Inspector) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := i.runtime.Status()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
