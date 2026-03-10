package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
	"net/http"
	"strings"
)

// AnthropicProvider implements Provider for Anthropic Claude.
type AnthropicProvider struct {
	*BaseProvider
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderAnthropic
	apiBase := cfg.APIBase
	// 处理默认值
	if apiBase == "" {
		apiBase = "https://api.anthropic.com/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "claude-3-5-sonnet-20241022"
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Anthropic.
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert messages to Anthropic format
	messages := make([]map[string]any, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		messages = append(messages, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	anthropicReq := map[string]any{
		"model":      req.Model,
		"max_tokens": 4096,
		"messages":   messages,
	}

	if systemPrompt != "" {
		anthropicReq["system"] = systemPrompt
	}

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	}

	// Convert tools
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Function.Name,
				"description":  t.Function.Description,
				"input_schema": t.Function.Parameters,
			})
		}
		anthropicReq["tools"] = tools
	}

	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := p.doRequestWithHeaders(ctx, "POST", "/messages", anthropicReq, headers)
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
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var content string
	for _, c := range result.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &ChatResponse{
		ID:      result.ID,
		Model:   result.Model,
		Content: content,
		Usage: Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}, nil
}

// ChatStream sends a streaming chat request to Anthropic.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	// Convert messages to Anthropic format
	messages := make([]map[string]any, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}
		messages = append(messages, map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	anthropicReq := map[string]any{
		"model":      req.Model,
		"max_tokens": 4096,
		"messages":   messages,
		"stream":     true,
	}

	if systemPrompt != "" {
		anthropicReq["system"] = systemPrompt
	}

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	}

	headers := map[string]string{
		"x-api-key":         p.apiKey,
		"anthropic-version": "2023-06-01",
	}

	resp, err := p.doRequestWithHeaders(ctx, "POST", "/messages", anthropicReq, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Message struct {
				StopReason string `json:"stop_reason"`
			} `json:"message"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				if err := callback(event.Delta.Text, "", nil, false); err != nil {
					return err
				}
			}
		case "message_stop":
			if err := callback("", "", nil, true); err != nil {
				return err
			}
			return nil
		}
	}

	return scanner.Err()
}
