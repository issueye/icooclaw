package react

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

// ChatStream 发送消息（流式）
func (a *ReActAgent) ChatStream(ctx context.Context, msg bus.InboundMessage, callback StreamCallback) (string, int, error) {
	// 会话键
	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	// 1. 获取供应商实例
	provider, modelName, err := a.GetDynamicProvider(ctx)
	if err != nil {
		if callback != nil {
			callback(StreamChunk{Error: err})
		}
		return "", 0, err
	}

	// 2. 构建消息
	messages, err := a.buildMessages(ctx, sessionKey, msg)
	if err != nil {
		if callback != nil {
			callback(StreamChunk{Error: err})
		}
		return "", 0, err
	}

	// 3. 运行LLM模型（流式）
	content, iteration, err := a.RunLLMStream(ctx, modelName, provider, messages, msg, callback)
	if err != nil {
		return "", iteration, err
	}

	// 4. 保存助手消息到记忆
	if a.memory != nil && content != "" {
		if err := a.memory.Save(ctx, sessionKey, consts.RoleAssistant.ToString(), content); err != nil {
			a.logger.With("name", "【智能体】").Warn("保存助手消息失败", "error", err)
		}
	}

	return content, iteration, nil
}

// RunLLMStream 运行LLM模型（流式）
func (a *ReActAgent) RunLLMStream(
	ctx context.Context,
	modelName string,
	provider providers.Provider,
	messages []providers.ChatMessage,
	msg bus.InboundMessage,
	callback StreamCallback,
) (string, int, error) {
	iteration := 0
	currentMessages := messages
	var err error

	// 调用钩子运行LLM模型前
	if a.hooks != nil {
		currentMessages, err = a.hooks.OnRunLLMBefore(ctx, msg, currentMessages)
		if err != nil {
			return "", iteration, err
		}
	}

	// 迭代调用LLM
	for iteration < a.maxToolIterations {
		iteration++

		// 1. 构建请求消息
		req := providers.ChatRequest{
			Model:    modelName,
			Messages: currentMessages,
		}

		// 2. 处理工具调用
		toolDefs := a.tools.ToProviderDefs()
		if len(toolDefs) > 0 {
			req.Tools = a.convertToolDefinitions(toolDefs)
		}

		// 3. 发送流式请求到提供商
		var collectedContent string
		var collectedReasoning string
		var collectedToolCalls []providers.ToolCall

		err = provider.ChatStream(ctx, req, func(chunk string, reasoning string, toolCalls []providers.ToolCall, done bool) error {
			// 收集内容
			collectedContent += chunk
			collectedReasoning += reasoning

			// 收集工具调用
			if len(toolCalls) > 0 {
				collectedToolCalls = append(collectedToolCalls, toolCalls...)
			}

			// 发送内容块到回调函数
			if callback != nil {
				// 发送推理过程
				if reasoning != "" {
					if err = callback(StreamChunk{
						Reasoning: reasoning,
						Iteration: iteration,
					}); err != nil {
						return err
					}
				}

				// 发送内容
				if chunk != "" {
					err = callback(StreamChunk{Content: chunk, Iteration: iteration})
					if err != nil {
						return err
					}
				}
			}

			return nil
		})

		if err != nil {
			if callback != nil {
				callback(StreamChunk{Error: err, Iteration: iteration})
			}
			return "", iteration, fmt.Errorf("LLM请求失败: %w", err)
		}

		// 4. 处理工具调用响应
		if len(collectedToolCalls) > 0 {
			// 合并工具调用
			mergedToolCalls := a.mergeToolCalls(collectedToolCalls)

			// 验证工具调用
			validToolCalls := a.validateToolCalls(mergedToolCalls)
			if len(validToolCalls) == 0 {
				// 没有有效工具调用，作为普通响应处理
				return collectedContent, iteration, nil
			}

			// 添加助手消息
			assistantMsg := providers.ChatMessage{
				Role:      consts.RoleAssistant.ToString(),
				Content:   collectedContent,
				ToolCalls: validToolCalls,
			}
			currentMessages = append(currentMessages, assistantMsg)

			// 5. 执行每个工具调用
			for _, tc := range validToolCalls {
				// 发送工具调用通知
				if callback != nil {
					if err := callback(StreamChunk{
						ToolName:  tc.Function.Name,
						Iteration: iteration,
					}); err != nil {
						return "", iteration, err
					}
				}

				// 执行工具调用
				toolResult, execErr := a.executeToolCall(ctx, tc, msg)
				if execErr != nil {
					toolResult = fmt.Sprintf("错误: %v", execErr)
				}

				// 发送工具结果通知
				if callback != nil {
					if err := callback(StreamChunk{
						ToolResult: toolResult,
						Iteration:  iteration,
					}); err != nil {
						return "", iteration, err
					}
				}

				// 添加工具调用结果消息
				currentMessages = append(currentMessages, providers.ChatMessage{
					Role:       consts.RoleTool.ToString(),
					Content:    toolResult,
					ToolCallID: tc.ID,
				})
			}

			// 继续下一个迭代
			continue
		}

		// 6. 没有工具调用，返回响应内容
		// 发送完成信号
		if callback != nil {
			if err := callback(StreamChunk{
				Content:   collectedContent,
				Done:      true,
				Iteration: iteration,
			}); err != nil {
				return "", iteration, err
			}
		}

		return collectedContent, iteration, nil
	}

	// 到达最大迭代次数
	err = fmt.Errorf("已达到最大工具迭代次数 (%d)", a.maxToolIterations)
	if callback != nil {
		callback(StreamChunk{Error: err, Iteration: iteration})
	}
	return "", iteration, err
}
