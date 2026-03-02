package hooks

import (
	"context"

	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
)

// DefaultHooks 默认空实现
type DefaultHooks struct{}

func (h *DefaultHooks) OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error {
	return nil
}
func (h *DefaultHooks) OnLLMChunk(ctx context.Context, content, thinking string) error { return nil }
func (h *DefaultHooks) OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error {
	return nil
}
func (h *DefaultHooks) OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error {
	return nil
}
func (h *DefaultHooks) OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error {
	return nil
}
func (h *DefaultHooks) OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error {
	return nil
}
func (h *DefaultHooks) OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error {
	return nil
}
func (h *DefaultHooks) OnError(ctx context.Context, err error) error { return nil }
func (h *DefaultHooks) OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error {
	return nil
}
