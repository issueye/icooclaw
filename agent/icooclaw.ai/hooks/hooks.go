package hooks

import (
	"context"

	"icooclaw.ai/provider"
	"icooclaw.ai/tools"
)

type OnChunkFunc func(content, thinking string)

type HookFunc func(ctx context.Context, data interface{}) error

type ReActHooks interface {
	OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error
	OnLLMChunk(ctx context.Context, content, thinking string) error
	OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error
	OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error
	OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error
	OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error
	OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error
	OnError(ctx context.Context, err error) error
	OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error
}
