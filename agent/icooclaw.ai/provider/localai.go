package provider

import "context"

// ============ LocalAI Provider ============

// LocalAIProvider LocalAI Provider 实现
type LocalAIProvider struct {
	*BaseProvider
}

// NewLocalAIProvider 创建 LocalAI Provider
func NewLocalAIProvider(apiBase, model string) *LocalAIProvider {
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &LocalAIProvider{
		BaseProvider: NewBaseProvider("localai", "", apiBase, model),
	}
}

// Chat 实现 Chat 方法
func (p *LocalAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// LocalAI 不需要 Authorization 头
	httpReq.Header.Del("Authorization")

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *LocalAIProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// LocalAI 不需要 Authorization 头
	httpReq.Header.Del("Authorization")

	return p.sendStreamRequest(ctx, httpReq, callback)
}
