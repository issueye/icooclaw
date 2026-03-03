// Package gateway 提供 HTTP REST API 网关
package gateway

import (
	"context"
	"net/http"
)

// Gateway HTTP 网关接口
type Gateway interface {
	// Start 启动网关服务
	Start(ctx context.Context) error
	// Stop 停止网关服务
	Stop() error
	// IsRunning 检查是否运行中
	IsRunning() bool
	// Router 获取路由器（用于挂载到外部服务）
	Router() http.Handler
}

// Config 网关配置
type Config interface {
	Enabled() bool
	Host() string
	Port() int
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}
