# icooclaw Agent 目标架构设计

> **设计原则**：不考虑向后兼容，直接采用最佳实践

## 一、目标架构概览

采用 **标准 Go 项目布局**，参考 picoclaw 的成熟设计：

```
agent/
├── cmd/                          # 应用入口
│   └── icooclaw/
│       └── main.go
├── pkg/                          # 公共库（单一模块）
│   ├── agent/                    # Agent 核心
│   │   ├── agent.go              # Agent 结构体（职责精简）
│   │   ├── loop.go               # Agent 循环
│   │   ├── context.go            # 上下文构建器
│   │   └── registry.go           # Agent 注册表
│   ├── providers/                # LLM Provider
│   │   ├── base.go               # Provider 接口
│   │   ├── factory.go            # Provider 工厂
│   │   ├── fallback.go           # Fallback 链
│   │   ├── openai.go             # OpenAI 实现
│   │   ├── anthropic.go          # Anthropic 实现
│   │   ├── deepseek.go           # DeepSeek 实现
│   │   └── ...
│   ├── tools/                    # 工具系统
│   │   ├── registry.go           # 工具注册表
│   │   ├── executor.go           # 同步/异步执行器
│   │   ├── context.go            # 工具上下文注入
│   │   └── builtin/              # 内置工具
│   ├── channels/                 # 消息通道
│   │   ├── manager.go            # 通道管理器
│   │   ├── worker.go             # 通道工作器
│   │   └── rate_limiter.go       # 速率限制
│   ├── bus/                      # 消息总线
│   │   ├── bus.go                # 总线实现
│   │   └── message.go            # 消息类型
│   ├── config/                   # 配置管理
│   │   ├── config.go             # 配置结构
│   │   ├── loader.go             # 配置加载
│   │   └── defaults.go           # 默认值
│   ├── storage/                  # 数据存储
│   │   ├── storage.go            # 存储接口
│   │   └── sqlite.go             # SQLite 实现
│   ├── memory/                   # 记忆系统
│   │   ├── loader.go             # 记忆加载器
│   │   └── summarizer.go         # 摘要生成
│   ├── mcp/                      # MCP 协议
│   │   ├── client.go             # MCP 客户端
│   │   └── tools.go              # MCP 工具适配
│   ├── skill/                    # 技能系统
│   │   ├── loader.go             # 技能加载器
│   │   └── executor.go           # 技能执行器
│   └── hooks/                    # 钩子系统
│       └── hooks.go              # 钩子接口
├── config/                       # 配置示例
│   └── config.example.toml
├── scripts/                      # 构建脚本
├── Makefile                      # 构建管理
├── .golangci.yaml               # Lint 配置
├── .goreleaser.yaml             # 发布配置
├── Dockerfile                    # 容器化
└── docker-compose.yaml          # 编排配置
```

## 二、核心模块设计

### 2.1 Agent 核心 (`pkg/agent/`)

#### 精简的 Agent 结构体

```go
// agent.go
type Agent struct {
    name      string
    workspace string
    sessionID string
    
    // 核心依赖（通过依赖注入）
    provider  providers.Provider      // 单一 Provider
    tools     *tools.Registry         // 工具注册表
    memory    memory.Loader           // 记忆加载器
    skills    skill.Loader            // 技能加载器
    storage   *storage.Storage        // 数据存储
    bus       *bus.MessageBus         // 消息总线
    
    logger    *slog.Logger
}

// 职责精简：仅负责消息处理循环
func (a *Agent) Run(ctx context.Context) error {
    for {
        select {
        case msg := <-a.bus.Inbound():
            go a.processMessage(ctx, msg)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

#### Agent 循环分离

```go
// loop.go - 参考 picoclaw 的 AgentLoop
type AgentLoop struct {
    bus            *bus.MessageBus
    provider       *providers.FallbackChain  // Fallback 链
    tools          *tools.Registry
    memory         memory.Loader
    skills         skill.Loader
    storage        *storage.Storage
    channelManager *channels.Manager
    
    running     atomic.Bool
    summarizing sync.Map
}

type processOptions struct {
    SessionKey      string
    Channel         string
    ChatID          string
    UserMessage     string
    Media           []string
    EnableSummary   bool
}
```

### 2.2 Provider 系统 (`pkg/providers/`)

#### Provider 接口

```go
// base.go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error
    GetDefaultModel() string
    GetName() string
}

