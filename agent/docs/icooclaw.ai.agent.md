# icooclaw.ai/agent 模块概要说明

## 模块概述

`icooclaw.ai/agent` 模块是 icooclaw AI 框架的核心 Agent 实现，采用 **ReAct（推理-行动）模式** 构建，提供了智能对话代理的完整生命周期管理能力。

## 架构设计

### 核心组件

```
┌─────────────────────────────────────────────────────────────┐
│                         Agent                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │   Provider  │  │   Tools     │  │   Memory    │          │
│  │  (LLM调用)  │  │  (工具注册) │  │  (记忆存储) │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  ContextBuilder                       │   │
│  │  (系统提示词构建 / 历史消息管理 / 记忆加载)           │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   ReActAgent                          │   │
│  │  (推理-行动循环 / 工具调用执行 / 流式响应处理)        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
              ┌────────────────────────┐
              │     MessageBus         │
              │   (消息输入/输出)       │
              └────────────────────────┘
```

---

## 文件说明

### 1. agent.go - Agent 核心实现

**主要职责：** Agent 实例管理、消息处理循环、生命周期控制

#### 核心结构体

```go
type Agent struct {
    name      string                    // Agent 名称
    provider  provider.Provider         // LLM 提供者
    tools     ToolRegistryInterface     // 工具注册表
    storage   StorageInterface          // 存储接口
    memory    MemoryStoreInterface      // 记忆存储
    skills    SkillLoaderInterface      // 技能加载器
    config    config.AgentSettings      // 配置
    bus       MessageBusInterface       // 消息总线
    logger    *slog.Logger              // 日志
    workspace string                    // 工作目录
}
```

#### 核心功能

| 方法 | 说明 |
|------|------|
| `NewAgent()` | 使用函数式选项模式创建 Agent 实例 |
| `Init()` | 初始化 Agent（加载技能、记忆） |
| `Run()` | 启动消息处理主循环 |
| `ProcessMessage()` | 处理单条消息（同步模式） |
| `SetProvider()` | 运行时切换 LLM 提供者 |
| `RegisterTool()` | 注册工具 |

#### 消息处理流程

```
InboundMessage → 获取/创建会话 → 保存用户消息 → 构建上下文
                                                      ↓
                                               ReActAgent.Run()
                                                      ↓
OutboundMessage ← 流式输出 ← 工具执行 ← LLM 响应 ← ────┘
```

---

### 2. react.go - ReAct 模式实现

**主要职责：** 实现 ReAct（推理-行动）循环逻辑

#### 核心结构体

```go
type ReActConfig struct {
    MaxIterations int                    // 最大迭代次数，默认 10
    Provider      provider.Provider      // LLM 提供者
    Tools         ToolRegistryInterface  // 工具注册表
    Session       *storage.Session       // 会话
    Logger        *slog.Logger           // 日志
    Hooks         hooks.ReActHooks       // 钩子接口
}

type ReActAgent struct {
    config *ReActConfig
}
```

#### ReAct 循环流程

```
┌─────────────────────────────────────────────────────────────┐
│                    ReAct Loop                               │
│                                                              │
│   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐ │
│   │ 构建    │───▶│ LLM     │───▶│ 解析    │───▶│ 有工具? │ │
│   │ Request │    │ 调用    │    │ 响应    │    │         │ │
│   └─────────┘    └─────────┘    └─────────┘    └────┬────┘ │
│                                                       │      │
│                    ┌──────────────────────────────────┤      │
│                    │ Yes                              │ No   │
│                    ▼                                  ▼      │
│              ┌─────────┐                       ┌──────────┐  │
│              │ 执行    │                       │ 返回     │  │
│              │ 工具    │                       │ 结果     │  │
│              └────┬────┘                       └──────────┘  │
│                   │                                           │
│                   ▼                                           │
│              ┌─────────┐                                      │
│              │ 添加到  │                                      │
│              │ 消息列表│                                      │
│              └────┬────┘                                      │
│                   │                                           │
│                   └──────────────────────────▶ 循环 ◀─────────┘
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### 流式响应处理

- 支持 OpenAI o1 风格的独立 `reasoning_content` 字段
- 支持 DeepSeek 风格的 `思考和内容` 标签解析
- 工具调用按 index 累积参数

---

### 3. context.go - 上下文构建器

**主要职责：** 构建系统提示词、管理对话上下文

#### 核心结构体

```go
type ContextBuilder struct {
    agent   AgentContext           // Agent 上下文接口
    session *storage.Session        // 会话
    logger  *slog.Logger           // 日志
    fs      FileSystemInterface    // 文件系统接口
}
```

#### 系统提示词构建

系统提示词按优先级组合以下内容：

| 优先级 | 来源 | 说明 |
|--------|------|------|
| 1 | `SOUL.md` | AI 身份与人格定义 |
| 2 | `USER.md` | 用户信息 |
| 3 | 会话角色提示词 | 用户设定的角色（最高优先） |
| 4 | 默认系统提示词 | Agent 配置 |
| 5 | 环境信息 | 操作系统、工作目录 |
| 6 | `memory/MEMORY.md` | 用户记忆 |
| 7 | 技能提示词 | 加载的技能内容 |
| 8 | 长期记忆 | Memory Store 中的记忆 |

#### 记忆文件结构 (MEMORY.md)

```markdown
# 记忆

