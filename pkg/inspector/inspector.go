package inspector

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fluxor-io/fluxor/pkg/runtime"
	"github.com/fluxor-io/fluxor/pkg/types"
)

// Inspector is a component that provides an HTTP endpoint for inspecting the runtime.
type Inspector struct {
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

func (i *Inspector) Name() string {
	return "inspector"
}

// OnStart starts the inspector's HTTP server.
func (i *Inspector) OnStart(ctx context.Context, b types.Bus) error {
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

// OnStop gracefully shuts down the inspector's HTTP server.
func (i *Inspector) OnStop(ctx context.Context) error {
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
