package httpx

import (
	"context"
	"net/http"

	"github.com/example/goflux/pkg/bus"
)

// Server is a simple HTTP server that can be used to trigger event bus calls.
type Server struct {
	bus bus.Bus
	srv *http.Server
}

// NewServer creates a new HTTP server.
func NewServer(addr string, bus bus.Bus) *Server {
	return &Server{
		bus: bus,
		srv: &http.Server{Addr: addr},
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

// HandleFunc registers a handler for the given pattern.
type HandlerFunc func(bus bus.Bus, w http.ResponseWriter, r *http.Request)

// Handle registers a handler for the given pattern. The handler is executed in a goroutine.
func (s *Server) Handle(pattern string, handler HandlerFunc) {
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		// The actual handling is done in a goroutine to avoid blocking the http server.
		// In a real application, you would want to use a worker pool to limit the number of concurrent requests.
		go handler(s.bus, w, r)
	})
}
