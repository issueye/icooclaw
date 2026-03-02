package channel

import (
	"context"
	"log/slog"
)

// Channel 消息通道接口
type Channel interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Send(ctx context.Context, msg OutboundMessage) error
	IsRunning() bool
}

// InboundHandler 接收消息处理
type InboundHandler interface {
	Handle(ctx context.Context, msg InboundMessage) error
}

// OutboundMessage 发送消息
type OutboundMessage struct {
	Channel   string
	ChatID    string
	Content   string
	ParseMode string // markdown, html, text
}

// InboundMessage 接收消息
type InboundMessage struct {
	Channel   string
	ChatID    string
	UserID    string
	Content   string
	MessageID string
}

// BaseChannel 基础通道实现
type BaseChannel struct {
	name    string
	running bool
	logger  *slog.Logger
}

// NewBaseChannel 创建基础通道
func NewBaseChannel(name string, logger *slog.Logger) *BaseChannel {
	return &BaseChannel{
		name:   name,
		logger: logger,
	}
}

// Name 获取名称
func (c *BaseChannel) Name() string {
	return c.name
}

// IsRunning 检查是否运行
func (c *BaseChannel) IsRunning() bool {
	return c.running
}

// SetRunning 设置运行状态
func (c *BaseChannel) SetRunning(running bool) {
	c.running = running
}
