package core

import (
	"encoding/json"
	"fmt"
)

// JSON is a convenience alias for JSON objects (Dev UX).
type JSON map[string]any

// JSONEncode encodes a value to JSON bytes (fail-fast).
// Uses standard encoding/json for JSON encoding.
// Note: Previously used Sonic for better performance, but switched to stdlib
// for Go 1.24 compatibility. Will switch back when Sonic supports Go 1.24.
func JSONEncode(v interface{}) ([]byte, error) {
	// Fail-fast: validate input
	if v == nil {
		return nil, &EventBusError{Code: "INVALID_INPUT", Message: "cannot encode nil value"}
	}

	// Use standard library json.Marshal
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}

	return data, nil
}

// JSONDecode decodes JSON bytes to a value (fail-fast).
// Uses standard encoding/json for JSON decoding.
// Note: Previously used Sonic for better performance, but switched to stdlib
// for Go 1.24 compatibility. Will switch back when Sonic supports Go 1.24.
func JSONDecode(data []byte, v interface{}) error {
	// Fail-fast: validate inputs
	if len(data) == 0 {
		return &EventBusError{Code: "INVALID_INPUT", Message: "cannot decode empty data"}
	}
	if v == nil {
		return &EventBusError{Code: "INVALID_INPUT", Message: "cannot decode into nil value"}
	}

	// Use standard library json.Unmarshal
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}
