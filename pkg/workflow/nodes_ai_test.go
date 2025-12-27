package workflow

import (
	"context"
	"testing"
)

func TestAINodeHandler_ProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI default", "openai", "https://api.openai.com/v1"},
		{"Cursor provider", "cursor", "https://api.openai.com/v1"},
		{"Anthropic provider", "anthropic", "https://api.anthropic.com/v1"},
		{"Unknown provider", "unknown", "https://api.openai.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := getProviderBaseURL(tt.provider)
			if baseURL != tt.expected {
				t.Errorf("Expected baseURL %q, got %q", tt.expected, baseURL)
			}
		})
	}
}

func TestAINodeHandler_DefaultModels(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI default model", "openai", "gpt-3.5-turbo"},
		{"Cursor default model", "cursor", "gpt-4"},
		{"Anthropic default model", "anthropic", "claude-3-sonnet-20240229"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := getProviderDefaultModel(tt.provider)
			if model != tt.expected {
				t.Errorf("Expected model %q, got %q", tt.expected, model)
			}
		})
	}
}

func TestAINodeHandler_EnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI env var", "openai", "OPENAI_API_KEY"},
		{"Cursor env var", "cursor", "CURSOR_API_KEY"},
		{"Anthropic env var", "anthropic", "ANTHROPIC_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVar := getProviderEnvVar(tt.provider)
			if envVar != tt.expected {
				t.Errorf("Expected env var %q, got %q", tt.expected, envVar)
			}
		})
	}
}

func TestAINodeHandler_ConfigValidation(t *testing.T) {
	// Test that handler validates required config
	input := &NodeInput{
		Config: map[string]interface{}{
			"provider": "cursor",
		},
		Data: map[string]interface{}{"prompt": "test"},
	}

	_, err := AINodeHandler(context.Background(), input)
	if err == nil {
		t.Error("Expected error when apiKey is missing")
	}
}

func TestAINodeHandler_EndpointSelection(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		expected string
	}{
		{"OpenAI chat model", "openai", "gpt-4", "/chat/completions"},
		{"OpenAI text model", "openai", "text-davinci-003", "/completions"},
		{"Cursor chat model", "cursor", "gpt-4", "/chat/completions"},
		{"Anthropic messages", "anthropic", "claude-3", "/messages"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := getProviderEndpoint(tt.provider, tt.model)
			if endpoint != tt.expected {
				t.Errorf("Expected endpoint %q, got %q", tt.expected, endpoint)
			}
		})
	}
}

