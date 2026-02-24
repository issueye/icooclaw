# Golang三方库文档

## 4.1 核心框架

| 库 | 用途 | 版本 |
|----|------|------|
| **golang.org/x/net** | net/http扩展（WebSocket等） | v0.20.x |
| **github.com/julienschmidt/httprouter** | 高性能HTTP路由（备选） | v1.3.x |

> **说明：** 使用 Golang 原生 `net/http` 标准库作为Web框架，无需额外依赖

## 4.2 配置与日志

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/spf13/viper** | 配置管理（TOML/JSON/YAML/ENV） | v1.18.x |
| **log/slog** | 结构化日志（Go 1.21+内置） | 内置 |

> **说明：** 日志使用 Go 1.21+ 内置的 `log/slog`，无需额外依赖

## 4.3 数据库

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/glebarez/go-sqlite** | SQLite GORM驱动（纯Go） | v1.10.x |
| **gorm.io/gorm** | GORM ORM框架 | v1.25.x |

> **说明：** 使用 `github.com/glebarez/go-sqlite` 而非 `mattn/go-sqlite3`（CGO），实现纯Go编译

## 4.4 异步与并发

| 库 | 用途 | 版本 |
|----|------|------|
| **go-redis/redis** | Redis客户端（分布式缓存/消息） | v9.x |
| **nats-io/nats.go** | NATS消息队列 | v1.x |

> **说明：** 使用 Go 原生 `sync`, `context`, `goroutine` 实现并发，无需额外依赖

## 4.5 网络与HTTP

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/go-resty/resty/v2** | HTTP客户端 | v2.x |
| **github.com/gorilla/websocket** | WebSocket支持 | v1.5.x |

## 4.6 LLM相关

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/sashabaranov/go-openai** | OpenAI SDK | v1.x |
| **github.com/anthropics/anthropic-sdk-go** | Anthropic SDK | v0.3.x |
| **自实现** | LiteLLM封装 | - |

> **说明：** 对于 LiteLLM 的多Provider支持，建议自实现简单的路由逻辑

## 4.7 数据与序列化

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/json-iterator/go** | 高性能JSON | v1.1.x |
| **github.com/go-playground/validator/v10** | 数据校验 | v10.x |
| **github.com/BurntSushi/toml** | TOML解析 | v1.x |

> **说明：** Viper 内置支持 TOML，但如需独立解析可使用 `BurntSushi/toml`

## 4.8 消息通道（WebSocket/Webhook）

| 库 | 用途 | 版本 |
|----|------|------|
| **golang.org/x/net/websocket** | WebSocket支持（标准库） | 内置 |
| **github.com/gorilla/websocket** | WebSocket支持（增强） | v1.5.x |

> **说明：** 当前版本仅支持 WebSocket 和 Webhook，其他聊天平台SDK后续添加

## 4.9 消息平台SDK（后续支持）

## 4.9 定时任务

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/adhocore/gronx** | Cron表达式解析（轻量、多时区、纯Go） | v1.x |

> **gronx 特点：**
> - 轻量级：无外部依赖，纯Go实现
> - 多时区支持：内置时区处理
> - 高性能：解析和执行效率高
> - 简单API：易于集成和使用

## 4.10 MCP支持

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/mark3labs/mcp-go** | MCP客户端/服务器实现 | v1.x |
| **github.com/sourcegraph/jsonrpc2** | JSON-RPC 2.0 | v0.13.x |

> **mcp-go 特点：**
> - 完整的MCP协议实现（客户端+服务器）
> - 支持Stdio和SSE传输方式
> - 易于集成和扩展

## 4.11 工具与工具函数

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/google/shlex** | Shell词法分析 | - |
| **github.com/mattn/go-shellwords** | Shell参数解析 | v1.x |
| **github.com/charmbracelet/lipgloss** | 终端样式 | v0.8.x |
| **github.com/pterm/pterm** | 终端UI/颜色 | v0.12.x |

## 4.12 测试

| 库 | 用途 | 版本 |
|----|------|------|
| **github.com/stretchr/testify** | 测试断言 | v1.8.x |
| **github.com/stretchr/testify/mock** | Mock框架 | v1.x |

## 4.13 推荐依赖组合

### 最小依赖集（推荐）
```go
// 核心功能最小依赖
require (
    github.com/spf13/viper v1.18.2
    gorm.io/gorm v1.25.7
    github.com/glebarez/go-sqlite v1.10.1
    github.com/sashabaranov/go-openai v1.19.1
    github.com/adhocore/gronx v1.1.0
    github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
    github.com/json-iterator/go v1.1.12
    github.com/stretchr/testify v1.8.4
    github.com/gorilla/websocket v1.5.1
    github.com/go-resty/resty/v2 v2.11.0
)
```

### 完整依赖示例（go.mod）
```go
module github.com/icooclaw/icooclaw

go 1.21

require (
    // 配置与日志
    github.com/spf13/viper v1.18.2
    
    // 数据库
    gorm.io/gorm v1.25.7
    github.com/glebarez/go-sqlite v1.10.1
    
    // LLM
    github.com/sashabaranov/go-openai v1.19.1
    github.com/anthropics/anthropic-sdk-go v0.3.0
    
    // 网络
    github.com/gorilla/websocket v1.5.1
    github.com/go-resty/resty/v2 v2.11.0
    
    // 定时任务
    github.com/adhocore/gronx v1.1.0
    
    // 工具
    github.com/google/shlex v0.0.0-20181106134648-c34317bd91bf
    github.com/json-iterator/go v1.1.12
    
    // 测试
    github.com/stretchr/testify v1.8.4
)
```

### 无外部依赖版本（纯标准库）
如果追求极致轻量，可以使用纯标准库实现核心功能：

```go
import (
    "context"
    "encoding/json"
    "encoding/base64"
    "fmt"
    "io"
    "log"
    "log/slog"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "sync"
    "time"
    
    // 需要单独引入的库
    "github.com/glebarez/go-sqlite"  // SQLite驱动（纯Go）
    "github.com/spf13/viper"         // 配置管理
)
```
