// Package hooks provides hook interfaces for icooclaw.
package hooks

import (
	"context"

	"icooclaw/pkg/providers"
)

// AgentHooks 定义智能体生命周期事件的钩子接口。
type AgentHooks interface {
	// OnStart 当智能体启动时调用。
	OnStart(ctx context.Context, channel string, sessionID string) error

	// OnStop 当智能体停止时调用。
	OnStop(ctx context.Context, channel string, sessionID string) error

	// OnMessageReceived 当收到消息时调用。
	OnMessageReceived(ctx context.Context, channel, sessionID, text string) error

	// OnMessageSent 当发送消息时调用。
	OnMessageSent(ctx context.Context, channel, sessionID, text string) error

	// OnToolCall 当调用工具时调用。
	OnToolCall(ctx context.Context, toolName string, args map[string]any) error

	// OnToolResult 当工具调用结果时调用。
	OnToolResult(ctx context.Context, toolName string, result string, err error) error

	// OnError 当错误发生时调用。
	OnError(ctx context.Context, err error, context map[string]any) error
}

// ProviderHooks 定义提供程序事件的钩子接口。
type ProviderHooks interface {
	// OnRequest 当提供程序请求时调用。
	OnRequest(ctx context.Context, provider string, req *providers.ChatRequest) error

	// OnResponse 当提供程序响应时调用。
	OnResponse(ctx context.Context, provider string, resp *providers.ChatResponse) error

	// OnStreamChunk 当提供程序响应流时调用。
	OnStreamChunk(ctx context.Context, provider string, chunk string) error

	// OnFailover 当提供程序故障时调用。
	OnFailover(ctx context.Context, fromProvider, toProvider string, reason error) error
}

// ReActHooks 定义ReAct循环事件的钩子接口。
type ReActHooks interface {
	// OnThought 当智能体思考时调用。
	OnThought(ctx context.Context, thought string) error

	// OnAction 当智能体执行操作时调用。
	OnAction(ctx context.Context, action string, args map[string]any) error

	// OnObservation 当智能体观察时调用。
	OnObservation(ctx context.Context, observation string) error

	// OnFinalAnswer 当智能体完成时调用。
	OnFinalAnswer(ctx context.Context, answer string) error
}

// ChannelHooks 定义通道事件的钩子接口。
type ChannelHooks interface {
	// OnChannelStart 当通道启动时调用。
	OnChannelStart(ctx context.Context, channel string) error

	// OnChannelStop 当通道停止时调用。
	OnChannelStop(ctx context.Context, channel string) error

	// OnMessageInbound 当通道收到消息时调用。
	OnMessageInbound(ctx context.Context, channel string, msg any) error

	// OnMessageOutbound 当通道发送消息时调用。
	OnMessageOutbound(ctx context.Context, channel string, msg any) error
}

// MemoryHooks 定义内存事件的钩子接口。
type MemoryHooks interface {
	// OnMemoryLoad 当内存加载时调用。
	OnMemoryLoad(ctx context.Context, sessionKey string, count int) error

	// OnMemorySave 当内存保存时调用。
	OnMemorySave(ctx context.Context, sessionKey, role, content string) error

	// OnMemoryClear 当内存清除时调用。
	OnMemoryClear(ctx context.Context, sessionKey string) error

	// OnSummary 当生成摘要时调用。
	OnSummary(ctx context.Context, sessionKey string, summary string) error
}

// CompositeHooks 组合钩子接口。
type CompositeHooks struct {
	agentHooks    []AgentHooks
	providerHooks []ProviderHooks
	reactHooks    []ReActHooks
	channelHooks  []ChannelHooks
	memoryHooks   []MemoryHooks
}

// NewCompositeHooks creates new composite hooks.
func NewCompositeHooks() *CompositeHooks {
	return &CompositeHooks{}
}

// AddAgentHooks adds agent hooks.
func (c *CompositeHooks) AddAgentHooks(h AgentHooks) {
	c.agentHooks = append(c.agentHooks, h)
}

