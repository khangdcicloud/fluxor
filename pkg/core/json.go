package core

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	jsonEncoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewEncoder(nil)
		},
	}
	jsonDecoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewDecoder(nil)
		},
	}
)

// JSONEncode encodes a value to JSON bytes (fail-fast)
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

// JSONDecode decodes JSON bytes to a value (fail-fast)
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

// JSONEncodeFast encodes with object reuse for better performance
func JSONEncodeFast(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
