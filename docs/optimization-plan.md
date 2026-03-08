# icooclaw Agent 项目优化计划

> **重构策略**：不考虑向后兼容，采用激进重构策略，直接采用最佳架构设计

## 一、项目现状分析

### 1.1 当前架构

```
agent/
├── go.work                    # Workspace 多模块配置
├── icooclaw.ai/              # AI 核心模块
│   ├── agent/                # Agent 实现（职责过重）
│   ├── provider/             # LLM Provider 抽象（缺少 Fallback）
│   ├── tools/                # 工具系统（仅同步执行）
│   ├── memory/               # 记忆系统
│   ├── skill/                # 技能加载
│   ├── mcp/                  # MCP 协议
│   └── hooks/                # 钩子接口
├── icooclaw.core/            # 核心基础设施
│   ├── bus/                  # 消息总线（缺少背压控制）
│   ├── channel/              # 消息通道
│   ├── config/               # 配置管理（结构分散）
│   ├── storage/              # 数据存储
│   ├── scheduler/            # 定时任务
│   └── utils/                # 工具函数
├── icooclaw.cmd/             # CLI 命令模块
└── icooclaw.gateway/         # HTTP Gateway 模块
```

### 1.2 主要问题

| 问题类别 | 具体问题 | 影响 |
|---------|---------|------|
| **架构设计** | Agent 结构体职责过重 | 难以测试和维护 |
| **架构设计** | 模块边界模糊，依赖关系复杂 | 代码耦合度高 |
| **Provider** | 缺少 Fallback 链机制 | 系统可靠性低 |
| **Provider** | 无错误分类和重试策略 | 故障恢复能力弱 |
| **工具系统** | 仅支持同步执行 | 长时间工具阻塞主流程 |
| **工具系统** | 无上下文注入机制 | 工具无法获取会话信息 |
| **消息总线** | 缺少背压控制 | 高负载时可能内存溢出 |
| **配置管理** | 配置结构分散 | 难以维护和扩展 |
| **工程化** | 缺少 CI/CD 配置 | 发布流程不规范 |
| **测试** | 核心逻辑测试覆盖不足 | 重构风险高 |

---

## 二、参考项目优秀实践（picoclaw）

### 2.1 架构设计

| 实践 | 说明 | 收益 |
|-----|------|-----|
| **标准 pkg/ 布局** | 清晰的包边界和职责划分 | 降低认知负担 |
| **Provider 工厂模式** | 支持零代码添加新模型 | 扩展性强 |
| **Fallback 链机制** | 自动重试和降级策略 | 提高可靠性 |
| **通道管理器** | 速率限制 + 重试 + TTL | 生产级稳定性 |

### 2.2 代码设计

| 实践 | 说明 | 收益 |
|-----|------|-----|
| **异步工具执行** | 支持长时间运行的工具 | 不阻塞主流程 |
| **上下文注入** | 工具可获取 channel/chatID | 增强工具能力 |
| **确定性工具排序** | 按名称排序工具列表 | LLM 缓存友好 |
| **分类错误处理** | 区分可重试/不可重试错误 | 精准故障恢复 |

### 2.3 工程化

| 实践 | 说明 | 收益 |
|-----|------|-----|
| **golangci-lint** | 全面的代码质量检查 | 代码规范统一 |
| **goreleaser** | 自动化多平台发布 | 发布效率高 |
| **Docker 支持** | 容器化部署 | 部署便捷 |
| **多语言文档** | README 多语言版本 | 社区友好 |

---

## 三、优化阶段规划

### 阶段一：架构重构（预计 2 周）

**目标**：简化模块结构，明确职责边界

#### 任务清单

- [ ] **T1.1** 重构 Agent 结构体，拆分职责
  - 提取 ProviderManager 管理 Provider 生命周期
  - 提取 SessionManager 管理会话状态
  - 提取 ToolExecutor 管理工具执行

- [ ] **T1.2** 重新规划模块边界
  - 合并 `icooclaw.ai` 和 `icooclaw.core` 为单一模块
  - 或明确接口边界，使用依赖注入

- [ ] **T1.3** 统一错误处理
  - 定义 `errors.go` 包含所有错误类型
  - 实现 `FailoverError` 支持错误分类
  - 添加错误包装和上下文传递

- [ ] **T1.4** 改进消息总线
  - 添加背压控制机制
  - 使用 `atomic.Bool` 管理状态
  - 实现安全的关闭机制

### 阶段二：Provider 增强（预计 1 周）

**目标**：提高 LLM 调用的可靠性和灵活性

#### 任务清单

