package web

import "github.com/fluxorio/fluxor/pkg/lite/fx"

type HandlerFunc func(c *fx.Context) error

type Router struct {
	routes map[string]HandlerFunc
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]HandlerFunc)}
}

func (r *Router) GET(path string, h HandlerFunc) {
	r.routes["GET "+path] = h
}

func (r *Router) Handle(c *fx.Context) error {
	key := c.R.Method + " " + c.R.URL.Path
	if h, ok := r.routes[key]; ok {
		return h(c)
	}
	return c.Error(404, "Not Found")
}
