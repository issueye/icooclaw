package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/storage"
	"io"
	"net/http"
)

// AzureOpenAIProvider implements Provider for Azure OpenAI.
type AzureOpenAIProvider struct {
	*BaseProvider
	apiVersion string
	deployment string
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider.
func NewAzureOpenAIProvider(cfg *storage.Provider) *AzureOpenAIProvider {
	apiVersion := "2024-02-15-preview"
	v, ok := cfg.Metadata["api_version"].(string)
	if ok && v != "" {
		apiVersion = v
	}

	deployment := cfg.DefaultModel
	if deployment == "" {
		deployment = "gpt-4o"
	}

	return &AzureOpenAIProvider{
		BaseProvider: NewBaseProvider("azure", cfg.APIKey, cfg.APIBase, deployment),
		apiVersion:   apiVersion,
		deployment:   deployment,
	}
}

// Chat sends a chat request to Azure OpenAI.
func (p *AzureOpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false

	// Azure OpenAI uses deployment name in URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.apiBase, p.deployment, p.apiVersion)

	var reqBody io.Reader
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	reqBody = bytes.NewReader(data)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
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

// ChatStream sends a streaming chat request to Azure OpenAI.
func (p *AzureOpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.apiBase, p.deployment, p.apiVersion)

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}

	return p.streamResponse(resp, callback)
}
