package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/storage"
	"net/http"
)

// OpenRouterProvider implements Provider for OpenRouter.
type OpenRouterProvider struct {
	*BaseProvider
	siteURL  string
	siteName string
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(cfg *storage.Provider) *OpenRouterProvider {
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://openrouter.ai/api/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "openai/gpt-4o"
	}

	return &OpenRouterProvider{
		BaseProvider: NewBaseProvider("openrouter", cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request via OpenRouter.
func (p *OpenRouterProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	headers := map[string]string{
		"Authorization": "Bearer " + p.apiKey,
	}
	if p.siteURL != "" {
		headers["HTTP-Referer"] = p.siteURL
	}
	if p.siteName != "" {
		headers["X-Title"] = p.siteName
	}

	resp, err := p.doRequestWithHeaders(ctx, "POST", "/chat/completions", req, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &ChatResponse{
		ID:        result.ID,
		Model:     result.Model,
		Content:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request via OpenRouter.
func (p *OpenRouterProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true

	headers := map[string]string{
		"Authorization": "Bearer " + p.apiKey,
	}
	if p.siteURL != "" {
		headers["HTTP-Referer"] = p.siteURL
	}
	if p.siteName != "" {
		headers["X-Title"] = p.siteName
	}

	resp, err := p.doRequestWithHeaders(ctx, "POST", "/chat/completions", req, headers)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}
