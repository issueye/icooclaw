# 架构设计

本文档描述 icooclaw 的系统架构和模块设计。

## 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Channels                        │
│         ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐     │
│         │ Feishu  │  │DingTalk │  │   Web   │  │   CLI   │     │
│         └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘     │
└──────────────┼────────────┼────────────┼────────────┼───────────┘
               │            │            │            │
               ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Channel Manager                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Channel Registry  │  Message Router  │  Rate Limiter   │   │
│  └─────────────────────────────────────────────────────────┘   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Message Bus                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Inbound    │  │   Outbound   │  │    Media     │          │
│  │   Channel    │  │   Channel    │  │   Channel    │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Agent Loop                              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  1. Route Message  →  2. Load Context  →  3. LLM Call   │   │
│  │  4. Tool Execute  ←  5. Parse Response ←  6. Iterate    │   │
│  └─────────────────────────────────────────────────────────┘   │
└──────────┬──────────────────────┬───────────────────────────────┘
           │                      │
           ▼                      ▼
┌──────────────────┐    ┌──────────────────────────────────────┐
│   Tool Registry  │    │           Provider Factory           │
│  ┌────────────┐  │    │  ┌─────────┐  ┌─────────┐           │
│  │  Built-in  │  │    │  │ OpenAI  │  │ Claude  │  ...      │
│  │  MCP Tool  │  │    │  └─────────┘  └─────────┘           │
│  │  Script    │  │    │  ┌─────────┐  ┌─────────┐           │
│  └────────────┘  │    │  │ Gemini  │  │DeepSeek │  ...      │
└──────────────────┘    │  └─────────┘  └─────────┘           │
                        │  ┌─────────────────────────────┐    │
                        │  │     Fallback Chain          │    │
                        │  └─────────────────────────────┘    │
                        └──────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Storage Layer                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │ Provider │ │ Channel  │ │ Session  │ │  Memory  │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │   Tool   │ │  Skill   │ │ Binding  │ │   MCP    │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                        SQLite + GORM                             │
└─────────────────────────────────────────────────────────────────┘
```

## 核心模块

### 1. Agent 模块 (`pkg/agent/`)

Agent 是框架的核心，负责消息处理和 LLM 交互。

#### AgentInstance

```go
type AgentInstance struct {
    config AgentConfig
    
    provider providers.Provider
    tools    *tools.Registry
    memory   memory.Loader
    skills   skill.Loader
    storage  *storage.Storage
    bus      *bus.MessageBus
    
    sessionCache sync.Map
    logger *slog.Logger
}
```

#### AgentLoop

消息处理循环，实现 ReAct 模式：

```go
type Loop struct {
    bus               *bus.MessageBus
    provider          providers.Provider
    tools             *tools.Registry
    memory            memory.Loader
    maxToolIterations int
    
    running atomic.Bool
}
```

**处理流程：**

1. 从 MessageBus 接收消息
2. 路由到对应的 Agent
3. 加载会话历史和上下文
4. 调用 LLM 获取响应
5. 执行工具调用（如有）
6. 保存会话记忆
7. 发送响应到 MessageBus

### 2. Message Bus (`pkg/bus/`)

消息总线负责组件间的异步通信。

```go
type MessageBus struct {
    inbound       chan InboundMessage
    outbound      chan OutboundMessage
    outboundMedia chan OutboundMediaMessage
    done          chan struct{}
    closed        atomic.Bool
}
```

**特性：**
- Context 支持，可取消和超时
- 优雅关闭，避免消息丢失
- 订阅模式，支持多消费者

### 3. Channels 模块 (`pkg/channels/`)

渠道模块负责与外部平台对接。

#### Channel 接口

```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Send(ctx context.Context, msg OutboundMessage) error
    IsRunning() bool
}
```

#### 扩展接口

```go
// 消息编辑
type MessageEditor interface {
    EditMessage(ctx context.Context, chatID, messageID, content string) error
}

