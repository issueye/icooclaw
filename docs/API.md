# icooclaw HTTP API 接口文档

## 概述

- **基础路径**: `/api/v1`
- **默认端口**: `8080`
- **响应格式**: JSON

## 通用响应结构

```json
{
  "code": 200,
  "message": "操作成功",
  "data": { ... }
}
```

## 分页参数

```json
{
  "page": 1,
  "size": 10,
  "total": 100
}
```

---

## 1. 健康检查

### GET /api/v1/health

健康检查接口。

**响应**: `200 OK`

```
OK
```

---

## 2. Session 会话管理

### POST /api/v1/sessions/page

分页查询会话列表。

**请求体**:
```json
{
  "page": {
    "page": 1,
    "size": 10
  },
  "key_word": "搜索关键词",
  "channel": "telegram",
  "user_id": "user123"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "会话列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 100 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "key": "telegram:chat123",
        "channel": "telegram",
        "chat_id": "chat123",
        "user_id": "user123",
        "last_consolidated": 0,
        "metadata": ""
      }
    ]
  }
}
```

### POST /api/v1/sessions/save

创建或更新会话（根据 ID 判断）。

**请求体**:
```json
{
  "id": 0,
  "key": "telegram:chat123",
  "channel": "telegram",
  "chat_id": "chat123",
  "user_id": "user123",
  "last_consolidated": 0,
  "metadata": ""
}
```

### POST /api/v1/sessions/delete

删除会话。

**请求体**:
```json
{
  "id": 1
}
```

### POST /api/v1/sessions/get

根据 ID 获取会话。

**请求体**:
```json
{
  "id": 1
}
```

---

## 3. Message 消息管理

### POST /api/v1/messages/page

分页查询消息列表。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "session_id": 1,
  "role": "user"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "消息列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 50 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "session_id": 1,
        "role": "user",
        "content": "Hello",
        "tool_calls": "",
        "tool_call_id": "",
        "tool_name": "",
        "reasoning_content": ""
      }
    ]
  }
}
```

### POST /api/v1/messages/create

创建消息。

**请求体**:
```json
{
  "session_id": 1,
  "role": "user",
  "content": "Hello",
  "tool_calls": "",
  "tool_call_id": "",
  "tool_name": "",
  "reasoning_content": ""
}
```

### POST /api/v1/messages/update

更新消息。

**请求体**: 同 create

### POST /api/v1/messages/delete

删除消息。

**请求体**: `{ "id": 1 }`

### POST /api/v1/messages/get

根据 ID 获取消息。

**请求体**: `{ "id": 1 }`

---

## 4. MCP 配置管理

### POST /api/v1/mcp/page

分页查询 MCP 配置。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "key_word": ""
}
```

**响应**:
```json
{
  "code": 200,
  "message": "MCP配置列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 5 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "name": "filesystem",
        "description": "文件系统 MCP",
        "type": "stdio",
        "args": ["/path/to/server", "--arg1"]
      }
    ]
  }
}
```

### POST /api/v1/mcp/create

创建 MCP 配置。

**请求体**:
```json
{
  "name": "filesystem",
  "description": "文件系统 MCP",
  "type": "stdio",
  "args": ["/path/to/server", "--arg1"]
}
```

**字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | MCP 名称，唯一 |
| description | string | MCP 描述 |
| type | string | 类型: `stdio` 或 `Streamable HTTP` |
| args | []string | 参数列表 |

### POST /api/v1/mcp/update

更新 MCP 配置。

**请求体**: 同 create，需包含 `id` 字段

### POST /api/v1/mcp/delete

删除 MCP 配置。

**请求体**: `{ "id": 1 }`

### POST /api/v1/mcp/get

根据 ID 获取 MCP 配置。

**请求体**: `{ "id": 1 }`

### GET /api/v1/mcp/all

获取所有 MCP 配置。

**响应**:
```json
{
  "code": 200,
  "message": "MCP配置列表获取成功",
  "data": [ ... ]
}
```

---

## 5. Memory 记忆管理

### POST /api/v1/memories/page

