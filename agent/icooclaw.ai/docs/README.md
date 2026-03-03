# icooclaw.ai 文档

本目录包含 `icooclaw.ai` 模块的架构和设计文档。

## 文档列表

| 文档 | 说明 |
|------|------|
| [architecture.md](./architecture.md) | 模块架构总览，包含架构图、模块说明、接口设计等 |
| [module-dependencies.md](./module-dependencies.md) | 模块依赖关系详解，包含依赖图、依赖矩阵、外部依赖等 |
| [optimization-suggestions.md](./optimization-suggestions.md) | 优化建议文档，包含代码质量、架构设计、性能优化、安全性改进等 |

## 快速导航

### 架构概览

模块采用分层架构设计：

```
┌─────────────────────────────────────┐
│           应用层 (Application)        │
│              subagent                │
├─────────────────────────────────────┤
│            核心层 (Core)              │
│      agent    react    context       │
├─────────────────────────────────────┤
│           服务层 (Service)            │
│  memory  skill  tools  provider  mcp │
├─────────────────────────────────────┤
│          基础层 (Foundation)          │
│        storage    config    consts   │
├─────────────────────────────────────┤
│          外部模块 (External)          │
│         icooclaw.bus    icooclaw.utils│
└─────────────────────────────────────┘
```

### 核心模块职责

| 模块 | 职责 |
|------|------|
| `agent` | Agent 核心实现，生命周期管理 |
| `react` | ReAct 循环，LLM 交互逻辑 |
| `context` | 上下文构建，提示词组装 |
| `provider` | LLM 提供者抽象 |
| `tools` | 工具注册和执行 |
| `memory` | 长期记忆系统 |
| `storage` | 数据持久化 |

### 关键接口

- `Provider` - LLM 提供者接口
- `Tool` - 工具接口
- `ReActHooks` - Agent 生命周期钩子

详细接口定义请参考 [architecture.md](./architecture.md)。

## 更新记录

- 2026-03-02: 初始版本，包含架构文档和依赖关系图