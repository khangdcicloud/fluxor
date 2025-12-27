package ai

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid OpenAI config with API key",
			config: Config{
				Provider: ProviderOpenAI,
				APIKey:   "test-key",
				Model:    "gpt-3.5-turbo",
			},
			wantErr: false,
		},
		{
			name: "valid Anthropic config",
			config: Config{
				Provider: ProviderAnthropic,
				APIKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
			},
			wantErr: false,
		},
		{
			name: "valid Cursor config",
			config: Config{
				Provider: ProviderCursor,
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "default provider when empty",
			config: Config{
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Errorf("NewClient() returned nil client")
			}
			if !tt.wantErr && client != nil {
				if client.Provider() != tt.config.Provider && tt.config.Provider != "" {
					t.Errorf("NewClient() provider = %v, want %v", client.Provider(), tt.config.Provider)
				}
			}
		})
	}
}

func TestAIClient_ChatRequest(t *testing.T) {
	// This test requires a real API key, so we'll skip it in CI
	// In a real scenario, you'd use a mock HTTP client
	t.Skip("Skipping test that requires real API key")

	config := Config{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-3.5-turbo",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
	}

	_, err = client.Chat(context.Background(), req)
	if err != nil {
		t.Errorf("Chat() error = %v", err)
	}
}

func TestAIClient_ChatRequest_Validation(t *testing.T) {
	config := Config{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	tests := []struct {
		name    string
		req     *ChatRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty messages",
			req: &ChatRequest{
				Messages: []Message{},
			},
			wantErr: true,
		},
		{
			name: "valid request",
			req: &ChatRequest{
				Messages: []Message{
					{
						Role:    "user",
						Content: "Hello",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Chat(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Chat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProviderBaseURL(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderOpenAI, "https://api.openai.com/v1"},
		{ProviderCursor, "https://api.openai.com/v1"},
		{ProviderAnthropic, "https://api.anthropic.com/v1"},
		{ProviderCustom, ""},
		{Provider("unknown"), "https://api.openai.com/v1"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := getProviderBaseURL(tt.provider)
			if got != tt.want {
				t.Errorf("getProviderBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetProviderDefaultModel(t *testing.T) {
	tests := []struct {
		provider Provider
		want     string
	}{
		{ProviderOpenAI, "gpt-3.5-turbo"},
		{ProviderCursor, "gpt-4"},
		{ProviderAnthropic, "claude-3-sonnet-20240229"},
		{Provider("unknown"), "gpt-3.5-turbo"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			got := getProviderDefaultModel(tt.provider)
			if got != tt.want {
				t.Errorf("getProviderDefaultModel() = %v, want %v", got, tt.want)
			}
		})
	}
}
