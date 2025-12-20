package core

import (
	"fmt"
	"time"
)

// ValidateAddress validates an event bus address
func ValidateAddress(address string) error {
	if address == "" {
		return &Error{Code: "INVALID_ADDRESS", Message: "address cannot be empty"}
	}
	if len(address) > 255 {
		return &Error{Code: "INVALID_ADDRESS", Message: "address too long (max 255 characters)"}
	}
	return nil
}

// ValidateTimeout validates a timeout duration
func ValidateTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return &Error{Code: "INVALID_TIMEOUT", Message: "timeout must be positive"}
	}
	if timeout > 5*time.Minute {
		return &Error{Code: "INVALID_TIMEOUT", Message: "timeout too large (max 5 minutes)"}
	}
	return nil
}

// ValidateVerticle validates a verticle before deployment
func ValidateVerticle(verticle Verticle) error {
	if verticle == nil {
		return &Error{Code: "INVALID_VERTICLE", Message: "verticle cannot be nil"}
	}
	return nil
}

// ValidateBody validates a message body
func ValidateBody(body interface{}) error {
	if body == nil {
		return &Error{Code: "INVALID_BODY", Message: "body cannot be nil"}
	}
	return nil
}

// FailFast panics with an error (fail-fast principle)
func FailFast(err error) {
	if err != nil {
		panic(fmt.Errorf("fail-fast: %w", err))
	}
}

// FailFastIf panics if condition is true
func FailFastIf(condition bool, message string) {
	if condition {
		panic(fmt.Errorf("fail-fast: %s", message))
	}
}
