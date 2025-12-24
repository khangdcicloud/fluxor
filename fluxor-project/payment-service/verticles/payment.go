package verticles

import (
	"github.com/fluxorio/fluxor/pkg/core"

	"github.com/quadgatefoundation/fluxor/fluxor-project/common/contracts"
)

type PaymentVerticle struct{}

func NewPaymentVerticle() *PaymentVerticle { return &PaymentVerticle{} }

func (v *PaymentVerticle) Start(ctx core.FluxorContext) error {
	// Example: consume events
	ctx.EventBus().Consumer("payments.create").Handler(func(c core.FluxorContext, msg core.Message) error {
		_ = ctx.EventBus().Publish("logs", contracts.LogEvent{Service: "payment-service", Message: "payment created"})
		return msg.Reply(map[string]any{"ok": true})
	})
	return nil
}

func (v *PaymentVerticle) Stop(ctx core.FluxorContext) error { return nil }
