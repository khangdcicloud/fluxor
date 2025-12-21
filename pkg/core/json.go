package core

import (
	"fmt"

	"github.com/bytedance/sonic"
)

// JSONEncode encodes a value to JSON bytes using Sonic (fail-fast)
// Uses Sonic (bytedance/sonic) for high-performance JSON encoding
// Sonic is significantly faster than the standard library's json.Marshal
// It uses JIT compilation and SIMD optimizations for better performance
func JSONEncode(v interface{}) ([]byte, error) {
	// Fail-fast: validate input
	if v == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cannot encode nil value"}
	}

	// Use Sonic Marshal (much faster than standard library)
	// Sonic internally handles pooling and optimizations
	data, err := sonic.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}

	return data, nil
}

// JSONDecode decodes JSON bytes to a value using Sonic (fail-fast)
// Uses Sonic (bytedance/sonic) for high-performance JSON decoding
// Sonic is significantly faster than the standard library's json.Unmarshal
// It uses JIT compilation and SIMD optimizations for better performance
func JSONDecode(data []byte, v interface{}) error {
	// Fail-fast: validate inputs
	if len(data) == 0 {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode empty data"}
	}
	if v == nil {
		return &Error{Code: "INVALID_INPUT", Message: "cannot decode into nil value"}
	}

	// Use Sonic Unmarshal (much faster than standard library)
	// Sonic internally handles optimizations
	if err := sonic.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}
