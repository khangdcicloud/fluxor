package web

import (
	"context"
	"net/http"
	
	"github.com/fluxorio/fluxor/pkg/core"
)

// httpServer implements Server
// Extends BaseServer for common lifecycle management
type httpServer struct {
	*core.BaseServer // Embed base server for lifecycle management
	router           *router
	httpServer       *http.Server
}

// NewServer creates a new HTTP server
func NewServer(vertx core.Vertx, addr string) Server {
	r := NewRouter().(*router)
	
	return &httpServer{
		BaseServer: core.NewBaseServer("http-server", vertx),
		router:     r,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}

// doStart is called by BaseServer.Start() - implements hook method
func (s *httpServer) doStart() error {
	// Inject Vertx and EventBus into router handlers
	// This is done by wrapping the router's ServeHTTP
	
	return s.httpServer.ListenAndServe()
}

// doStop is called by BaseServer.Stop() - implements hook method
func (s *httpServer) doStop() error {
	ctx := context.Background()
	return s.httpServer.Shutdown(ctx)
}

func (s *httpServer) Router() Router {
	return s.router
}

// InjectVertx injects Vertx and EventBus into request context
func (s *httpServer) InjectVertx(ctx *RequestContext) {
	ctx.Vertx = s.Vertx()
	ctx.EventBus = s.EventBus()
}

