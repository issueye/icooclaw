package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
	"net/http"
)

// QwenProvider implements Provider for Alibaba Qwen (通义千问).
type QwenProvider struct {
	*BaseProvider
}

// NewQwenProvider creates a new Qwen provider.
func NewQwenProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderQwen
	apiBase := cfg.APIBase
	// 处理默认值
	if apiBase == "" {
		apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// 处理编码计划
	if cfg.Type == consts.ProviderQwenCodingPlan {
		apiBase = "https://coding.dashscope.aliyuncs.com/v1"
		providerName = consts.ProviderQwenCodingPlan
	}

	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "qwen-plus"
	}

	return &QwenProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Qwen.
func (p *QwenProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
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
				Reasoning string     `json:"reasoning_content"`
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
		Reasoning: result.Choices[0].Message.Reasoning,
		ToolCalls: result.Choices[0].Message.ToolCalls,
		Usage:     result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Qwen.
func (p *QwenProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	resp, err := p.doRequest(ctx, "POST", "/chat/completions", req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}
