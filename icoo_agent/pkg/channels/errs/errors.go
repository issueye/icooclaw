// Package channels provides channel management for icooclaw.
package errs

import (
	"errors"
	"fmt"
)

// Sentinel errors for channel operations.
var (
	ErrNotRunning      = errors.New("通道未运行")
	ErrRateLimit       = errors.New("请求频率受限")
	ErrTemporary       = errors.New("临时故障")
	ErrSendFailed      = errors.New("发送失败")
	ErrChannelNotFound = errors.New("通道未找到")
)

// ClassifySendError classifies an HTTP error for retry decisions.
func ClassifySendError(statusCode int, rawErr error) error {
	switch {
	case statusCode == 429: // Too Many Requests
		return fmt.Errorf("%w: %v", ErrRateLimit, rawErr)
	case statusCode >= 500: // Server Error
		return fmt.Errorf("%w: %v", ErrTemporary, rawErr)
	case statusCode >= 400: // Client Error
		return fmt.Errorf("%w: %v", ErrSendFailed, rawErr)
	default:
		return rawErr
	}
}

// ClassifyNetError classifies a network error for retry decisions.
func ClassifyNetError(err error) error {
	return fmt.Errorf("%w: %v", ErrTemporary, err)
}

// IsRetriable returns true if the error is retriable.
func IsRetriable(err error) bool {
	return errors.Is(err, ErrRateLimit) || errors.Is(err, ErrTemporary)
}

// IsPermanent returns true if the error is permanent (not retriable).
func IsPermanent(err error) bool {
	return errors.Is(err, ErrNotRunning) || errors.Is(err, ErrSendFailed)
}
