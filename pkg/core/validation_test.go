package core

import (
	"testing"
	"time"
)

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", "test.address", false},
		{"empty address", "", true},
		{"long address", string(make([]byte, 256)), true},
		{"normal address", "api.users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"valid timeout", 5 * time.Second, false},
		{"zero timeout", 0, true},
		{"negative timeout", -1 * time.Second, true},
		{"too large timeout", 10 * time.Minute, true},
		{"max valid timeout", 5 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBody(t *testing.T) {
	tests := []struct {
		name    string
		body    interface{}
		wantErr bool
	}{
		{"valid body", "test", false},
		{"nil body", nil, true},
		{"map body", map[string]string{"key": "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBody(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailFast(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("FailFast() should panic")
		}
	}()

	FailFast(&EventBusError{Code: "TEST", Message: "test error"})
}
