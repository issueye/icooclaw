// Package bus provides message bus for inter-component communication.
package bus

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/errors"
)

// SenderInfo contains information about the message sender.
type SenderInfo struct {
	ID       string
	Name     string
	Username string
	IsBot    bool
}

// InboundMessage represents a message received from a channel.
type InboundMessage struct {
	Channel   string
	ChatID    string
	Sender    SenderInfo
	Text      string
	Media     []string
	ReplyTo   string
	Timestamp time.Time
	Metadata  map[string]any
}

// OutboundMessage represents a message to be sent to a channel.
type OutboundMessage struct {
	Channel  string
	ChatID   string
	Text     string
	Media    []string
	ReplyTo  string
	EditID   string
	Metadata map[string]any
}

// OutboundMediaMessage represents a media message to be sent.
type OutboundMediaMessage struct {
	Channel  string
	ChatID   string
	Media    []string
	Caption  string
	Metadata map[string]any
}

const defaultBusBufferSize = 64

// MessageBus provides pub/sub messaging between components.
type MessageBus struct {
	inbound       chan InboundMessage
	outbound      chan OutboundMessage
	outboundMedia chan OutboundMediaMessage
	done          chan struct{}
	closed        atomic.Bool

	// Backpressure control
	inboundCapacity  int
	outboundCapacity int
	dropCount        atomic.Int64

	// Subscribers
	inboundSubs  map[string]chan InboundMessage
	outboundSubs map[string]chan OutboundMessage
	mu           sync.RWMutex
}

// Config contains configuration for MessageBus.
type Config struct {
	InboundCapacity  int
	OutboundCapacity int
}

// DefaultConfig returns default configuration.
func DefaultConfig() Config {
	return Config{
		InboundCapacity:  defaultBusBufferSize,
		OutboundCapacity: defaultBusBufferSize,
	}
}

// NewMessageBus creates a new MessageBus.
func NewMessageBus(cfg Config) *MessageBus {
	if cfg.InboundCapacity <= 0 {
		cfg.InboundCapacity = defaultBusBufferSize
	}
	if cfg.OutboundCapacity <= 0 {
		cfg.OutboundCapacity = defaultBusBufferSize
	}

	return &MessageBus{
		inbound:          make(chan InboundMessage, cfg.InboundCapacity),
		outbound:         make(chan OutboundMessage, cfg.OutboundCapacity),
		outboundMedia:    make(chan OutboundMediaMessage, cfg.InboundCapacity),
		done:             make(chan struct{}),
		inboundCapacity:  cfg.InboundCapacity,
		outboundCapacity: cfg.OutboundCapacity,
		inboundSubs:      make(map[string]chan InboundMessage),
		outboundSubs:     make(map[string]chan OutboundMessage),
	}
}