// 移除敏感方法：GetAPIKey(), GetAPIBase()
```

#### Provider 工厂

```go
// factory.go
type Factory struct {
    configs map[string]ProviderConfig
}

func (f *Factory) Create(model string) (Provider, error) {
    // 1. 从配置查找 Provider
    // 2. 支持从模型名称推断（如 "openrouter/xxx" -> OpenRouter）
    // 3. 支持 model_list 零代码添加模型
}

func (f *Factory) CreateChain(models ...string) (*FallbackChain, error) {
    // 创建 Fallback 链
}
```

#### Fallback 链

```go
// fallback.go
type FailoverReason string

const (
    FailoverAuth       FailoverReason = "auth"
    FailoverRateLimit  FailoverReason = "rate_limit"
    FailoverTimeout    FailoverReason = "timeout"
    FailoverFormat     FailoverReason = "format"
)

type FailoverError struct {
    Reason   FailoverReason
    Provider string
    Model    string
    Status   int
    Wrapped  error
}

func (e *FailoverError) IsRetriable() bool {
    return e.Reason != FailoverFormat
}

type FallbackChain struct {
    providers   []Provider
    cooldowns   map[string]time.Time
    mu          sync.RWMutex
}

func (fc *FallbackChain) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // 自动重试和降级
}
```

### 2.3 工具系统 (`pkg/tools/`)

#### 工具注册表

```go
// registry.go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]Parameter
    Execute(ctx context.Context, args map[string]any) *ToolResult
}

type AsyncExecutor interface {
    ExecuteAsync(ctx context.Context, args map[string]any, callback AsyncCallback) *ToolResult
}

type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

// 确定性排序（LLM 缓存友好）
func (r *Registry) ListTools() []Tool {
    names := r.sortedToolNames()
    // ...
}

// 带上下文执行
func (r *Registry) ExecuteWithContext(
    ctx context.Context,
    name string,
    args map[string]any,
    channel, chatID string,
    asyncCallback AsyncCallback,
) *ToolResult {
    ctx = WithToolContext(ctx, channel, chatID)
    // ...
}
```

#### 工具上下文

```go
// context.go
type toolContextKey struct{}

type ToolContext struct {
    Channel string
    ChatID  string
}

func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
    return context.WithValue(ctx, toolContextKey{}, ToolContext{
        Channel: channel,
        ChatID:  chatID,
    })
}

func GetToolContext(ctx context.Context) *ToolContext {
    if tc, ok := ctx.Value(toolContextKey{}).(ToolContext); ok {
        return &tc
    }
    return nil
}
```

### 2.4 通道管理器 (`pkg/channels/`)

```go
// manager.go
type Manager struct {
    channels   map[string]Channel
    workers    map[string]*channelWorker
    bus        *bus.MessageBus
    httpServer *http.Server  // 共享 HTTP 服务器（Webhook）
    
    mu sync.RWMutex
}

type channelWorker struct {
    channel  Channel
    limiter  *rate.Limiter
    inbox    chan bus.OutboundMessage
    done     chan struct{}
}

// 速率限制配置
var channelRateConfig = map[string]float64{
    "telegram": 20,
    "discord":  1,
    "slack":    1,
    "web":      100,
}

// 带重试的发送
func (m *Manager) sendWithRetry(ctx context.Context, name string, w *channelWorker, msg bus.OutboundMessage) {
    // 速率限制
    // 指数退避重试
}
```

### 2.5 消息总线 (`pkg/bus/`)

```go
// bus.go
type MessageBus struct {
    inbound       chan InboundMessage
    outbound      chan OutboundMessage
    outboundMedia chan OutboundMediaMessage
    done          chan struct{}
    closed        atomic.Bool
    
    // 背压控制
    inboundCapacity  int
    outboundCapacity int
    dropCount        atomic.Int64
}

func (mb *MessageBus) Close() {
    if mb.closed.CompareAndSwap(false, true) {
        close(mb.done)
        // 排空缓冲通道
    }
}