- [ ] **T2.1** 实现 Provider 工厂模式
  - 支持从配置自动选择 Provider
  - 支持从模型名称推断 Provider
  - 支持自定义模型配置（model_list）

- [ ] **T2.2** 实现 Fallback 链机制
  - 定义 FailoverReason 枚举（auth, rate_limit, timeout, format）
  - 实现冷却追踪防止频繁重试
  - 支持配置降级策略

- [ ] **T2.3** 增强错误处理
  - 区分可重试/不可重试错误
  - 实现指数退避重试
  - 添加错误日志和监控

### 阶段三：工具系统增强（预计 1 周）

**目标**：支持更灵活的工具执行模式

#### 任务清单

- [ ] **T3.1** 支持异步工具执行
  - 定义 `AsyncExecutor` 接口
  - 实现异步执行和回调机制
  - 添加超时控制

- [ ] **T3.2** 实现上下文注入
  - 定义 `ToolContext` 结构体
  - 在执行时注入 channel/chatID
  - 支持工具获取会话信息

- [ ] **T3.3** 确定性工具排序
  - 按名称排序工具列表
  - 保证 KV 缓存稳定性

### 阶段四：Channels 模块重构（预计 2 周）

**目标**：参考 picoclaw 的 channels 设计，实现生产级多通道消息系统

#### 设计参考（picoclaw channels）

picoclaw 的 channels 包是一个设计精良的多通道消息系统，核心特点：

| 特性 | 说明 |
|-----|------|
| **Manager 统一编排** | 集中管理速率限制、重试、消息分割 |
| **能力发现机制** | 通过接口断言发现可选能力（Typing、Edit、Reaction 等） |
| **工厂注册模式** | 每个通道独立子包，通过工厂注册解耦 |
| **错误分类重试** | 区分永久失败、速率限制、临时错误，采用不同重试策略 |
| **共享 HTTP 服务器** | 统一管理 Webhook 和健康检查端点 |
| **TTL 清理机制** | 定期清理过期的状态条目，防止内存泄漏 |

#### 目标架构

```
pkg/channels/
├── manager.go              # 通道管理器（核心编排）
├── worker.go               # 通道工作器（消息队列 + 速率限制）
├── base.go                 # Channel 基础接口
├── interfaces.go           # 可选能力接口
├── errors.go               # 错误定义和分类
├── registry.go             # 工厂注册表
├── message.go              # 消息分割等工具函数
├── errutil.go              # 错误分类工具
└── telegram/               # Telegram 通道实现
    ├── telegram.go
    ├── init.go             # 工厂注册
    └── ...
```

#### 任务清单

- [ ] **T4.1** 定义核心接口 (`base.go`, `interfaces.go`)
  - `Channel` 基础接口：Name, Start, Stop, Send, IsRunning, IsAllowed
  - 可选能力接口：
    - `TypingCapable`：打字指示器
    - `MessageEditor`：消息编辑
    - `ReactionCapable`：消息反应
    - `PlaceholderCapable`：占位符消息
    - `MediaSender`：媒体发送
    - `WebhookHandler`：Webhook 处理

- [ ] **T4.2** 实现 Manager (`manager.go`)
  - 通道生命周期管理：`initChannels()`, `StartAll()`, `StopAll()`
  - 消息调度：`dispatchOutbound()`, `dispatchOutboundMedia()`
  - 速率限制：每个 Worker 独立的 `rate.Limiter`
  - 重试策略：`sendWithRetry()` 根据错误类型决定重试
  - 消息分割：`SplitMessage()` 保持代码块完整性
  - TTL 清理：`runTTLJanitor()` 定期清理过期状态

- [ ] **T4.3** 实现 Worker (`worker.go`)
  - 消息队列：`chan OutboundMessage` + `chan OutboundMediaMessage`
  - 速率限制器：根据通道类型配置不同速率
  - 优雅关闭：正确处理队列排空

- [ ] **T4.4** 实现错误处理 (`errors.go`, `errutil.go`)
  - 哨兵错误：`ErrNotRunning`, `ErrRateLimit`, `ErrTemporary`, `ErrSendFailed`
  - 错误分类：`ClassifySendError()`, `ClassifyNetError()`
  - 重试策略：
    - `ErrNotRunning` / `ErrSendFailed`：不重试
    - `ErrRateLimit`：固定延迟（1秒）
    - `ErrTemporary`：指数退避（500ms → 8s）

- [ ] **T4.5** 实现工厂注册 (`registry.go`)
  - `RegisterFactory()` 注册通道工厂
  - `GetFactory()` 获取通道工厂
  - 支持通道自注册（通过 `init()`）

