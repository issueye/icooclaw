# icooclaw

AI Agent 框架 - 支持多种 LLM Provider 和消息通道

## 功能特性

### 多 LLM Provider 支持
- **OpenAI** - GPT-4o, GPT-4 Turbo, GPT-3.5 Turbo
- **Anthropic** - Claude-3-Opus, Claude-3-Sonnet, Claude-3-Haiku
- **OpenRouter** - 统一网关，支持多种模型
- **DeepSeek** - DeepSeek Chat, DeepSeek Coder
- **Ollama** - 本地 LLM (Llama2, Codellama, Mistral 等)
- **Azure OpenAI** - 微软 Azure 云服务
- **LocalAI** - 本地 OpenAI 替代方案
- **OneAPI** - API 聚合服务
- **Custom** - 自定义端点
- **OpenAI Compatible** - 任何兼容 OpenAI API 的服务

### 多消息通道
- WebSocket 实时通信
- Webhook
- CLI 交互模式 (REPL)

### Agent 核心功能
- 工具调用 (Function Calling)
- 记忆系统 (SQLite 持久化)
- 技能加载
- MCP 支持
- 定时任务调度

### 丰富的内置工具
- **文件操作**: file_read, file_write, file_list, file_delete, file_edit
- **文件搜索**: grep (正则), find, tree, wc, read_part
- **网络请求**: http_request, web_search, web_fetch
- **计算**: calculator
- **系统交互**: exec (Shell 执行)
- **消息**: message (多通道发送)

## 快速开始

### 1. 配置

复制 `config.example.toml` 为 `config.toml` 并配置你的 API Key：

```toml
[providers.openrouter]
enabled = true
api_key = "sk-or-v1-xxxx"
model = "anthropic/claude-sonnet-4-20250514"
```

### 2. 运行

```bash
# 运行 CLI 交互模式
./icooclaw.exe

# 或启动 WebSocket 服务
./icooclaw.exe serve
```

### 3. 使用 CLI

```
> Hello!  # 直接输入消息
< AI 响应

> help    # 查看帮助
> model gpt-4o  # 切换模型
> providers  # 查看可用 provider
> quit     # 退出
```

## 配置说明

### Provider 配置

```toml
# OpenRouter (推荐)
[providers.openrouter]
enabled = true
api_key = "sk-or-v1-xxxx"
model = "anthropic/claude-sonnet-4-20250514"

# OpenAI
[providers.openai]
enabled = false
api_key = "sk-xxxx"
model = "gpt-4o"

# Anthropic
[providers.anthropic]
enabled = false
api_key = "sk-ant-xxxx"
model = "claude-sonnet-4-20250514"

# DeepSeek
[providers.deepseek]
enabled = false
api_key = "sk-xxxx"
model = "deepseek-chat"

# Ollama (本地 LLM)
[providers.ollama]
enabled = false
api_base = "http://localhost:11434"
model = "llama2"

# Azure OpenAI
[providers.azure_openai]
enabled = false
api_key = "your-azure-key"
endpoint = "https://your-resource.openai.azure.com"
deployment = "gpt-4"
api_version = "2024-02-15-preview"

# LocalAI (本地)
[providers.localai]
enabled = false
api_base = "http://localhost:8080"
model = "gpt-3.5-turbo"

# OneAPI
[providers.oneapi]
enabled = false
api_key = "sk-xxxx"
api_base = "https://api.oneapi.icu/v1"
model = "gpt-3.5-turbo"

# 自定义 OpenAI 兼容端点
[providers.custom]
enabled = false
api_key = "no-key"
api_base = "http://localhost:8000/v1"
model = "qwen2.5"

# 多个 OpenAI 兼容 LLM (支持额外请求头)
[[providers.compatible]]
enabled = false
name = "siliconflow"
api_key = "your-api-key"
api_base = "https://api.siliconflow.cn/v1"
model = "Qwen/Qwen2-7B-Instruct"

[[providers.compatible]]
enabled = false
name = "local-llm"
api_key = "no-key"
api_base = "http://localhost:8080/v1"
model = "llama2-7b"
```

### 工具配置

