package bus

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type MessageType string

const (
	MessageTypeMessage    MessageType = "message"
	MessageTypeChunk      MessageType = "chunk"
	MessageTypeEnd        MessageType = "chunk_end"
	MessageTypeToolCall   MessageType = "tool_call"
	MessageTypeToolResult MessageType = "tool_result"
	MessageTypeError      MessageType = "error"
	MessageTypeThinking   MessageType = "thinking"
)

func (m MessageType) ToString() string {
	return string(m)
}

type MessageStatus string

const (
	MessageStatusRunning   MessageStatus = "running"
	MessageStatusCompleted MessageStatus = "completed"
	MessageStatusError     MessageStatus = "error"
)

func (s MessageStatus) ToString() string {
	return string(s)
}

// InboundMessage 接收消息
type InboundMessage struct {
	ID        string         `json:"id,omitempty"`         // 消息ID
	SessionID string         `json:"session_id,omitempty"` // 会话 ID
	Channel   string         `json:"channel,omitempty"`    // 通道
	ChatID    string         `json:"chat_id,omitempty"`    // 会话ID
	UserID    string         `json:"user_id,omitempty"`    // 用户ID
	Content   string         `json:"content,omitempty"`    // 内容
	Timestamp time.Time      `json:"timestamp,omitempty"`  // 时间戳
	Metadata  map[string]any `json:"metadata,omitempty"`   // 元数据
}

// OutboundMessage 发送消息
type OutboundMessage struct {
	ID         string         `json:"id,omitempty"`           // 消息ID
	Type       MessageType    `json:"type,omitempty"`         // 消息类型
	Channel    string         `json:"channel,omitempty"`      // 通道
	ChatID     string         `json:"chat_id,omitempty"`      // 会话ID
	Content    string         `json:"content,omitempty"`      // 内容
	Thinking   string         `json:"thinking,omitempty"`     // 思考内容
	ToolName   string         `json:"tool_name,omitempty"`    // 工具名称
	ToolCallID string         `json:"tool_call_id,omitempty"` // 工具调用ID
	Arguments  string         `json:"arguments,omitempty"`    // 工具调用参数
	Status     MessageStatus  `json:"status,omitempty"`       // 状态
	Error      string         `json:"error,omitempty"`        // 错误信息
	Timestamp  time.Time      `json:"timestamp,omitempty"`    // 时间戳
	Metadata   map[string]any `json:"metadata,omitempty"`     // 元数据
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
	inboundDropped      int64
	outboundDropped     int64
	droppedAlerted      bool
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
	// 获取客户端 ID 用于追踪
	clientID := ""
	if msg.Metadata != nil {
		if id, ok := msg.Metadata["client_id"].(string); ok {
			clientID = id
		}
	}

	content := msg.Content
	if len(content) > 100 {
		content = content[:100] + "..."
	}

	b.logger.Info("[消息总线] 发布入站消息",
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"user_id", msg.UserID,
		"client_id", clientID,
		"content_length", len(msg.Content),
		"content_preview", content,
	)

	select {
	case b.inbound <- msg:
		b.logger.Debug("[消息总线] 入站消息已写入通道",
			"channel", msg.Channel,
			"chat_id", msg.ChatID,
		)
		return nil
	case <-ctx.Done():
		b.logger.Warn("[消息总线] 上下文已取消，无法发布入站消息",
			"channel", msg.Channel,
			"chat_id", msg.ChatID,
		)
		return ctx.Err()
	default:
		b.mu.Lock()
		b.inboundDropped++
		dropped := b.inboundDropped
		b.mu.Unlock()

		b.logger.Error("[消息总线] 入站通道已满，消息被丢弃",
			"channel", msg.Channel,
			"chat_id", msg.ChatID,
			"buffer_size", b.bufferSize,
			"total_dropped", dropped,
		)

		if !b.droppedAlerted && dropped >= int64(b.bufferSize) {
			b.droppedAlerted = true
			b.logger.Error("[消息总线] 消息丢弃超过缓冲区大小，请检查系统负载或增加缓冲区大小",
				"buffer_size", b.bufferSize,
				"dropped_count", dropped,
			)
		}

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
	b.mu.Lock()
	defer b.mu.Unlock()

	// 获取客户端 ID 用于追踪
	clientID := ""
	if msg.Metadata != nil {
		if id, ok := msg.Metadata["client_id"].(string); ok {
			clientID = id
		}
	}

	content := msg.Content
	if len(content) > 100 {
		content = content[:100] + "..."
	}

	b.logger.Info("[消息总线] 发布出站消息",
		"type", msg.Type,
		"channel", msg.Channel,
		"chat_id", msg.ChatID,
		"client_id", clientID,
		"content_length", len(content),
		"content_preview", content,
	)

	// 优先分发给订阅者
	for _, ch := range b.outboundSubscribers {
		select {
		case ch <- msg:
			b.logger.Debug("[消息总线] 出站消息已分发给订阅者",
				"type", msg.Type,
				"channel", msg.Channel,
				"chat_id", msg.ChatID,
			)
		default:
			b.logger.Warn("[消息总线] 订阅者通道已满，消息被丢弃",
				"type", msg.Type,
				"channel", msg.Channel,
				"chat_id", msg.ChatID,
			)
		}
	}

	select {
	case b.outbound <- msg:
		b.logger.Debug("[消息总线] 出站消息已写入通道",
			"type", msg.Type,
			"channel", msg.Channel,
			"chat_id", msg.ChatID,
		)
		return nil
	case <-ctx.Done():
		b.logger.Warn("[消息总线] 上下文已取消，无法发布出站消息",
			"type", msg.Type,
			"channel", msg.Channel,
		)
		return ctx.Err()
	default:
		b.outboundDropped++
		dropped := b.outboundDropped

		b.logger.Error("[消息总线] 出站通道已满，消息被丢弃",
			"type", msg.Type,
			"channel", msg.Channel,
			"buffer_size", b.bufferSize,
			"total_dropped", dropped,
		)

		if !b.droppedAlerted && dropped >= int64(b.bufferSize) {
			b.droppedAlerted = true
			b.logger.Error("[消息总线] 消息丢弃超过缓冲区大小，请检查系统负载或增加缓冲区大小",
				"buffer_size", b.bufferSize,
				"dropped_count", dropped,
			)
		}

		return ErrChannelFull
	}
}

// ConsumeInbound 消费接收消息
func (b *MessageBus) ConsumeInbound(ctx context.Context) (InboundMessage, error) {
	select {
	case msg := <-b.inbound:
		clientID := ""
		if msg.Metadata != nil {
			if id, ok := msg.Metadata["client_id"].(string); ok {
				clientID = id
			}
		}

		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}

		b.logger.Info("[消息总线] AI Agent 消费入站消息",
			"channel", msg.Channel,
			"chat_id", msg.ChatID,
			"user_id", msg.UserID,
			"client_id", clientID,
			"content_length", len(msg.Content),
			"content_preview", content,
		)
		return msg, nil
	case <-ctx.Done():
		b.logger.Warn("[消息总线] 上下文已取消，无法消费入站消息")
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

// GetDroppedStats 获取丢弃消息统计
func (b *MessageBus) GetDroppedStats() (inbound, outbound int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.inboundDropped, b.outboundDropped
}

// ResetDroppedAlert 重置丢弃告警状态
func (b *MessageBus) ResetDroppedAlert() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.droppedAlerted = false
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
