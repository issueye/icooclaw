package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// OnChunkFunc 流式 token 回调
type OnChunkFunc func(chunk string)

// Loop Agent Loop实现ReAct模式的对话循环
type Loop struct {
	agent   *Agent
	session *storage.Session
	logger  *slog.Logger
	onChunk OnChunkFunc
}

// NewLoop 创建Agent Loop
func NewLoop(agent *Agent, session *storage.Session, logger *slog.Logger) *Loop {
	return &Loop{
		agent:   agent,
		session: session,
		logger:  logger,
	}
}

// NewLoopWithStream 创建支持流式的 Agent Loop
func NewLoopWithStream(agent *Agent, session *storage.Session, logger *slog.Logger, onChunk OnChunkFunc) *Loop {
	return &Loop{
		agent:   agent,
		session: session,
		logger:  logger,
		onChunk: onChunk,
	}
}

// Run 运行Agent Loop
// 迭代逻辑：
// 1. 发送消息到LLM
// 2. 检查是否有tool_calls
//   - 有：执行工具，添加结果，继续循环
//   - 无：返回content，结束循环
func (l *Loop) Run(ctx context.Context, messages []provider.Message, systemPrompt string) (string, []provider.ToolCall, error) {
	maxIterations := 10
	toolDefs := l.agent.Tools().ToDefinitions()

	for iteration := 0; iteration < maxIterations; iteration++ {
		// 构建请求
		req := provider.ChatRequest{
			Messages:    messages,
			Model:       l.agent.Provider().GetDefaultModel(),
			Temperature: l.agent.Config().Temperature,
			MaxTokens:   l.agent.Config().MaxTokens,
		}

		// 添加工具定义
		if len(toolDefs) > 0 {
			// 转换工具定义
			tools := make([]provider.ToolDefinition, len(toolDefs))
			for i, t := range toolDefs {
				tools[i] = provider.ToolDefinition{
					Type: t.Type,
					Function: provider.FunctionDefinition{
						Name:        t.Function.Name,
						Description: t.Function.Description,
						Parameters:  t.Function.Parameters,
					},
				}
			}
			req.Tools = tools
		}

		// 添加系统提示词
		if systemPrompt != "" && len(messages) > 0 && messages[0].Role == "system" {
			messages[0].Content = systemPrompt
		} else if systemPrompt != "" {
			messages = append([]provider.Message{
				{Role: "system", Content: systemPrompt},
			}, messages...)
		}

		l.logger.Debug("Sending request to LLM", "iteration", iteration, "message_count", len(messages))

		// 记录发送给 LLM 的完整请求
		l.logLLMRequest(iteration, req, systemPrompt)

		// 调用LLM
		var content string
		var toolCalls []provider.ToolCall
		var reasoningContent string

		if l.onChunk != nil {
			var fullContent strings.Builder
			// 累积 tool_calls (按 index 分组) - 使用指针避免复制
			toolCallsData := make(map[int]*struct {
				id   string
				typ  string
				name string
				args strings.Builder
			})
			err := l.agent.Provider().ChatStream(ctx, req, func(chunk provider.StreamChunk) error {
				if chunk.Content != "" {
					fullContent.WriteString(chunk.Content)
					l.onChunk(chunk.Content)
				}
				if chunk.ReasoningContent != "" {
					reasoningContent += chunk.ReasoningContent
				}
				// 收集 tool_calls
				for _, tc := range chunk.ToolCalls {
					index := tc.Index
					if existing, ok := toolCallsData[index]; ok {
						// 累积 arguments (只有非空时才拼接)
						if tc.Arguments != "" {
							existing.args.WriteString(tc.Arguments)
						}
						// 更新 ID 和 Type (如果存在)
						if tc.ID != "" {
							existing.id = tc.ID
						}
						if tc.Type != "" {
							existing.typ = tc.Type
						}
						if tc.Name != "" {
							existing.name = tc.Name
						}
					} else {
						tcData := &struct {
							id   string
							typ  string
							name string
							args strings.Builder
						}{
							id:   tc.ID,
							typ:  tc.Type,
							name: tc.Name,
						}
						tcData.args.WriteString(tc.Arguments)
						toolCallsData[index] = tcData
					}
				}
				return nil
			})
			if err != nil {
				return "", nil, fmt.Errorf("failed to call LLM stream: %w", err)
			}
			content = fullContent.String()
			// 转换为 slice
			for _, tc := range toolCallsData {
				toolCalls = append(toolCalls, provider.ToolCall{
					ID:   tc.id,
					Type: tc.typ,
					Function: provider.ToolCallFunction{
						Name:      tc.name,
						Arguments: tc.args.String(),
					},
				})
			}
		} else {
			resp, err := l.agent.Provider().Chat(ctx, req)
			if err != nil {
				return "", nil, fmt.Errorf("failed to call LLM: %w", err)
			}

			// 检查响应
			if len(resp.Choices) == 0 {
				return "", nil, fmt.Errorf("no choices in response")
			}

			choice := resp.Choices[0]
			content = choice.Message.Content
			toolCalls = choice.Message.ToolCalls
			reasoningContent = choice.Message.ReasoningContent
		}

		// 记录 LLM 响应
		l.logLLMResponse(iteration, content, toolCalls, reasoningContent)

		// 检查 finish reason
		if len(toolCalls) == 0 && (l.onChunk == nil || content != "") {
			l.logger.Info("Agent loop completed", "iterations", iteration+1)
			return content, nil, nil
		}

		// 检查tool_calls
		if len(toolCalls) > 0 {
			l.logger.Info("Executing tools", "count", len(toolCalls))

			// 添加助手消息（包含tool_calls）
			toolCallsJSON, _ := json.Marshal(toolCalls)
			l.session.AddMessage("assistant", content, string(toolCallsJSON), "", "", reasoningContent)

			// 首先添加 assistant 消息（包含所有 tool_calls）
			messages = append(messages, provider.Message{
				Role:             "assistant",
				Content:          content,
				ToolCalls:        toolCalls,
				ReasoningContent: reasoningContent,
			})

			// 执行工具并添加工具结果消息
			for _, call := range toolCalls {
				// 转换为 tools.ToolCall
				toolCall := tools.ToolCall{
					ID:   call.ID,
					Type: call.Type,
					Function: tools.ToolCallFunction{
						Name:      call.Function.Name,
						Arguments: call.Function.Arguments,
					},
				}
				result := l.agent.Tools().Execute(ctx, toolCall)

				// 记录工具执行
				l.logToolExecution(call.Function.Name, call.Function.Arguments, result)

				var resultContent string
				if result.Error != nil {
					resultContent = fmt.Sprintf("Error: %v", result.Error)
				} else {
					resultContent = result.Content
				}

				// 添加工具结果消息到 session
				l.session.AddMessage("tool", resultContent, "", result.ToolCallID, call.Function.Name, "")

				// 添加工具结果消息到消息列表
				messages = append(messages, provider.Message{
					Role:       "tool",
					Content:    resultContent,
					ToolCallID: call.ID,
				})
			}

			// 继续循环
			continue
		}

		return content, nil, nil
	}

	return "", nil, fmt.Errorf("max iterations exceeded")
}

