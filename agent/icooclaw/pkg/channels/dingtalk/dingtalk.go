// Package dingtalk provides DingTalk channel implementation for icooclaw.
package dingtalk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
)

// Config contains DingTalk channel configuration.
type Config struct {
	Enabled         bool     `json:"enabled" mapstructure:"enabled"`
	ClientID        string   `json:"client_id" mapstructure:"client_id"`
	ClientSecret    string   `json:"client_secret" mapstructure:"client_secret"`
	AgentID         int64    `json:"agent_id" mapstructure:"agent_id"`
	AllowFrom       []string `json:"allow_from" mapstructure:"allow_from"`
	ReasoningChatID string   `json:"reasoning_chat_id" mapstructure:"reasoning_chat_id"`
}

// Channel implements the channels.Channel interface for DingTalk.
type Channel struct {
	config       Config
	bus          *bus.MessageBus
	logger       *slog.Logger
	clientID     string
	clientSecret string
	streamClient *client.StreamClient
	ctx          context.Context
	cancel       context.CancelFunc

	// Map to store session webhooks for each chat
	sessionWebhooks sync.Map // chatID -> sessionWebhook

	running atomic.Bool
}

// New creates a new DingTalk channel instance.
func New(cfg Config, b *bus.MessageBus, logger *slog.Logger) (*Channel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk client_id and client_secret are required")
	}

	return &Channel{
		config:       cfg,
		bus:          b,
		logger:       logger,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}, nil
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "dingtalk"
}

// Start initializes the DingTalk channel with Stream Mode.
func (c *Channel) Start(ctx context.Context) error {
	c.logger.Info("Starting DingTalk channel (Stream Mode)...")

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Create credential config
	cred := client.NewAppCredentialConfig(c.clientID, c.clientSecret)

	// Create the stream client with options
	c.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// Register chatbot callback handler
	c.streamClient.RegisterChatBotCallbackRouter(c.onChatBotMessageReceived)

	// Start the stream client
	if err := c.streamClient.Start(c.ctx); err != nil {
		return fmt.Errorf("failed to start stream client: %w", err)
	}

	c.running.Store(true)
	c.logger.Info("DingTalk channel started (Stream Mode)")
	return nil
}

// Stop gracefully stops the DingTalk channel.
func (c *Channel) Stop(ctx context.Context) error {
	c.logger.Info("Stopping DingTalk channel...")

	if c.cancel != nil {
		c.cancel()
	}

	if c.streamClient != nil {
		c.streamClient.Close()
	}

	c.running.Store(false)
	c.logger.Info("DingTalk channel stopped")
	return nil
}

// IsRunning returns true if the channel is running.
func (c *Channel) IsRunning() bool {
	return c.running.Load()
}

// IsAllowed checks if a sender is allowed.
func (c *Channel) IsAllowed(senderID string) bool {
	if len(c.config.AllowFrom) == 0 {
		return true
	}

	for _, allowed := range c.config.AllowFrom {
		if senderID == allowed {
			return true
		}
	}
	return false
}

// IsAllowedSender checks if a sender is allowed (with full info).
func (c *Channel) IsAllowedSender(sender channels.SenderInfo) bool {
	return c.IsAllowed(sender.ID)
}

// ReasoningChannelID returns the channel ID for reasoning messages.
func (c *Channel) ReasoningChannelID() string {
	return c.config.ReasoningChatID
}

// Send sends a message to DingTalk via the chatbot reply API.
func (c *Channel) Send(ctx context.Context, msg channels.OutboundMessage) error {
	if !c.IsRunning() {
		return channels.ErrNotRunning
	}

	// Get session webhook from storage
	sessionWebhookRaw, ok := c.sessionWebhooks.Load(msg.ChatID)
	if !ok {
		return fmt.Errorf("no session_webhook found for chat %s, cannot send message", msg.ChatID)
	}

	sessionWebhook, ok := sessionWebhookRaw.(string)
	if !ok {
		return fmt.Errorf("invalid session_webhook type for chat %s", msg.ChatID)
	}

	c.logger.Debug("Sending message", "chat_id", msg.ChatID, "preview", truncate(msg.Text, 100))

	// Use the session webhook to send the reply
	return c.SendDirectReply(ctx, sessionWebhook, msg.Text)
}

// onChatBotMessageReceived implements the IChatBotMessageHandler function signature.
func (c *Channel) onChatBotMessageReceived(
	ctx context.Context,
	data *chatbot.BotCallbackDataModel,
) ([]byte, error) {
	// Extract message content from Text field
	content := data.Text.Content
	if content == "" {
		// Try to extract from Content interface{} if Text is empty
		if contentMap, ok := data.Content.(map[string]any); ok {
			if textContent, ok := contentMap["content"].(string); ok {
				content = textContent
			}
		}
	}

	if content == "" {
		return nil, nil // Ignore empty messages
	}

	senderID := data.SenderStaffId
	senderNick := data.SenderNick
	chatID := senderID
	if data.ConversationType != "1" {
		// For group chats
		chatID = data.ConversationId
	}

	// Store the session webhook for this chat so we can reply later
	c.sessionWebhooks.Store(chatID, data.SessionWebhook)

	// Check allowlist
	if !c.IsAllowed(senderID) {
		return nil, nil
	}

	metadata := map[string]any{
		"sender_name":       senderNick,
		"conversation_id":   data.ConversationId,
		"conversation_type": data.ConversationType,
		"platform":          "dingtalk",
		"session_webhook":   data.SessionWebhook,
	}

	c.logger.Debug("Received message",
		"sender_nick", senderNick,
		"sender_id", senderID,
		"preview", truncate(content, 50),
	)

	// Build inbound message
	inboundMsg := bus.InboundMessage{
		Channel:   c.Name(),
		ChatID:    chatID,
		Sender:    bus.SenderInfo{ID: senderID, Name: senderNick},
		Text:      content,
		Metadata:  metadata,
	}

	// Publish to bus
	if err := c.bus.PublishInbound(ctx, inboundMsg); err != nil {
		c.logger.Error("Failed to publish inbound message", "error", err)
	}

	// Return nil to indicate we've handled the message asynchronously
	return nil, nil
}

// SendDirectReply sends a direct reply using the session webhook.
func (c *Channel) SendDirectReply(ctx context.Context, sessionWebhook, content string) error {
	replier := chatbot.NewChatbotReplier()

	// Convert string content to []byte for the API
	contentBytes := []byte(content)
	titleBytes := []byte("AI Assistant")

	// Send markdown formatted reply
	err := replier.SimpleReplyMarkdown(
		ctx,
		sessionWebhook,
		titleBytes,
		contentBytes,
	)
	if err != nil {
		return fmt.Errorf("dingtalk send: %w", channels.ErrTemporary)
	}

	return nil
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 0 {
		return ""
	}
	return s[:maxLen]
}