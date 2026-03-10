package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
	"net/http"
)

// SiliconFlowProvider implements Provider for SiliconFlow.
type SiliconFlowProvider struct {
	*BaseProvider
}

// NewSiliconFlowProvider creates a new SiliconFlow provider.
func NewSiliconFlowProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderSiliconFlow
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.siliconflow.cn/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "Qwen/Qwen2.5-7B-Instruct"
	}

	return &SiliconFlowProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to SiliconFlow.
func (p *SiliconFlowProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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

// ChatStream sends a streaming chat request to SiliconFlow.
func (p *SiliconFlowProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
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
