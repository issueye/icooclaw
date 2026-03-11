// Package channels provides channel management for icooclaw.
package channels

import (
	"time"
)

// OutboundMessage represents a message to be sent to a channel.
type OutboundMessage struct {
	Channel   string         `json:"channel"`
	SessionID string         `json:"session_id"`
	Text      string         `json:"text"`
	Media     []string       `json:"media,omitempty"`
	ReplyTo   string         `json:"reply_to,omitempty"`
	EditID    string         `json:"edit_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// OutboundMediaMessage represents a media message to be sent.
type OutboundMediaMessage struct {
	Channel   string         `json:"channel"`
	SessionID string         `json:"session_id"`
	Media     []string       `json:"media"`
	Caption   string         `json:"caption,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// InboundMessage represents a message received from a channel.
type InboundMessage struct {
	Channel   string         `json:"channel"`
	SessionID string         `json:"session_id"`
	Sender    SenderInfo     `json:"sender"`
	Text      string         `json:"text"`
	Media     []string       `json:"media,omitempty"`
	ReplyTo   string         `json:"reply_to,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}