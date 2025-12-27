package verticles

import (
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"

	"github.com/quadgatefoundation/fluxor/examples/fluxor-project/common/contracts"
)

// ApiGatewayVerticle is an HTTP server
// It's a process using context from FluxorContext
type ApiGatewayVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	server             *web.FastHTTPServer
	addr               string
}

func NewApiGatewayVerticle() *ApiGatewayVerticle {
	return &ApiGatewayVerticle{
		BaseVerticle: core.NewBaseVerticle("api-gateway"),
	}
}

// Start overrides BaseVerticle.Start - single entry point for initialization
// Setup HTTP server
func (v *ApiGatewayVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// API Gateway verticle is a process using context from FluxorContext
	// All setup uses the context provided by the framework
	gocmd := ctx.GoCMD()

	// Setup HTTP server address from context config
	v.addr = ":8080"
	if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" {
		v.addr = val
	}

	// Create FastHTTPServer using context's GoCMD
	logger := core.NewDefaultLogger()
	logger.Info("Setting up HTTP server on:", v.addr)

	cfg := web.DefaultFastHTTPServerConfig(v.addr)
	v.server = web.NewFastHTTPServer(gocmd, cfg)

	// Setup routes on the server's router
	r := v.server.FastRouter()

	r.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]any{"status": "ok"})
	})

	r.GETFast("/hello", func(c *web.FastRequestContext) error {
		return c.Text(200, "/hello")
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

	// Start HTTP server - blocking operation, but framework calls Start() in goroutine
	logger.Info("Starting HTTP server on:", v.addr)
	if err := v.server.Start(); err != nil {
		logger.Error("HTTP server failed to start:", err)
		return err
	}

	return nil
}

// Stop overrides BaseVerticle.Stop - single entry point for cleanup
func (v *ApiGatewayVerticle) Stop(ctx core.FluxorContext) error {
	// Call base Stop first
	if err := v.BaseVerticle.Stop(ctx); err != nil {
		return err
	}

	// Stop HTTP server
	if v.server != nil {
		logger := core.NewDefaultLogger()
		logger.Info("Stopping HTTP server")
		if err := v.server.Stop(); err != nil {
			logger.Error("HTTP server stop error:", err)
			return err
		}
		logger.Info("HTTP server stopped successfully")
	}

	return nil
}