## 重要事实
<!-- 重要事实和信息 -->

## 用户偏好
<!-- 用户偏好和设置 -->

## 学到的知识
<!-- 从对话中学习的知识 -->

## 最后更新
<!-- 最后记忆更新的时间戳 -->
```

---

### 4. interfaces.go - 接口定义

**主要职责：** 定义核心接口，实现模块解耦

#### 核心接口

```go
// StorageInterface - 存储接口
type StorageInterface interface {
    GetOrCreateSession(channel, chatID, userID string) (*storage.Session, error)
    AddMessage(sessionID uint, role, content, toolCalls, toolCallID, toolName, reasoningContent string) (*storage.Message, error)
    GetSession(sessionID uint) (*storage.Session, error)
    UpdateSessionMetadata(sessionID uint, metadata string) error
    CreateTask(task *storage.Task) error
    GetTaskByName(name string) (*storage.Task, error)
    GetAllTasks() ([]storage.Task, error)
    DeleteTask(id uint) error
}

// ToolRegistryInterface - 工具注册表接口
type ToolRegistryInterface interface {
    Register(tool tools.Tool)
    Get(name string) (tools.Tool, error)
    ToDefinitions() []tools.ToolDefinition
    Execute(ctx context.Context, call interface{}) tools.ToolResult
}

// MemoryStoreInterface - 记忆存储接口
type MemoryStoreInterface interface {
    Load(ctx context.Context) error
    GetAll() ([]storage.Memory, error)
    Add(memType, key, content string) error
    Consolidate(session *storage.Session) error
    RememberHistory(key, content string) error
}

// SkillLoaderInterface - 技能加载接口
type SkillLoaderInterface interface {
    Load(ctx context.Context) error
    GetLoaded() []skill.Skill
    GetByName(name string) *skill.Skill
}

// MessageBusInterface - 消息总线接口
type MessageBusInterface interface {
    ConsumeInbound(ctx context.Context) (icooclawbus.InboundMessage, error)
    PublishOutbound(ctx context.Context, msg icooclawbus.OutboundMessage) error
}

// AgentContext - Agent 上下文接口
type AgentContext interface {
    Logger() *slog.Logger
    Workspace() string
    Skills() SkillLoaderInterface
    Memory() MemoryStoreInterface
    Config() config.AgentSettings
    GetSessionRolePrompt(sessionID uint) (string, error)
    GetSystemPrompt() string
}

// FileSystemInterface - 文件系统接口
type FileSystemInterface interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte, perm os.FileMode) error
    Stat(path string) (os.FileInfo, error)
}
```

#### 接口依赖关系

```
Agent
  ├── StorageInterface     → storage.Storage
  ├── ToolRegistryInterface → tools.Registry
  ├── MemoryStoreInterface  → memory.MemoryStore
  ├── SkillLoaderInterface  → skill.Loader
  ├── MessageBusInterface   → bus.MessageBus
  └── FileSystemInterface   → OsFileSystem