分页查询记忆。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "type": "memory",
  "key_word": "",
  "user_id": "",
  "session_id": null
}
```

**响应**:
```json
{
  "code": 200,
  "message": "记忆列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 20 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "type": "memory",
        "key": "user_preference",
        "content": "用户喜欢简洁的回答",
        "session_id": null,
        "user_id": "user123",
        "tags": ["preference"],
        "importance": 5,
        "is_pinned": false,
        "is_deleted": false,
        "expires_at": null
      }
    ]
  }
}
```

**type 字段可选值**:
- `memory` - 长期记忆
- `history` - 会话历史
- `session` - 会话记忆
- `user` - 用户记忆

### POST /api/v1/memories/create

创建记忆。

**请求体**:
```json
{
  "type": "memory",
  "key": "user_preference",
  "content": "用户喜欢简洁的回答",
  "session_id": null,
  "user_id": "user123",
  "tags": ["preference"],
  "importance": 5,
  "is_pinned": false,
  "expires_at": null
}
```

### POST /api/v1/memories/update

更新记忆。

### POST /api/v1/memories/delete

删除记忆。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/get

根据 ID 获取记忆。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/pin

置顶记忆。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/unpin

取消置顶。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/soft-delete

软删除记忆。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/restore

恢复软删除的记忆。

**请求体**: `{ "id": 1 }`

### POST /api/v1/memories/search

搜索记忆。

**请求体**:
```json
{
  "query": "搜索关键词"
}
```

---

## 6. Task 任务管理

### POST /api/v1/tasks/page

分页查询任务。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "key_word": "",
  "enabled": true
}
```

**响应**:
```json
{
  "code": 200,
  "message": "任务列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 5 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "name": "daily_report",
        "description": "每日报告",
        "type": 1,
        "cron_expr": "0 9 * * *",
        "interval": 0,
        "message": "生成每日报告",
        "channel": "telegram",
        "chat_id": "chat123",
        "enabled": true,
        "next_run_at": "2026-03-04T09:00:00Z",
        "last_run_at": "2026-03-03T09:00:00Z"
      }
    ]
  }
}
```

**type 字段说明**:
- `0` - 立即执行任务
- `1` - cron 定时任务

### POST /api/v1/tasks/create

创建任务。

**请求体**:
```json
{
  "name": "daily_report",
  "description": "每日报告",
  "type": 1,
  "cron_expr": "0 9 * * *",
  "interval": 0,
  "message": "生成每日报告",
  "channel": "telegram",
  "chat_id": "chat123",
  "enabled": true
}
```

### POST /api/v1/tasks/update

更新任务。

### POST /api/v1/tasks/delete

删除任务。

**请求体**: `{ "id": 1 }`

### POST /api/v1/tasks/get

根据 ID 获取任务。

**请求体**: `{ "id": 1 }`

### POST /api/v1/tasks/toggle

切换任务启用状态。

**请求体**: `{ "id": 1 }`

### GET /api/v1/tasks/all

获取所有任务。

### GET /api/v1/tasks/enabled

获取所有启用的任务。

---

## 7. Provider 配置管理

### POST /api/v1/providers/page

分页查询 Provider 配置。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "key_word": "",
  "enabled": true
}
```

**响应**:
```json
{
  "code": 200,
  "message": "Provider配置列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 3 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "name": "openai",
        "enabled": true,
        "config": "{\"api_key\":\"sk-xxx\",\"model\":\"gpt-4\"}"
      }
    ]
  }
}
```

### POST /api/v1/providers/create

创建 Provider 配置。

**请求体**:
```json
{
  "name": "openai",
  "enabled": true,
  "config": "{\"api_key\":\"sk-xxx\",\"model\":\"gpt-4\"}"
}
```

### POST /api/v1/providers/update

更新 Provider 配置。

### POST /api/v1/providers/delete

删除 Provider 配置。

**请求体**: `{ "id": 1 }`

### POST /api/v1/providers/get

根据 ID 获取 Provider 配置。

**请求体**: `{ "id": 1 }`

### GET /api/v1/providers/all

获取所有 Provider 配置。

### GET /api/v1/providers/enabled

获取所有启用的 Provider 配置。

---

## 8. Skill 技能管理

### POST /api/v1/skills/page

分页查询技能。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "key_word": "",
  "enabled": true,
  "source": "workspace"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "技能列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 10 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "name": "crud",
        "source": "workspace",
        "description": "CRUD 操作技能",
        "content": "# CRUD Skill\n...",
        "enabled": true,
        "always_load": false,
        "metadata": ""
      }
    ]
  }
}
```

### POST /api/v1/skills/create

