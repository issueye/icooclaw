package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"icooclaw.ai/hooks"
	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
	"icooclaw.core/bus"
	"icooclaw.core/consts"
	"icooclaw.core/storage"
)

// LoopHooks ReAct 钩子实现（解耦版本）
type LoopHooks struct {
	storage  *storage.Storage
	bus      *bus.MessageBus
	onChunk  hooks.OnChunkFunc
	chatID   string
	clientID string
	session  *storage.Session
	logger   *slog.Logger
}

// NewLoopHooks 创建解耦的 LoopHooks
func NewLoopHooks(
	storage *storage.Storage,
	bus *bus.MessageBus,
	onChunk hooks.OnChunkFunc,
	chatID, clientID string,
	session *storage.Session,
	logger *slog.Logger,
) *LoopHooks {
	return &LoopHooks{
		storage:  storage,
		bus:      bus,
		onChunk:  onChunk,
		chatID:   chatID,
		clientID: clientID,
		session:  session,
		logger:   logger,
	}
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
	if h.storage != nil && h.session != nil {
		// 创建或更新工具调用消息
		saveData := storage.NewMessage()
		saveData.SessionID = h.session.ID
		saveData.Role = consts.RoleToolCall
		saveData.ToolCallID = toolCallID
		saveData.ToolName = toolName
		saveData.ToolArguments = arguments
		err := h.storage.Message().CreateOrUpdate(saveData)
		if err != nil {
			h.logger.Error("保存工具调用消息失败", "tool", toolName, "error", err)
		}
	}

	if h.bus != nil {
		metadata := map[string]any{"client_id": h.clientID}

		msg := bus.OutboundMessage{
			Type:       bus.MessageTypeToolCall,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Arguments:  arguments,
			Status:     bus.MessageStatusRunning,
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   metadata,
		}

		h.bus.PublishOutbound(ctx, msg)
	}
	return nil
}

func (h *LoopHooks) OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error {
	if h.storage != nil && h.session != nil {
		resultContent := result.Content
		if result.Error != nil {
			resultContent = fmt.Sprintf("错误内容: %v", result.Error)
		}

		saveData := storage.NewMessage()
		saveData.SessionID = h.session.ID
		saveData.Role = consts.RoleToolResult
		saveData.ToolCallID = toolCallID
		saveData.ToolName = toolName
		saveData.ToolResult = resultContent
		saveData.ToolResultError = result.Error.Error()

		err := h.storage.Message().Create(saveData)
		if err != nil {
			h.logger.Error("保存工具结果消息失败", "tool", toolName, "error", err)
		}
	}

	if h.bus != nil {
		status := bus.MessageStatusCompleted
		errorMsg := ""
		if result.Error != nil {
			status = bus.MessageStatusError
			errorMsg = result.Error.Error()
		}

		metadata := map[string]any{"client_id": h.clientID}

		msg := bus.OutboundMessage{
			Type:       bus.MessageTypeToolResult,
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Content:    result.Content,
			Error:      errorMsg,
			Status:     status,
			ChatID:     h.chatID,
			Timestamp:  time.Now(),
			Metadata:   metadata,
		}
		h.bus.PublishOutbound(ctx, msg)
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