- [ ] **T4.6** 实现消息工具 (`message.go`)
  - `SplitMessage()`：智能分割超长消息，保持代码块完整性
  - `FormatMessage()`：消息格式化

- [ ] **T4.7** 迁移现有通道实现
  - 迁移 Telegram 通道
  - 添加工厂注册
  - 实现可选能力接口

- [ ] **T4.8** 共享 HTTP 服务器
  - 统一管理 Webhook 端点
  - 健康检查端点
  - 优雅关闭

#### 速率限制配置

```go
var channelRateConfig = map[string]float64{
    "telegram": 20,   // 20 msg/s
    "discord":  1,    // 1 msg/s
    "slack":    1,    // 1 msg/s
    "web":      100,  // 100 msg/s
}
```

#### 消息发送流程

```
AgentLoop.PublishOutbound()
        │
        ▼
MessageBus.SubscribeOutbound()
        │
        ▼
Manager.dispatchOutbound()
        │
        ├── 检查通道是否存在
        └── 入队到 Worker.queue
        │
        ▼
Manager.runWorker()
        │
        ├── limiter.Wait() (速率限制)
        ├── preSend() (Typing/Reaction/Placeholder)
        ├── SplitMessage() (超长消息分割)
        └── sendWithRetry() (错误分类重试)
```

---

### 阶段五：配置管理与数据存储重构（预计 1 周）

**目标**：配置文件仅保留基本配置，动态数据通过 SQLite 数据库存储

#### 设计原则

| 存储位置 | 内容 | 说明 |
|---------|------|------|
| **配置文件** | 基本配置信息 | 启动时加载，运行时不变 |
| **SQLite 数据库** | 动态数据 | 运行时读写，持久化存储 |

#### 配置文件内容（config.toml）

```toml
# 基本配置信息
[agent]
workspace = "./workspace"
default_model = "gpt-4"
default_provider = "openai"

[database]
path = "./data/icooclaw.db"

[gateway]
enabled = true
port = 8080

[logging]
level = "info"
format = "json"
```

#### SQLite 数据库存储内容

| 表名 | 内容 | 说明 |
|-----|------|------|
| `providers` | Provider 配置 | API Key、Endpoint、模型列表等敏感信息 |
| `channels` | 通道配置 | Token、Webhook URL、权限列表等 |
| `sessions` | 会话状态 | 会话历史、上下文、摘要等 |
| `tools` | 工具配置 | 自定义工具定义、权限等 |
| `skills` | 技能配置 | 技能定义、参数等 |
| `memory` | 记忆数据 | 长期记忆、用户偏好等 |
| `bindings` | Agent 绑定 | 通道与 Agent 的绑定关系 |

#### 数据库表结构设计

```sql
-- Provider 配置表
CREATE TABLE providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,           -- openai, anthropic, deepseek, etc.
    api_key TEXT,                 -- 加密存储
    api_base TEXT,
    default_model TEXT,
    models TEXT,                  -- JSON 数组
    config TEXT,                  -- JSON 扩展配置
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 通道配置表
CREATE TABLE channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,           -- telegram, discord, slack, etc.
    enabled BOOLEAN DEFAULT true,
    config TEXT,                  -- JSON 配置（token, webhook_url 等）
    permissions TEXT,             -- JSON 权限列表
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 会话表
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL UNIQUE,
    channel TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    agent_name TEXT,
    context TEXT,                 -- JSON 上下文
    summary TEXT,
    last_active DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Agent 绑定表
CREATE TABLE bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel TEXT NOT NULL,
    chat_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(channel, chat_id)
);

-- 记忆表
CREATE TABLE memory (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,           -- user, assistant, system
    content TEXT NOT NULL,
    metadata TEXT,                -- JSON 元数据
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 工具配置表
CREATE TABLE tools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,           -- builtin, mcp, custom
    definition TEXT,              -- JSON 工具定义
    config TEXT,                  -- JSON 配置
    enabled BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 技能配置表
CREATE TABLE skills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    prompt TEXT,
    tools TEXT,                   -- JSON 工具列表
    config TEXT,                  -- JSON 配置
    enabled BOOLEAN DEFAULT true,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### 任务清单

- [ ] **T5.1** 设计数据库 Schema
  - 定义所有表结构
  - 添加索引优化查询
  - 设计迁移机制

- [ ] **T5.2** 实现数据库访问层
  - 创建 `pkg/storage/database.go`
  - 实现 CRUD 操作
  - 添加事务支持

- [ ] **T5.3** 重构配置加载
  - 配置文件仅保留基本配置
  - 启动时从数据库加载动态配置
  - 支持配置热更新

- [ ] **T5.4** 实现敏感信息加密
  - API Key 加密存储
  - Token 加密存储
  - 密钥管理机制

- [ ] **T5.5** 迁移现有配置
  - 分析现有配置结构
  - 编写迁移脚本
  - 测试迁移流程

- [ ] **T5.6** 添加配置管理 API
  - Provider 配置 CRUD
  - 通道配置 CRUD
  - 绑定关系管理

#### 配置加载流程

```
启动
  │
  ├── 1. 加载 config.toml（基本配置）
  │       ├── workspace 路径
  │       ├── 数据库路径
  │       ├── 日志配置
  │       └── 网关配置
  │
  ├── 2. 初始化数据库连接
  │
  ├── 3. 从数据库加载动态配置
  │       ├── providers 表 → Provider 配置
  │       ├── channels 表 → 通道配置
  │       ├── bindings 表 → Agent 绑定
  │       ├── tools 表 → 工具配置
  │       └── skills 表 → 技能配置
  │
  └── 4. 初始化各模块
