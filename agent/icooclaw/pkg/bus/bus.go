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
		InboundCapacity:  1000,
		OutboundCapacity: 1000,
	}
}

// NewMessageBus creates a new MessageBus.
func NewMessageBus(cfg Config) *MessageBus {
	if cfg.InboundCapacity <= 0 {
		cfg.InboundCapacity = 1000
	}
	if cfg.OutboundCapacity <= 0 {
		cfg.OutboundCapacity = 1000
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

// PublishInbound publishes an inbound message.
func (mb *MessageBus) PublishInbound(msg InboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
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
	default:
		mb.dropCount.Add(1)
		return errors.ErrBufferFull
	}
}

// PublishOutbound publishes an outbound message.
func (mb *MessageBus) PublishOutbound(msg OutboundMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	select {
	case mb.outbound <- msg:
		return nil
	default:
		mb.dropCount.Add(1)
		return errors.ErrBufferFull
	}
}

// PublishOutboundMedia publishes an outbound media message.
func (mb *MessageBus) PublishOutboundMedia(msg OutboundMediaMessage) error {
	if mb.closed.Load() {
		return errors.ErrNotRunning
	}

	select {
	case mb.outboundMedia <- msg:
		return nil
	default:
		mb.dropCount.Add(1)
		return errors.ErrBufferFull
	}
}

// Inbound returns the inbound message channel.
func (mb *MessageBus) Inbound() <-chan InboundMessage {
	return mb.inbound
}

// Outbound returns the outbound message channel.
func (mb *MessageBus) Outbound() <-chan OutboundMessage {
	return mb.outbound
}

// OutboundMedia returns the outbound media message channel.
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

// Close closes the message bus.
func (mb *MessageBus) Close() {
	if mb.closed.CompareAndSwap(false, true) {
		close(mb.done)

		// Drain buffered channels
		for len(mb.inbound) > 0 {
			<-mb.inbound
		}
		for len(mb.outbound) > 0 {
			<-mb.outbound
		}
		for len(mb.outboundMedia) > 0 {
			<-mb.outboundMedia
		}

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
