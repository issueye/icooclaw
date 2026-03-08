// Package agent provides the core agent implementation for icooclaw.
package agent

import (
	"context"
	"log/slog"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
)

// Agent represents an AI agent.
type Agent struct {
	name      string
	workspace string
	sessionID string

	// Core dependencies
	provider providers.Provider
	tools    *tools.Registry
	memory   memory.Loader
	skills   skill.Loader
	storage  *storage.Storage
	bus      *bus.MessageBus

	logger *slog.Logger
}

// New creates a new Agent.
func New(name string, opts ...Option) *Agent {
	a := &Agent{
		name:   name,
		tools:  tools.NewRegistry(),
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Option is a functional option for Agent.
type Option func(*Agent)

// WithProvider sets the provider.
func WithProvider(p providers.Provider) Option {
	return func(a *Agent) { a.provider = p }
}

// WithTools sets the tools registry.
func WithTools(t *tools.Registry) Option {
	return func(a *Agent) { a.tools = t }
}

// WithMemory sets the memory loader.
func WithMemory(m memory.Loader) Option {
	return func(a *Agent) { a.memory = m }
}

// WithSkills sets the skill loader.
func WithSkills(s skill.Loader) Option {
	return func(a *Agent) { a.skills = s }
}

// WithStorage sets the storage.
func WithStorage(s *storage.Storage) Option {
	return func(a *Agent) { a.storage = s }
}

// WithBus sets the message bus.
func WithBus(b *bus.MessageBus) Option {
	return func(a *Agent) { a.bus = b }
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(a *Agent) { a.logger = l }
}

// WithWorkspace sets the workspace.
func WithWorkspace(w string) Option {
	return func(a *Agent) { a.workspace = w }
}

// Name returns the agent name.
func (a *Agent) Name() string {
	return a.name
}

// Run starts the agent message processing loop.
func (a *Agent) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-a.bus.Inbound():
			go a.processMessage(ctx, msg)
		}
	}
}

// processMessage processes an inbound message.
func (a *Agent) processMessage(ctx context.Context, msg bus.InboundMessage) {
	// Build context
	sessionKey := a.getSessionKey(msg.Channel, msg.ChatID)

	// Load memory
	var history []providers.ChatMessage
	if a.memory != nil {
		mem, err := a.memory.Load(ctx, sessionKey)
		if err != nil {
			a.logger.Warn("failed to load memory", "error", err)
		} else {
			history = mem
		}
	}

	// Add user message
	history = append(history, providers.ChatMessage{
		Role:    "user",
		Content: msg.Text,
	})

	// Build request
	req := providers.ChatRequest{
		Model:    a.provider.GetDefaultModel(),
		Messages: history,
		Tools:    a.getToolDefinitions(),
	}

	// Send to provider
	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		a.logger.Error("failed to get response", "error", err)
		return
	}

	// Handle tool calls
	if len(resp.ToolCalls) > 0 {
		a.handleToolCalls(ctx, msg, resp.ToolCalls, history)
		return
	}

	// Send response
	a.bus.PublishOutbound(bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Text:    resp.Content,
	})

	// Save memory
	if a.memory != nil {
		a.memory.Save(ctx, sessionKey, "user", msg.Text)
		a.memory.Save(ctx, sessionKey, "assistant", resp.Content)
	}
}

// handleToolCalls handles tool calls from the provider.
func (a *Agent) handleToolCalls(ctx context.Context, msg bus.InboundMessage, toolCalls []providers.ToolCall, history []providers.ChatMessage) {
	for _, tc := range toolCalls {
		result := a.tools.ExecuteWithContext(ctx, tc.Function.Name, nil, msg.Channel, msg.ChatID, nil)

		// Add tool result to history
		history = append(history, providers.ChatMessage{
			Role:    "assistant",
			Content: "",
		})
		history = append(history, providers.ChatMessage{
			Role:    "tool",
			Content: result.Content,
			Name:    tc.Function.Name,
		})
	}

	// Continue conversation with tool results
	req := providers.ChatRequest{
		Model:    a.provider.GetDefaultModel(),
		Messages: history,
		Tools:    a.getToolDefinitions(),
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		a.logger.Error("failed to get response after tool calls", "error", err)
		return
	}

	// Send response
	a.bus.PublishOutbound(bus.OutboundMessage{
		Channel: msg.Channel,
		ChatID:  msg.ChatID,
		Text:    resp.Content,
	})
}

// getToolDefinitions returns tool definitions for the provider.
func (a *Agent) getToolDefinitions() []providers.Tool {
	defs := a.tools.GetToolDefinitions()
	tools := make([]providers.Tool, 0, len(defs))
	for _, def := range defs {
		if fn, ok := def["function"].(map[string]any); ok {
			t := providers.Tool{
				Type: "function",
				Function: providers.Function{
					Name:        fn["name"].(string),
					Description: fn["description"].(string),
					Parameters:  fn["parameters"].(map[string]any),
				},
			}
			tools = append(tools, t)
		}
	}
	return tools
}

// getSessionKey returns a session key for the given channel and chat ID.
func (a *Agent) getSessionKey(channel, chatID string) string {
	return channel + ":" + chatID
}
