package react

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers"
)

// Chat 发送消息
func (a *ReActAgent) Chat(ctx context.Context, msg bus.InboundMessage) (string, int, error) {

	// 会话键
	sessionKey := consts.GetSessionKey(msg.Channel, msg.SessionID)

	// 1. 获取供应商实例
	provider, modelName, err := a.GetDynamicProvider(ctx)
	if err != nil {
		return "", 0, err
	}

	// 2. 构建消息
	messages, err := a.buildMessages(ctx, sessionKey, msg)
	if err != nil {
		return "", 0, err
	}

	// 3. 运行LLM模型
	content, iteration, err := a.RunLLM(ctx, modelName, provider, messages, msg)
	if err != nil {
		return "", 0, err
	}

	return content, iteration, nil
}

// RunLLM 运行LLM模型
func (a *ReActAgent) RunLLM(
	ctx context.Context, // 上下文
	modelName string, // 模型名称
	provider providers.Provider, // 提供商
	messages []providers.ChatMessage, // 消息列表
	msg bus.InboundMessage, // 原始消息
) (string, int, error) {
	iteration := 0
	currentMessages := messages
	var err error

	// 调用钩子运行LLM模型前
	if a.hooks != nil {
		currentMessages, err = a.hooks.RunLLMBefore(ctx, msg, currentMessages)
		if err != nil {
			return "", iteration, err
		}
	}

	// 1. 迭代调用LLM
	for iteration < a.maxToolIterations {
		iteration++

		// 1. 构建请求消息
		req := providers.ChatRequest{
			Model:    modelName,
			Messages: currentMessages,
		}

		// 2. 处理工具调用
		// 添加工具调用如果可用
		toolDefs := a.tools.ToProviderDefs()
		if len(toolDefs) > 0 {
			req.Tools = a.convertToolDefinitions(toolDefs)
		}

		// 3. 发送请求到提供商
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return "", iteration, fmt.Errorf("LLM请求失败: %w", err)
		}

		// 4. 处理工具调用响应
		if len(resp.ToolCalls) > 0 {
			// 处理工具调用响应
			assistantMsg := providers.ChatMessage{
				Role:      consts.RoleAssistant.ToString(),
				Content:   resp.Content,
				ToolCalls: resp.ToolCalls,
			}

			currentMessages = append(currentMessages, assistantMsg)

			// 5. 执行每个工具调用
			for _, tc := range resp.ToolCalls {
				// 执行工具调用
				toolResult, err := a.executeToolCall(ctx, tc, msg)
				if err != nil {
					toolResult = fmt.Sprintf("错误: %v", err)
				}

				// 添加工具调用结果消息
				currentMessages = append(currentMessages, providers.ChatMessage{
					Role:       consts.RoleTool.ToString(),
					Content:    toolResult,
					ToolCallID: tc.ID,
				})

				continue
			}
		}

		// 6. 返回响应内容
		return resp.Content, iteration, nil
	}

	// 到达最大迭代次数
	return "", iteration, fmt.Errorf("已达到最大工具迭代次数 (%d)", a.maxToolIterations)
}