// 背压感知发送
func (mb *MessageBus) TrySend(msg OutboundMessage) error {
    select {
    case mb.outbound <- msg:
        return nil
    default:
        mb.dropCount.Add(1)
        return ErrBufferFull
    }
}
```

### 2.6 配置管理 (`pkg/config/`)

```go
// config.go
type Config struct {
    Agents    AgentsConfig    `json:"agents"`
    Bindings  []AgentBinding  `json:"bindings,omitempty"`
    Session   SessionConfig   `json:"session,omitempty"`
    Channels  ChannelsConfig  `json:"channels"`
    Providers ProvidersConfig `json:"providers,omitempty"`
    ModelList []ModelConfig   `json:"model_list"`      // 零代码添加模型
    Gateway   GatewayConfig   `json:"gateway"`
    Tools     ToolsConfig     `json:"tools"`
}

type AgentDefaults struct {
    Workspace string `json:"workspace" env:"ICOOCLAW_WORKSPACE"`
    ModelName string `json:"model_name" env:"ICOOCLAW_MODEL"`
    Provider  string `json:"provider" env:"ICOOCLAW_PROVIDER"`
}

// 环境变量覆盖
func Load(path string) (*Config, error) {
    // 1. 加载配置文件
    // 2. 环境变量覆盖
    // 3. 验证配置
}
```

## 三、错误处理统一

```go
// pkg/errors/errors.go
package errors

import "errors"

// 错误类型
var (
    ErrProviderUnavailable = errors.New("provider unavailable")
    ErrRateLimited         = errors.New("rate limited")
    ErrTimeout             = errors.New("timeout")
    ErrInvalidConfig       = errors.New("invalid config")
    ErrToolNotFound        = errors.New("tool not found")
    ErrSessionNotFound     = errors.New("session not found")
)

// 分类错误
type ClassifiedError struct {
    Code     string
    Message  string
    Retriable bool
    Cause    error
}

func (e *ClassifiedError) Error() string {
    return e.Message
}

func (e *ClassifiedError) Unwrap() error {
    return e.Cause
}

// 错误分类
func Classify(err error) *ClassifiedError {
    // 根据错误类型返回分类
}
```

## 四、依赖关系

```
cmd/icooclaw
    └── pkg/agent
            ├── pkg/providers
            ├── pkg/tools
            ├── pkg/channels
            ├── pkg/bus
            ├── pkg/config
            ├── pkg/storage
            ├── pkg/memory
            ├── pkg/mcp
            ├── pkg/skill
            └── pkg/hooks
```

**原则**：
- 单向依赖，无循环
- 接口定义在使用方
- 依赖注入，便于测试

## 五、与现有架构对比

| 维度 | 现有架构 | 目标架构 |
|-----|---------|---------|
| **模块组织** | Go Workspace 多模块 | 单一模块 pkg/ 布局 |
| **Agent 职责** | 过重（管理 Provider 缓存等） | 精简（仅消息处理） |
| **Provider** | 简单注册表 | 工厂 + Fallback 链 |
| **工具执行** | 仅同步 | 同步/异步 + 上下文注入 |
| **通道管理** | 基础启动/停止 | 速率限制 + 重试 + TTL |
| **消息总线** | 无背压控制 | 背压感知 + 安全关闭 |
| **配置** | 分散 | 统一 + 环境变量覆盖 |
| **错误处理** | 不一致 | 统一分类 + 重试策略 |

## 六、迁移策略

由于不考虑兼容，采用 **重写迁移** 策略：

1. **保留**：业务逻辑代码（如 ReAct 循环、工具实现）
2. **重写**：架构层代码（模块组织、接口定义）
3. **参考**：picoclaw 的成熟实现

### 迁移步骤

```
Step 1: 创建新的 pkg/ 目录结构
Step 2: 迁移并重构 core 模块（bus, config, storage）
Step 3: 迁移并重构 providers（添加工厂和 Fallback）
Step 4: 迁移并重构 tools（添加异步和上下文）
Step 5: 迁移并重构 agent（精简职责）
Step 6: 迁移 channels（添加管理器）
Step 7: 更新 cmd/ 入口
Step 8: 添加工程化配置
Step 9: 删除旧模块
```

## 七、测试策略

```go
// 使用 mock Provider 进行单元测试
type MockProvider struct {
    Response *ChatResponse
    Error    error
}

func (m *MockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    return m.Response, m.Error
}

// 集成测试使用测试服务器
func TestAgentLoop(t *testing.T) {
    // 启动 mock LLM 服务器
    // 创建 AgentLoop
    // 发送测试消息
    // 验证响应
}
```