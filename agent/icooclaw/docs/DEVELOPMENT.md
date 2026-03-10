# 开发指南

本文档描述如何参与 icooclaw 的开发和贡献。

## 环境准备

### 系统要求

- Go 1.23+
- Git
- Make (可选)

### 安装 Go

```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang-go

# Windows
# 从 https://go.dev/dl/ 下载安装
```

### 克隆项目

```bash
git clone https://github.com/your-org/icooclaw.git
cd icooclaw
```

### 安装依赖

```bash
go mod download
```

## 项目结构

```
icooclaw/
├── cmd/
│   └── icooclaw/          # 入口程序
├── pkg/
│   ├── agent/             # Agent 核心
│   ├── bus/               # 消息总线
│   ├── channels/          # 渠道实现
│   ├── config/            # 配置管理
│   ├── errors/            # 错误定义
│   ├── gateway/           # HTTP Gateway
│   ├── hooks/             # 钩子系统
│   ├── mcp/               # MCP 协议
│   ├── memory/            # 记忆管理
│   ├── providers/         # LLM 提供商
│   ├── scheduler/         # 任务调度
│   ├── script/            # 脚本引擎
│   ├── skill/             # 技能系统
│   ├── storage/           # 数据存储
│   └── tools/             # 工具系统
├── docs/                  # 文档
├── go.mod
└── Makefile
```

## 开发流程

### 1. 创建分支

```bash
git checkout -b feature/your-feature
```

### 2. 编写代码

遵循 Go 代码规范：

- 使用 `gofmt` 格式化代码
- 添加必要的注释
- 编写单元测试

### 3. 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/agent/...

# 运行带覆盖率的测试
go test -cover ./...
```

### 4. 构建项目

```bash
# 构建
go build -o icooclaw ./cmd/icooclaw

# 或使用 Makefile
make build
```

### 5. 提交代码

```bash
git add .
git commit -m "feat: add your feature"
git push origin feature/your-feature
```

## 代码规范

### 命名约定

```go
// 包名：小写单词
package agent

// 导出类型：大写开头
type Agent struct {}

// 私有类型：小写开头
type agentConfig struct {}

// 接口：动词或 -er 后缀
type Provider interface {}
type Executor interface {}

// 常量：大写或驼峰
const (
    DefaultTimeout = 30 * time.Second
    maxRetries     = 3
)
```

### 错误处理

```go
// 使用 errors.New 创建简单错误
var ErrNotFound = errors.New("not found")

// 使用 fmt.Errorf 创建格式化错误
return nil, fmt.Errorf("failed to connect: %w", err)

// 使用自定义错误类型
type ValidationError struct {
    Field string
    Msg   string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Msg)
}
```

### 日志记录

```go
import "log/slog"

// 使用结构化日志
logger.Info("message processed",
    "channel", msg.Channel,
    "chat_id", msg.ChatID,
    "duration", time.Since(start),
)

// 日志级别
logger.Debug("debug info")
logger.Info("normal info")
logger.Warn("warning")
logger.Error("error occurred", "error", err)
```

### 并发安全

```go
type SafeMap struct {
    mu sync.RWMutex
    m  map[string]string
}

func (s *SafeMap) Get(key string) (string, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    v, ok := s.m[key]
    return v, ok
}

func (s *SafeMap) Set(key, value string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.m[key] = value
}
```

## 添加新功能

### 添加新渠道

1. 创建目录 `pkg/channels/mychannel/`
2. 实现 `Channel` 接口
3. 注册到 `Registry`

```go
// pkg/channels/mychannel/mychannel.go
package mychannel

import (
    "context"
    "icooclaw/pkg/bus"
    "icooclaw/pkg/channels"
)

type MyChannel struct {
    config Config
    bus    *bus.MessageBus
}

func New(cfg Config, bus *bus.MessageBus) (*MyChannel, error) {
    return &MyChannel{config: cfg, bus: bus}, nil
}

func (c *MyChannel) Name() string { return "my_channel" }
func (c *MyChannel) Start(ctx context.Context) error { return nil }
func (c *MyChannel) Stop(ctx context.Context) error { return nil }
func (c *MyChannel) Send(ctx context.Context, msg channels.OutboundMessage) error { return nil }
func (c *MyChannel) IsRunning() bool { return true }
```

```go
// pkg/channels/mychannel/register.go
package mychannel

