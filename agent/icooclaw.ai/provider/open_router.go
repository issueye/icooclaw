package provider

import "context"

// ============ OpenRouter Provider ============

// OpenRouterProvider OpenRouter Provider 实现
type OpenRouterProvider struct {
	*BaseProvider
}

// NewOpenRouterProvider 创建OpenRouter Provider
func NewOpenRouterProvider(apiKey, model string) *OpenRouterProvider {
	apiBase := "https://openrouter.ai/api/v1"
	if model == "" {
		model = "anthropic/claude-3.5-sonnet"
	}
	return &OpenRouterProvider{
		BaseProvider: NewBaseProvider("openrouter", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法 - OpenRouter 使用 OpenAI 兼容格式
func (p *OpenRouterProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 添加OpenRouter特定的请求头
	httpReq.Header.Set("HTTP-Referer", "https://github.com/icooclaw")
	httpReq.Header.Set("X-Title", "icooclaw")

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *OpenRouterProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// 添加OpenRouter特定的请求头
	httpReq.Header.Set("HTTP-Referer", "https://github.com/icooclaw")
	httpReq.Header.Set("X-Title", "icooclaw")

	return p.sendStreamRequest(ctx, httpReq, callback)
}
