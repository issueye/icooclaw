package provider

import "context"

// ============ DeepSeek Provider ============

// DeepSeekProvider DeepSeek Provider 实现
type DeepSeekProvider struct {
	*BaseProvider
}

// NewDeepSeekProvider 创建DeepSeek Provider
func NewDeepSeekProvider(apiKey, model string) *DeepSeekProvider {
	apiBase := "https://api.deepseek.com/v1"
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekProvider{
		BaseProvider: NewBaseProvider("deepseek", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法 - DeepSeek 使用 OpenAI 兼容格式
func (p *DeepSeekProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *DeepSeekProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	return p.sendStreamRequest(ctx, httpReq, callback)
}
