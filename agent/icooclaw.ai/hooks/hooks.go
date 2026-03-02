package hooks

import (
	"context"
	"fmt"
	"time"

	"icooclaw.ai/agent/tools"
	"icooclaw.ai/consts"
	"icooclaw.ai/provider"
	"icooclaw.ai/storage"
	bus "icooclaw.bus"
)

// ============ Hook 接口定义 ============

// OnChunkFunc 流式 token 回调
// content: 内容块, thinking: 思考内容
type OnChunkFunc func(content, thinking string)

// HookFunc 通用钩子函数类型
type HookFunc func(ctx context.Context, data interface{}) error

// ReActHooks ReAct 循环的钩子接口
type ReActHooks interface {
	// OnLLMRequest LLM 请求发送前调用
	OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error

	// OnLLMChunk 流式输出内容块时调用
	OnLLMChunk(ctx context.Context, content, thinking string) error

	// OnLLMResponse LLM 响应后调用
	OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error

	// OnToolCall 工具调用前调用
	OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error

	// OnToolResult 工具执行后调用
	OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error

	// OnIterationStart 每次迭代开始时调用
	OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error

	// OnIterationEnd 每次迭代结束时调用
	OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error

	// OnError 发生错误时调用
	OnError(ctx context.Context, err error) error

	// OnComplete 循环完成时调用
	OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error
}

// ============ Loop Hooks 实现 (供 Loop 使用) ============

// LoopHooks 实现 ReActHooks 接口，用于将 Loop 的 onChunk 回调连接到 ReAct
type LoopHooks struct {
	agent    *Agent
	onChunk  OnChunkFunc
	chatID   string
	clientID string
	session  *storage.Session
}

func (h *LoopHooks) OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error {
	return nil
}

func (h *LoopHooks) OnLLMChunk(ctx context.Context, content, thinking string) error {
	if h.onChunk != nil {
		h.onChunk(content, thinking)
	}
	return nil
}

func (h *LoopHooks) OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error {
	return nil
}

func (h *LoopHooks) OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error {
	// 保存工具调用消息到数据库
	if h.agent.storage != nil && h.session != nil {
		_, err := h.agent.storage.AddMessage(
			h.session.ID,
			consts.RoleToolCall.ToString(),
			"",         // content 为空
			arguments,  // tool_calls 字段存储参数
			toolCallID, // tool_call_id
			toolName,   // tool_name
			"",         // reasoning_content 为空
		)
		if err != nil {
			h.agent.logger.Error("保存工具调用消息失败", "error", err, "tool", toolName)
		}
	}

	// 发布到消息总线
	if h.agent.bus != nil {
		h.agent.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Type:       "tool_call",
			ID:         toolCallID,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Arguments:  arguments,
			Status:     "running",
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   map[string]interface{}{"client_id": h.clientID},
		})
	}
	return nil
}

func (h *LoopHooks) OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error {
	// 保存工具结果消息到数据库
	if h.agent.storage != nil && h.session != nil {
		resultContent := result.Content
		if result.Error != nil {
			resultContent = fmt.Sprintf("Error: %v", result.Error)
		}
		_, err := h.agent.storage.AddMessage(
			h.session.ID,
			consts.RoleTool.ToString(),
			resultContent, // content 存储结果
			"",            // tool_calls 为空
			toolCallID,    // tool_call_id
			toolName,      // tool_name
			"",            // reasoning_content 为空
		)
		if err != nil {
			h.agent.logger.Error("保存工具结果消息失败", "error", err, "tool", toolName)
		}
	}

	// 发布到消息总线
	if h.agent.bus != nil {
		status := "completed"
		errorMsg := ""
		if result.Error != nil {
			status = "error"
			errorMsg = result.Error.Error()
		}
		h.agent.bus.PublishOutbound(ctx, bus.OutboundMessage{
			Type:       "tool_result",
			ID:         toolCallID,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Content:    result.Content,
			Error:      errorMsg,
			Status:     status,
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   map[string]interface{}{"client_id": h.clientID},
		})
	}
	return nil
}

func (h *LoopHooks) OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error {
	return nil
}

func (h *LoopHooks) OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error {
	return nil
}

func (h *LoopHooks) OnError(ctx context.Context, err error) error { return nil }

func (h *LoopHooks) OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error {
	return nil
}