// 打字指示器
type TypingCapable interface {
    StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// 消息反应
type ReactionCapable interface {
    ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

// 媒体发送
type MediaSender interface {
    SendMedia(ctx context.Context, msg OutboundMediaMessage) error
}
```

### 4. Providers 模块 (`pkg/providers/`)

提供商模块封装各种 LLM API。

#### Provider 接口

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error
    GetDefaultModel() string
    GetName() string
}
```

#### 支持的提供商

| 提供商 | 模型示例 | 特性 |
|--------|----------|------|
| OpenAI | gpt-4, gpt-4o | 流式输出、工具调用 |
| Anthropic | claude-3-opus | 流式输出、工具调用 |
| Gemini | gemini-pro | 流式输出 |
| DeepSeek | deepseek-chat | 推理模型支持 |
| OpenRouter | openrouter/* | 多模型代理 |
| Groq | llama-* | 高速推理 |
| Mistral | mistral-* | 开源模型 |
| Ollama | * | 本地模型 |
| Azure OpenAI | gpt-* | 企业部署 |

#### Fallback Chain

自动故障转移机制：

```go
type FallbackChain struct {
    providers []Provider
    cooldowns map[string]time.Time
}
```

### 5. Tools 模块 (`pkg/tools/`)

工具系统支持 LLM 调用外部功能。

#### Tool 接口

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any
    Execute(ctx context.Context, args map[string]any) *Result
}
```

#### 内置工具

| 工具 | 说明 |
|------|------|
| `http_request` | HTTP 请求 |
| `web_search` | Web 搜索 (DuckDuckGo) |
| `datetime` | 日期时间 |
| `read_file` | 读取文件 |
| `write_file` | 写入文件 |
| `list_dir` | 列出目录 |
| `exec` | 执行命令 |
| `script` | 执行 JavaScript |

### 6. MCP 模块 (`pkg/mcp/`)

支持 Model Context Protocol，可扩展工具生态。

```go
type Manager struct {
    clients map[string]*Client
    tools   *tools.Registry
}

func (m *Manager) LoadFromConfig(ctx context.Context, cfg MCPConfig) error
```

### 7. Gateway 模块 (`pkg/gateway/`)

HTTP Gateway 提供 RESTful API 和 WebSocket 支持。

#### 组件

- **Server** - HTTP 服务器
- **WebSocket Manager** - WebSocket 连接管理
- **SSE Broker** - Server-Sent Events
- **Handlers** - API 处理器
- **Middleware** - 认证、日志等

### 8. Storage 模块 (`pkg/storage/`)

数据持久化层，使用 SQLite + GORM。

#### 数据模型

| 模型 | 说明 |
|------|------|
| Provider | 提供商配置 |
| Channel | 渠道配置 |
| Session | 会话信息 |
| Memory | 对话记忆 |
| Tool | 工具配置 |
| Skill | 技能配置 |
| Binding | Agent 绑定 |
| MCP | MCP 服务器配置 |

### 9. Scheduler 模块 (`pkg/scheduler/`)

任务调度器，支持 Cron 表达式。

```go
type Scheduler struct {
    cron  *cron.Cron
    tasks map[string]*Task
}

func (s *Scheduler) AddTask(task *Task) error
func (s *Scheduler) RunTask(id string) error
```

### 10. Script 模块 (`pkg/script/`)

JavaScript 脚本引擎，基于 goja。

```go
type Engine struct {
    vm       *goja.Runtime
    cfg      *Config
    builtins []Builtin
}
```

**内置模块：**
- `console` - 控制台输出
- `fs` - 文件系统
- `http` - HTTP 请求
- `shell` - Shell 命令
- `crypto` - 加密函数

## 数据流

### 消息处理流程

```
1. 用户发送消息
   ↓
2. Channel 接收并转换为 InboundMessage
   ↓
3. MessageBus.PublishInbound()
   ↓
4. AgentLoop 从 MessageBus 消费消息
   ↓
5. 路由到对应的 Agent
   ↓
6. 加载会话历史和记忆
   ↓
7. 构建消息上下文
   ↓
8. 调用 LLM Provider
   ↓
9. 解析响应，执行工具调用（如有）
   ↓
10. 生成最终响应
   ↓
11. MessageBus.PublishOutbound()
   ↓
12. Channel 发送响应给用户
```

### 工具调用流程

```
1. LLM 返回 ToolCall
   ↓
2. AgentLoop 检测到 ToolCall
   ↓
3. 从 ToolRegistry 获取工具
   ↓
4. 执行工具，获取结果
   ↓
5. 将结果添加到消息历史
   ↓
6. 重新调用 LLM
   ↓
7. 重复直到无 ToolCall 或达到最大迭代次数
```

## 扩展机制

### 添加新渠道

1. 实现 `Channel` 接口
2. 注册到 `Registry`

```go
func init() {
    channels.RegisterFactory("my_channel", func(cfg map[string]any, bus *bus.MessageBus) (channels.Channel, error) {
        return NewMyChannel(cfg, bus)
    })
}
```

### 添加新提供商

1. 实现 `Provider` 接口
2. 注册到 `Registry`

```go
func init() {
    providers.RegisterFactory("my_provider", NewMyProvider)
}
```

### 添加新工具

1. 实现 `Tool` 接口
2. 注册到 `ToolRegistry`

```go
registry.Register(NewMyTool())
```

## 性能考虑

### 并发安全

- 所有共享状态使用 `sync.Map` 或 `sync.RWMutex` 保护
- 使用 `atomic.Bool` 进行状态标记

### 资源管理

- MessageBus 使用带缓冲的 channel
- 工具执行支持超时控制
- HTTP 请求支持连接池

### 内存优化

- 会话历史限制最大条目数
- 记忆支持摘要压缩
- 工具定义排序确保 KV Cache 稳定性