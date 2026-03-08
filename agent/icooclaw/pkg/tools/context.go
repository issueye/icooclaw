// Package tools provides tool management for icooclaw.
package tools

import (
	"context"
)

// Context key for tool context.
type contextKey struct{}

// Context contains context information for tool execution.
type Context struct {
	Channel string
	ChatID  string
}

// WithToolContext injects tool context into a context.
func WithToolContext(ctx context.Context, channel, chatID string) context.Context {
	return context.WithValue(ctx, contextKey{}, Context{
		Channel: channel,
		ChatID:  chatID,
	})
}

// GetToolContext extracts tool context from a context.
func GetToolContext(ctx context.Context) *Context {
	if tc, ok := ctx.Value(contextKey{}).(Context); ok {
		return &tc
	}
	return nil
}

// GetChannel extracts channel from context.
func GetChannel(ctx context.Context) string {
	if tc := GetToolContext(ctx); tc != nil {
		return tc.Channel
	}
	return ""
}

// GetChatID extracts chat ID from context.
func GetChatID(ctx context.Context) string {
	if tc := GetToolContext(ctx); tc != nil {
		return tc.ChatID
	}
	return ""
}