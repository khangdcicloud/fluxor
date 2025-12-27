package failfast

import (
	"errors"
	"testing"
)

func TestErr(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		Err(nil)
	})

	t.Run("with error", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error() == "" {
				t.Error("Expected error message")
			}
		}()
		Err(errors.New("test error"))
	})
}

func TestIf(t *testing.T) {
	t.Run("condition true", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		If(true, "should not panic")
	})

	t.Run("condition false", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error() == "" {
				t.Error("Expected error message")
			}
		}()
		If(false, "test message")
	})

	t.Run("formatted message", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			expected := "fail-fast: value is 42"
			if err.Error() != expected {
				t.Errorf("Expected %q, got %q", expected, err.Error())
			}
		}()
		If(false, "value is %d", 42)
	})
}

func TestNotNil(t *testing.T) {
	t.Run("not nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		val := "test"
		NotNil(&val, "val")
	})

	t.Run("nil pointer", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			expected := "fail-fast: ptr is nil"
			if err.Error() != expected {
				t.Errorf("Expected %q, got %q", expected, err.Error())
			}
		}()
		var ptr *string
		NotNil(ptr, "ptr")
	})

	t.Run("nil interface", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
		}()
		var val interface{}
		NotNil(val, "val")
	})
}
