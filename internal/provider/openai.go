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

// ============ OpenAI Provider ============

// OpenAIProvider OpenAI Provider 实现
type OpenAIProvider struct {
	*BaseProvider
}

// NewOpenAIProvider 创建OpenAI Provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	apiBase := "https://api.openai.com/v1"
	if model == "" {
		model = "gpt-4o"
	}
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider("openai", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// OpenAI 特定请求头
	httpReq.Header.Set("OpenAI-Beta", "assistants=v2")

	return p.sendRequest(ctx, httpReq)
}

// ============ Anthropic Provider ============

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest Anthropic 请求格式
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Tools       []ToolDefinition   `json:"tools,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

// AnthropicResponse Anthropic 响应格式
type AnthropicResponse struct {
	Type         string         `json:"type"`
	ID           string         `json:"id"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

// ContentBlock 内容块
type ContentBlock struct {
	Type       string `json:"type"`
	Text       string `json:"text"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Input      string `json:"input"`
	ToolUseID  string `json:"tool_use_id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolInput  any    `json:"tool_input,omitempty"`
}

// AnthropicUsage Anthropic 使用量
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicProvider Anthropic Provider 实现
type AnthropicProvider struct {
	*BaseProvider
	Version string // Anthropic API 版本
}

// NewAnthropicProvider 创建Anthropic Provider
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	apiBase := "https://api.anthropic.com/v1"
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicProvider{
		BaseProvider: NewBaseProvider("anthropic", apiKey, apiBase, model),
		Version:      "2023-06-01",
	}
}

// Chat 实现Chat方法 - Anthropic 使用不同的 API 格式
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 转换消息格式
	messages := make([]AnthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Anthropic 使用特殊的 system 消息
			messages = append([]AnthropicMessage{{Role: "system", Content: msg.Content}}, messages...)
		} else {
			messages = append(messages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// 设置默认 max_tokens
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// 构建 Anthropic 请求
	anthReq := AnthropicRequest{
		Model:       p.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       req.Tools,
		Stream:      false,
	}

	// 序列化请求
	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 构建 HTTP 请求
	url := strings.TrimSuffix(p.APIBase, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", p.Version)

	// 发送请求
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

	// 解析 Anthropic 响应
	var anthResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 转换为标准响应格式
	return p.convertResponse(anthResp), nil
}

// convertResponse 转换 Anthropic 响应为标准格式
func (p *AnthropicProvider) convertResponse(anthResp AnthropicResponse) *ChatResponse {
	var content string
	var toolCalls []ToolCall

	for _, block := range anthResp.Content {
		if block.Type == "text" {
			content = block.Text
		} else if block.Type == "tool_use" {
			// Anthropic 工具调用
			argsBytes, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: ToolCallFunction{
					Name:      block.Name,
					Arguments: argsBytes,
				},
			})
		}
	}

	finishReason := anthResp.StopReason
	if finishReason == "tool_use" {
		finishReason = "tool_calls"
	}

	return &ChatResponse{
		ID:      anthResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   anthResp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      anthResp.Role,
					Content:   content,
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: Usage{
			PromptTokens:     anthResp.Usage.InputTokens,
			CompletionTokens: anthResp.Usage.OutputTokens,
			TotalTokens:      anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		},
	}
}

// ============ DeepSeek Provider ============

// DeepSeekProvider DeepSeek Provider 实现
type DeepSeekProvider struct {
	*BaseProvider
}

// NewDeepSeekProvider 创建DeepSeek Provider
func NewDeepSeekProvider(apiKey, model string) *DeepSeekProvider {
	apiBase := "https://api.deepseek.com/v1"
	if model == "" {
		model = "deepseek-chat"
	}
	return &DeepSeekProvider{
		BaseProvider: NewBaseProvider("deepseek", apiKey, apiBase, model),
	}
}

// Chat 实现Chat方法 - DeepSeek 使用 OpenAI 兼容格式
func (p *DeepSeekProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 设置模型
	if req.Model == "" {
		req.Model = p.Model
	}

	httpReq, err := p.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return p.sendRequest(ctx, httpReq)
}

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
