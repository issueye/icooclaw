// Package errors provides unified error handling for icooclaw.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors
var (
	// Provider errors
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrRateLimited         = errors.New("rate limited")
	ErrTimeout             = errors.New("timeout")
	ErrAuthFailed          = errors.New("authentication failed")

	// Config errors
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrConfigNotFound = errors.New("configuration not found")

	// Tool errors
	ErrToolNotFound    = errors.New("tool not found")
	ErrToolExecution   = errors.New("tool execution failed")
	ErrToolTimeout     = errors.New("tool execution timeout")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")

	// Channel errors
	ErrChannelNotRunning = errors.New("channel not running")
	ErrChannelNotFound   = errors.New("channel not found")
	ErrSendFailed        = errors.New("send failed")

	// Storage errors
	ErrStorageFailed   = errors.New("storage operation failed")
	ErrRecordNotFound  = errors.New("record not found")
	ErrDuplicateRecord = errors.New("duplicate record")

	// Memory errors
	ErrMemoryLoadFailed = errors.New("memory load failed")

	// MCP errors
	ErrMCPConnectionFailed = errors.New("MCP connection failed")
	ErrMCPToolNotFound     = errors.New("MCP tool not found")

	// Generic errors
	ErrBufferFull   = errors.New("buffer full")
	ErrNotRunning   = errors.New("not running")
	ErrTemporary    = errors.New("temporary failure")
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
		return fmt.Sprintf("failover [%s]: %s (provider=%s, model=%s, status=%d)",
			e.Reason, e.Wrapped.Error(), e.Provider, e.Model, e.Status)
	}
	return fmt.Sprintf("failover [%s]: provider=%s, model=%s, status=%d",
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