import "icooclaw/pkg/channels"

func init() {
    channels.RegisterFactory("my_channel", func(cfg map[string]any, bus *bus.MessageBus) (channels.Channel, error) {
        return New(parseConfig(cfg), bus)
    })
}
```

### 添加新提供商

1. 在 `pkg/providers/` 创建文件
2. 实现 `Provider` 接口
3. 注册到 `Registry`

```go
// pkg/providers/myprovider.go
package providers

import "context"

type MyProvider struct {
    apiKey string
    baseURL string
}

func NewMyProvider(cfg *storage.Provider) *MyProvider {
    return &MyProvider{
        apiKey:  cfg.APIKey,
        baseURL: cfg.APIBase,
    }
}

func (p *MyProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // 实现 API 调用
}

func (p *MyProvider) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) error {
    // 实现流式调用
}

func (p *MyProvider) GetDefaultModel() string { return "my-model" }
func (p *MyProvider) GetName() string { return "my_provider" }
```

```go
// 在 registry.go 中注册
func (r *Registry) RegisterBuiltins() {
    // ...
    r.RegisterFactory(consts.ProviderMyProvider, NewMyProvider)
}
```

### 添加新工具

1. 实现 `Tool` 接口
2. 注册到 `ToolRegistry`

```go
// pkg/tools/builtin/mytool.go
package builtin

import (
    "context"
    "icooclaw/pkg/tools"
)

type MyTool struct{}

func NewMyTool() *MyTool {
    return &MyTool{}
}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "My custom tool description"
}

func (t *MyTool) Parameters() map[string]any {
    return map[string]any{
        "input": map[string]any{
            "type":        "string",
            "description": "Input parameter",
        },
    }
}

func (t *MyTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
    input, _ := args["input"].(string)
    // 实现工具逻辑
    return &tools.Result{
        Success: true,
        Content: "result",
    }
}
```

## 测试

### 单元测试

```go
// pkg/agent/agent_test.go
package agent

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAgent_ProcessMessage(t *testing.T) {
    // 准备
    agent := NewAgentInstance(AgentConfig{Name: "test"})
    
    // 执行
    result, err := agent.ProcessMessage(ctx, "hello")
    
    // 断言
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

### 集成测试

```go
// pkg/agent/integration_test.go
//go:build integration

package agent

import (
    "testing"
)

func TestAgent_FullFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // 完整流程测试
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -run TestAgent ./pkg/agent/...

# 运行集成测试
go test -tags=integration ./...

# 查看覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 调试

### 日志级别

```toml
[logging]
level = "debug"  # debug, info, warn, error
```

### 使用 Delve

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试
dlv debug ./cmd/icooclaw -- start -c config.toml
```

### 性能分析

```go
import (
    "net/http"
    _ "net/http/pprof"
)

func init() {
    go http.ListenAndServe(":6060", nil)
}
```

```bash
# CPU 分析
go tool pprof http://localhost:6060/debug/pprof/profile

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap
```

## 发布流程

### 1. 更新版本

```bash
# 更新版本号
git tag v1.0.0
git push origin v1.0.0
```

### 2. 构建发布

```bash
# 构建多平台
make build-all

# 或手动构建
GOOS=linux GOARCH=amd64 go build -o icooclaw-linux-amd64 ./cmd/icooclaw
GOOS=darwin GOARCH=amd64 go build -o icooclaw-darwin-amd64 ./cmd/icooclaw
GOOS=windows GOARCH=amd64 go build -o icooclaw-windows-amd64.exe ./cmd/icooclaw
```

### 3. 创建 Release

在 GitHub 上创建 Release，上传构建产物。

## 贡献指南

### 提交信息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型：**
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具

**示例：**

```
feat(channel): add Telegram channel support

- Implement Telegram bot API
- Support message editing and reactions
- Add inline keyboard support

Closes #123
```

### Pull Request 流程

1. Fork 项目
2. 创建分支
3. 提交更改
4. 创建 Pull Request
5. 等待代码审查
6. 合并代码

### 代码审查标准

- 代码质量
- 测试覆盖
- 文档完整
- 向后兼容