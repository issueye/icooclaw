package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message 消息结构
type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction 工具调用函数
type ToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Usage 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatRequest 请求结构
type ChatRequest struct {
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	TopP        float64          `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
}

// ChatResponse 响应结构
type ChatResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	ReasoningContent  string   `json:"reasoning_content,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice 选择
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Provider LLM Provider接口
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	GetDefaultModel() string
	GetName() string
}

// BaseProvider 基础Provider
type BaseProvider struct {
	Name       string
	APIKey     string
	APIBase    string
	Model      string
	httpClient *http.Client
}

// NewBaseProvider 创建基础Provider
func NewBaseProvider(name, apiKey, apiBase, model string) *BaseProvider {
	return &BaseProvider{
		Name:    name,
		APIKey:  apiKey,
		APIBase: apiBase,
		Model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GetDefaultModel 获取默认模型
func (p *BaseProvider) GetDefaultModel() string {
	return p.Model
}

// GetName 获取名称
func (p *BaseProvider) GetName() string {
	return p.Name
}

// Chat 实现Provider接口
func (p *BaseProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

// buildRequest 构建请求
func (p *BaseProvider) buildRequest(ctx context.Context, req ChatRequest) (*http.Request, error) {
	url := strings.TrimSuffix(p.APIBase, "/") + "/chat/completions"

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
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	return httpReq, nil
}

// sendRequest 发送请求
func (p *BaseProvider) sendRequest(ctx context.Context, req *http.Request) (*ChatResponse, error) {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s %s", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}
