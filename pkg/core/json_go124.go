//go:build go1.24
// +build go1.24

package core

import (
	"encoding/json"
	"fmt"
)

// JSONEncode encodes a value to JSON bytes (fail-fast).
//
// Go 1.24+: Sonic's JIT loader is not compatible with the Go runtime ABI changes,
// so we use the standard library as a safe fallback.
func JSONEncode(v interface{}) ([]byte, error) {
	// Fail-fast: validate input
	if v == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cannot encode nil value"}
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}
	return data, nil
}

// JSONDecode decodes JSON bytes to a value (fail-fast).
//
// Go 1.24+: use standard library decoder (see JSONEncode note).
func JSONDecode(data []byte, v interface{}) error {
	// Fail-fast: validate inputs
	if len(data) == 0 {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode empty data"}
	}
	if v == nil {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode into nil value"}
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}

