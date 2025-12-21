package web

import (
	"net/http"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fx"
)

type HttpVerticle struct {
	port   string
	router Router
	server *http.Server
}

func NewHttpVerticle(port string, router Router) *HttpVerticle {
	return &HttpVerticle{port: port, router: router}
}

func (v *HttpVerticle) OnStart(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("HttpVerticle Listening", "port", v.port)

	handler := func(w http.ResponseWriter, r *http.Request) {
		c := fx.NewContext(w, r, ctx)
		// Router interface should implement http.Handler
		if httpHandler, ok := v.router.(http.Handler); ok {
			httpHandler.ServeHTTP(c.W, c.R)
		}
	}

	v.server = &http.Server{Addr: ":" + v.port, Handler: http.HandlerFunc(handler)}
	
	go func() { 
		if err := v.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server crashed", "err", err)
		}
	}()
	return nil
}

func (v *HttpVerticle) OnStop() error {
	if v.server != nil { return v.server.Close() }
	return nil
}
