package provider

import "context"

// ============ OpenAI Compatible Provider 通用模板 ============

// OpenAICompatibleProvider 通用的 OpenAI 兼容 Provider
// 支持任何实现了 OpenAI ChatCompletions API 的服务
type OpenAICompatibleProvider struct {
	*BaseProvider
	ExtraHeaders map[string]string // 额外的请求头
}

// NewOpenAICompatibleProvider 创建 OpenAI 兼容 Provider
func NewOpenAICompatibleProvider(name, apiKey, apiBase, model string) *OpenAICompatibleProvider {
	if apiBase == "" {
		apiBase = "http://localhost:8000/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &OpenAICompatibleProvider{
		BaseProvider: NewBaseProvider(name, apiKey, apiBase, model),
		ExtraHeaders: make(map[string]string),
	}
}

// SetHeader 设置额外的请求头
func (p *OpenAICompatibleProvider) SetHeader(key, value string) {
	p.ExtraHeaders[key] = value
}

// Chat 实现 Chat 方法
func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 添加额外的请求头
	for key, value := range p.ExtraHeaders {
		httpReq.Header.Set(key, value)
	}

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *OpenAICompatibleProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// 添加额外的请求头
	for key, value := range p.ExtraHeaders {
		httpReq.Header.Set(key, value)
	}

	return p.sendStreamRequest(ctx, httpReq, callback)
}
