package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// Loader loads configuration from various sources
type Loader interface {
	Load(path string, target interface{}) error
}

// Manager manages configuration with validation and environment variable support
type Manager struct {
	config     interface{}
	validators []Validator
}

// Validator validates configuration
type Validator interface {
	Validate(config interface{}) error
}

// ValidatorFunc is a function that validates configuration
type ValidatorFunc func(config interface{}) error

func (f ValidatorFunc) Validate(config interface{}) error {
	return f(config)
}

// Load loads configuration from a file (YAML or JSON)
// Automatically detects file type by extension
func Load(path string, target interface{}) error {
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return LoadYAML(path, target)
	}
	if strings.HasSuffix(path, ".json") {
		return LoadJSON(path, target)
	}
	// Default to YAML
	return LoadYAML(path, target)
}

// LoadWithEnv loads configuration from file and applies environment variable overrides
// Environment variables use format: PREFIX_FIELD_SUBFIELD (e.g., APP_DATABASE_DSN)
func LoadWithEnv(path string, prefix string, target interface{}) error {
	// Load from file first
	if err := Load(path, target); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	// Apply environment variable overrides
	if err := ApplyEnvOverrides(prefix, target); err != nil {
		return fmt.Errorf("failed to apply env overrides: %w", err)
	}

	return nil
}

// ApplyEnvOverrides applies environment variable overrides to configuration struct
// Uses reflection to set struct fields from environment variables
func ApplyEnvOverrides(prefix string, target interface{}) error {
	if prefix == "" {
		prefix = "APP"
	}

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	return applyEnvToStruct(prefix, val.Elem())
}

// applyEnvToStruct recursively applies environment variables to struct fields
func applyEnvToStruct(prefix string, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Build environment variable name: PREFIX_FIELDNAME
		envKey := prefix + "_" + strings.ToUpper(fieldType.Name)
		envKey = strings.ReplaceAll(envKey, "-", "_")

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := applyEnvToStruct(envKey, field); err != nil {
				return err
			}
			continue
		}

		// Handle pointers to structs
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			if err := applyEnvToStruct(envKey, field.Elem()); err != nil {
				return err
			}
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(envKey)
		if envValue == "" {
			continue // No override for this field
		}

		// Set field value based on type
		if err := setFieldFromEnv(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, envKey, err)
		}
	}

	return nil
}

// setFieldFromEnv sets a struct field value from environment variable string
func setFieldFromEnv(field reflect.Value, envValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(envValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		if _, err := fmt.Sscanf(envValue, "%d", &intVal); err != nil {
			return fmt.Errorf("invalid integer value: %s", envValue)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		if _, err := fmt.Sscanf(envValue, "%d", &uintVal); err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", envValue)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		var floatVal float64
		if _, err := fmt.Sscanf(envValue, "%f", &floatVal); err != nil {
			return fmt.Errorf("invalid float value: %s", envValue)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal := strings.ToLower(envValue) == "true" || envValue == "1"
		field.SetBool(boolVal)
	case reflect.Slice:
		// For slices, split by comma
		parts := strings.Split(envValue, ",")
		sliceType := field.Type().Elem()
		slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
		for i, part := range parts {
			part = strings.TrimSpace(part)
			elem := reflect.New(sliceType).Elem()
			if err := setFieldFromEnv(elem, part); err != nil {
				return err
			}
			slice.Index(i).Set(elem)
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// Validate validates configuration using registered validators
func Validate(config interface{}, validators ...Validator) error {
	for _, validator := range validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}
	return nil
}

// NewManager creates a new configuration manager
func NewManager(config interface{}) *Manager {
	return &Manager{
		config:     config,
		validators: make([]Validator, 0),
	}
}

// AddValidator adds a validator to the manager
func (m *Manager) AddValidator(validator Validator) {
	m.validators = append(m.validators, validator)
}

// Validate validates the configuration
func (m *Manager) Validate() error {
	return Validate(m.config, m.validators...)
}

// Get returns the configuration
func (m *Manager) Get() interface{} {
	return m.config
}

// GetTyped returns the configuration as the specified type
func GetTyped[T any](config interface{}) (T, error) {
	var zero T
	val, ok := config.(T)
	if !ok {
		return zero, fmt.Errorf("configuration type mismatch: expected %T, got %T", zero, config)
	}
	return val, nil
}

// MustGetTyped returns the configuration as the specified type, panics on error
func MustGetTyped[T any](config interface{}) T {
	val, err := GetTyped[T](config)
	if err != nil {
		panic(err)
	}
	return val
}

