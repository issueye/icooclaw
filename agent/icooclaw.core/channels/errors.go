// Package channels 提供消息通道的抽象和管理
package channels

import "errors"

// 通道错误哨兵
var (
	// ErrNotRunning 表示通道未运行，Manager 不会重试
	ErrNotRunning = errors.New("channel not running")

	// ErrRateLimit 表示平台返回速率限制响应（如 HTTP 429）
	// Manager 会等待固定延迟后重试
	ErrRateLimit = errors.New("rate limited")

	// ErrTemporary 表示临时故障（如网络超时、5xx 错误）
	// Manager 会使用指数退避重试
	ErrTemporary = errors.New("temporary failure")

	// ErrSendFailed 表示永久失败（如无效 ChatID、4xx 非 429 错误）
	// Manager 不会重试
	ErrSendFailed = errors.New("send failed")

	// ErrChannelNotFound 表示通道未找到
	ErrChannelNotFound = errors.New("channel not found")

	// ErrNoChannelsEnabled 表示没有启用的通道
	ErrNoChannelsEnabled = errors.New("no channels enabled")
)