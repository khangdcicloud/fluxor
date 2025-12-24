package contracts

// Contracts are shared message types for EventBus payloads.
// Keep these stable to avoid coupling services too tightly.

type LogEvent struct {
	Service string `json:"service"`
	Message string `json:"message"`
}

