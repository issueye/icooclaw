package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/icooclaw/icooclaw/internal/provider"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// Loop Agent Loop实现ReAct模式的对话循环
type Loop struct {
	agent   *Agent
	session *storage.Session
	logger  *slog.Logger
}

// NewLoop 创建Agent Loop
func NewLoop(agent *Agent, session *storage.Session, logger *slog.Logger) *Loop {
	return &Loop{
		agent:   agent,
		session: session,
		logger:  logger,
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

		// 调用LLM
		resp, err := l.agent.Provider().Chat(ctx, req)
		if err != nil {
			return "", nil, fmt.Errorf("failed to call LLM: %w", err)
		}

		// 检查响应
		if len(resp.Choices) == 0 {
			return "", nil, fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		content := choice.Message.Content

		// 检查finish reason
		if choice.FinishReason == "stop" {
			l.logger.Info("Agent loop completed", "iterations", iteration+1)
			return content, nil, nil
		}

		// 检查tool_calls
		if len(choice.Message.ToolCalls) > 0 {
			l.logger.Info("Executing tools", "count", len(choice.Message.ToolCalls))

			// 添加助手消息（包含tool_calls）
			toolCallsJSON, _ := json.Marshal(choice.Message.ToolCalls)
			l.session.AddMessage("assistant", content, string(toolCallsJSON), "", "", choice.Message.ReasoningContent)

			// 执行工具
			for _, call := range choice.Message.ToolCalls {
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
				var resultContent string
				if result.Error != nil {
					resultContent = fmt.Sprintf("Error: %v", result.Error)
				} else {
					resultContent = result.Content
				}

				// 添加工具结果消息
				l.session.AddMessage("tool", resultContent, "", result.ToolCallID, call.Function.Name, "")

				// 添加到消息列表
				messages = append(messages, choice.Message)
				messages = append(messages, provider.Message{
					Role:       "tool",
					Content:    resultContent,
					ToolCallID: result.ToolCallID,
				})
			}

			// 继续循环
			continue
		}

		// 其他finish reason
		return content, nil, nil
	}

	return "", nil, fmt.Errorf("max iterations exceeded")
}
