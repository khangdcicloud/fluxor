package core

import (
	"encoding/json"
	"fmt"
)

// JSONEncode encodes a value to JSON bytes (fail-fast)
// Uses the standard library's json.Marshal for reliable JSON encoding
// For production deployments requiring higher performance, consider
// using bytedance/sonic when Go 1.24 support is available
func JSONEncode(v interface{}) ([]byte, error) {
	// Fail-fast: validate input
	if v == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cannot encode nil value"}
	}

	// Use standard library Marshal
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}

	return data, nil
}

// JSONDecode decodes JSON bytes to a value (fail-fast)
// Uses the standard library's json.Unmarshal for reliable JSON decoding
// For production deployments requiring higher performance, consider
// using bytedance/sonic when Go 1.24 support is available
func JSONDecode(data []byte, v interface{}) error {
	// Fail-fast: validate inputs
	if len(data) == 0 {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode empty data"}
	}
	if v == nil {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode into nil value"}
	}

	// Use standard library Unmarshal
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}
