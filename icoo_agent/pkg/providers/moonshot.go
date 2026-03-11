package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/storage"
	"net/http"
)

// MoonshotProvider implements Provider for Moonshot (月之暗面).
type MoonshotProvider struct {
	*BaseProvider
}

// NewMoonshotProvider creates a new Moonshot provider.
func NewMoonshotProvider(cfg *storage.Provider) Provider {
	providerName := consts.ProviderMoonshot
	apiBase := cfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.moonshot.cn/v1"
	}
	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = "moonshot-v1-8k"
	}

	return &MoonshotProvider{
		BaseProvider: NewBaseProvider(providerName.ToString(), cfg.APIKey, apiBase, defaultModel),
	}
}

// Chat sends a chat request to Moonshot.
func (p *MoonshotProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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
				Role    string `json:"role"`
				Content string `json:"content"`
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
		ID:      result.ID,
		Model:   result.Model,
		Content: result.Choices[0].Message.Content,
		Usage:   result.Usage,
	}, nil
}

// ChatStream sends a streaming chat request to Moonshot.
func (p *MoonshotProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
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
