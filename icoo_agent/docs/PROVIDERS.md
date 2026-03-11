# 提供商配置

本文档描述如何配置各种 LLM 提供商。

## 目录

- [OpenAI](#openai)
- [Anthropic (Claude)](#anthropic-claude)
- [Google Gemini](#google-gemini)
- [DeepSeek](#deepseek)
- [OpenRouter](#openrouter)
- [Groq](#groq)
- [Mistral](#mistral)
- [Ollama](#ollama)
- [Azure OpenAI](#azure-openai)
- [其他提供商](#其他提供商)

---

## OpenAI

### 功能特性

- ✅ Chat Completions API
- ✅ 流式输出
- ✅ 工具调用 (Function Calling)
- ✅ Vision 支持

### 配置示例

```toml
[providers.openai]
api_key = "sk-xxxxxxxxxxxxxxxx"
api_base = "https://api.openai.com/v1"  # 可选
default_model = "gpt-4o"
models = ["gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"]
```

### 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | string | 是 | API 密钥 |
| api_base | string | 否 | API 基础 URL |
| default_model | string | 否 | 默认模型 |
| models | []string | 否 | 可用模型列表 |

### 支持的模型

- `gpt-4o` - 最新多模态模型
- `gpt-4-turbo` - GPT-4 Turbo
- `gpt-4` - GPT-4
- `gpt-3.5-turbo` - GPT-3.5 Turbo

---

## Anthropic (Claude)

### 功能特性

- ✅ Messages API
- ✅ 流式输出
- ✅ 工具调用
- ✅ Vision 支持
- ✅ 长上下文 (200K)

### 配置示例

```toml
[providers.anthropic]
api_key = "sk-ant-xxxxxxxxxxxxxxxx"
api_base = "https://api.anthropic.com/v1"  # 可选
default_model = "claude-3-opus-20240229"
```

### 支持的模型

- `claude-3-opus-20240229` - Claude 3 Opus
- `claude-3-sonnet-20240229` - Claude 3 Sonnet
- `claude-3-haiku-20240307` - Claude 3 Haiku
- `claude-3-5-sonnet-20241022` - Claude 3.5 Sonnet

---

## Google Gemini

### 功能特性

- ✅ Gemini API
- ✅ 流式输出
- ✅ 多模态支持
- ✅ 长上下文

### 配置示例

```toml
[providers.gemini]
api_key = "AIzaSyxxxxxxxxxxxxxxxx"
api_base = "https://generativelanguage.googleapis.com/v1beta"  # 可选
default_model = "gemini-pro"
```

### 支持的模型

- `gemini-pro` - Gemini Pro
- `gemini-pro-vision` - Gemini Pro Vision
- `gemini-1.5-pro` - Gemini 1.5 Pro
- `gemini-1.5-flash` - Gemini 1.5 Flash

---

## DeepSeek

### 功能特性

- ✅ Chat API
- ✅ 流式输出
- ✅ 推理模型 (DeepSeek Reasoner)
- ✅ 低成本

### 配置示例

```toml
[providers.deepseek]
api_key = "sk-xxxxxxxxxxxxxxxx"
api_base = "https://api.deepseek.com/v1"  # 可选
default_model = "deepseek-chat"
```

### 支持的模型

- `deepseek-chat` - DeepSeek Chat
- `deepseek-reasoner` - DeepSeek Reasoner (推理模型)

---

## OpenRouter

### 功能特性

- ✅ 多模型代理
- ✅ 统一 API
- ✅ 流式输出
- ✅ 工具调用

### 配置示例

```toml
[providers.openrouter]
api_key = "sk-or-xxxxxxxxxxxxxxxx"
api_base = "https://openrouter.ai/api/v1"  # 可选
default_model = "openai/gpt-4o"
```

### 模型格式

OpenRouter 使用 `provider/model` 格式：

- `openai/gpt-4o`
- `anthropic/claude-3-opus`
- `google/gemini-pro`
- `meta-llama/llama-3-70b-instruct`
- `deepseek/deepseek-chat`

---

## Groq

### 功能特性

- ✅ 超高速推理
- ✅ 流式输出
- ✅ 工具调用

### 配置示例

```toml
[providers.groq]
api_key = "gsk_xxxxxxxxxxxxxxxx"
api_base = "https://api.groq.com/openai/v1"  # 可选
default_model = "llama-3.1-70b-versatile"
```

### 支持的模型

- `llama-3.1-70b-versatile`
- `llama-3.1-8b-instant`
- `mixtral-8x7b-32768`
- `gemma2-9b-it`

---

## Mistral

### 功能特性

- ✅ Mistral API
- ✅ 流式输出
- ✅ 工具调用

### 配置示例

```toml
[providers.mistral]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://api.mistral.ai/v1"  # 可选
default_model = "mistral-large-latest"
```

### 支持的模型

- `mistral-large-latest`
- `mistral-medium-latest`
- `mistral-small-latest`
- `open-mixtral-8x22b`
- `open-mistral-7b`

---

## Ollama

### 功能特性

- ✅ 本地模型
- ✅ 流式输出
- ✅ 无需 API Key
- ✅ 自定义模型

### 配置示例

```toml
[providers.ollama]
api_base = "http://localhost:11434/v1"
default_model = "llama3"
```

### 使用说明

1. 安装 Ollama: https://ollama.ai
2. 拉取模型: `ollama pull llama3`
3. 启动服务: `ollama serve`

### 常用模型

- `llama3` - Llama 3
- `llama3:70b` - Llama 3 70B
- `mistral` - Mistral
- `codellama` - Code Llama
- `qwen2` - Qwen 2

---

## Azure OpenAI

### 功能特性

- ✅ 企业级部署
- ✅ 流式输出
- ✅ 工具调用
- ✅ 私有端点

### 配置示例

```toml
[providers.azure]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://your-resource.openai.azure.com/"
api_version = "2024-02-15-preview"
default_model = "gpt-4"
deployment_name = "gpt-4-deployment"
```

### 配置说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| api_key | string | 是 | API 密钥 |
| api_base | string | 是 | Azure 资源 URL |
| api_version | string | 否 | API 版本 |
| default_model | string | 是 | 模型名称 |
| deployment_name | string | 是 | 部署名称 |

---

## 其他提供商

### 智谱 AI (Zhipu)

```toml
[providers.zhipu]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://open.bigmodel.cn/api/paas/v4"
default_model = "glm-4"
```

### Moonshot (Kimi)

```toml
[providers.moonshot]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://api.moonshot.cn/v1"
default_model = "moonshot-v1-8k"
```

### 通义千问 (Qwen)

```toml
[providers.qwen]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://dashscope.aliyuncs.com/compatible-mode/v1"
default_model = "qwen-turbo"
```

### 硅基流动 (SiliconFlow)

```toml
[providers.siliconflow]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://api.siliconflow.cn/v1"
default_model = "Qwen/Qwen2.5-7B-Instruct"
```

### Grok (xAI)

```toml
[providers.grok]
api_key = "xxxxxxxxxxxxxxxx"
api_base = "https://api.x.ai/v1"
default_model = "grok-beta"
```

---

## Fallback Chain

配置自动故障转移：

```toml
[agent]
default_provider = "openai"
fallback_providers = ["anthropic", "deepseek", "openrouter"]
```

当主提供商失败时，会自动切换到备用提供商。

### 故障转移策略

| 错误类型 | 处理方式 |
|----------|----------|
| 认证失败 | 冷却 5 分钟后重试 |
| 速率限制 | 冷却 60 秒后重试 |
| 超时 | 冷却 30 秒后重试 |
| 服务器错误 | 冷却 10 秒后重试 |

---

## 模型选择建议

### 通用对话

| 场景 | 推荐模型 |
|------|----------|
| 高质量 | GPT-4o, Claude 3 Opus |
| 平衡 | GPT-4 Turbo, Claude 3.5 Sonnet |
| 经济 | GPT-3.5 Turbo, Claude 3 Haiku |

### 代码生成

| 场景 | 推荐模型 |
|------|----------|
| 高质量 | GPT-4o, Claude 3.5 Sonnet |
| 快速 | DeepSeek Coder, Code Llama |

### 推理任务

| 场景 | 推荐模型 |
|------|----------|
| 复杂推理 | DeepSeek Reasoner, o1 |
| 快速推理 | Groq Llama 3 |

### 长文本

| 场景 | 推荐模型 |
|------|----------|
| 超长上下文 | Claude 3 (200K), Gemini 1.5 Pro (1M) |

---

## 常见问题

### API Key 无效

1. 检查 Key 是否正确复制
2. 检查 Key 是否过期
3. 检查账户余额

### 速率限制

1. 添加多个提供商作为备用
2. 配置 Fallback Chain
3. 联系提供商提升配额

### 响应超时

1. 检查网络连接
2. 增加超时时间
3. 使用更快的模型