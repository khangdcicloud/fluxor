package core

import (
	"encoding/json"
	"testing"
)

func TestJSONEncode(t *testing.T) {
	tests := []struct {
		name    string
		v       interface{}
		wantErr bool
	}{
		{"valid map", map[string]string{"key": "value"}, false},
		{"valid string", "test", false},
		{"nil value", nil, true},
		{"valid struct", struct{ Name string }{"test"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := JSONEncode(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONEncode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONEncode_Pooling(t *testing.T) {
	// Test that encoding works correctly with Sonic
	data1 := map[string]string{"key1": "value1"}
	data2 := map[string]int{"key2": 42}

	encoded1, err := JSONEncode(data1)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	encoded2, err := JSONEncode(data2)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	// Verify both encodings are correct using standard json for compatibility check
	var decoded1 map[string]string
	if err := json.Unmarshal(encoded1, &decoded1); err != nil {
		t.Errorf("Failed to decode encoded1: %v", err)
	}
	if decoded1["key1"] != "value1" {
		t.Errorf("decoded1 = %v, want map[key1:value1]", decoded1)
	}

	var decoded2 map[string]int
	if err := json.Unmarshal(encoded2, &decoded2); err != nil {
		t.Errorf("Failed to decode encoded2: %v", err)
	}
	if decoded2["key2"] != 42 {
		t.Errorf("decoded2 = %v, want map[key2:42]", decoded2)
	}
}

func TestJSONDecode(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		v       interface{}
		wantErr bool
	}{
		{"valid json", []byte(`{"key":"value"}`), &map[string]string{}, false},
		{"empty data", []byte{}, &map[string]string{}, true},
		{"nil target", []byte(`{"key":"value"}`), nil, true},
		{"invalid json", []byte(`{invalid}`), &map[string]string{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONDecode(tt.data, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSONDecode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONEncodeDecode(t *testing.T) {
	original := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"nested": map[string]string{
			"key": "value",
		},
	}

	encoded, err := JSONEncode(original)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	var decoded map[string]interface{}
	err = JSONDecode(encoded, &decoded)
	if err != nil {
		t.Fatalf("JSONDecode() error = %v", err)
	}

	if decoded["name"] != original["name"] {
		t.Errorf("decoded name = %v, want %v", decoded["name"], original["name"])
	}
}
