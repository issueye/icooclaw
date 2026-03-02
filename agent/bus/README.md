# Bus 模块

消息总线模块，提供异步消息队列和事件处理功能。

## 功能特性

- **消息队列**：支持 Inbound/Outbound 消息的发布和订阅
- **缓冲区管理**：可配置的缓冲区大小，防止消息丢失
- **事件系统**：支持多种事件类型（Agent 生命周期、工具调用、心跳等）
- **上下文支持**：支持 context 取消和超时控制

## 核心组件

### 消息类型

```go
const (
    MessageTypeMessage    MessageType = "message"      // 普通消息
    MessageTypeChunk      MessageType = "chunk"         // 流式消息块
    MessageTypeEnd        MessageType = "chunk_end"     // 流式消息结束
    MessageTypeToolCall   MessageType = "tool_call"    // 工具调用
    MessageTypeToolResult MessageType = "tool_result"  // 工具结果
    MessageTypeError      MessageType = "error"        // 错误
    MessageTypeThinking  MessageType = "thinking"     // 思考中
)
```

### 消息格式

```go
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
    ID         string
    Type       MessageType
    Channel    string
    ChatID     string
    Content    string
    Thinking   string
    ToolName   string
    ToolCallID string
    Arguments  string
    Status     string
    Error      string
    Timestamp  time.Time
    Metadata   map[string]interface{}
}
```

## 使用示例

### 创建消息总线

```go
// 使用默认缓冲区大小（100）
bus := NewMessageBus()

// 自定义缓冲区大小
bus := NewMessageBus(200)
```

### 发布消息

```go
// 发布接收消息
inboundMsg := InboundMessage{
    Channel:   "telegram",
    ChatID:    "chat_123",
    UserID:    "user_456",
    Content:   "Hello",
    Timestamp: time.Now(),
}

err := bus.PublishInbound(ctx, inboundMsg)
if err != nil {
    // 处理错误
}
```

### 订阅消息

```go
// 订阅接收消息
inboundCh := bus.SubscribeInbound("my-handler")
go func() {
    for msg := range inboundCh {
        // 处理消息
    }
}()

// 订阅发送消息
outboundCh := bus.SubscribeOutbound("my-handler")
go func() {
    for msg := range outboundCh {
        // 处理消息
    }
}()
```

### 发送消息

```go
// 发送消息
outboundMsg := OutboundMessage{
    Type:    MessageTypeMessage,
    Channel: "telegram",
    ChatID:  "chat_123",
    Content: "Hello!",
}

bus.PublishOutbound(outboundMsg)
```

## 事件系统

### 事件类型

```go
const (
    EventAgentStart         EventType = "agent_start"          // Agent 启动
    EventAgentStop          EventType = "agent_stop"           // Agent 停止
    EventAgentResponse     EventType = "agent_response"      // Agent 响应
    EventToolCall          EventType = "tool_call"           // 工具调用
    EventToolResult        EventType = "tool_result"          // 工具结果
    EventError             EventType = "error"               // 错误
    EventMessageReceived   EventType = "message_received"    // 消息接收
    EventMessageSent       EventType = "message_sent"         // 消息发送
    EventTaskStart         EventType = "task_start"          // 任务开始
    EventTaskComplete      EventType = "task_complete"        // 任务完成
    EventHeartbeat         EventType = "heartbeat"            // 心跳
)
```

### 事件结构

```go
type Event struct {
    Type      EventType
    Channel   string
    ChatID    string
    SessionID uint
    Data      interface{}
    Timestamp time.Time
}
```

### 发布事件

```go
event := NewEvent(
    EventAgentStart,
    "telegram",
    "chat_123",
    1,
    map[string]interface{}{"name": "my-agent"},
)
bus.PublishEvent(event)
```

### 订阅事件

```go
eventCh := bus.SubscribeEvent(EventAgentStart)
go func() {
    for event := range eventCh {
        // 处理事件
    }
}()
```

## 错误处理

```go
var (
    ErrChannelFull = errors.New("channel is full")
    ErrSubscriberNotFound = errors.New("subscriber not found")
)

// 检查错误
if err == ErrChannelFull {
    // 缓冲区已满，处理消息积压
}
```

## 设置日志

```go
// 使用自定义 logger
customLogger := slog.New(...)
bus.SetLogger(customLogger)
```

## 完整示例

```go
package main

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/icooclaw/bus"
)

func main() {
    // 创建消息总线
    bus := bus.NewMessageBus(100)
    bus.SetLogger(slog.Default())
    
    ctx := context.Background()
    
    // 启动消费者
    go consumeInbound(bus, "handler1")
    go consumeOutbound(bus, "handler2")
    
    // 发布消息
    inbound := bus.InboundMessage{
        Channel: "telegram",
        ChatID:  "chat_123",
        Content: "Hello",
    }
    bus.PublishInbound(ctx, inbound)
    
    // 等待处理
    time.Sleep(time.Second)
}

func consumeInbound(b *bus.MessageBus, name string) {
    ch := b.SubscribeInbound(name)
    for msg := range ch {
        slog.Info("Received inbound", "channel", msg.Channel, "content", msg.Content)
    }
}

func consumeOutbound(b *bus.MessageBus, name string) {
    ch := b.SubscribeOutbound(name)
    for msg := range ch {
        slog.Info("Received outbound", "channel", msg.Channel, "content", msg.Content)
    }
}
```

## 文件结构

```
bus/
├── bus.go        # 消息总线实现
├── bus_test.go   # 消息总线测试
├── events.go     # 事件系统
└── README.md     # 本文档
```

## 运行测试

```bash
cd agent/bus
go test -v ./...
```
