package bus

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// InboundMessage 接收消息
type InboundMessage struct {
	ID        string
	Channel   string
	ChatID    string
	UserID    string
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// OutboundMessage 发送消息
type OutboundMessage struct {
	ID         string                 `json:"id,omitempty"`
	Type       string                 `json:"type"` // "message", "chunk", "chunk_end", "tool_call", "tool_result", "error", "thinking"
	Channel    string                 `json:"channel,omitempty"`
	ChatID     string                 `json:"chat_id,omitempty"`
	Content    string                 `json:"content,omitempty"`
	Thinking   string                 `json:"thinking,omitempty"`
	ToolName   string                 `json:"tool_name,omitempty"`
	ToolCallID string                 `json:"tool_call_id,omitempty"`
	Arguments  string                 `json:"arguments,omitempty"`
	Status     string                 `json:"status,omitempty"` // "running", "completed", "error"
	Error      string                 `json:"error,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MessageBus 异步消息队列
type MessageBus struct {
	inbound             chan InboundMessage
	outbound            chan OutboundMessage
	subscribers         map[string]chan InboundMessage
	outboundSubscribers map[string]chan OutboundMessage
	mu                  sync.RWMutex
	logger              *slog.Logger
	bufferSize          int
}

// NewMessageBus 创建消息总线
func NewMessageBus(bufferSize ...int) *MessageBus {
	size := 100
	if len(bufferSize) > 0 {
		size = bufferSize[0]
	}

	return &MessageBus{
		inbound:             make(chan InboundMessage, size),
		outbound:            make(chan OutboundMessage, size),
		subscribers:         make(map[string]chan InboundMessage),
		outboundSubscribers: make(map[string]chan OutboundMessage),
		logger:              slog.Default(),
		bufferSize:          size,
	}
}

// SetLogger 设置日志
func (b *MessageBus) SetLogger(logger *slog.Logger) {
	b.logger = logger
}

// PublishInbound 发布接收消息
func (b *MessageBus) PublishInbound(ctx context.Context, msg InboundMessage) error {
	select {
	case b.inbound <- msg:
		b.logger.Debug("Published inbound message", "channel", msg.Channel, "chat_id", msg.ChatID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		b.logger.Warn("Inbound channel full, dropping message")
		return ErrChannelFull
	}
}

// SubscribeInbound 订阅接收消息
func (b *MessageBus) SubscribeInbound(channel string) <-chan InboundMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[channel]; !ok {
		b.subscribers[channel] = make(chan InboundMessage, b.bufferSize)
	}
	return b.subscribers[channel]
}

// UnsubscribeInbound 取消订阅
func (b *MessageBus) UnsubscribeInbound(channel string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subscribers[channel]; ok {
		close(ch)
		delete(b.subscribers, channel)
	}
}

// SubscribeOutbound 订阅发送消息
func (b *MessageBus) SubscribeOutbound(channel string) <-chan OutboundMessage {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.outboundSubscribers[channel]; !ok {
		b.outboundSubscribers[channel] = make(chan OutboundMessage, b.bufferSize)
	}
	return b.outboundSubscribers[channel]
}

// UnsubscribeOutbound 取消订阅发送消息
func (b *MessageBus) UnsubscribeOutbound(channel string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.outboundSubscribers[channel]; ok {
		close(ch)
		delete(b.outboundSubscribers, channel)
	}
}

// PublishOutbound 发布发送消息
func (b *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 优先分发给订阅者
	for _, ch := range b.outboundSubscribers {
		select {
		case ch <- msg:
		default:
			// 忽略阻塞的订阅者或进行日志记录
		}
	}

	select {
	case b.outbound <- msg:
		b.logger.Debug("Published outbound message", "channel", msg.Channel, "chat_id", msg.ChatID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 如果主通道满了，但订阅者可能已经收到了，这里为了简单返回成功或 Warn
		b.logger.Warn("Outbound channel full")
		return nil
	}
}

// ConsumeInbound 消费接收消息
func (b *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, error) {
	select {
	case msg := <-b.inbound:
		return msg, nil
	case <-ctx.Done():
		return InboundMessage{}, ctx.Err()
	}
}

// ConsumeOutbound 消费发送消息
func (b *MessageBus) ConsumeOutbound(ctx context.Context) (OutboundMessage, error) {
	select {
	case msg := <-b.outbound:
		return msg, nil
	case <-ctx.Done():
		return OutboundMessage{}, ctx.Err()
	}
}

// InboundChannel 返回接收消息通道
func (b *MessageBus) InboundChannel() <-chan InboundMessage {
	return b.inbound
}

// OutboundChannel 返回发送消息通道
func (b *MessageBus) OutboundChannel() <-chan OutboundMessage {
	return b.outbound
}

// Close 关闭消息总线
func (b *MessageBus) Close() {
	close(b.inbound)
	close(b.outbound)

	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		close(ch)
	}
	b.subscribers = make(map[string]chan InboundMessage)

	for _, ch := range b.outboundSubscribers {
		close(ch)
	}
	b.outboundSubscribers = make(map[string]chan OutboundMessage)
}

// 错误定义
var (
	ErrChannelFull = &BusError{"channel is full"}
)

type BusError struct {
	msg string
}

func (e *BusError) Error() string {
	return e.msg
}