```

#### 优势

| 优势 | 说明 |
|-----|------|
| **安全性** | 敏感信息加密存储，不暴露在配置文件中 |
| **灵活性** | 运行时动态修改配置，无需重启 |
| **可管理性** | 通过 API 或 CLI 管理配置 |
| **版本控制友好** | 配置文件可提交 Git，不含敏感信息 |
| **多实例支持** | 多个实例共享同一数据库，配置一致 |

### 阶段六：工程化完善（预计 3 天）

**目标**：建立规范的工程化流程

#### 任务清单

- [ ] **T6.1** 添加 CI/CD 配置
  - 配置 golangci-lint
  - 配置 GitHub Actions
  - 配置 goreleaser

- [ ] **T6.2** 完善构建系统
  - 支持多平台构建
  - 版本信息注入
  - MIPS 特殊处理（如需要）

- [ ] **T6.3** 添加容器化支持
  - 编写 Dockerfile
  - 配置 docker-compose

### 阶段七：测试完善（持续）

**目标**：提高代码质量和重构信心

#### 任务清单

- [ ] **T7.1** 单元测试
  - Agent 核心逻辑测试
  - Provider 测试（使用 mock）
  - 工具系统测试

- [ ] **T7.2** 集成测试
  - 端到端消息处理测试
  - Fallback 链测试
  - 并发场景测试

- [ ] **T7.3** 性能测试
  - 消息总线压力测试
  - 并发处理能力测试

---

## 四、优先级矩阵

| 优先级 | 阶段 | 理由 |
|-------|------|-----|
| **P0** | 阶段一：架构重构 | 是后续工作的基础 |
| **P0** | 阶段二：Provider 增强 | 直接影响系统可靠性 |
| **P1** | 阶段三：工具系统增强 | 提升功能完整性 |
| **P1** | 阶段四：Channels 模块重构 | 生产级稳定性保障 |
| **P1** | 阶段五：配置管理与数据存储重构 | 安全性和灵活性保障 |
| **P2** | 阶段六：工程化完善 | 长期维护保障 |
| **P2** | 阶段七：测试完善 | 贯穿整个开发周期 |

---

## 五、风险评估

| 风险 | 可能性 | 影响 | 缓解措施 |
|-----|-------|-----|---------|
| 重构引入新 Bug | 高 | 高 | 编写充分测试，小步迭代 |
| 模块合并导致依赖冲突 | 中 | 中 | 使用 go mod tidy 解决 |
| Fallback 逻辑复杂 | 中 | 中 | 参考成熟实现，充分测试 |

> **注意**：不考虑向后兼容，配置格式可直接变更，无需迁移支持

---

## 六、里程碑

| 里程碑 | 完成标准 | 预计时间 |
|-------|---------|---------|
| **M1** | 架构重构完成，所有测试通过 | 第 2 周末 |
| **M2** | Provider 增强完成，Fallback 链可用 | 第 3 周末 |
| **M3** | 工具系统增强完成 | 第 4 周末 |
| **M4** | Channels 模块重构完成，支持多通道 | 第 6 周末 |
| **M5** | 配置管理与数据存储重构完成 | 第 7 周末 |
| **M6** | 工程化完善，发布 v1.0 | 第 8 周末 |

---

## 七、参考资源

- [picoclaw 项目](../picoclaw/) - 架构和设计参考
- [Go 项目布局标准](https://github.com/golang-standards/project-layout)
- [golangci-lint 配置](https://golangci-lint.run/usage/configuration/)
- [goreleaser 配置](https://goreleaser.com/customization/)