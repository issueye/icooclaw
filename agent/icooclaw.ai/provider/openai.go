package provider

import (
	"context"
)

// ============ OpenAI Provider ============

// OpenAIProvider OpenAI Provider 实现
type OpenAIProvider struct {
	*BaseProvider
}

// NewOpenAIProvider 创建OpenAI Provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	apiBase := "https://api.openai.com/v1"
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider("openai", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// OpenAI 特定请求头
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// OpenAI 特定请求头
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	return p.sendStreamRequest(ctx, httpReq, callback)
}
