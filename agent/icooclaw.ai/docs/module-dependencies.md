# 模块依赖关系图

## 完整依赖图

```mermaid
graph LR
    %% 定义节点样式
    classDef core fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef infra fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef integration fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef extension fill:#fff3e0,stroke:#f57c00,stroke-width:2px
    classDef external fill:#fafafa,stroke:#616161,stroke-width:1px,stroke-dasharray: 5 5

    %% 核心模块
    agent:::core
    react:::core
    context:::core

    %% 基础设施模块
    config:::infra
    consts:::infra
    storage:::infra

    %% 集成模块
    provider:::integration
    tools:::integration
    mcp:::integration
    hooks:::integration

    %% 扩展模块
    memory:::extension
    skill:::extension
    subagent:::extension

    %% 外部模块
    bus[icooclaw.bus]:::external
    utils[icooclaw.utils]:::external

    %% 依赖关系 - agent
    agent --> react
    agent --> context
    agent --> hooks
    agent --> memory
    agent --> skill
    agent --> tools
    agent --> provider
    agent --> storage
    agent --> bus
    agent --> consts

    %% 依赖关系 - react
    react --> provider
    react --> tools
    react --> hooks
    react --> consts
    react --> storage

    %% 依赖关系 - context
    context --> storage
    context --> consts
    context --> memory
    context --> skill

    %% 依赖关系 - tools
    tools --> config
    tools --> provider

    %% 依赖关系 - hooks
    hooks --> provider
    hooks --> tools

    %% 依赖关系 - mcp
    mcp --> config
    mcp --> tools

    %% 依赖关系 - memory
    memory --> storage

    %% 依赖关系 - skill
    skill --> storage

    %% 依赖关系 - subagent
    subagent --> agent

    %% 依赖关系 - storage
    storage --> config

    %% 依赖关系 - config
    config --> utils
```

## 分层依赖图

```mermaid
graph TB
    subgraph Layer4["应用层"]
        subagent
    end

    subgraph Layer3["核心层"]
        agent
        react
        context
    end

    subgraph Layer2["服务层"]
        memory
        skill
        tools
        provider
        mcp
        hooks
    end

    subgraph Layer1["基础层"]
        storage
        config
        consts
    end

    subgraph Layer0["外部模块"]
        bus[icooclaw.bus]
        utils[icooclaw.utils]
    end

    %% 层间依赖（自上而下）
    subagent --> agent

    agent --> memory
    agent --> skill
    agent --> tools
    agent --> provider
    agent --> hooks
    agent --> bus

    react --> tools
    react --> provider
    react --> hooks

    context --> memory
    context --> skill

    memory --> storage
    skill --> storage
    tools --> config
    provider --> config
    mcp --> config
    mcp --> tools

    storage --> config
    config --> utils
```

## 模块依赖矩阵

| 模块 | agent | react | context | config | consts | storage | provider | tools | mcp | hooks | memory | skill | subagent |
|------|:-----:|:-----:|:-------:|:------:|:------:|:-------:|:--------:|:-----:|:---:|:-----:|:------:|:-----:|:--------:|
| agent | - | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ | ❌ |
| react | ❌ | - | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ❌ | ❌ |
| context | ❌ | ❌ | - | ❌ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ |
| config | ❌ | ❌ | ❌ | - | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| consts | ❌ | ❌ | ❌ | ❌ | - | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| storage | ❌ | ❌ | ❌ | ✅ | ❌ | - | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| provider | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | - | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| tools | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | - | ❌ | ❌ | ❌ | ❌ | ❌ |
| mcp | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ | - | ❌ | ❌ | ❌ | ❌ |
| hooks | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ❌ | - | ❌ | ❌ | ❌ |
| memory | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | - | ❌ | ❌ |
| skill | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | - | ❌ |
| subagent | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | - |

## 循环依赖检查

当前模块依赖关系**无循环依赖**。依赖关系为有向无环图 (DAG)。

```
依赖深度:
- Level 0: consts, bus, utils (外部依赖)
- Level 1: config
- Level 2: storage, provider
- Level 3: tools, hooks, memory, skill, mcp
- Level 4: react, context
- Level 5: agent
- Level 6: subagent
```

## 外部依赖

### Go 标准库
- `context` - 上下文管理
- `encoding/json` - JSON 序列化
- `fmt` - 格式化输出
- `log/slog` - 结构化日志
- `net/http` - HTTP 客户端
- `os` - 文件系统操作
- `path/filepath` - 路径处理
- `sync` - 并发原语
- `time` - 时间处理

### 第三方库
| 库 | 版本 | 用途 | 使用模块 |
|---|------|------|---------|
| `gorm.io/gorm` | v1.31.1 | ORM | storage |
| `github.com/glebarez/sqlite` | v1.21.2 | SQLite 驱动 | storage |
| `github.com/spf13/viper` | v1.21.0 | 配置管理 | config |
| `github.com/dop251/goja` | v0.0.0-20260226184354-913bd86fb70c | JS 运行时 | tools |
| `github.com/mark3labs/mcp-go` | v0.44.1 | MCP SDK | mcp |
| `github.com/stretchr/testify` | v1.11.1 | 测试框架 | 全局 |

### 内部模块依赖
| 模块 | 依赖 |
|------|------|
| icooclaw.ai | icooclaw.bus |
| icooclaw.ai | icooclaw.utils |