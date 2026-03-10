// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"context"
)

// ChatMessage represents a message in a chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Tools       []Tool        `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// Tool represents a tool definition.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition.
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall represents a tool call in a response.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	ID        string     `json:"id"`
	Model     string     `json:"model"`
	Content   string     `json:"content,omitempty"`
	Reasoning string     `json:"reasoning,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage,omitempty"`
}

// Usage represents token usage.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamCallback is called for each chunk in a streaming response.
type StreamCallback func(chunk string, reasoning string, toolCalls []ToolCall, done bool) error

// Provider is the interface for LLM providers.
type Provider interface {
	// Chat sends a chat request and returns the response.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream sends a chat request and streams the response.
	ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error

	// GetDefaultModel returns the default model for this provider.
	GetDefaultModel() string

	// GetName returns the provider name.
	GetName() string
}

// ProviderConfig represents provider configuration.
type ProviderConfig struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	APIKey       string         `json:"api_key"`
	APIBase      string         `json:"api_base"`
	DefaultModel string         `json:"default_model"`
	Models       []string       `json:"models"`
	Config       map[string]any `json:"config"`
}
