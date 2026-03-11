# 渠道配置指南

本文档描述如何配置和使用各种消息渠道。

## 目录

- [飞书 (Feishu/Lark)](#飞书-feishulark)
- [钉钉 (DingTalk)](#钉钉-dingtalk)
- [WebSocket](#websocket)
- [HTTP API](#http-api)

---

## 飞书 (Feishu/Lark)

### 功能特性

- ✅ WebSocket 长连接模式
- ✅ Interactive Card 消息
- ✅ 消息编辑
- ✅ 占位符消息
- ✅ 消息反应（表情）
- ✅ 媒体消息下载（图片、文件、音频、视频）
- ✅ 白名单过滤

### 创建飞书应用

1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 获取 **App ID** 和 **App Secret**
4. 配置权限：
   - `im:message` - 获取与发送消息
   - `im:message:send_as_bot` - 以应用身份发消息
   - `im:resource` - 获取与上传图片或文件资源

### 配置示例

```toml
[channels.feishu]
enabled = true
app_id = "cli_xxxxxxxxxxxx"
app_secret = "xxxxxxxxxxxxxxxxxxxxxxxx"
encrypt_key = "xxxxxxxxxxxxxxxxxxxxxxxx"
verification_token = "xxxxxxxxxxxxxxxxxxxxxxxx"
allow_from = ["ou_xxxxx", "ou_yyyyy"]  # 可选：白名单用户
reasoning_chat_id = ""  # 可选：推理过程发送的聊天
```

### 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | bool | 是 | 是否启用 |
| app_id | string | 是 | 应用 ID |
| app_secret | string | 是 | 应用密钥 |
| encrypt_key | string | 否 | 加密密钥（用于消息加密） |
| verification_token | string | 否 | 验证令牌 |
| allow_from | []string | 否 | 白名单用户 Open ID |
| reasoning_chat_id | string | 否 | 推理过程发送的目标聊天 |

### 消息格式

飞书渠道支持发送 Interactive Card 消息，支持 Markdown 格式：

```json
{
  "type": "interactive",
  "card": {
    "elements": [
      {
        "tag": "markdown",
        "content": "**标题**\n内容"
      }
    ]
  }
}
```

### 使用示例

```bash
# 启动服务后，飞书机器人会自动连接
# 用户在飞书中发送消息即可与 AI 对话
```

---

## 钉钉 (DingTalk)

### 功能特性

- ✅ Stream 模式（推荐）
- ✅ Markdown 消息
- ✅ Session Webhook 管理
- ✅ 白名单过滤

### 创建钉钉应用

1. 访问 [钉钉开放平台](https://open.dingtalk.com/)
2. 创建企业内部应用
3. 获取 **Client ID** 和 **Client Secret**
4. 获取 **Agent ID**
5. 配置权限：
   - `qyapi_get_member` - 获取成员信息
   - `qyapi_get_dept_member` - 获取部门成员

### 配置示例

```toml
[channels.dingtalk]
enabled = true
client_id = "dingxxxxxxxxxxxx"
client_secret = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
agent_id = 123456789
allow_from = ["user123", "user456"]  # 可选：白名单用户
reasoning_chat_id = ""  # 可选：推理过程发送的聊天
```

### 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | bool | 是 | 是否启用 |
| client_id | string | 是 | 应用 Client ID |
| client_secret | string | 是 | 应用 Client Secret |
| agent_id | int64 | 是 | Agent ID |
| allow_from | []string | 否 | 白名单用户 ID |
| reasoning_chat_id | string | 否 | 推理过程发送的目标聊天 |

### Stream 模式

钉钉渠道使用 Stream 模式接收消息，无需配置 Webhook：

1. 应用会自动建立长连接
2. 实时接收用户消息
3. 支持自动重连

### 消息格式

钉钉渠道支持 Markdown 消息：

```json
{
  "msgtype": "markdown",
  "markdown": {
    "title": "标题",
    "text": "### 标题\n内容"
  }
}
```

---

## WebSocket

### 功能特性

- ✅ 实时双向通信
- ✅ 心跳检测
- ✅ 会话管理
- ✅ 流式响应

### 连接方式

```javascript
// 前端 JavaScript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('WebSocket 连接成功');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到消息:', data);
};

ws.onerror = (error) => {
  console.error('WebSocket 错误:', error);
};

ws.onclose = () => {
  console.log('WebSocket 连接关闭');
};
```

### 发送消息

```javascript
// 发送聊天消息
ws.send(JSON.stringify({
  type: 'chat',
  message: '你好，请介绍一下自己',
  session_id: 'session-123'
}));
```

### 消息类型

#### 请求消息

| 类型 | 说明 | 字段 |
|------|------|------|
| chat | 聊天消息 | message, session_id |
| ping | 心跳 | - |

#### 响应消息

| 类型 | 说明 | 字段 |
|------|------|------|
| response | 响应消息 | session_id, content |
| tool_call | 工具调用 | name, args |
| error | 错误 | message |
| pong | 心跳响应 | - |

### 流式响应

```javascript
ws.send(JSON.stringify({
  type: 'chat',
  message: '写一首诗',
  session_id: 'session-123',
  stream: true
}));

// 会收到多个响应
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'stream') {
    // 流式内容块
    process.stdout.write(data.content);
  } else if (data.type === 'response') {
    // 完成响应
    console.log('\n完成');
  }
};
```

---

## HTTP API

### 功能特性

- ✅ RESTful API
- ✅ SSE 流式响应
- ✅ 会话管理

### 基本使用

```bash
# 发送聊天消息
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好",
    "session_id": "test-session"
  }'
```

### 流式响应 (SSE)

```bash
# 使用 SSE 流式响应
curl -N http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": "写一首诗",
    "session_id": "test-session"
  }'
```

**响应格式：**

```
event: message
data: {"content": "春"}

event: message
data: {"content": "风"}

event: done
data: {}
```

---

## 渠道开发指南

### 实现 Channel 接口

```go
package mychannel

import (
    "context"
    "icooclaw/pkg/bus"
    "icooclaw/pkg/channels"
)

type MyChannel struct {
    config Config
    bus    *bus.MessageBus
    running atomic.Bool
}

func New(cfg Config, bus *bus.MessageBus) (*MyChannel, error) {
    return &MyChannel{
        config: cfg,
        bus:    bus,
    }, nil
}

// Name 返回渠道名称
func (c *MyChannel) Name() string {
    return "my_channel"
}

// Start 启动渠道
func (c *MyChannel) Start(ctx context.Context) error {
    c.running.Store(true)
    // 启动消息接收逻辑
    go c.receiveMessages(ctx)
    return nil
}

// Stop 停止渠道
func (c *MyChannel) Stop(ctx context.Context) error {
    c.running.Store(false)
    return nil
}

// Send 发送消息
func (c *MyChannel) Send(ctx context.Context, msg channels.OutboundMessage) error {
    // 实现消息发送逻辑
    return nil
}

// IsRunning 检查是否运行中
func (c *MyChannel) IsRunning() bool {
    return c.running.Load()
}

// 接收消息并发布到 MessageBus
func (c *MyChannel) receiveMessages(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // 接收平台消息
            msg := c.platformReceive()
            
            // 转换为 InboundMessage
            inbound := bus.InboundMessage{
                Channel:  c.Name(),
                ChatID:   msg.ChatID,
                SenderID: msg.UserID,
                Content:  msg.Text,
            }
            
            // 发布到 MessageBus
            c.bus.PublishInbound(ctx, inbound)
        }
    }
}
```

### 注册渠道

```go
package mychannel

import "icooclaw/pkg/channels"

func init() {
    channels.RegisterFactory("my_channel", func(cfg map[string]any, bus *bus.MessageBus) (channels.Channel, error) {
        config := parseConfig(cfg)
        return New(config, bus)
    })
}
```

### 实现扩展接口

```go
// 消息编辑
func (c *MyChannel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
    // 实现消息编辑
    return nil
}

// 打字指示器
func (c *MyChannel) StartTyping(ctx context.Context, chatID string) (stop func(), err error) {
    // 开始打字动画
    return func() {
        // 停止打字动画
    }, nil
}

// 消息反应
func (c *MyChannel) ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error) {
    // 添加反应
    return func() {
        // 移除反应
    }, nil
}

// 媒体发送
func (c *MyChannel) SendMedia(ctx context.Context, msg channels.OutboundMediaMessage) error {
    // 发送媒体消息
    return nil
}
```

---

## 常见问题

### 飞书消息收不到

1. 检查 App ID 和 App Secret 是否正确
2. 检查应用权限配置
3. 检查是否在白名单中

### 钉钉 Stream 连接失败

1. 检查 Client ID 和 Client Secret 是否正确
2. 检查网络连接
3. 查看日志中的错误信息

### WebSocket 连接断开

1. 检查心跳是否正常
2. 检查网络稳定性
3. 查看服务器日志