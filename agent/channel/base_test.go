package channel

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseChannel tests for BaseChannel
func TestBaseChannel_NewBaseChannel(t *testing.T) {
	logger := slog.Default()
	ch := NewBaseChannel("test", logger)
	require.NotNil(t, ch)
	assert.Equal(t, "test", ch.Name())
}

func TestBaseChannel_IsRunning(t *testing.T) {
	ch := NewBaseChannel("test", nil)

	// Initially not running
	assert.False(t, ch.IsRunning())

	// Set running
	ch.SetRunning(true)
	assert.True(t, ch.IsRunning())

	// Set not running
	ch.SetRunning(false)
	assert.False(t, ch.IsRunning())
}

// TestOutboundMessage tests for OutboundMessage struct
func TestOutboundMessage_Structure(t *testing.T) {
	msg := OutboundMessage{
		Channel:   "telegram",
		ChatID:    "chat_123",
		Content:   "Hello",
		ParseMode: "markdown",
	}

	assert.Equal(t, "telegram", msg.Channel)
	assert.Equal(t, "chat_123", msg.ChatID)
	assert.Equal(t, "Hello", msg.Content)
	assert.Equal(t, "markdown", msg.ParseMode)
}

// TestInboundMessage tests for InboundMessage struct
func TestInboundMessage_Structure(t *testing.T) {
	msg := InboundMessage{
		Channel:   "telegram",
		ChatID:    "chat_123",
		UserID:    "user_456",
		Content:   "Hello",
		MessageID: "msg_789",
	}

	assert.Equal(t, "telegram", msg.Channel)
	assert.Equal(t, "chat_123", msg.ChatID)
	assert.Equal(t, "user_456", msg.UserID)
	assert.Equal(t, "Hello", msg.Content)
	assert.Equal(t, "msg_789", msg.MessageID)
}

// TestBaseChannelMethods tests the methods on BaseChannel
func TestBaseChannelMethods(t *testing.T) {
	ch := NewBaseChannel("test", nil)

	// Name
	assert.Equal(t, "test", ch.Name())

	// IsRunning
	assert.False(t, ch.IsRunning())
	ch.SetRunning(true)
	assert.True(t, ch.IsRunning())
	ch.SetRunning(false)
	assert.False(t, ch.IsRunning())
}
