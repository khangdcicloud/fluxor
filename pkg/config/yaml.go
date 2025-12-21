package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML loads configuration from a YAML file
func LoadYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read YAML file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return nil
}

// SaveYAML saves configuration to a YAML file
func SaveYAML(path string, config interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}

