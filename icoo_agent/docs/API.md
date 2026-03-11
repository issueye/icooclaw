# API 文档

本文档描述 icooclaw 的 RESTful API 接口。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **Content-Type**: `application/json`
- **认证方式**: API Key (可选)

## 目录

- [健康检查](#健康检查)
- [聊天接口](#聊天接口)
- [会话管理](#会话管理)
- [消息管理](#消息管理)
- [提供商管理](#提供商管理)
- [渠道管理](#渠道管理)
- [工具管理](#工具管理)
- [技能管理](#技能管理)
- [MCP 管理](#mcp-管理)
- [记忆管理](#记忆管理)
- [任务管理](#任务管理)
- [绑定管理](#绑定管理)

---

## 健康检查

### GET /health

检查服务健康状态。

**响应示例：**

```json
{
  "code": 200,
  "message": "OK",
  "data": {
    "status": "healthy"
  }
}
```

---

## 聊天接口

### POST /chat

发送聊天消息（HTTP 方式）。

**请求体：**

```json
{
  "message": "你好，请介绍一下自己",
  "session_id": "session-123",
  "channel": "web",
  "chat_id": "chat-456"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| message | string | 是 | 用户消息 |
| session_id | string | 否 | 会话 ID（不提供则自动生成） |
| channel | string | 否 | 渠道标识，默认 `web` |
| chat_id | string | 否 | 聊天 ID |

**响应示例：**

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "session_id": "session-123",
    "response": "你好！我是一个 AI 助手...",
    "tool_calls": []
  }
}
```

### POST /chat/stream

流式聊天（SSE）。

**请求体：** 同 `/chat`

**响应：** Server-Sent Events 流

```
event: message
data: {"content": "你"}

event: message
data: {"content": "好"}

event: done
data: {}
```

### GET /chat/status

获取连接状态。

**响应示例：**

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "websocket_connections": 5,
    "active_sessions": 10
  }
}
```

### GET /chat/queue

获取队列状态。

**响应示例：**

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "pending": 3,
    "processing": 2,
    "max_concurrent": 10
  }
}
```

### POST /chat/queue/max

设置最大并发数。

**请求体：**

```json
{
  "max": 20
}
```

### GET /chat/agents

获取 Agent 状态。

**响应示例：**

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "agent_count": 3,
    "active_count": 2,
    "max_agents": 10,
    "agent_infos": [
      {"id": "default", "status": "active"},
      {"id": "assistant", "status": "idle"}
    ]
  }
}
```

---

## WebSocket

### GET /ws

建立 WebSocket 连接。

**连接 URL：** `ws://localhost:8080/ws`

**消息格式：**

```json
{
  "type": "chat",
  "message": "你好",
  "session_id": "session-123"
}
```

**响应格式：**

```json
{
  "type": "response",
  "session_id": "session-123",
  "content": "你好！有什么可以帮助你的？"
}
```

---

## 会话管理

### POST /sessions/page

分页查询会话。

**请求体：**

```json
{
  "page": 1,
  "page_size": 10,
  "channel": "web"
}
```

**响应示例：**

```json
{
  "code": 200,
  "message": "Success",
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 10,
    "items": [
      {
        "id": "session-123",
        "channel": "web",
        "chat_id": "chat-456",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

### POST /sessions/create

创建新会话。

**请求体：**

```json
{
  "channel": "web",
  "chat_id": "chat-456",
  "user_id": "user-789"
}
```

### POST /sessions/save

保存会话。

### POST /sessions/get

获取单个会话。

**请求体：**

```json
{
  "id": "session-123"
}
```

### POST /sessions/delete

删除会话。

**请求体：**

```json
{
  "id": "session-123"
}
```

---

## 消息管理

### POST /messages/page

分页查询消息。

**请求体：**

```json
{
  "page": 1,
  "page_size": 20,
  "session_id": "session-123"
}
```

### POST /messages/create

创建消息。

**请求体：**

```json
{
  "session_id": "session-123",
  "role": "user",
  "content": "你好"
}
```

### POST /messages/get

获取单个消息。

### POST /messages/delete

删除消息。

---

## 提供商管理

### POST /providers/page

分页查询提供商。

### POST /providers/create

创建提供商配置。

**请求体：**

```json
{
  "name": "openai-main",
  "type": "openai",
  "api_key": "sk-xxx",
  "api_base": "https://api.openai.com/v1",
  "default_model": "gpt-4",
  "models": ["gpt-4", "gpt-4o", "gpt-3.5-turbo"]
}
```

### POST /providers/update

更新提供商配置。

### POST /providers/delete

删除提供商配置。

### GET /providers/all

获取所有提供商。

### GET /providers/enabled

获取已启用的提供商。

---

## 渠道管理

### POST /channels/page

分页查询渠道。

### POST /channels/create

创建渠道配置。

**请求体：**

```json
{
  "name": "feishu-main",
  "type": "feishu",
  "enabled": true,
  "config": {
    "app_id": "cli_xxx",
    "app_secret": "xxx"
  }
}
```

### POST /channels/update

更新渠道配置。

### POST /channels/delete

删除渠道配置。

### GET /channels/all

获取所有渠道。

### GET /channels/enabled

获取已启用的渠道。

---

## 工具管理

### POST /tools/page

分页查询工具。

### POST /tools/create

创建工具配置。

**请求体：**

```json
{
  "name": "weather",
  "type": "custom",
  "enabled": true,
  "definition": {
    "name": "weather",
    "description": "获取天气信息",
    "parameters": {
      "type": "object",
      "properties": {
        "city": {
          "type": "string",
          "description": "城市名称"
        }
      }
    }
  }
}
```

### POST /tools/update

更新工具配置。

### POST /tools/delete

删除工具配置。

### GET /tools/all

获取所有工具。

### GET /tools/enabled

获取已启用的工具。

---

## 技能管理

### POST /skills/page

分页查询技能。

### POST /skills/create

创建技能。

**请求体：**

```json
{
  "name": "translator",
  "description": "翻译助手",
  "prompt": "你是一个专业的翻译助手...",
  "tools": ["web_search"],
  "enabled": true
}
```

### POST /skills/upsert

创建或更新技能。

### POST /skills/get-by-name

按名称获取技能。

### GET /skills/all

获取所有技能。

### GET /skills/enabled

获取已启用的技能。

---

## MCP 管理

### POST /mcp/page

分页查询 MCP 服务器。

### POST /mcp/create

创建 MCP 服务器配置。

**请求体：**

```json
{
  "name": "filesystem",
  "command": "mcp-filesystem",
  "args": ["/path/to/workspace"],
  "enabled": true
}
```

### POST /mcp/update

更新 MCP 服务器配置。

### POST /mcp/delete

删除 MCP 服务器配置。

### GET /mcp/all

获取所有 MCP 服务器。

---

## 记忆管理

### POST /memories/page

分页查询记忆。

### POST /memories/search

搜索记忆。

**请求体：**

```json
{
  "session_id": "session-123",
  "query": "关键词",
  "limit": 10
}
```

### POST /memories/create

创建记忆条目。

### POST /memories/delete

删除记忆条目。

---

## 任务管理

### POST /tasks/page

分页查询任务。

### POST /tasks/create

创建定时任务。

**请求体：**

```json
{
  "name": "daily-report",
  "schedule": "0 9 * * *",
  "description": "每日报告",
  "enabled": true
}
```

### POST /tasks/toggle

启用/禁用任务。

**请求体：**

```json
{
  "id": "task-123",
  "enabled": false
}
```

### GET /tasks/all

获取所有任务。

### GET /tasks/enabled

获取已启用的任务。

---

## 绑定管理

### POST /bindings/page

分页查询绑定。

### POST /bindings/create

创建 Agent 绑定。

**请求体：**

```json
{
  "channel": "feishu",
  "chat_id": "chat-123",
  "agent_name": "assistant",
  "enabled": true
}
```

### POST /bindings/update

更新绑定配置。

### POST /bindings/delete

删除绑定。

### GET /bindings/all

获取所有绑定。

---

## 错误响应

所有错误响应格式：

```json
{
  "code": 400,
  "message": "Invalid request",
  "error": "详细错误信息"
}
```

### 常见错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 认证

### API Key 认证

在请求头中添加：

```
Authorization: Bearer your-api-key
```

或

```
X-API-Key: your-api-key
```