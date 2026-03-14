package hooks

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// ReactHooks React钩子接口
type ReactHooks interface {
	// GetProvider 获取供应商实例
	GetProvider(ctx context.Context, defaultModel string, storage *storage.ProviderStorage) (providers.Provider, string, error)

	MessageHooks // MessageHooks 消息钩子
	LLMHooks     // LLMHooks LLM钩子
	ToolHooks    // ToolHooks 工具钩子
}

// MessageHooks 消息钩子
type MessageHooks interface {
	// BuildMessagesBefore 构建消息列表前钩子
	BuildMessagesBefore(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error)
	// BuildMessagesAfter 构建消息列表后钩子
	BuildMessagesAfter(ctx context.Context, sessionKey string, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error)
}

// LLMHooks LLM钩子
type LLMHooks interface {
	// RunLLMBefore 运行LLM模型前钩子
	RunLLMBefore(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error)
	// RunLLMAfter 运行LLM模型后钩子
	RunLLMAfter(ctx context.Context, msg bus.InboundMessage, history []providers.ChatMessage) ([]providers.ChatMessage, error)
}

// ToolHooks 工具钩子
type ToolHooks interface {
	// ToolCallBefore 工具调用前钩子
	ToolCallBefore(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (providers.ToolCall, error)
	// ToolCallAfter 工具调用后钩子
	ToolCallAfter(ctx context.Context, toolName string, msg bus.InboundMessage, result *tools.Result) error
	// ToolParseArguments 工具参数解析钩子
	ToolParseArguments(ctx context.Context, toolName string, tc providers.ToolCall, msg bus.InboundMessage) (map[string]any, error)
}
