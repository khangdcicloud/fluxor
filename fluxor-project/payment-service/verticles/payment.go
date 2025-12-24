package verticles

import (
	"github.com/fluxorio/fluxor/pkg/core"

	"github.com/quadgatefoundation/fluxor/fluxor-project/common/contracts"
)

type PaymentVerticle struct{}

func NewPaymentVerticle() *PaymentVerticle { return &PaymentVerticle{} }

func (v *PaymentVerticle) Start(ctx core.FluxorContext) error {
	bus := ctx.EventBus()

	// Example: Request/Reply from api-gateway.
	bus.Consumer(contracts.AddressPaymentsAuthorize).Handler(func(c core.FluxorContext, msg core.Message) error {
		body, ok := msg.Body().([]byte)
		if !ok {
			_ = bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "invalid payload type"})
			return msg.Reply(contracts.PaymentAuthorizeReply{OK: false, Error: "invalid_request"})
		}

		var req contracts.PaymentAuthorizeRequest
		if err := core.JSONDecode(body, &req); err != nil {
			_ = bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "invalid json"})
			return msg.Reply(contracts.PaymentAuthorizeReply{OK: false, Error: "invalid_request"})
		}

		// Simulate authorization.
		_ = bus.Publish(contracts.AddressLogs, contracts.LogEvent{Service: "payment-service", Message: "authorized " + req.PaymentID})
		return msg.Reply(contracts.PaymentAuthorizeReply{OK: true, AuthID: "auth_" + req.PaymentID})
	})
	return nil
}

func (v *PaymentVerticle) Stop(ctx core.FluxorContext) error { return nil }
