package channel

import (
	"context"
	"io"
	"log/slog"
	"time"
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
	ParseMode string
}

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

// MessageBus 接口 - 消息总线
type MessageBus interface {
	// PublishInbound 发布接收消息
	PublishInbound(ctx context.Context, msg InboundMessage) error
	// Publish 发布消息
	Publish(event interface{}) error
	// Subscribe 订阅消息
	Subscribe(handler interface{}) error
}

// ConfigReader 接口 - 配置读取
type ConfigReader interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
}

// StorageReader 接口 - 存储读取
type StorageReader interface {
	// GetSessions 获取会话列表
	GetSessions(userID, channel string) (interface{}, error)
	// GetSessionMessages 获取会话消息
	GetSessionMessages(sessionID uint, limit int) (interface{}, error)
	// DeleteSession 删除会话
	DeleteSession(sessionID string) error
}

// Logger 接口 - 日志
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// RequestReader 接口 - HTTP 请求读取
type RequestReader interface {
	GetHeader(key string) string
	GetBody() (io.ReadCloser, error)
	GetQuery(key string) string
	GetPathVar(key string) string
}

// ChannelConfig 接口 - 通道配置
type ChannelConfig interface {
	WebSocketConfig() WebSocketConfig
	WebhookConfig() WebhookConfig
	// FeishuConfig 飞书通道配置
	FeishuConfig() FeishuConfig
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig interface {
	Enabled() bool
	Host() string
	Port() int
}

// WebhookConfig Webhook 配置
type WebhookConfig interface {
	Enabled() bool
	Host() string
	Port() int
	Path() string
	Secret() string
	Extra() map[string]interface{}
}

// FeishuConfig 飞书通道配置
type FeishuConfig interface {
	Enabled() bool
	// Host 监听地址，默认 0.0.0.0
	Host() string
	// Port 监听端口，默认 8082
	Port() int
	// Path Webhook 接收路径，默认 /feishu/webhook
	Path() string
	// VerificationToken 飞书事件订阅验证 Token
	VerificationToken() string
	// EncryptKey 飞书消息体加密密钥（可选）
	EncryptKey() string
	// AppID 飞书应用 AppID（用于发送消息）
	AppID() string
	// AppSecret 飞书应用 AppSecret
	AppSecret() string
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
