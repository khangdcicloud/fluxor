package config

import (
	"fmt"
	"reflect"
	"strings"
)

// RequiredFields validates that required fields are not empty
func RequiredFields(fields ...string) Validator {
	return ValidatorFunc(func(config interface{}) error {
		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if val.Kind() != reflect.Struct {
			return fmt.Errorf("config must be a struct")
		}

		missing := make([]string, 0)

		for _, fieldName := range fields {
			// Support nested field paths
			fieldVal := getNestedField(val, fieldName)
			if !fieldVal.IsValid() {
				return fmt.Errorf("field %s not found in config struct", fieldName)
			}

			if isEmpty(fieldVal) {
				missing = append(missing, fieldName)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("required fields are missing: %s", strings.Join(missing, ", "))
		}

		return nil
	})
}

// isEmpty checks if a reflect.Value is empty (zero value)
func isEmpty(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return val.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return val.IsNil()
	default:
		return false
	}
}

// RangeValidator validates that a numeric field is within a range
// Supports nested fields using dot notation (e.g., "Database.MaxConns")
func RangeValidator(fieldName string, min, max float64) Validator {
	return ValidatorFunc(func(config interface{}) error {
		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Support nested field paths (e.g., "Database.MaxConns")
		fieldVal := getNestedField(val, fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		var numVal float64
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			numVal = float64(fieldVal.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			numVal = float64(fieldVal.Uint())
		case reflect.Float32, reflect.Float64:
			numVal = fieldVal.Float()
		default:
			return fmt.Errorf("field %s is not numeric", fieldName)
		}

		if numVal < min || numVal > max {
			return fmt.Errorf("field %s value %f is out of range [%f, %f]", fieldName, numVal, min, max)
		}

		return nil
	})
}

// getNestedField gets a field value, supporting nested paths with dot notation
func getNestedField(val reflect.Value, fieldPath string) reflect.Value {
	parts := strings.Split(fieldPath, ".")
	current := val

	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		current = current.FieldByName(part)
		if !current.IsValid() {
			return reflect.Value{}
		}
	}
	return current
}

// StringLengthValidator validates that a string field has a specific length range
// Supports nested fields using dot notation
func StringLengthValidator(fieldName string, minLen, maxLen int) Validator {
	return ValidatorFunc(func(config interface{}) error {
		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		fieldVal := getNestedField(val, fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		if fieldVal.Kind() != reflect.String {
			return fmt.Errorf("field %s is not a string", fieldName)
		}

		strVal := fieldVal.String()
		length := len(strVal)

		if length < minLen || length > maxLen {
			return fmt.Errorf("field %s length %d is out of range [%d, %d]", fieldName, length, minLen, maxLen)
		}

		return nil
	})
}

// OneOfValidator validates that a field value is one of the allowed values
func OneOfValidator(fieldName string, allowedValues ...interface{}) Validator {
	return ValidatorFunc(func(config interface{}) error {
		val := reflect.ValueOf(config)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		fieldVal := val.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			return fmt.Errorf("field %s not found", fieldName)
		}

		fieldInterface := fieldVal.Interface()

		for _, allowed := range allowedValues {
			if reflect.DeepEqual(fieldInterface, allowed) {
				return nil
			}
		}

		return fmt.Errorf("field %s value %v is not one of allowed values: %v", fieldName, fieldInterface, allowedValues)
	})
}
