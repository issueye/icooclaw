# icooclaw

AI Agent 框架 - 支持多种 LLM Provider 和消息通道

## 功能特性

- **多 LLM Provider 支持**
  - OpenAI (GPT-4o, GPT-4 Turbo, GPT-3.5 Turbo)
  - Anthropic (Claude-3-Opus, Claude-3-Sonnet, Claude-3-Haiku)
  - OpenRouter (统一网关，支持多种模型)
  - DeepSeek (DeepSeek Chat, DeepSeek Coder)
  - Custom (支持本地模型如 vLLM, LM Studio, Ollama)

- **多消息通道**
  - WebSocket 实时通信
  - CLI 交互模式 (REPL)

- **Agent 核心功能**
  - 工具调用 (Function Calling)
  - 记忆系统
  - 技能加载

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

# Custom (本地模型)
[providers.custom]
enabled = false
api_key = "no-key"
api_base = "http://localhost:8000/v1"
model = "qwen2.5"
```

### WebSocket 配置

```toml
[channels.websocket]
enabled = true
host = "0.0.0.0"
port = 8080
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
├── cmd/icooclaw/      # CLI 主程序
├── internal/
│   ├── agent/         # Agent 核心
│   ├── bus/           # 消息总线
│   ├── channel/       # 通道实现
│   ├── config/        # 配置管理
│   ├── provider/      # LLM Provider
│   ├── scheduler/      # 定时任务
│   └── storage/       # 数据存储
└── config.example.toml # 配置示例
```
