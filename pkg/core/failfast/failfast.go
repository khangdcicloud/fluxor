package failfast

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

// Err panics if err != nil (fail-fast principle)
// Includes stack trace for debugging
func Err(err error) {
	if err != nil {
		panic(fmt.Errorf("fail-fast: %w\n%s", err, debug.Stack()))
	}
}

// If panics if condition is false
// Allows formatted messages with args
func If(condition bool, message string, args ...interface{}) {
	if !condition {
		panic(fmt.Errorf("fail-fast: "+message, args...))
	}
}

// NotNil panics if ptr is nil
// Useful for validating required pointers/values
// Handles both untyped nil and typed nil pointers correctly
func NotNil(ptr interface{}, name string) {
	if ptr == nil {
		panic(fmt.Errorf("fail-fast: %s is nil", name))
	}
	// Check for typed nil pointers (e.g., *string that is nil)
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		panic(fmt.Errorf("fail-fast: %s is nil", name))
	}
}
