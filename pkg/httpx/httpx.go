package httpx

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/reactor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is a simple HTTP server that can be used to trigger event bus calls.
type Server struct {
	bus     bus.Bus
	reactor *reactor.Reactor
	srv     *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(port int, reactor *reactor.Reactor, bus bus.Bus) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	return &Server{
		bus:     bus,
		reactor: reactor,
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: r,
		},
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
