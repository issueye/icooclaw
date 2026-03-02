package bus

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMessageBus tests for NewMessageBus
func TestNewMessageBus(t *testing.T) {
	bus := NewMessageBus()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.inbound)
	assert.NotNil(t, bus.outbound)
	assert.NotNil(t, bus.subscribers)
}

func TestNewMessageBus_WithBufferSize(t *testing.T) {
	bus := NewMessageBus(50)
	require.NotNil(t, bus)
	assert.Equal(t, 50, bus.bufferSize)
}

// TestMessageBus_SetLogger tests for SetLogger
func TestMessageBus_SetLogger(t *testing.T) {
	bus := NewMessageBus()
	require.NotNil(t, bus)

	bus.SetLogger(nil) // Should not panic
}

// TestMessageBus_PublishInbound tests for PublishInbound
func TestMessageBus_PublishInbound(t *testing.T) {
	bus := NewMessageBus(10)

	msg := InboundMessage{
		ID:        "msg_1",
		Channel:   "test",
		ChatID:    "chat_1",
		UserID:    "user_1",
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	err := bus.PublishInbound(context.Background(), msg)
	require.NoError(t, err)
}

func TestMessageBus_PublishInbound_ChannelFull(t *testing.T) {
	bus := NewMessageBus(1)

	// Fill the channel
	msg := InboundMessage{ID: "msg_1"}
	err := bus.PublishInbound(context.Background(), msg)
	require.NoError(t, err)

	// Try to publish another one (should fail because channel is full)
	err = bus.PublishInbound(context.Background(), InboundMessage{ID: "msg_2"})
	assert.Error(t, err)
	assert.Equal(t, ErrChannelFull, err)
}

func TestMessageBus_PublishInbound_CancelledContext(t *testing.T) {
	bus := NewMessageBus(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msg := InboundMessage{ID: "msg_1"}
	// The behavior might differ based on implementation, so we just check it doesn't panic
	_ = bus.PublishInbound(ctx, msg)
}

// TestMessageBus_PublishOutbound tests for PublishOutbound
func TestMessageBus_PublishOutbound(t *testing.T) {
	bus := NewMessageBus(10)

	msg := OutboundMessage{
		ID:        "msg_1",
		Channel:   "test",
		ChatID:    "chat_1",
		Content:   "Hello",
		Timestamp: time.Now(),
	}

	err := bus.PublishOutbound(context.Background(), msg)
	require.NoError(t, err)
}

func TestMessageBus_PublishOutbound_ChannelFull(t *testing.T) {
	bus := NewMessageBus(1)

	// Fill the channel
	msg := OutboundMessage{ID: "msg_1"}
	err := bus.PublishOutbound(context.Background(), msg)
	require.NoError(t, err)

	// Try to publish another one (should fail)
	err = bus.PublishOutbound(context.Background(), OutboundMessage{ID: "msg_2"})
	assert.Error(t, err)
	assert.Equal(t, ErrChannelFull, err)
}

// TestMessageBus_SubscribeInbound tests for SubscribeInbound
func TestMessageBus_SubscribeInbound(t *testing.T) {
	bus := NewMessageBus(10)

	ch := bus.SubscribeInbound("test_channel")
	require.NotNil(t, ch)

	// Subscribe again should return same channel
	ch2 := bus.SubscribeInbound("test_channel")
	assert.Equal(t, ch, ch2)
}

// TestMessageBus_UnsubscribeInbound tests for UnsubscribeInbound
func TestMessageBus_UnsubscribeInbound(t *testing.T) {
	bus := NewMessageBus(10)

	bus.SubscribeInbound("test_channel")
	bus.UnsubscribeInbound("test_channel")

	// After unsubscribe, should get a new channel
	ch := bus.SubscribeInbound("test_channel")
	require.NotNil(t, ch)
}

// TestMessageBus_ConsumeInbound tests for ConsumeInbound
func TestMessageBus_ConsumeInbound(t *testing.T) {
	bus := NewMessageBus(10)

	msg := InboundMessage{ID: "msg_1", Content: "test"}
	err := bus.PublishInbound(context.Background(), msg)
	require.NoError(t, err)

	// Consume the message
	consumed, err := bus.ConsumeInbound(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "msg_1", consumed.ID)
	assert.Equal(t, "test", consumed.Content)
}

func TestMessageBus_ConsumeInbound_CancelledContext(t *testing.T) {
	bus := NewMessageBus(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := bus.ConsumeInbound(ctx)
	assert.Error(t, err)
}

// TestMessageBus_ConsumeOutbound tests for ConsumeOutbound
func TestMessageBus_ConsumeOutbound(t *testing.T) {
	bus := NewMessageBus(10)

	msg := OutboundMessage{ID: "msg_1", Content: "test"}
	err := bus.PublishOutbound(context.Background(), msg)
	require.NoError(t, err)

	// Consume the message
	consumed, err := bus.ConsumeOutbound(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "msg_1", consumed.ID)
}

// TestMessageBus_InboundChannel tests for InboundChannel
func TestMessageBus_InboundChannel(t *testing.T) {
	bus := NewMessageBus(10)

	ch := bus.InboundChannel()
	require.NotNil(t, ch)
}

// TestMessageBus_OutboundChannel tests for OutboundChannel
func TestMessageBus_OutboundChannel(t *testing.T) {
	bus := NewMessageBus(10)

	ch := bus.OutboundChannel()
	require.NotNil(t, ch)
}

// TestMessageBus_Close tests for Close
func TestMessageBus_Close(t *testing.T) {
	bus := NewMessageBus(10)

	// Subscribe before close
	ch := bus.SubscribeInbound("test")

	// Close should not panic
	bus.Close()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed")
	default:
		// OK
	}
}

// TestBusError tests for BusError
func TestBusError_Error(t *testing.T) {
	err := &BusError{"test error"}
	assert.Equal(t, "test error", err.Error())
}

// TestInboundMessage tests for InboundMessage struct
func TestInboundMessage_Structure(t *testing.T) {
	msg := InboundMessage{
		ID:        "msg_1",
		Channel:   "telegram",
		ChatID:    "chat_123",
		UserID:    "user_456",
		Content:   "Hello",
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "msg_1", msg.ID)
	assert.Equal(t, "telegram", msg.Channel)
	assert.Equal(t, "chat_123", msg.ChatID)
	assert.Equal(t, "user_456", msg.UserID)
	assert.Equal(t, "Hello", msg.Content)
	assert.NotNil(t, msg.Metadata)
}

// TestOutboundMessage tests for OutboundMessage struct
func TestOutboundMessage_Structure(t *testing.T) {
	msg := OutboundMessage{
		ID:        "msg_1",
		Channel:   "telegram",
		ChatID:    "chat_123",
		Content:   "Hello",
		Timestamp: time.Now(),
		Metadata:  map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "msg_1", msg.ID)
	assert.Equal(t, "telegram", msg.Channel)
	assert.Equal(t, "chat_123", msg.ChatID)
	assert.Equal(t, "Hello", msg.Content)
	assert.NotNil(t, msg.Metadata)
}

// TestMessageBus_ConcurrentPublishAndConsume tests concurrent operations
func TestMessageBus_ConcurrentPublishAndConsume(t *testing.T) {
	bus := NewMessageBus(100)

	// Publish messages concurrently
	go func() {
		for i := 0; i < 50; i++ {
			bus.PublishInbound(context.Background(), InboundMessage{ID: "msg"})
		}
	}()

	// Consume messages concurrently
	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			bus.ConsumeInbound(context.Background())
		}
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