// AddProviderHooks adds provider hooks.
func (c *CompositeHooks) AddProviderHooks(h ProviderHooks) {
	c.providerHooks = append(c.providerHooks, h)
}

// AddReActHooks adds ReAct hooks.
func (c *CompositeHooks) AddReActHooks(h ReActHooks) {
	c.reactHooks = append(c.reactHooks, h)
}

// AddChannelHooks adds channel hooks.
func (c *CompositeHooks) AddChannelHooks(h ChannelHooks) {
	c.channelHooks = append(c.channelHooks, h)
}

// AddMemoryHooks adds memory hooks.
func (c *CompositeHooks) AddMemoryHooks(h MemoryHooks) {
	c.memoryHooks = append(c.memoryHooks, h)
}

// OnMessageReceived calls all agent hooks.
func (c *CompositeHooks) OnMessageReceived(ctx context.Context, channel, sessionID, text string) error {
	for _, h := range c.agentHooks {
		if err := h.OnMessageReceived(ctx, channel, sessionID, text); err != nil {
			return err
		}
	}
	return nil
}

// OnMessageSent calls all agent hooks.
func (c *CompositeHooks) OnMessageSent(ctx context.Context, channel, sessionID, text string) error {
	for _, h := range c.agentHooks {
		if err := h.OnMessageSent(ctx, channel, sessionID, text); err != nil {
			return err
		}
	}
	return nil
}

// OnToolCall calls all agent hooks.
func (c *CompositeHooks) OnToolCall(ctx context.Context, toolName string, args map[string]any) error {
	for _, h := range c.agentHooks {
		if err := h.OnToolCall(ctx, toolName, args); err != nil {
			return err
		}
	}
	return nil
}

// OnToolResult calls all agent hooks.
func (c *CompositeHooks) OnToolResult(ctx context.Context, toolName string, result string, err error) error {
	for _, h := range c.agentHooks {
		if err := h.OnToolResult(ctx, toolName, result, err); err != nil {
			return err
		}
	}
	return nil
}

// OnError calls all agent hooks.
func (c *CompositeHooks) OnError(ctx context.Context, err error, context map[string]any) error {
	for _, h := range c.agentHooks {
		if e := h.OnError(ctx, err, context); e != nil {
			return e
		}
	}
	return nil
}

// OnRequest calls all provider hooks.
func (c *CompositeHooks) OnRequest(ctx context.Context, provider string, req *providers.ChatRequest) error {
	for _, h := range c.providerHooks {
		if err := h.OnRequest(ctx, provider, req); err != nil {
			return err
		}
	}
	return nil
}

// OnResponse calls all provider hooks.
func (c *CompositeHooks) OnResponse(ctx context.Context, provider string, resp *providers.ChatResponse) error {
	for _, h := range c.providerHooks {
		if err := h.OnResponse(ctx, provider, resp); err != nil {
			return err
		}
	}
	return nil
}

// OnStreamChunk calls all provider hooks.
func (c *CompositeHooks) OnStreamChunk(ctx context.Context, provider string, chunk string) error {
	for _, h := range c.providerHooks {
		if err := h.OnStreamChunk(ctx, provider, chunk); err != nil {
			return err
		}
	}
	return nil
}

// OnFailover calls all provider hooks.
func (c *CompositeHooks) OnFailover(ctx context.Context, fromProvider, toProvider string, reason error) error {
	for _, h := range c.providerHooks {
		if err := h.OnFailover(ctx, fromProvider, toProvider, reason); err != nil {
			return err
		}
	}
	return nil
}

// OnThought calls all ReAct hooks.
func (c *CompositeHooks) OnThought(ctx context.Context, thought string) error {
	for _, h := range c.reactHooks {
		if err := h.OnThought(ctx, thought); err != nil {
			return err
		}
	}
	return nil
}

