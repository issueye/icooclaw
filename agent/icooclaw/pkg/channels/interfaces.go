// Package channels provides channel management for icooclaw.
package channels

import (
	"context"
	"net/http"
)

// TypingCapable is an optional interface for channels that support typing indicators.
type TypingCapable interface {
	StartTyping(ctx context.Context, chatID string) (stop func(), err error)
}

// MessageEditor is an optional interface for channels that support message editing.
type MessageEditor interface {
	EditMessage(ctx context.Context, chatID string, messageID string, content string) error
}

// ReactionCapable is an optional interface for channels that support message reactions.
type ReactionCapable interface {
	ReactToMessage(ctx context.Context, chatID, messageID string) (undo func(), err error)
}

// PlaceholderCapable is an optional interface for channels that support placeholder messages.
type PlaceholderCapable interface {
	SendPlaceholder(ctx context.Context, chatID string) (messageID string, err error)
}

// MediaSender is an optional interface for channels that support media sending.
type MediaSender interface {
	SendMedia(ctx context.Context, msg OutboundMediaMessage) error
}

// WebhookHandler is an optional interface for channels that handle webhooks.
type WebhookHandler interface {
	WebhookPath() string
	http.Handler
}

// HealthChecker is an optional interface for channels with health endpoints.
type HealthChecker interface {
	HealthPath() string
	HealthHandler(w http.ResponseWriter, r *http.Request)
}

// PlaceholderRecorder records placeholder messages for later editing.
type PlaceholderRecorder interface {
	RecordPlaceholder(channel, chatID, messageID string)
	GetPlaceholder(channel, chatID string) string
	DeletePlaceholder(channel, chatID string)
}