```toml
[tools]
enabled = true
allow_file_read = true
allow_file_write = true
allow_file_edit = true
allow_file_delete = true
allow_exec = true
exec_timeout = 30
workspace = "./workspace"
```

### WebSocket 配置

```toml
[channels.websocket]
enabled = true
host = "0.0.0.0"
port = 8080
```

### 记忆系统配置

```toml
[agents.default]
memory_enabled = true
consolidation_threshold = 50
max_memory_age = 30
max_session_memories = 100
max_user_memories = 500
```

## 工具使用

### web_search
```json
{
  "query": "Go语言教程",
  "num_results": 5,
  "engine": "duckduckgo"  // duckduckgo, brave, google
}
```

### web_fetch
```json
{
  "url": "https://example.com",
  "extract_text": true,
  "max_length": 10000,
  "selector": ".article-content",
  "query": "data.items.0.title"
}
```

### grep
```json
{
  "pattern": "func\\s+\\w+\\(",
  "path": "./src",
  "glob": "*.go",
  "ignore_case": false,
  "recursive": true,
  "max_results": 100
}
```

### find
```json
{
  "path": "./src",
  "name": "*.go",
  "type": "f",
  "max_depth": 5
}
```

## 开发

```bash
# 构建
go build -o icooclaw.exe ./cmd/icooclaw

# 测试
go test ./...

# 代码检查
go vet ./...
```

## 项目结构

```
icooclaw/
├── cmd/icooclaw/           # CLI 主程序入口
│   ├── main.go
│   └── commands/           # CLI 命令
│
├── internal/
│   ├── agent/             # Agent 核心
│   │   ├── agent.go      # Agent 主结构
│   │   ├── loop.go       # ReAct 对话循环
│   │   ├── context.go    # 上下文构建
│   │   ├── memory.go     # 记忆系统
│   │   ├── skills.go     # 技能加载器
│   │   └── tools/        # 工具集
│   │       ├── base.go       # 工具基类
│   │       ├── registry.go   # 工具注册表
│   │       ├── http.go      # HTTP 请求工具
│   │       ├── search.go    # 搜索工具
│   │       ├── grep.go     # grep/find/tree 工具
│   │       ├── calculator.go # 计算器工具
│   │       ├── shell.go    # Shell 执行工具
│   │       ├── file.go    # 文件操作工具
│   │       └── message.go # 消息工具
│   │
│   ├── bus/              # 消息总线
│   ├── channel/           # 消息通道
│   ├── config/            # 配置管理
│   ├── provider/          # LLM Provider
│   ├── scheduler/         # 定时任务调度
│   ├── storage/          # 数据存储 (SQLite)
│   └── mcp/              # MCP 协议支持
│
├── templates/             # 模板文件
└── config.example.toml   # 配置示例
```

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    CLI / WebSocket Server                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Component Initialization                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │  Config  │ │   DB     │ │ Message  │ │ Provider │     │
│  │          │ │ (SQLite) │ │   Bus    │ │Registry  │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                    │
│  │ Channel  │ │  Agent   │ │ Scheduler│                    │
│  │ Manager  │ │ Instance │ │          │                    │
│  └──────────┘ └──────────┘ └──────────┘                    │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  WebSocket      │  │  Webhook     │  │  Scheduler     │
│  Channel        │  │  Channel     │  │  (Cron Jobs)   │
└────────┬────────┘  └────────┬────────┘  └────────┬────────┘
         └────────────────────┼────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Message Bus                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Agent Loop                           │
│                  (ReAct Pattern Implementation)             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Build       │  │    Call      │  │   Execute    │   │
│  │  Context     │──▶│    LLM      │──▶│   Tools     │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Built-in Tools │  │    Skills     │  │   MCP Servers  │
│  - HTTP        │  │  (Custom LLM  │  │  (External     │
│  - Search      │  │   Prompts)    │  │   Capabilities) │
│  - Calculator   │  │                │  │                 │
│  - File Ops    │  │                │  │                 │
│  - Shell       │  │                │  │                 │
└─────────────────┘  └─────────────────┘  └─────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Storage Layer                          │
│        (SQLite: Sessions, Messages, Tasks, Memories)       │
└─────────────────────────────────────────────────────────────┘
```

## License

MIT
