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
	ID        string
	Channel   string
	ChatID    string
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// MessageBus 异步消息队列
type MessageBus struct {
	inbound     chan InboundMessage
	outbound    chan OutboundMessage
	subscribers map[string]chan InboundMessage
	mu          sync.RWMutex
	logger      *slog.Logger
	bufferSize  int
}

// NewMessageBus 创建消息总线
func NewMessageBus(bufferSize ...int) *MessageBus {
	size := 100
	if len(bufferSize) > 0 {
		size = bufferSize[0]
	}

	return &MessageBus{
		inbound:     make(chan InboundMessage, size),
		outbound:    make(chan OutboundMessage, size),
		subscribers: make(map[string]chan InboundMessage),
		logger:      slog.Default(),
		bufferSize:  size,
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

// PublishOutbound 发布发送消息
func (b *MessageBus) PublishOutbound(ctx context.Context, msg OutboundMessage) error {
	select {
	case b.outbound <- msg:
		b.logger.Debug("Published outbound message", "channel", msg.Channel, "chat_id", msg.ChatID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		b.logger.Warn("Outbound channel full, dropping message")
		return ErrChannelFull
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
