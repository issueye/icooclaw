package provider

import "context"

// ============ Custom Provider ============

// CustomProvider 自定义端点 Provider 实现
type CustomProvider struct {
	*BaseProvider
}

// NewCustomProvider 创建Custom Provider
func NewCustomProvider(apiKey, apiBase, model string) *CustomProvider {
	if apiBase == "" {
		apiBase = "http://localhost:8000/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &CustomProvider{
		BaseProvider: NewBaseProvider("custom", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法 - Custom 使用 OpenAI 兼容格式
func (p *CustomProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Custom 端点可能不需要 Bearer token
	if p.APIKey != "" && p.APIKey != "no-key" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *CustomProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// Custom 端点可能不需要 Bearer token
	if p.APIKey != "" && p.APIKey != "no-key" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	return p.sendStreamRequest(ctx, httpReq, callback)
}
