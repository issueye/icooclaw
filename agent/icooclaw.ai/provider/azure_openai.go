package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ============ Azure OpenAI Provider ============

// AzureOpenAIProvider Azure OpenAI Provider 实现
type AzureOpenAIProvider struct {
	*BaseProvider
	APIVersion string // Azure API 版本
}

// NewAzureOpenAIProvider 创建 Azure OpenAI Provider
func NewAzureOpenAIProvider(apiKey, endpoint, deployment, apiVersion string) *AzureOpenAIProvider {
	if apiVersion == "" {
		apiVersion = "2024-02-15-preview"
	}
	// Azure 使用特殊的 URL 格式
	apiBase := strings.TrimSuffix(endpoint, "/")
	if deployment != "" {
		apiBase += "/openai/deployments/" + deployment
	}
	return &AzureOpenAIProvider{
		BaseProvider: NewBaseProvider("azure-openai", apiKey, apiBase, deployment),
		APIVersion:   apiVersion,
	}
}

// Chat 实现 Chat 方法
func (p *AzureOpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Azure 使用 deployment 名称作为模型
	if req.Model == "" {
		req.Model = p.Model
	}

	// 构建 URL，包含 API 版本
	url := fmt.Sprintf("%s?api-version=%s", p.APIBase, p.APIVersion)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("api-key", p.APIKey)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s %s", resp.Status, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// ChatStream 实现ChatStream方法
func (p *AzureOpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	req.Stream = true
	// Azure 使用 deployment 名称作为模型
	if req.Model == "" {
		req.Model = p.Model
	}

	// 构建 URL，包含 API 版本
	url := fmt.Sprintf("%s?api-version=%s", p.APIBase, p.APIVersion)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("api-key", p.APIKey)
	}

	return p.sendStreamRequest(ctx, httpReq, callback)
}