// logLLMRequest 记录发送给 LLM 的请求
func (l *Loop) logLLMRequest(iteration int, req provider.ChatRequest, systemPrompt string) {
	messagesJSON, _ := json.MarshalIndent(req.Messages, "", "  ")
	toolsJSON, _ := json.MarshalIndent(req.Tools, "", "  ")

	l.logger.Info("=== LLM Request ===",
		"iteration", iteration,
		"model", req.Model,
		"temperature", req.Temperature,
		"max_tokens", req.MaxTokens,
		"message_count", len(req.Messages),
		"tool_count", len(req.Tools))

	l.logger.Debug("System Prompt", "content", systemPrompt)
	l.logger.Debug("Messages", "messages", string(messagesJSON))
	if len(req.Tools) > 0 {
		l.logger.Debug("Tools", "tools", string(toolsJSON))
	}
}

// logLLMResponse 记录 LLM 的响应
func (l *Loop) logLLMResponse(iteration int, content string, toolCalls []provider.ToolCall, reasoningContent string) {
	l.logger.Info("=== LLM Response ===",
		"iteration", iteration,
		"content_length", len(content),
		"tool_call_count", len(toolCalls),
		"reasoning_length", len(reasoningContent))

	if content != "" {
		l.logger.Debug("Response Content", "content", content)
	}
	if reasoningContent != "" {
		l.logger.Debug("Reasoning Content", "reasoning", reasoningContent)
	}
	if len(toolCalls) > 0 {
		toolCallsJSON, _ := json.MarshalIndent(toolCalls, "", "  ")
		l.logger.Debug("Tool Calls", "tool_calls", string(toolCallsJSON))
	}
}

// logToolExecution 记录工具执行
func (l *Loop) logToolExecution(toolName string, arguments string, result tools.ToolResult) {
	l.logger.Info("=== Tool Execution ===",
		"tool_name", toolName,
		"tool_call_id", result.ToolCallID,
		"has_error", result.Error != nil)

	l.logger.Debug("Tool Arguments", "arguments", arguments)
	if result.Error != nil {
		l.logger.Debug("Tool Error", "error", result.Error.Error())
	} else {
		l.logger.Debug("Tool Result", "result", result.Content)
	}
}
