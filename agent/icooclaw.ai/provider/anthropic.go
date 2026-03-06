package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"icooclaw.core/consts"
)

// ============ Anthropic Provider ============

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    consts.RoleType `json:"role"`
	Content string          `json:"content"`
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
	Type         string          `json:"type"`
	ID           string          `json:"id"`
	Role         consts.RoleType `json:"role"`
	Content      []ContentBlock  `json:"content"`
	Model        string          `json:"model"`
	StopReason   string          `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        AnthropicUsage  `json:"usage"`
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
			messages = append([]AnthropicMessage{{Role: consts.RoleSystem, Content: msg.Content}}, messages...)
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

// ChatStream 实现ChatStream方法
func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	// 转换消息格式
	messages := make([]AnthropicMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			messages = append([]AnthropicMessage{{Role: "system", Content: msg.Content}}, messages...)
		} else {
			messages = append(messages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	anthReq := AnthropicRequest{
		Model:       p.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       req.Tools,
		Stream:      true,
	}

	body, err := json.Marshal(anthReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(p.APIBase, "/") + "/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", p.Version)

	return p.sendAnthropicStreamRequest(ctx, httpReq, callback)
}

// sendAnthropicStreamRequest 处理 Anthropic 专有的 SSE 流
func (p *AnthropicProvider) sendAnthropicStreamRequest(ctx context.Context, req *http.Request, callback StreamCallback) error {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s %s", resp.Status, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	var lastID string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event:") {
			// Anthropic SSE 包含 event 和 data 两行
			eventType := strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			dataLine, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			if !strings.HasPrefix(dataLine, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(dataLine, "data:"))

			switch eventType {
			case "message_start":
				var start struct {
					Message struct {
						ID string `json:"id"`
					} `json:"message"`
				}
				json.Unmarshal([]byte(data), &start)
				lastID = start.Message.ID

			case "content_block_delta":
				var delta struct {
					Delta struct {
						Text string `json:"text"`
					} `json:"delta"`
				}
				if err := json.Unmarshal([]byte(data), &delta); err == nil {
					callback(StreamChunk{
						ID:      lastID,
						Content: delta.Delta.Text,
					})
				}

			case "message_delta":
				var delta struct {
					Delta struct {
						StopReason string `json:"stop_reason"`
					} `json:"delta"`
					Usage struct {
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
				}
				if err := json.Unmarshal([]byte(data), &delta); err == nil {
					finishReason := delta.Delta.StopReason
					if finishReason == "tool_use" {
						finishReason = "tool_calls"
					}
					callback(StreamChunk{
						ID:           lastID,
						FinishReason: finishReason,
						Usage: &Usage{
							CompletionTokens: delta.Usage.OutputTokens,
						},
					})
				}

			case "message_stop":
				return nil

			case "error":
				return fmt.Errorf("anthropic stream error: %s", data)
			}
		}
	}

	return nil
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
					Arguments: string(argsBytes),
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
