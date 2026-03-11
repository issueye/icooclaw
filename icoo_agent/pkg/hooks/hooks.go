// Package hooks provides hook interfaces for icooclaw.
package hooks

import (
	"context"

	"icooclaw/pkg/providers"
)

// AgentHooks defines hooks for agent lifecycle events.
type AgentHooks interface {
	// OnMessageReceived is called when a message is received.
	OnMessageReceived(ctx context.Context, channel, sessionID, text string) error

	// OnMessageSent is called when a message is sent.
	OnMessageSent(ctx context.Context, channel, sessionID, text string) error

	// OnToolCall is called when a tool is invoked.
	OnToolCall(ctx context.Context, toolName string, args map[string]any) error

	// OnToolResult is called when a tool returns a result.
	OnToolResult(ctx context.Context, toolName string, result string, err error) error

	// OnError is called when an error occurs.
	OnError(ctx context.Context, err error, context map[string]any) error
}

// ProviderHooks defines hooks for provider events.
type ProviderHooks interface {
	// OnRequest is called before a provider request.
	OnRequest(ctx context.Context, provider string, req *providers.ChatRequest) error

	// OnResponse is called after a provider response.
	OnResponse(ctx context.Context, provider string, resp *providers.ChatResponse) error

	// OnStreamChunk is called for each streaming chunk.
	OnStreamChunk(ctx context.Context, provider string, chunk string) error

	// OnFailover is called when a failover occurs.
	OnFailover(ctx context.Context, fromProvider, toProvider string, reason error) error
}

// ReActHooks defines hooks for ReAct loop events.
type ReActHooks interface {
	// OnThought is called when the agent thinks.
	OnThought(ctx context.Context, thought string) error

	// OnAction is called when the agent takes an action.
	OnAction(ctx context.Context, action string, args map[string]any) error

	// OnObservation is called when an observation is made.
	OnObservation(ctx context.Context, observation string) error

	// OnFinalAnswer is called when the final answer is ready.
	OnFinalAnswer(ctx context.Context, answer string) error
}

// ChannelHooks defines hooks for channel events.
type ChannelHooks interface {
	// OnChannelStart is called when a channel starts.
	OnChannelStart(ctx context.Context, channel string) error

	// OnChannelStop is called when a channel stops.
	OnChannelStop(ctx context.Context, channel string) error

	// OnMessageInbound is called for inbound messages.
	OnMessageInbound(ctx context.Context, channel string, msg any) error

	// OnMessageOutbound is called for outbound messages.
	OnMessageOutbound(ctx context.Context, channel string, msg any) error
}

// MemoryHooks defines hooks for memory events.
type MemoryHooks interface {
	// OnMemoryLoad is called when memory is loaded.
	OnMemoryLoad(ctx context.Context, sessionKey string, count int) error

	// OnMemorySave is called when memory is saved.
	OnMemorySave(ctx context.Context, sessionKey, role, content string) error

	// OnMemoryClear is called when memory is cleared.
	OnMemoryClear(ctx context.Context, sessionKey string) error

	// OnSummary is called when a summary is generated.
	OnSummary(ctx context.Context, sessionKey string, summary string) error
}

// CompositeHooks combines multiple hooks.
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
