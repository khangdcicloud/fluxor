package web

import (
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

// fastRouter implements Router for fasthttp
type fastRouter struct {
	routes     []*fastRoute
	middleware []FastMiddleware
	mu         sync.RWMutex
}

type fastRoute struct {
	method  string
	path    string
	handler FastRequestHandler
}

// FastRequestHandler handles fasthttp requests
type FastRequestHandler func(ctx *FastRequestContext) error

// FastMiddleware is middleware for fasthttp
type FastMiddleware func(handler FastRequestHandler) FastRequestHandler

// newFastRouter creates a new fasthttp router
func newFastRouter() *fastRouter {
	return &fastRouter{
		routes:     make([]*fastRoute, 0),
		middleware: make([]FastMiddleware, 0),
	}
}

// ServeFastHTTP implements fasthttp request handler
func (r *fastRouter) ServeFastHTTP(ctx *FastRequestContext) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	method := string(ctx.Method())
	path := string(ctx.Path())

	for _, route := range r.routes {
		if route.method == method && r.matchPath(route.path, path) {
			// Extract params
			r.extractParams(route.path, path, ctx.Params)

			// Apply middleware chain
			handler := route.handler
			for i := len(r.middleware) - 1; i >= 0; i-- {
				handler = r.middleware[i](handler)
			}

			// Execute handler
			if err := handler(ctx); err != nil {
				ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			}
			return
		}
	}

	// Not found
	ctx.Error("Not Found", fasthttp.StatusNotFound)
}

func (r *fastRouter) GETFast(path string, handler FastRequestHandler) {
	r.RouteFast("GET", path, handler)
}

func (r *fastRouter) POSTFast(path string, handler FastRequestHandler) {
	r.RouteFast("POST", path, handler)
}

func (r *fastRouter) PUTFast(path string, handler FastRequestHandler) {
	r.RouteFast("PUT", path, handler)
}

func (r *fastRouter) DELETEFast(path string, handler FastRequestHandler) {
	r.RouteFast("DELETE", path, handler)
}

func (r *fastRouter) PATCHFast(path string, handler FastRequestHandler) {
	r.RouteFast("PATCH", path, handler)
}

// Implement Router interface for compatibility (not used with fasthttp)
func (r *fastRouter) GET(path string, handler RequestHandler) {
	// Not implemented for standard http - use GETFast instead
}

func (r *fastRouter) POST(path string, handler RequestHandler) {
	// Not implemented for standard http - use POSTFast instead
}

func (r *fastRouter) PUT(path string, handler RequestHandler) {
	// Not implemented for standard http
}

func (r *fastRouter) DELETE(path string, handler RequestHandler) {
	// Not implemented for standard http
}

func (r *fastRouter) PATCH(path string, handler RequestHandler) {
	// Not implemented for standard http
}

// RouteFast registers a fast handler
func (r *fastRouter) RouteFast(method, path string, handler FastRequestHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Apply middleware
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	r.routes = append(r.routes, &fastRoute{
		method:  method,
		path:    path,
		handler: handler,
	})
}

func (r *fastRouter) Route(method, path string, handler RequestHandler) {
	// Convert to FastRequestHandler
	r.RouteFast(method, path, func(ctx *FastRequestContext) error {
		reqCtx := &RequestContext{
			Request:  nil,
			Response: nil,
			Vertx:    ctx.Vertx,
			EventBus: ctx.EventBus,
			Params:   ctx.Params,
		}
		return handler(reqCtx)
	})
}

func (r *fastRouter) Use(middleware Middleware) {
	// Convert middleware to FastMiddleware
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			reqCtx := &RequestContext{
				Request:  nil,
				Response: nil,
				Vertx:    ctx.Vertx,
				EventBus: ctx.EventBus,
				Params:   ctx.Params,
			}
			wrapped := middleware(func(reqCtx *RequestContext) error {
				return next(ctx)
			})
			return wrapped(reqCtx)
		}
	})
}

func (r *fastRouter) matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			continue // Parameter
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}

func (r *fastRouter) extractParams(pattern, path string, params map[string]string) {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			if i < len(pathParts) {
				params[paramName] = pathParts[i]
			}
		}
	}
}
