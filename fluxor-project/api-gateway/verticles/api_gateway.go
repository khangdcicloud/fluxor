package verticles

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// ApiGatewayVerticle is an example verticle exposing HTTP endpoints.
type ApiGatewayVerticle struct {
	server *web.FastHTTPServer
}

func NewApiGatewayVerticle() *ApiGatewayVerticle {
	return &ApiGatewayVerticle{}
}

func (v *ApiGatewayVerticle) Start(ctx core.FluxorContext) error {
	vertx := ctx.Vertx()

	cfg := web.DefaultFastHTTPServerConfig(":8080")
	v.server = web.NewFastHTTPServer(vertx, cfg)

	r := v.server.FastRouter()
	r.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]any{"status": "ok"})
	})

	go func() { _ = v.server.Start() }()
	return nil
}

func (v *ApiGatewayVerticle) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
