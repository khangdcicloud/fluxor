package verticles

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"

	"github.com/quadgatefoundation/fluxor/examples/fluxor-project/common/contracts"
)

// PaymentVerticle is both an HTTP server and EventBus consumer
// It's a process using context from FluxorContext
type PaymentVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	server             *web.FastHTTPServer
	bus                core.EventBus
	addr               string
}

func NewPaymentVerticle() *PaymentVerticle {
	return &PaymentVerticle{
		BaseVerticle: core.NewBaseVerticle("payment-service"),
	}
}

// Start overrides BaseVerticle.Start - single entry point for initialization
// Setup HTTP server and EventBus consumer
func (v *PaymentVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Payment verticle is a process using context from FluxorContext
	// All setup uses the context provided by the framework
	v.bus = ctx.EventBus()
	gocmd := ctx.GoCMD()

	// Print config from context
	config := ctx.Config()
	configJSON, err := core.JSONEncode(config)
	if err != nil {
		core.NewDefaultLogger().Error("Failed to marshal config:", err)
	} else {
		core.NewDefaultLogger().Info("Payment Service Config:", string(configJSON))
	}

	// Setup HTTP server address from context config
	v.addr = "127.0.0.1:8081"
	if val, ok := ctx.Config()["payment_http_addr"].(string); ok && val != "" {
		v.addr = val
	} else if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" && val != ":8080" {
		// Only use shared http_addr if it's not the API Gateway port
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
		return c.JSON(200, map[string]any{"status": "ok", "service": "payment-service"})
	})

	// Setup EventBus consumer using context's EventBus
	// Consumer is started when verticle starts - it listens for payment authorization requests
	logger.Info("Starting EventBus consumer on address:", contracts.AddressPaymentsAuthorize)

	consumer := v.bus.Consumer(contracts.AddressPaymentsAuthorize)
	consumer.Handler(func(c core.FluxorContext, msg core.Message) error {
		body, ok := msg.Body().([]byte)
		if !ok {
			_ = v.bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "invalid payload type"})
			return msg.Reply(contracts.PaymentAuthorizeReply{OK: false, Error: "invalid_request"})
		}

		var req contracts.PaymentAuthorizeRequest
		if err := core.JSONDecode(body, &req); err != nil {
			_ = v.bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "invalid json"})
			return msg.Reply(contracts.PaymentAuthorizeReply{OK: false, Error: "invalid_request"})
		}

		// Simulate authorization.
		_ = v.bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "authorized " + req.PaymentID})
		return msg.Reply(contracts.PaymentAuthorizeReply{OK: true, AuthID: "auth_" + req.PaymentID})
	})

	logger.Info("EventBus consumer started successfully on address:", contracts.AddressPaymentsAuthorize)

	// Start HTTP server - blocking operation, but framework calls Start() in goroutine
	logger.Info("Starting HTTP server on:", v.addr)
	if err := v.server.Start(); err != nil {
		logger.Error("HTTP server failed to start:", err)
		return err
	}

	return nil
}

// Stop overrides BaseVerticle.Stop - single entry point for cleanup
func (v *PaymentVerticle) Stop(ctx core.FluxorContext) error {
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
