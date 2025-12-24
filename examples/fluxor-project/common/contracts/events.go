package contracts

// Contracts are shared message types for EventBus payloads.
// Keep these stable to avoid coupling services too tightly.

type LogEvent struct {
	Service string `json:"service"`
	Message string `json:"message"`
}

// EventBus addresses shared by services.
const (
	AddressLogs              = "logs"
	AddressPaymentsAuthorize = "payments.authorize"
)

// PaymentAuthorizeRequest is sent by api-gateway to payment-service.
type PaymentAuthorizeRequest struct {
	PaymentID string `json:"paymentId"`
	UserID    string `json:"userId"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

// PaymentAuthorizeReply is returned by payment-service.
type PaymentAuthorizeReply struct {
	OK     bool   `json:"ok"`
	AuthID string `json:"authId,omitempty"`
	Error  string `json:"error,omitempty"`
}
