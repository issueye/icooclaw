// Package channels provides channel management for icooclaw.
package channels

import (
	"context"
)

// SenderInfo contains information about the message sender.
type SenderInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	IsBot    bool   `json:"is_bot"`
}

// Channel is the interface for message channels.
type Channel interface {
	// Name returns the channel name.
	Name() string

	// Start starts the channel.
	Start(ctx context.Context) error

	// Stop stops the channel.
	Stop(ctx context.Context) error

	// Send sends a message to the channel.
	Send(ctx context.Context, msg OutboundMessage) error

	// IsRunning returns true if the channel is running.
	IsRunning() bool

	// IsAllowed checks if a sender is allowed.
	IsAllowed(senderID string) bool

	// IsAllowedSender checks if a sender is allowed (with full info).
	IsAllowedSender(sender SenderInfo) bool

	// ReasoningChannelID returns the channel ID for reasoning messages.
	ReasoningChannelID() string
}