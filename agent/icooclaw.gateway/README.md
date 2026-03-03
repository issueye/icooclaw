# icooclaw.gateway

HTTP REST API 网关模块，提供统一的 API 接口层。

## 功能

- 健康检查 `/api/v1/health`
- 会话管理 `/api/v1/sessions`
- Provider 信息 `/api/v1/providers`
- 技能管理 `/api/v1/skills`

## 使用

```go
import "icooclaw.gateway"

// 创建网关
gateway := gateway.NewRESTGateway(cfg, storage, agent, skills, logger)

// 启动
gateway.Start(ctx)

// 或挂载到现有路由器
gateway.Mount(r, "/api")
```

## 接口

### StorageReader
```go
type StorageReader interface {
    GetSessions(userID, channel string) (interface{}, error)
    GetSessionMessages(sessionID uint, limit int) (interface{}, error)
    DeleteSession(sessionID string) error
}
```

### AgentReader
```go
type AgentReader interface {
    GetProvider() ProviderReader
}
```

### SkillReader
```go
type SkillReader interface {
    GetAllSkills() (interface{}, error)
    GetSkillByName(name string) (interface{}, error)
}
```