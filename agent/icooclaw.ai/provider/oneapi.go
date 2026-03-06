package provider

import "context"

// ============通配 AI (OneAPI) Provider ============

// OneAPIProvider OneAPI 通配 AI 接口 Provider
type OneAPIProvider struct {
	*BaseProvider
}

// NewOneAPIProvider 创建 OneAPI Provider
func NewOneAPIProvider(apiKey, apiBase, model string) *OneAPIProvider {
	if apiBase == "" {
		apiBase = "https://api.oneapi.icu/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &OneAPIProvider{
		BaseProvider: NewBaseProvider("oneapi", apiKey, apiBase, model),
	}
}

// Chat 实现 Chat 方法
func (p *OneAPIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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
func (p *OneAPIProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
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
