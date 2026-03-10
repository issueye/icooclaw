# icooclaw - AI Agent Framework

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)

**一个轻量级、可扩展的 AI Agent 框架，支持多渠道接入和多 LLM 提供商**

[快速开始](#快速开始) · [架构设计](./ARCHITECTURE.md) · [API 文档](./API.md) · [渠道配置](./CHANNELS.md)

</div>

---

## ✨ 特性

- 🤖 **多 Agent 支持** - 支持创建和管理多个 Agent 实例
- 🔌 **多渠道接入** - 支持飞书、钉钉、WebSocket、HTTP 等
- 🧠 **多 LLM 提供商** - 支持 OpenAI、Anthropic、Gemini、DeepSeek 等 15+ 提供商
- 🛠️ **工具系统** - 内置 HTTP 请求、Web 搜索、文件操作等工具
- 📦 **MCP 协议** - 支持 Model Context Protocol，可扩展工具生态
- 🔄 **Fallback Chain** - 自动故障转移，提高服务可用性
- 💾 **持久化存储** - SQLite 存储，支持会话、记忆、配置等
- 🌐 **HTTP Gateway** - RESTful API + WebSocket + SSE 流式响应
- 📜 **脚本引擎** - 内置 JavaScript 运行时，支持自定义脚本
- ⏰ **任务调度** - Cron 表达式支持，定时执行任务

## 📦 安装

### 从源码构建

```bash
git clone https://github.com/your-org/icooclaw.git
cd icooclaw
go build -o icooclaw ./cmd/icooclaw
```

### 使用 Go Install

```bash
go install github.com/your-org/icooclaw/cmd/icooclaw@latest
```

## 🚀 快速开始

### 1. 创建配置文件

创建 `config.toml` 文件：

```toml
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

[channels.feishu]
enabled = false
app_id = ""
app_secret = ""

[channels.dingtalk]
enabled = false
client_id = ""
client_secret = ""
agent_id = 0
```

### 2. 启动服务

```bash
./icooclaw start -c config.toml
```

### 3. 使用 HTTP API

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 发送聊天消息
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, how are you?", "session_id": "test"}'
```

### 4. 使用 WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'chat',
    message: 'Hello!',
    session_id: 'test'
  }));
};
ws.onmessage = (event) => {
  console.log('Received:', JSON.parse(event.data));
};
```

## 📁 项目结构

```
icooclaw/
├── cmd/
│   └── icooclaw/          # 入口程序
│       └── main.go
├── pkg/
│   ├── agent/             # Agent 核心实现
│   │   ├── agent.go       # Agent 实例和注册表
│   │   └── loop.go        # 消息处理循环
│   ├── bus/               # 消息总线
│   ├── channels/          # 渠道实现
│   │   ├── feishu/        # 飞书渠道
│   │   ├── dingtalk/      # 钉钉渠道
│   │   └── ...
│   ├── config/            # 配置管理
│   ├── errors/            # 错误定义
│   ├── gateway/           # HTTP Gateway
│   │   ├── handlers/      # API 处理器
│   │   ├── middleware/    # 中间件
│   │   ├── sse/           # Server-Sent Events
│   │   └── websocket/     # WebSocket 支持
│   ├── hooks/             # 钩子系统
│   ├── mcp/               # MCP 协议支持
│   ├── memory/            # 记忆管理
│   ├── providers/         # LLM 提供商
│   ├── scheduler/         # 任务调度
│   ├── script/            # JavaScript 引擎
│   ├── skill/             # 技能系统
│   ├── storage/           # 数据存储
│   └── tools/             # 工具系统
│       └── builtin/       # 内置工具
├── docs/                  # 文档
├── config.toml           # 配置文件
└── go.mod
```

## 🔧 配置说明

### Agent 配置

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `workspace` | string | 工作目录 | `./workspace` |
| `default_model` | string | 默认模型 | `gpt-4` |
| `default_provider` | string | 默认提供商 | `openai` |

### Gateway 配置

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `enabled` | bool | 是否启用 | `true` |
| `port` | int | 监听端口 | `8080` |

### Logging 配置

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `level` | string | 日志级别 | `info` |
| `format` | string | 日志格式 | `json` |

## 📚 文档

- [架构设计](./ARCHITECTURE.md) - 系统架构和模块设计
- [API 文档](./API.md) - RESTful API 接口说明
- [渠道配置](./CHANNELS.md) - 飞书、钉钉等渠道配置指南
- [提供商配置](./PROVIDERS.md) - LLM 提供商配置说明
- [开发指南](./DEVELOPMENT.md) - 开发和贡献指南

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License