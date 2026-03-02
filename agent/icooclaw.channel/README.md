# Channel 模块

消息通道模块，提供多种消息通道的实现和管理。

## 功能特性

- **WebSocket 通道**：支持 WebSocket 长连接实时通信
- **Webhook 通道**：支持 HTTP Webhook 回调接收消息
- **通道管理**：统一的通道注册、启动、停止管理
- **消息格式**：统一的 Inbound/Outbound 消息格式

## 核心组件

### 通道接口

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Send(ctx context.Context, msg OutboundMessage) error
    IsRunning() bool
}
```

### 消息格式

```go
// OutboundMessage 发送消息
type OutboundMessage struct {
    Channel   string
    ChatID    string
    Content   string
    ParseMode string // markdown, html, text
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
```

## 使用示例

### 创建 Webhook 通道

```go
// 定义配置
type MyWebhookConfig struct{}

func (c *MyWebhookConfig) Enabled() bool { return true }
func (c *MyWebhookConfig) Host() string { return "0.0.0.0" }
func (c *MyWebhookConfig) Port() int { return 8081 }
func (c *MyWebhookConfig) Path() string { return "/webhook" }
func (c *MyWebhookConfig) Secret() string { return "my-secret" }
func (c *MyWebhookConfig) Extra() map[string]interface{} { return nil }

// 创建通道
channel := NewWebhookChannel(webhookConfig, messageBus, logger)

// 启动
ctx := context.Background()
err := channel.Start(ctx)

// 发送消息
err := channel.Send(ctx, OutboundMessage{
    Channel: "webhook",
    ChatID:  "chat_123",
    Content: "Hello",
})
```

### 创建 WebSocket 通道

```go
// 定义配置
type MyWebSocketConfig struct{}

func (c *MyWebSocketConfig) Enabled() bool { return true }
func (c *MyWebSocketConfig) Host() string { return "0.0.0.0" }
func (c *MyWebSocketConfig) Port() int { return 8080 }

// 创建通道
channel := NewWebSocketChannel(wsConfig, messageBus, storage, logger)

// 设置 Agent（可选）
channel.SetAgent(agent)

// 启动
ctx := context.Background()
err := channel.Start(ctx)
```

### 使用通道管理器

```go
// 创建消息总线
bus := channel.NewMessageBus(...)

// 创建配置
config := &MyChannelConfig{...}

// 创建管理器
manager := channel.NewManager(bus, config, db, logger)

// 启动所有通道
err := manager.StartAll()

// 获取通道
ch, err := manager.Get("webhook")

// 停止所有通道
ctx := context.Background()
manager.StopAll(ctx)
```

## 接口定义

### Logger 接口

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

### StorageReader 接口

```go
type StorageReader interface {
    GetChatByID(id uint) (interface{}, error)
    CreateChat(chat interface{}) error
    UpdateChat(chat interface{}) error
    GetUserByID(id uint) (interface{}, error)
    CreateUser(user interface{}) error
}
```

### MessageBus 接口

```go
type MessageBus interface {
    PublishInbound(ctx context.Context, msg InboundMessage) error
    Publish(event interface{}) error
    Subscribe(handler interface{}) error
}
```

## WebSocket 消息格式

```go
type WebSocketMessage struct {
    Type     string          `json:"type"`
    Content  string          `json:"content"`
    Thinking string          `json:"thinking,omitempty"`
    ChatID   string          `json:"chat_id,omitempty"`
    UserID   string          `json:"user_id,omitempty"`
    Data     json.RawMessage `json:"data,omitempty"`
}
```

## Webhook API

### 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/webhook` | POST | 接收消息 |
| `/health` | GET | 健康检查 |
| `/status` | GET | 状态检查 |

### 请求格式

```json
{
    "content": "消息内容",
    "chat_id": "聊天ID",
    "user_id": "用户ID",
    "message_id": "消息ID",
    "extra": {}
}
```

### 响应格式

```json
{
    "success": true,
    "message": "Message received",
    "data": {
        "message_id": "xxx"
    }
}
```

## WebSocket API

### 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/ws` | WebSocket | WebSocket 连接 |
| `/api/v1/chat` | POST | 聊天 |
| `/api/v1/chat/stream` | POST | 流式聊天 |
| `/api/v1/health` | GET | 健康检查 |

## 文件结构

```
channel/
├── base.go          # 基础定义和接口
├── base_test.go     # 基础测试
├── manager.go       # 通道管理器
├── webhook.go       # Webhook 通道实现
└── websocket.go     # WebSocket 通道实现
```

## 运行测试

```bash
cd agent/channel
go test -v ./...
```
