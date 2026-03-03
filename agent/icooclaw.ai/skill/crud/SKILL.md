# CRUD API Skill

## 描述
这个技能允许 AI 助手通过统一的 API 接口管理项目中的各种资源，包括会话(Sessions)、消息(Messages)、MCP配置、记忆(Memories)、任务(Tasks)、Provider配置、技能(Skills)和通道配置(Channels)。

## 支持的资源

| 资源 | 说明 | 支持的操作 |
|------|------|-----------|
| sessions | 会话管理 | page, create, update, delete, get |
| messages | 消息管理 | page, create, update, delete, get |
| mcp | MCP配置管理 | page, create, update, delete, get, get-all |
| memories | 记忆管理 | page, create, update, delete, get, pin, unpin, soft-delete, restore, search |
| tasks | 任务管理 | page, create, update, delete, get, toggle, get-all, get-enabled |
| providers | Provider配置管理 | page, create, update, delete, get, get-all, get-enabled |
| skills | 技能管理 | page, create, update, delete, get, get-by-name, upsert, get-all, get-enabled |
| channels | 通道配置管理 | page, create, update, delete, get, get-all, get-enabled |

## API 端点格式

所有 CRUD API 端点遵循以下格式：
```
POST /api/v1/{resource}/{action}
GET /api/v1/{resource}/{action}  (用于获取列表等只读操作)
```

## 使用示例

### 分页查询会话
```json
POST /api/v1/sessions/page
{
  "page": {
    "page": 1,
    "size": 10
  },
  "key_word": "telegram"
}
```

### 创建新会话
```json
POST /api/v1/sessions/create
{
  "key": "telegram:123456",
  "channel": "telegram",
  "chat_id": "123456",
  "user_id": "user001"
}
```

### 获取单个资源
```json
POST /api/v1/sessions/get
{
  "id": 1
}
```

### 更新资源
```json
POST /api/v1/sessions/update
{
  "id": 1,
  "metadata": "{\"title\":\"新标题\"}"
}
```

### 删除资源
```json
POST /api/v1/sessions/delete
{
  "id": 1
}
```

## 数据模型

### Session (会话)
```go
type Session struct {
    ID               uint      `json:"id"`
    Key              string    `json:"key"`              // channel:chat_id
    Channel          string    `json:"channel"`          // telegram, discord, feishu...
    ChatID           string    `json:"chat_id"`          // 用户/群组ID
    UserID           string    `json:"user_id"`          // 用户唯一标识
    LastConsolidated int       `json:"last_consolidated"`// 已整合的消息数
    Metadata         string    `json:"metadata"`         // JSON元数据
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### Message (消息)
```go
type Message struct {
    ID               uint   `json:"id"`
    SessionID        uint   `json:"session_id"`
    Role             string `json:"role"`              // user, assistant, system, tool
    Content          string `json:"content"`
    ToolCalls        string `json:"tool_calls"`        // JSON数组
    ToolCallID       string `json:"tool_call_id"`      // 工具调用ID
    ToolName         string `json:"tool_name"`         // 工具名称
    ReasoningContent string `json:"reasoning_content"` // 思考过程
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### MCPConfig (MCP配置)
```go
type MCPConfig struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`        // MCP 名称
    Description string    `json:"description"` // MCP 描述
    Type        string    `json:"type"`        // stdio 或 Streamable HTTP
    Args        []string  `json:"args"`        // MCP 参数
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Memory (记忆)
```go
type Memory struct {
    ID         uint      `json:"id"`
    Type       string    `json:"type"`        // memory, history, session, user
    Key        string    `json:"key"`         // 记忆键
    Content    string    `json:"content"`     // 记忆内容
    SessionID  *uint     `json:"session_id"`  // 关联会话ID
    UserID     string    `json:"user_id"`     // 用户ID
    Tags       []string  `json:"tags"`        // 标签
    Importance int       `json:"importance"`  // 重要性级别 0-10
    IsPinned   bool      `json:"is_pinned"`   // 是否置顶
    IsDeleted  bool      `json:"is_deleted"`  // 软删除标记
    ExpiresAt  *time.Time `json:"expires_at"` // 过期时间
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

### Task (任务)
```go
type Task struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`        // 任务名称
    Description string    `json:"description"` // 任务描述
    Type        int       `json:"type"`        // 0: 立即执行, 1: cron任务
    CronExpr    string    `json:"cron_expr"`   // Cron表达式
    Interval    int       `json:"interval"`    // 固定间隔(秒)
    Message     string    `json:"message"`     // 触发消息
    Channel     string    `json:"channel"`     // 投递通道
    ChatID      string    `json:"chat_id"`     // 投递目标
    Enabled     bool      `json:"enabled"`     // 是否启用
    NextRunAt   time.Time `json:"next_run_at"` // 下次执行时间
    LastRunAt   time.Time `json:"last_run_at"` // 上次执行时间
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### ProviderConfig (Provider配置)
```go
type ProviderConfig struct {
    ID       uint      `json:"id"`
    Name     string    `json:"name"`     // openai, anthropic...
    Enabled  bool      `json:"enabled"`  // 是否启用
    Config   string    `json:"config"`   // JSON配置
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Skill (技能)
```go
type Skill struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`        // 技能名称
    Source      string    `json:"source"`      // builtin, workspace, remote
    Description string    `json:"description"` // 技能描述
    Content     string    `json:"content"`     // SKILL.md内容
    Enabled     bool      `json:"enabled"`     // 是否启用
    AlwaysLoad  bool      `json:"always_load"` // 是否总是加载
    Metadata    string    `json:"metadata"`    // JSON元数据
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### ChannelConfig (通道配置)
```go
type ChannelConfig struct {
    ID        uint      `json:"id"`
    Name      string    `json:"name"`      // telegram, discord...
    Enabled   bool      `json:"enabled"`   // 是否启用
    Config    string    `json:"config"`    // JSON配置
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

## 注意事项

1. 所有创建和更新操作都会自动设置 `created_at` 和 `updated_at` 时间戳
2. 分页查询的 `page` 参数从 1 开始
3. 记忆(Memory)支持软删除，使用 `soft-delete` 操作而不是 `delete`
4. 任务(Task)可以通过 `toggle` 操作快速切换启用状态
5. 技能(Skill)支持 `upsert` 操作，可以根据名称自动判断创建或更新