// OnAction calls all ReAct hooks.
func (c *CompositeHooks) OnAction(ctx context.Context, action string, args map[string]any) error {
	for _, h := range c.reactHooks {
		if err := h.OnAction(ctx, action, args); err != nil {
			return err
		}
	}
	return nil
}

// OnObservation calls all ReAct hooks.
func (c *CompositeHooks) OnObservation(ctx context.Context, observation string) error {
	for _, h := range c.reactHooks {
		if err := h.OnObservation(ctx, observation); err != nil {
			return err
		}
	}
	return nil
}

// OnFinalAnswer calls all ReAct hooks.
func (c *CompositeHooks) OnFinalAnswer(ctx context.Context, answer string) error {
	for _, h := range c.reactHooks {
		if err := h.OnFinalAnswer(ctx, answer); err != nil {
			return err
		}
	}
	return nil
}

// LoggingHooks logs all hook events.
type LoggingHooks struct {
	logger interface {
		Debug(msg string, args ...any)
		Info(msg string, args ...any)
		Error(msg string, args ...any)
	}
}

// OnMessageReceived logs message received.
func (h *LoggingHooks) OnMessageReceived(ctx context.Context, channel, sessionID, text string) error {
	h.logger.Debug("message received", "channel", channel, "session_id", sessionID, "text", text)
	return nil
}

// OnMessageSent logs message sent.
func (h *LoggingHooks) OnMessageSent(ctx context.Context, channel, sessionID, text string) error {
	h.logger.Debug("message sent", "channel", channel, "session_id", sessionID)
	return nil
}

// OnToolCall logs tool call.
func (h *LoggingHooks) OnToolCall(ctx context.Context, toolName string, args map[string]any) error {
	h.logger.Debug("tool call", "tool", toolName, "args", args)
	return nil
}

// OnToolResult logs tool result.
func (h *LoggingHooks) OnToolResult(ctx context.Context, toolName string, result string, err error) error {
	if err != nil {
		h.logger.Error("tool result", "tool", toolName, "error", err)
	} else {
		h.logger.Debug("tool result", "tool", toolName, "result", result)
	}
	return nil
}

// OnError logs error.
func (h *LoggingHooks) OnError(ctx context.Context, err error, context map[string]any) error {
	h.logger.Error("error", "error", err, "context", context)
	return nil
}

// OnRequest logs provider request.
func (h *LoggingHooks) OnRequest(ctx context.Context, provider string, req *providers.ChatRequest) error {
	h.logger.Debug("provider request", "provider", provider, "model", req.Model)
	return nil
}

// OnResponse logs provider response.
func (h *LoggingHooks) OnResponse(ctx context.Context, provider string, resp *providers.ChatResponse) error {
	h.logger.Debug("provider response", "provider", provider, "model", resp.Model)
	return nil
}

// OnStreamChunk logs stream chunk.
func (h *LoggingHooks) OnStreamChunk(ctx context.Context, provider string, chunk string) error {
	// Don't log every chunk, too noisy
	return nil
}

// OnFailover logs failover.
func (h *LoggingHooks) OnFailover(ctx context.Context, fromProvider, toProvider string, reason error) error {
	h.logger.Info("provider failover", "from", fromProvider, "to", toProvider, "reason", reason)
	return nil
}

// OnThought logs thought.
func (h *LoggingHooks) OnThought(ctx context.Context, thought string) error {
	h.logger.Debug("thought", "content", thought)
	return nil
}

// OnAction logs action.
func (h *LoggingHooks) OnAction(ctx context.Context, action string, args map[string]any) error {
	h.logger.Debug("action", "action", action, "args", args)
	return nil
}

// OnObservation logs observation.
func (h *LoggingHooks) OnObservation(ctx context.Context, observation string) error {
	h.logger.Debug("observation", "content", observation)
	return nil
}

// OnFinalAnswer logs final answer.
func (h *LoggingHooks) OnFinalAnswer(ctx context.Context, answer string) error {
	h.logger.Info("final answer", "content", answer)
	return nil
}
