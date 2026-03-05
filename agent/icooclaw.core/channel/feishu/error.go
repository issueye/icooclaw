package feishu

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// FeishuErrorCode 飞书 API 错误码
// 文档: https://open.feishu.cn/document/server-docs/getting-started/error-code
type FeishuErrorCode int

const (
	ErrCodeSuccess          FeishuErrorCode = 0
	ErrCodeTokenInvalid     FeishuErrorCode = 99991663
	ErrCodeTokenExpired     FeishuErrorCode = 99991664
	ErrCodeRateLimit        FeishuErrorCode = 99991400
	ErrCodePermissionDenied FeishuErrorCode = 99991661
	ErrCodeMessageNotFound  FeishuErrorCode = 230001
	ErrCodeChatNotFound     FeishuErrorCode = 230002
	ErrCodeParamInvalid     FeishuErrorCode = 99991600
)

// FeishuError 飞书 API 错误
type FeishuError struct {
	Code    FeishuErrorCode `json:"code"`
	Message string          `json:"msg"`
}

func (e *FeishuError) Error() string {
	return fmt.Sprintf("feishu error [%d]: %s", e.Code, e.Message)
}

// IsRetryable 判断错误是否可重试
func (e *FeishuError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeTokenExpired, ErrCodeRateLimit:
		return true
	default:
		return false
	}
}

// IsTokenError 判断是否为 Token 相关错误
func (e *FeishuError) IsTokenError() bool {
	return e.Code == ErrCodeTokenInvalid || e.Code == ErrCodeTokenExpired
}

// IsSuccess 判断是否成功
func (e *FeishuError) IsSuccess() bool {
	return e.Code == ErrCodeSuccess
}

// NewFeishuError 创建飞书错误
func NewFeishuError(code int, message string) *FeishuError {
	return &FeishuError{
		Code:    FeishuErrorCode(code),
		Message: message,
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int           // 最大重试次数
	InitialWait time.Duration // 初始等待时间
	MaxWait     time.Duration // 最大等待时间
	Multiplier  float64       // 等待时间乘数
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	InitialWait: 100 * time.Millisecond,
	MaxWait:     5 * time.Second,
	Multiplier:  2.0,
}

// doWithRetry 带重试的请求执行
func doWithRetry(ctx context.Context, fn func() error, cfg RetryConfig, onRetry func(attempt int, err error)) error {
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		var feishuErr *FeishuError
		if errors.As(err, &feishuErr) {
			if !feishuErr.IsRetryable() {
				return err
			}
		}

		lastErr = err

		if onRetry != nil {
			onRetry(attempt, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		wait = time.Duration(float64(wait) * cfg.Multiplier)
		if wait > cfg.MaxWait {
			wait = cfg.MaxWait
		}
	}

	return fmt.Errorf("重试 %d 次后仍失败: %w", cfg.MaxAttempts, lastErr)
}