```

---

### 5. context_test.go - 单元测试

**主要职责：** ContextBuilder 结构体验证

```go
func TestContextBuilder_Structure(t *testing.T) {
    cb := &ContextBuilder{}
    assert.NotNil(t, cb)
}
```

---

## 设计模式

### 1. 函数式选项模式 (Functional Options)

用于 Agent 初始化，提供灵活的配置方式：

```go
agent := NewAgent(
    name,
    provider,
    storage,
    config,
    logger,
    workspace,
    WithTools(customRegistry),
    WithMemoryStore(memoryStore),
    WithSkillLoader(skillLoader),
    WithMessageBus(messageBus),
)
```

### 2. 钩子模式 (Hooks)

用于在 ReAct 循环的关键节点注入自定义逻辑：

```go
type ReActHooks interface {
    OnLLMRequest(ctx context.Context, req *provider.ChatRequest, iteration int) error
    OnLLMChunk(ctx context.Context, content, thinking string) error
    OnLLMResponse(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iteration int) error
    OnToolCall(ctx context.Context, toolCallID string, toolName string, arguments string) error
    OnToolResult(ctx context.Context, toolCallID string, toolName string, result tools.ToolResult) error
    OnIterationStart(ctx context.Context, iteration int, messages []provider.Message) error
    OnIterationEnd(ctx context.Context, iteration int, hasToolCalls bool) error
    OnError(ctx context.Context, err error) error
    OnComplete(ctx context.Context, content, reasoningContent string, toolCalls []provider.ToolCall, iterations int) error
}
```

### 3. 依赖注入

通过接口实现模块解耦，便于测试和替换实现：

```go
type AgentContext interface {
    Logger() *slog.Logger
    Workspace() string
    // ...
}
```

---

## 消息流转

```
用户输入
    │
    ▼
┌────────────────┐
│  MessageBus    │
│ ConsumeInbound │
└───────┬────────┘
        │
        ▼
┌────────────────┐
│     Agent      │
│ handleMessage  │
└───────┬────────┘
        │
        ▼
┌────────────────┐     ┌──────────────┐
│ContextBuilder  │────▶│ Storage      │
│    Build       │     │ GetOrCreate  │
└───────┬────────┘     │ Session      │
        │              └──────────────┘
        ▼
┌────────────────┐
│  ReActAgent    │
│     Run        │
└───────┬────────┘
        │
        ├──────────────────────┐
        │                      │
        ▼                      ▼
┌────────────────┐    ┌──────────────┐
│    Provider    │    │    Tools     │
│  ChatStream    │    │   Execute    │
└───────┬────────┘    └──────┬───────┘
        │                    │
        └─────────┬──────────┘
                  │
                  ▼
        ┌────────────────┐
        │  MessageBus    │
        │PublishOutbound │
        └────────────────┘
                  │
                  ▼
              用户输出
```

---

## 配置说明

### AgentSettings

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `SystemPrompt` | string | 默认系统提示词 |
| `MemoryWindow` | int | 记忆窗口大小 |
| 其他配置 | - | 继承自 config.AgentSettings |

---

## 使用示例

### 基本使用

```go
// 创建 Agent
agent := agent.NewAgent(
    "my-agent",
    provider,
    storage,
    config,
    logger,
    workspace,
    agent.WithTools(tools.NewRegistry()),
    agent.WithMemoryStore(memoryStore),
)

// 初始化
if err := agent.Init(ctx); err != nil {
    log.Fatal(err)
}

// 启动消息循环
go agent.Run(ctx, messageBus)
```

### 同步处理单条消息

```go
response, err := agent.ProcessMessage(ctx, "你好，请介绍一下自己")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response)
```

---

## 依赖关系

```
icooclaw.ai/agent
    ├── icooclaw.ai/hooks       (钩子接口)
    ├── icooclaw.ai/provider    (LLM 提供者)
    ├── icooclaw.ai/skill       (技能加载)
    ├── icooclaw.ai/tools       (工具注册)
    ├── icooclaw.core/bus       (消息总线)
    ├── icooclaw.core/config    (配置)
    ├── icooclaw.core/consts    (常量)
    └── icooclaw.core/storage   (存储)
```

---

## 扩展指南

### 添加新工具

```go
// 实现工具接口
type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "我的自定义工具" }
func (t *MyTool) Parameters() tools.Parameters { return tools.Parameters{...} }
func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) tools.ToolResult {
    // 工具逻辑
    return tools.ToolResult{Content: "结果"}
}

// 注册工具
agent.RegisterTool(&MyTool{})
```

### 添加新钩子

```go
// 实现钩子接口
type MyHooks struct{}

func (h *MyHooks) OnToolCall(ctx context.Context, toolCallID, toolName, arguments string) error {
    // 工具调用前逻辑
    return nil
}

// 配置 ReAct
reactCfg := agent.NewReActConfig()
reactCfg.Hooks = &MyHooks{}
```

---

## 版本信息

- **模块路径**: `icooclaw.ai/agent`
- **Go 版本**: 1.21+
- **最后更新**: 2026-03