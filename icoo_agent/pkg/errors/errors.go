// Package errors provides unified error handling for icooclaw.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	// Provider errors
	ErrProviderUnavailable = errors.New("提供商不可用")
	ErrRateLimited         = errors.New("请求频率受限")
	ErrTimeout             = errors.New("请求超时")
	ErrAuthFailed          = errors.New("认证失败")

	// Config errors
	ErrInvalidConfig = errors.New("配置无效")
	ErrConfigNotFound = errors.New("配置未找到")

	// Tool errors
	ErrToolNotFound    = errors.New("工具未找到")
	ErrToolExecution   = errors.New("工具执行失败")
	ErrToolTimeout     = errors.New("工具执行超时")

	// Session errors
	ErrSessionNotFound = errors.New("会话未找到")
	ErrSessionExpired  = errors.New("会话已过期")

	// Channel errors
	ErrChannelNotRunning = errors.New("通道未运行")
	ErrChannelNotFound   = errors.New("通道未找到")
	ErrSendFailed        = errors.New("发送失败")

	// Storage errors
	ErrStorageFailed   = errors.New("存储操作失败")
	ErrRecordNotFound  = errors.New("记录未找到")
	ErrDuplicateRecord = errors.New("记录重复")

	// Memory errors
	ErrMemoryLoadFailed = errors.New("记忆加载失败")

	// MCP errors
	ErrMCPConnectionFailed = errors.New("MCP连接失败")
	ErrMCPToolNotFound     = errors.New("MCP工具未找到")

	// Generic errors
	ErrBufferFull   = errors.New("缓冲区已满")
	ErrNotRunning   = errors.New("未在运行")
	ErrTemporary    = errors.New("临时故障")
)

// FailoverReason represents the reason for provider failover.
type FailoverReason string

const (
	FailoverAuth      FailoverReason = "auth"
	FailoverRateLimit FailoverReason = "rate_limit"
	FailoverTimeout   FailoverReason = "timeout"
	FailoverFormat    FailoverReason = "format"
	FailoverUnknown   FailoverReason = "unknown"
)

// FailoverError represents an error that may trigger provider failover.
type FailoverError struct {
	Reason   FailoverReason
	Provider string
	Model    string
	Status   int
	Wrapped  error
}

func (e *FailoverError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("故障转移 [%s]: %s (提供商=%s, 模型=%s, 状态=%d)",
			e.Reason, e.Wrapped.Error(), e.Provider, e.Model, e.Status)
	}
	return fmt.Sprintf("故障转移 [%s]: 提供商=%s, 模型=%s, 状态=%d",
		e.Reason, e.Provider, e.Model, e.Status)
}

func (e *FailoverError) Unwrap() error {
	return e.Wrapped
}

// IsRetriable returns true if the error is retriable.
func (e *FailoverError) IsRetriable() bool {
	return e.Reason != FailoverFormat
}

// NewFailoverError creates a new FailoverError.
func NewFailoverError(reason FailoverReason, provider, model string, status int, wrapped error) *FailoverError {
	return &FailoverError{
		Reason:   reason,
		Provider: provider,
		Model:    model,
		Status:   status,
		Wrapped:  wrapped,
	}
}

// ClassifiedError represents a classified error with retry information.
type ClassifiedError struct {
	Code      string
	Message   string
	Retriable bool
	Cause     error
}

func (e *ClassifiedError) Error() string {
	return e.Message
}

func (e *ClassifiedError) Unwrap() error {
	return e.Cause
}

// NewClassifiedError creates a new ClassifiedError.
func NewClassifiedError(code, message string, retriable bool, cause error) *ClassifiedError {
	return &ClassifiedError{
		Code:      code,
		Message:   message,
		Retriable: retriable,
		Cause:     cause,
	}
}

// Wrap wraps an error with context.
func Wrap(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context.
func Wrapf(err error, format string, args ...any) error {
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}