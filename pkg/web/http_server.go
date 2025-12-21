package web

import (
	"context"
	"net/http"
	"sync"
	
	"github.com/fluxorio/fluxor/pkg/core"
)

// httpServer implements Server
type httpServer struct {
	vertx      core.Vertx
	router     *router
	httpServer *http.Server
	mu         sync.RWMutex
}

// NewServer creates a new HTTP server
func NewServer(vertx core.Vertx, addr string) Server {
	r := NewRouter().(*router)
	
	return &httpServer{
		vertx: vertx,
		router: r,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}

func (s *httpServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Inject Vertx and EventBus into router handlers
	// This is done by wrapping the router's ServeHTTP
	
	return s.httpServer.ListenAndServe()
}

func (s *httpServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	ctx := context.Background()
	return s.httpServer.Shutdown(ctx)
}

func (s *httpServer) Router() Router {
	return s.router
}

// InjectVertx injects Vertx and EventBus into request context
func (s *httpServer) InjectVertx(ctx *RequestContext) {
	ctx.Vertx = s.vertx
	ctx.EventBus = s.vertx.EventBus()
}

