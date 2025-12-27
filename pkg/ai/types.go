package ai

import (
	"context"
)

// Provider represents an AI provider type
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderCursor    Provider = "cursor"
	ProviderCustom    Provider = "custom"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // Message content
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`                 // Model name
	Messages    []Message `json:"messages"`              // Chat messages
	Temperature float64   `json:"temperature,omitempty"` // Temperature (0-2)
	MaxTokens   int       `json:"max_tokens,omitempty"`  // Max tokens
	TopP        float64   `json:"top_p,omitempty"`       // Top P
	Stream      bool      `json:"stream,omitempty"`      // Stream response
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Client is the interface for AI clients
type Client interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatSimple sends a simple chat message and returns the response text
	ChatSimple(ctx context.Context, prompt string) (string, error)

	// Provider returns the provider type
	Provider() Provider
}

// Config represents AI client configuration
type Config struct {
	Provider Provider `json:"provider"` // AI provider
	APIKey   string   `json:"apiKey"`   // API key (or use env var)
	BaseURL  string   `json:"baseURL"`  // Base URL (optional, provider-specific defaults)
	Model    string   `json:"model"`    // Default model
	Timeout  string   `json:"timeout"`  // Request timeout (default: 60s)
}
