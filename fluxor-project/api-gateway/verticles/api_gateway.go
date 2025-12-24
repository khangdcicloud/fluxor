package verticles

import (
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"

	"github.com/quadgatefoundation/fluxor/fluxor-project/common/contracts"
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

	addr := ":8080"
	if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" {
		addr = val
	}
	cfg := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(vertx, cfg)

	r := v.server.FastRouter()
	r.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]any{"status": "ok"})
	})

	// POST /payments/authorize -> request payment-service via EventBus (NATS cluster).
	r.POSTFast("/payments/authorize", func(c *web.FastRequestContext) error {
		var req contracts.PaymentAuthorizeRequest
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(400, map[string]any{"error": "invalid_request"})
		}
		if req.PaymentID == "" || req.UserID == "" || req.Amount <= 0 || req.Currency == "" {
			return c.JSON(400, map[string]any{"error": "invalid_request"})
		}

		reply, err := c.EventBus.Request(contracts.AddressPaymentsAuthorize, req, 2*time.Second)
		if err != nil {
			return c.JSON(502, map[string]any{"error": "payment_service_unavailable"})
		}

		body, ok := reply.Body().([]byte)
		if !ok {
			return c.JSON(502, map[string]any{"error": "bad_response"})
		}
		var resp contracts.PaymentAuthorizeReply
		if err := core.JSONDecode(body, &resp); err != nil {
			return c.JSON(502, map[string]any{"error": "bad_response"})
		}

		if !resp.OK {
			return c.JSON(402, resp)
		}
		return c.JSON(200, resp)
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
