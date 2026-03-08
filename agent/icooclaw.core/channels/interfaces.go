package channels

import (
	"context"
	"net/http"

	"icooclaw.core/bus"
)

// TypingCapable - 可以显示打字/思考指示器的通道
// StartTyping 开始指示器并返回停止函数
// 停止函数必须是幂等的且可以安全多次调用
type TypingCapable interface {
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// MessageEditor - 可以编辑现有消息的通道
// messageID 始终为字符串；通道在内部转换平台特定类型
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID string, messageID string, content string) error
}

// ReactionCapable - 可以对入站消息添加反应（如 👀）的通道
// ReactToMessage 添加反应并返回撤销函数以移除它
// 撤销函数必须是幂等的且可以安全多次调用
type ReactionCapable interface {
	ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

// PlaceholderCapable - 可以发送占位符消息（如 "Thinking... 💭"）
// 的通道，该消息稍后将被编辑为实际响应
// 通道还必须实现 MessageEditor 才能使占位符有用
// SendPlaceholder 返回占位符的平台消息 ID，以便 Manager.preSend
// 稍后可以通过 MessageEditor.EditMessage 编辑它
type PlaceholderCapable interface {
	SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

// PlaceholderRecorder 由 Manager 注入到通道中
// 通道在入站时调用这些方法来注册打字/占位符状态
// Manager 在出站时使用注册的状态来停止打字和编辑占位符
type PlaceholderRecorder interface {
	RecordPlaceholder(channel, chatID, placeholderID string)
	RecordTypingStop(channel, chatID string, stop func())
	RecordReactionUndo(channel, chatID string, undo func())
}

// MediaSender - 可以发送媒体附件（图片、文件、音频、视频）的通道
// Manager 通过类型断言发现实现此接口的通道，并将 OutboundMediaMessage 路由到它们
type MediaSender interface {
	SendMedia(ctx context.Context, msg bus.OutboundMediaMessage) error
}

// WebhookHandler - 通过 HTTP webhook 接收消息的通道的可选接口
// Manager 发现实现此接口的通道，并在共享 HTTP 服务器上注册它们
type WebhookHandler interface {
	// WebhookPath 返回在共享服务器上挂载此处理程序的路径
	// 例如："/webhook/telegram", "/webhook/feishu"
	WebhookPath() string
	http.Handler // ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// HealthChecker - 在共享 HTTP 服务器上公开健康检查端点的通道的可选接口
type HealthChecker interface {
	HealthPath() string
	HealthHandler(w http.ResponseWriter, r *http.Request)
}

// MessageLengthProvider - 通道实现的可选接口，用于广播其最大消息长度
// Manager 通过类型断言使用此接口来决定是否分割出站消息
type MessageLengthProvider interface {
	MaxMessageLength() int
}