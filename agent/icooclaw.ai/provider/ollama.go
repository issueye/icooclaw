package provider

import "context"

// ============ Ollama Provider ============

// OllamaProvider Ollama 本地 LLM Provider 实现
type OllamaProvider struct {
	*BaseProvider
}

// NewOllamaProvider 创建 Ollama Provider
func NewOllamaProvider(apiBase, model string) *OllamaProvider {
	if apiBase == "" {
		apiBase = "http://localhost:11434"
	}
	if model == "" {
		model = "llama2"
	}
	return &OllamaProvider{
		BaseProvider: NewBaseProvider("ollama", "", apiBase, model),
	}
}

// Chat 实现 Chat 方法 - Ollama 使用 OpenAI 兼容格式
func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Ollama 不需要 Authorization 头
	httpReq.Header.Del("Authorization")

	return p.sendRequest(ctx, httpReq)
}

// ChatStream 实现ChatStream方法
func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return err
	}

	// Ollama 不需要 Authorization 头
	httpReq.Header.Del("Authorization")

	return p.sendStreamRequest(ctx, httpReq, callback)
}