// PublishInbound publishes an inbound message with context support.
// Returns ErrBusClosed if the bus is closed, or ctx.Err() if context is canceled.
func (mb *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	// Check context before attempting to send
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.inbound <- msg:
		// Also forward to subscribers
		mb.mu.RLock()
		if sub, ok := mb.inboundSubs["all"]; ok {
			select {
			case sub <- msg:
			default:
				// Subscriber buffer full, skip
			}
		}
		mb.mu.RUnlock()
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishInboundNoCtx publishes an inbound message without context (for backward compatibility).
// Deprecated: Use PublishInbound with context instead.
func (mb *MessageBus) PublishInboundNoCtx(msg InboundMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mb.PublishInbound(ctx, msg)
}

// ConsumeInbound consumes an inbound message from the bus.
// Returns the message and true if successful, or empty message and false if the bus is closed or context is canceled.
func (mb *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, bool) {
	select {
	case msg, ok := <-mb.inbound:
		return msg, ok
	case <-mb.done:
		return InboundMessage{}, false
	case <-ctx.Done():
		return InboundMessage{}, false
	}
}

// PublishOutbound publishes an outbound message with context support.
// Returns ErrBusClosed if the bus is closed, or ctx.Err() if context is canceled.
func (mb *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	// Check context before attempting to send
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.outbound <- msg:
		// Also forward to subscribers
		mb.mu.RLock()
		if sub, ok := mb.outboundSubs["all"]; ok {
			select {
			case sub <- msg:
			default:
				// Subscriber buffer full, skip
			}
		}
		mb.mu.RUnlock()
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// PublishOutboundNoCtx publishes an outbound message without context (for backward compatibility).
// Deprecated: Use PublishOutbound with context instead.
func (mb *MessageBus) PublishOutboundNoCtx(msg OutboundMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mb.PublishOutbound(ctx, msg)
}

// ConsumeOutbound consumes an outbound message from the bus.
// Returns the message and true if successful, or empty message and false if the bus is closed or context is canceled.
func (mb *MessageBus) ConsumeOutbound(ctx context.Context) (OutboundMessage, bool) {
	select {
	case msg, ok := <-mb.outbound:
		return msg, ok
	case <-mb.done:
		return OutboundMessage{}, false
	case <-ctx.Done():
		return OutboundMessage{}, false
	}
}

// PublishOutboundMedia publishes an outbound media message with context support.
func (mb *MessageBus) PublishOutboundMedia(ctx context.Context, msg OutboundMediaMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case mb.outboundMedia <- msg:
		return nil
	case <-mb.done:
		return errors.ErrNotRunning
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ConsumeOutboundMedia consumes an outbound media message from the bus.
func (mb *MessageBus) ConsumeOutboundMedia(ctx context.Context) (OutboundMediaMessage, bool) {
	select {
	case msg, ok := <-mb.outboundMedia:
		return msg, ok
	case <-mb.done:
		return OutboundMediaMessage{}, false
	case <-ctx.Done():
		return OutboundMediaMessage{}, false
	}
}

// Inbound returns the inbound message channel.
// Deprecated: Use ConsumeInbound for safer consumption with context.
func (mb *MessageBus) Inbound() <-chan InboundMessage {
	return mb.inbound
}

// Outbound returns the outbound message channel.
// Deprecated: Use ConsumeOutbound for safer consumption with context.
func (mb *MessageBus) Outbound() <-chan OutboundMessage {
	return mb.outbound
}

// OutboundMedia returns the outbound media message channel.
// Deprecated: Use ConsumeOutboundMedia for safer consumption with context.
func (mb *MessageBus) OutboundMedia() <-chan OutboundMediaMessage {
	return mb.outboundMedia
}

// SubscribeInbound subscribes to inbound messages.
func (mb *MessageBus) SubscribeInbound(name string, buffer int) <-chan InboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if buffer <= 0 {
		buffer = 100
	}

	ch := make(chan InboundMessage, buffer)
	mb.inboundSubs[name] = ch
	return ch
}

// SubscribeOutbound subscribes to outbound messages.
func (mb *MessageBus) SubscribeOutbound(name string, buffer int) <-chan OutboundMessage {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if buffer <= 0 {
		buffer = 100
	}

	ch := make(chan OutboundMessage, buffer)
	mb.outboundSubs[name] = ch
	return ch
}

// UnsubscribeInbound unsubscribes from inbound messages.
func (mb *MessageBus) UnsubscribeInbound(name string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if ch, ok := mb.inboundSubs[name]; ok {
		close(ch)
		delete(mb.inboundSubs, name)
	}
}

// UnsubscribeOutbound unsubscribes from outbound messages.
func (mb *MessageBus) UnsubscribeOutbound(name string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if ch, ok := mb.outboundSubs[name]; ok {
		close(ch)
		delete(mb.outboundSubs, name)
	}
}

// Close closes the message bus gracefully.
// It drains buffered messages to avoid send-on-closed panics from concurrent publishers.
// Channels are NOT closed to avoid send-on-closed panics from concurrent publishers.
func (mb *MessageBus) Close() {
	if mb.closed.CompareAndSwap(false, true) {
		close(mb.done)

		// Drain buffered channels so messages aren't silently lost.
		// Channels are NOT closed to avoid send-on-closed panics from concurrent publishers.
		drained := 0
		for {
			select {
			case <-mb.inbound:
				drained++
			default:
				goto doneInbound
			}
		}
	doneInbound:
		for {
			select {
			case <-mb.outbound:
				drained++
			default:
				goto doneOutbound
			}
		}
	doneOutbound:
		for {
			select {
			case <-mb.outboundMedia:
				drained++
			default:
				goto doneMedia
			}
		}
	doneMedia:
		_ = drained // Avoid unused variable warning

		// Close subscriber channels
		mb.mu.Lock()
		for _, ch := range mb.inboundSubs {
			close(ch)
		}
		for _, ch := range mb.outboundSubs {
			close(ch)
		}
		mb.inboundSubs = make(map[string]chan InboundMessage)
		mb.outboundSubs = make(map[string]chan OutboundMessage)
		mb.mu.Unlock()
	}
}

// Done returns the done channel.
func (mb *MessageBus) Done() <-chan struct{} {
	return mb.done
}

// IsClosed returns true if the bus is closed.
func (mb *MessageBus) IsClosed() bool {
	return mb.closed.Load()
}

// DropCount returns the number of dropped messages.
func (mb *MessageBus) DropCount() int64 {
	return mb.dropCount.Load()
}

// Run starts the message bus (for compatibility, does nothing as channels are already active).
func (mb *MessageBus) Run(ctx context.Context) error {
	<-ctx.Done()
	mb.Close()
	return ctx.Err()
}