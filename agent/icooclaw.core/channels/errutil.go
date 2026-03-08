package channels

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ClassifySendError 根据 HTTP 状态码将原始错误包装为适当的哨兵错误。
// 执行 HTTP API 调用的通道应在 Send 路径中使用此函数。
func ClassifySendError(statusCode int, rawErr error) error {
	switch {
	case statusCode == http.StatusTooManyRequests:
		return fmt.Errorf("%w: %v", ErrRateLimit, rawErr)
	case statusCode >= 500:
		return fmt.Errorf("%w: %v", ErrTemporary, rawErr)
	case statusCode >= 400:
		return fmt.Errorf("%w: %v", ErrSendFailed, rawErr)
	default:
		return rawErr
	}
}

// ClassifyNetError 将网络/超时错误包装为 ErrTemporary。
func ClassifyNetError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrTemporary, err)
}

// IsTemporaryError 检查错误是否为临时错误（可重试）。
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	// 检查哨兵错误
	if errors.Is(err, ErrTemporary) || errors.Is(err, ErrRateLimit) {
		return true
	}
	// 检查网络临时错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// IsPermanentError 检查错误是否为永久错误（不应重试）。
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotRunning) || errors.Is(err, ErrSendFailed)
}