# icooclaw 项目总结

> 文档生成日期：2026年3月10日

## 项目概述

**icooclaw** 是一个用 Go 语言编写的 AI Agent 框架，支持多渠道接入、多 LLM 提供商、工具调用、MCP 协议等特性。

## 项目统计

| 指标 | 数值 |
|------|------|
| Go 文件数 | 103 |
| 代码行数 | ~15,000+ |
| 模块数 | 15 |
| 提供商数 | 15+ |
| 渠道数 | 2 (飞书、钉钉) |

## 模块清单

### 核心模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Agent | `pkg/agent/` | Agent 实例和消息处理循环 |
| Bus | `pkg/bus/` | 消息总线，组件间通信 |
| Config | `pkg/config/` | 配置管理 |
| Errors | `pkg/errors/` | 统一错误定义 |
| Storage | `pkg/storage/` | SQLite 数据持久化 |

### 渠道模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Channels | `pkg/channels/` | 渠道接口和管理 |
| Feishu | `pkg/channels/feishu/` | 飞书/Lark 渠道 |
| DingTalk | `pkg/channels/dingtalk/` | 钉钉渠道 |

### 提供商模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Providers | `pkg/providers/` | LLM 提供商抽象 |
| OpenAI | `pkg/providers/openai.go` | OpenAI GPT |
| Anthropic | `pkg/providers/anthropic.go` | Claude |
| Gemini | `pkg/providers/gemini.go` | Google Gemini |
| DeepSeek | `pkg/providers/deepseek.go` | DeepSeek |
| OpenRouter | `pkg/providers/openrouter.go` | 多模型代理 |
| Groq | `pkg/providers/groq.go` | 高速推理 |
| Mistral | `pkg/providers/mistral.go` | Mistral AI |
| Ollama | `pkg/providers/ollama.go` | 本地模型 |
| Azure | `pkg/providers/azure_openai.go` | Azure OpenAI |
| Zhipu | `pkg/providers/zhipu.go` | 智谱 AI |
| Moonshot | `pkg/providers/moonshot.go` | Kimi |
| Qwen | `pkg/providers/qwen.go` | 通义千问 |
| SiliconFlow | `pkg/providers/silicon_flow.go` | 硅基流动 |
| Grok | `pkg/providers/grok.go` | xAI Grok |

### Gateway 模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Server | `pkg/gateway/server.go` | HTTP 服务器 |
| Handlers | `pkg/gateway/handlers/` | API 处理器 |
| WebSocket | `pkg/gateway/websocket/` | WebSocket 支持 |
| SSE | `pkg/gateway/sse/` | Server-Sent Events |
| Middleware | `pkg/gateway/middleware/` | 认证中间件 |

### 工具模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Tools | `pkg/tools/` | 工具注册和执行 |
| Builtin | `pkg/tools/builtin/` | 内置工具 |
| MCP | `pkg/mcp/` | MCP 协议支持 |
| Script | `pkg/script/` | JavaScript 引擎 |

### 其他模块

| 模块 | 路径 | 说明 |
|------|------|------|
| Memory | `pkg/memory/` | 记忆管理 |
| Scheduler | `pkg/scheduler/` | 任务调度 |
| Skill | `pkg/skill/` | 技能系统 |
| Hooks | `pkg/hooks/` | 钩子系统 |

## API 端点

### 核心 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/health` | GET | 健康检查 |
| `/api/v1/chat` | POST | HTTP 聊天 |
| `/api/v1/chat/stream` | POST | SSE 流式聊天 |
| `/ws` | GET | WebSocket 连接 |

### CRUD API

| 资源 | 端点 | 操作 |
|------|------|------|
| Sessions | `/api/v1/sessions/*` | page, create, save, delete, get |
| Messages | `/api/v1/messages/*` | page, create, update, delete, get |
| Providers | `/api/v1/providers/*` | page, create, update, delete, get, all, enabled |
| Channels | `/api/v1/channels/*` | page, create, update, delete, get, all, enabled |
| Tools | `/api/v1/tools/*` | page, create, update, delete, get, all, enabled |
| Skills | `/api/v1/skills/*` | page, create, update, delete, get, all, enabled |
| MCP | `/api/v1/mcp/*` | page, create, update, delete, get, all |
| Memory | `/api/v1/memories/*` | page, create, update, delete, get, search |
| Tasks | `/api/v1/tasks/*` | page, create, update, delete, get, toggle, all, enabled |
| Bindings | `/api/v1/bindings/*` | page, create, update, delete, get, all |

