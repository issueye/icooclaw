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
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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

// StreamToolCall 用于流式传输中的 tool_call 累积
type StreamToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Index     int    `json:"index"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// StreamChunk 流式响应片段
type StreamChunk struct {
	ID               string           `json:"id"`
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	FinishReason     string           `json:"finish_reason,omitempty"`
	Usage            *Usage           `json:"usage,omitempty"`
	ToolCalls        []StreamToolCall `json:"tool_calls,omitempty"`
}

// StreamCallback 流式回调函数
type StreamCallback func(chunk StreamChunk) error

// Provider LLM Provider接口
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error
	GetDefaultModel() string
	GetName() string
}

type OpenAIToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Index    int              `json:"index"`
	Function ToolCallFunction `json:"function"`
}

type OpenAIDelta struct {
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content"`
	ToolCalls        []OpenAIToolCall `json:"tool_calls"`
	Refusal          string           `json:"refusal"`
	FunctionCall     json.RawMessage  `json:"function_call"`
}

type OpenAIChoice struct {
	Index        int         `json:"index"`
	Delta        OpenAIDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type OpenAIChunk struct {
	ID      string         `json:"id"`      // 流式响应的唯一标识符
	Choices []OpenAIChoice `json:"choices"` // 流式响应中的选择
	Usage   *Usage         `json:"usage"`
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

// ChatStream 实现Provider接口
func (p *BaseProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
	return fmt.Errorf("not implemented")
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

// sendStreamRequest 发送流式请求并处理 SSE
func (p *BaseProvider) sendStreamRequest(ctx context.Context, req *http.Request, callback StreamCallback) error {
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
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		if data == "[DONE]" {
			break
		}

		fmt.Println("请求[sendStreamRequest] 数据:", data)

		// 解析 OpenAI 格式的流式块
		var chunk OpenAIChunk

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 忽略解析失败的行，可能是非 JSON 或格式不匹配
			continue
		}

		if len(chunk.Choices) > 0 {
			c := chunk.Choices[0]

			// 转换 tool_calls 到 StreamToolCall
			var streamToolCalls []StreamToolCall
			for _, tc := range c.Delta.ToolCalls {
				streamToolCalls = append(streamToolCalls, StreamToolCall{
					ID:        tc.ID,
					Type:      tc.Type,
					Index:     tc.Index,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}

			streamChunk := StreamChunk{
				ID:               chunk.ID,
				Content:          c.Delta.Content,
				ReasoningContent: c.Delta.ReasoningContent,
				FinishReason:     c.FinishReason,
				Usage:            chunk.Usage,
				ToolCalls:        streamToolCalls,
			}
			if err := callback(streamChunk); err != nil {
				return err
			}
		}
	}

	return nil
}

// ============ 思考内容解析器 ============

// ThinkTagParser 思考内容标签解析器
// 用于解析不同模型返回的思考内容标签
type ThinkTagParser struct {
	// 思考内容开始标签
	StartTag string
	// 思考内容结束标签
	EndTag string
}

// 常见的思考标签解析器
var (
	// DeepSeek 使用的思考标签
	DeepSeekParser = &ThinkTagParser{
		StartTag: "<think>",
		EndTag:   "</think>",
	}

	// Kimi 使用的思考标签
	KimiParser = &ThinkTagParser{
		StartTag: "<|start_header_id|>reasoning<|end_header_id|>",
		EndTag:   "<|start_header_id|>assistant<|end_header_id|>",
	}

	// MiniMax 使用的思考标签
	MiniMaxParser = &ThinkTagParser{
		StartTag: "<think>",
		EndTag:   "</think>",
	}

	// 默认解析器 - 支持多种标签
	DefaultParser = &ThinkTagParser{
		StartTag: "<think>",
		EndTag:   "</think>",
	}
)

// ExtractThinkingContent 从内容中提取思考内容并清理内容
// 返回: (清理后的内容, 思考内容)
func ExtractThinkingContent(content, reasoningContent string) (string, string) {
	// 1. 如果有独立的 reasoning_content 字段，优先使用
	if reasoningContent != "" {
		return content, reasoningContent
	}

	// 2. 从 content 中解析思考标签 (DeepSeek-R1 等模型)
	thinking := extractFromTags(content, DeepSeekParser)
	if thinking != "" {
		return content, thinking
	}

	// 3. 尝试 Kimi 格式
	thinking = extractFromTags(content, KimiParser)
	if thinking != "" {
		return content, thinking
	}

	// 没有找到思考内容
	return content, ""
}

// extractFromTags 使用指定的标签解析器提取思考内容
func extractFromTags(content string, parser *ThinkTagParser) string {
	startIdx := strings.Index(content, parser.StartTag)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(content[startIdx:], parser.EndTag)
	if endIdx == -1 {
		return ""
	}
	endIdx += startIdx

	// 提取思考内容
	startIdx += len(parser.StartTag)
	thinking := strings.TrimSpace(content[startIdx:endIdx])

	return thinking
}

// CleanThinkingTags 从内容中移除思考标签
// 返回: 清理后的内容
func CleanThinkingTags(content string) string {
	content = strings.ReplaceAll(content, "<think>", "")
	content = strings.ReplaceAll(content, "</think>", "")
	content = strings.ReplaceAll(content, "<|start_header_id|>reasoning<|end_header_id|>", "")
	content = strings.ReplaceAll(content, "<|start_header_id|>assistant<|end_header_id|>", "")
	return strings.TrimSpace(content)
}
