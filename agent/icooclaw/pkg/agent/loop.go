package agent

import (
	"context"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/memory"
	"icooclaw/pkg/providers"
	"icooclaw/pkg/skill"
	"icooclaw/pkg/storage"
	"icooclaw/pkg/tools"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Loop represents the main agent processing loop.
type Loop struct {
	bus            *bus.MessageBus
	provider       *providers.FallbackChain
	tools          *tools.Registry
	memory         memory.Loader
	skills         skill.Loader
	storage        *storage.Storage
	channelManager *channels.Manager

	running     atomic.Bool
	summarizing sync.Map

	logger *slog.Logger
}

// NewLoop creates a new agent loop.
func NewLoop(opts ...LoopOption) *Loop {
	l := &Loop{
		tools:  tools.NewRegistry(),
		logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LoopOption is a functional option for Loop.
type LoopOption func(*Loop)

// WithLoopBus sets the message bus.
func WithLoopBus(b *bus.MessageBus) LoopOption {
	return func(l *Loop) { l.bus = b }
}

// WithLoopProvider sets the provider chain.
func WithLoopProvider(p *providers.FallbackChain) LoopOption {
	return func(l *Loop) { l.provider = p }
}

// WithLoopTools sets the tools registry.
func WithLoopTools(t *tools.Registry) LoopOption {
	return func(l *Loop) { l.tools = t }
}

// WithLoopMemory sets the memory loader.
func WithLoopMemory(m memory.Loader) LoopOption {
	return func(l *Loop) { l.memory = m }
}

// WithLoopSkills sets the skill loader.
func WithLoopSkills(s skill.Loader) LoopOption {
	return func(l *Loop) { l.skills = s }
}

// WithLoopStorage sets the storage.
func WithLoopStorage(s *storage.Storage) LoopOption {
	return func(l *Loop) { l.storage = s }
}

// WithLoopChannelManager sets the channel manager.
func WithLoopChannelManager(cm *channels.Manager) LoopOption {
	return func(l *Loop) { l.channelManager = cm }
}

// WithLoopLogger sets the logger.
func WithLoopLogger(log *slog.Logger) LoopOption {
	return func(l *Loop) { l.logger = log }
}

// Run starts the agent loop.
func (l *Loop) Run(ctx context.Context) error {
	l.running.Store(true)
	defer l.running.Store(false)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-l.bus.Inbound():
			go l.processMessage(ctx, msg)
		}
	}
}

// processMessage processes an inbound message.
func (l *Loop) processMessage(ctx context.Context, msg bus.InboundMessage) {
	// Get binding
	binding, err := l.storage.Binding().GetBinding(msg.Channel, msg.ChatID)
	if err != nil {
		l.logger.Debug("no binding found", "channel", msg.Channel, "chat_id", msg.ChatID)
		return
	}

	// Process with agent
	l.processWithAgent(ctx, binding.AgentName, msg)
}

// processWithAgent processes a message with a specific agent.
func (l *Loop) processWithAgent(ctx context.Context, agentName string, msg bus.InboundMessage) {
	// Implementation similar to Agent.processMessage
	l.logger.Info("processing message", "agent", agentName, "channel", msg.Channel, "chat_id", msg.ChatID)

	// Process with agent
}

// IsRunning returns true if the loop is running.
func (l *Loop) IsRunning() bool {
	return l.running.Load()
}