## 配置项

### Agent 配置

```toml
[agent]
workspace = "./workspace"
default_model = "gpt-4"
default_provider = "openai"
```

### Gateway 配置

```toml
[gateway]
enabled = true
port = 8080
```

### 渠道配置

```toml
[channels.feishu]
enabled = true
app_id = "cli_xxx"
app_secret = "xxx"

[channels.dingtalk]
enabled = true
client_id = "dingxxx"
client_secret = "xxx"
agent_id = 123456
```

## 依赖清单

### 核心依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/spf13/cobra | v1.9.1 | CLI 框架 |
| github.com/spf13/viper | v1.19.0 | 配置管理 |
| gorm.io/gorm | v1.25.12 | ORM |
| gorm.io/driver/sqlite | v1.5.7 | SQLite 驱动 |

### 渠道依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/larksuite/oapi-sdk-go/v3 | v3.5.3 | 飞书 SDK |
| github.com/open-dingtalk/dingtalk-stream-sdk-go | v0.9.1 | 钉钉 SDK |

### 工具依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/dop251/goja | v0.0.0-20260226184354 | JavaScript 引擎 |
| github.com/mark3labs/mcp-go | v0.44.1 | MCP 协议 |
| github.com/robfig/cron/v3 | v3.0.1 | Cron 调度 |

### Gateway 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| github.com/go-chi/chi/v5 | v5.x | HTTP 路由 |
| github.com/gorilla/websocket | v1.5.3 | WebSocket |

## 开发进度

### 已完成 ✅

- [x] Agent 核心实现
- [x] 消息总线
- [x] 工具系统
- [x] 存储层
- [x] 配置管理
- [x] 错误处理
- [x] 钩子系统
- [x] 任务调度
- [x] 脚本引擎
- [x] MCP 协议
- [x] 技能系统
- [x] 记忆管理
- [x] HTTP Gateway
- [x] WebSocket 支持
- [x] SSE 流式响应
- [x] 认证中间件
- [x] 飞书渠道
- [x] 钉钉渠道
- [x] 15+ LLM 提供商

### 待完成 📋

- [ ] Telegram 渠道
- [ ] Discord 渠道
- [ ] Slack 渠道
- [ ] WhatsApp 渠道
- [ ] QQ 渠道
- [ ] 单元测试覆盖
- [ ] 集成测试
- [ ] 性能优化
- [ ] 文档完善

## 文档索引

| 文档 | 路径 | 说明 |
|------|------|------|
| README | `docs/README.md` | 项目概述和快速开始 |
| 架构设计 | `docs/ARCHITECTURE.md` | 系统架构和模块设计 |
| API 文档 | `docs/API.md` | RESTful API 接口说明 |
| 渠道配置 | `docs/CHANNELS.md` | 飞书、钉钉等渠道配置 |
| 提供商配置 | `docs/PROVIDERS.md` | LLM 提供商配置说明 |
| 开发指南 | `docs/DEVELOPMENT.md` | 开发和贡献指南 |
| 分析报告 | `ANALYSIS_REPORT.md` | 与 PicoClaw 对比分析 |

## 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/your-org/icooclaw.git
cd icooclaw

# 2. 安装依赖
go mod download

# 3. 创建配置
cat > config.toml << EOF
[agent]
workspace = "./workspace"
default_model = "gpt-4"
default_provider = "openai"

[database]
path = "./data/icooclaw.db"

[gateway]
enabled = true
port = 8080
EOF

# 4. 运行
go run ./cmd/icooclaw start -c config.toml

# 5. 测试
curl http://localhost:8080/api/v1/health
```

## 联系方式

- **Issues**: https://github.com/your-org/icooclaw/issues
- **Pull Requests**: https://github.com/your-org/icooclaw/pulls

## 许可证

MIT License