创建技能。

**请求体**:
```json
{
  "name": "crud",
  "source": "workspace",
  "description": "CRUD 操作技能",
  "content": "# CRUD Skill\n...",
  "enabled": true,
  "always_load": false,
  "metadata": ""
}
```

### POST /api/v1/skills/update

更新技能。

### POST /api/v1/skills/delete

删除技能。

**请求体**: `{ "id": 1 }`

### POST /api/v1/skills/get

根据 ID 获取技能。

**请求体**: `{ "id": 1 }`

### POST /api/v1/skills/get-by-name

根据名称获取技能。

**请求体**:
```json
{
  "name": "crud"
}
```

### POST /api/v1/skills/upsert

创建或更新技能（根据 name 判断）。

**请求体**: 同 create

### GET /api/v1/skills/all

获取所有技能。

### GET /api/v1/skills/enabled

获取所有启用的技能。

---

## 9. Channel 通道配置管理

### POST /api/v1/channels/page

分页查询通道配置。

**请求体**:
```json
{
  "page": { "page": 1, "size": 10 },
  "key_word": "",
  "enabled": true
}
```

**响应**:
```json
{
  "code": 200,
  "message": "通道配置列表获取成功",
  "data": {
    "page": { "page": 1, "size": 10, "total": 2 },
    "records": [
      {
        "id": 1,
        "created_at": "2026-03-03T10:00:00Z",
        "updated_at": "2026-03-03T10:00:00Z",
        "name": "telegram",
        "enabled": true,
        "config": "{\"token\":\"xxx\"}"
      }
    ]
  }
}
```

### POST /api/v1/channels/create

创建通道配置。

**请求体**:
```json
{
  "name": "telegram",
  "enabled": true,
  "config": "{\"token\":\"xxx\"}"
}
```

### POST /api/v1/channels/update

更新通道配置。

### POST /api/v1/channels/delete

删除通道配置。

**请求体**: `{ "id": 1 }`

### POST /api/v1/channels/get

根据 ID 获取通道配置。

**请求体**: `{ "id": 1 }`

### GET /api/v1/channels/all

获取所有通道配置。

### GET /api/v1/channels/enabled

获取所有启用的通道配置。

---

## 10. Config 配置管理

### GET /api/v1/config/

获取当前完整配置。

**响应**:
```json
{
  "code": 200,
  "message": "获取配置成功",
  "data": {
    "gateway": {
      "port": 8080,
      "host": "0.0.0.0",
      "enabled": true,
      "key": ""
    },
    "workspace": "./workspace",
    "database": { "path": "./data/icooclaw.db" },
    "log": { "level": "info", "format": "text", "output": "" },
    ...
  }
}
```

### POST /api/v1/config/update

更新配置（部分更新）。

**请求体**:
```json
{
  "config": {
    "workspace": "/new/workspace",
    "log.level": "debug"
  }
}
```

### POST /api/v1/config/overwrite

覆盖配置文件（完整替换 TOML 内容）。

**请求体**:
```json
{
  "content": "[gateway]\nport = 8080\n..."
}
```

**说明**:
- 此接口会用新的 TOML 内容完全覆盖配置文件
- 会自动备份原配置文件为 `.bak`
- 如果新配置无效，会自动回滚

### GET /api/v1/config/file

获取配置文件原始内容。

**响应**:
```json
{
  "code": 200,
  "message": "获取配置文件成功",
  "data": {
    "content": "[gateway]\nport = 8080\n...",
    "path": "/path/to/config.toml"
  }
}
```

### GET /api/v1/config/json

获取 JSON 格式的配置（用于前端展示）。

---

## 11. Workspace 工作区管理

### GET /api/v1/workspace/

获取当前工作区路径。

**响应**:
```json
{
  "code": 200,
  "message": "获取工作区成功",
  "data": {
    "workspace": "./workspace"
  }
}
```

### POST /api/v1/workspace/set

设置工作区路径。

**请求体**:
```json
{
  "workspace": "/new/workspace/path"
}
```

**说明**:
- 设置后会自动保存到配置文件
- 需要 **重启服务** 才能生效

---

## 错误响应

当请求失败时，返回格式如下：

```json
{
  "code": 400,
  "message": "错误信息",
  "data": null
}
```

常见错误码：
| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## CORS 配置

API 支持 CORS，允许跨域访问：
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`