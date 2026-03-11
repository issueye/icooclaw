// Package feishu provides Feishu/Lark channel implementation for icooclaw.
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels"
	"icooclaw/pkg/channels/errs"
)

// Config contains Feishu channel configuration.
type Config struct {
	Enabled           bool     `json:"enabled" mapstructure:"enabled"`
	AppID             string   `json:"app_id" mapstructure:"app_id"`
	AppSecret         string   `json:"app_secret" mapstructure:"app_secret"`
	EncryptKey        string   `json:"encrypt_key" mapstructure:"encrypt_key"`
	VerificationToken string   `json:"verification_token" mapstructure:"verification_token"`
	AllowFrom         []string `json:"allow_from" mapstructure:"allow_from"`
	ReasoningChatID   string   `json:"reasoning_chat_id" mapstructure:"reasoning_chat_id"`
}

// Channel implements the channels.Channel interface for Feishu/Lark.
type Channel struct {
	config   Config
	bus      *bus.MessageBus
	client   *lark.Client
	wsClient *larkws.Client
	logger   *slog.Logger

	botOpenID atomic.Value // stores string; populated lazily for @mention detection

	running atomic.Bool
	mu      sync.Mutex
	cancel  context.CancelFunc
}

// New creates a new Feishu channel instance.
func New(cfg Config, b *bus.MessageBus, logger *slog.Logger) (*Channel, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("feishu app_id and app_secret are required")
	}

	return &Channel{
		config: cfg,
		bus:    b,
		client: lark.NewClient(cfg.AppID, cfg.AppSecret),
		logger: logger,
	}, nil
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "feishu"
}

// Start starts the Feishu channel.
func (c *Channel) Start(ctx context.Context) error {
	if c.config.AppID == "" || c.config.AppSecret == "" {
		return fmt.Errorf("feishu app_id or app_secret is empty")
	}

	// Fetch bot open_id via API for reliable @mention detection.
	if err := c.fetchBotOpenID(ctx); err != nil {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", err)
		return fmt.Errorf("获取机器人open_id失败：%w", err)
	}

	dispatcher := larkdispatcher.NewEventDispatcher(c.config.VerificationToken, c.config.EncryptKey).
		OnP2MessageReceiveV1(c.handleMessageReceive)

	runCtx, cancel := context.WithCancel(ctx)

	c.mu.Lock()
	c.cancel = cancel
	c.wsClient = larkws.NewClient(
		c.config.AppID,
		c.config.AppSecret,
		larkws.WithEventHandler(dispatcher),
	)
	wsClient := c.wsClient
	c.mu.Unlock()

	c.running.Store(true)
	c.logger.With("name", "【飞书】").Info("启动通道...（流模式）")

	go func() {
		if err := wsClient.Start(runCtx); err != nil {
			c.logger.With("name", "【飞书】").Error("启动通道失败：", "error", err)
		}
	}()

	return nil
}

// Stop stops the Feishu channel.
func (c *Channel) Stop(ctx context.Context) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.wsClient = nil
	c.mu.Unlock()

	c.running.Store(false)
	c.logger.With("name", "【飞书】").Info("通道已停止")
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

// Send sends a message using Interactive Card format for markdown rendering.
func (c *Channel) Send(ctx context.Context, msg channels.OutboundMessage) error {
	if !c.IsRunning() {
		return errs.ErrNotRunning
	}

	if msg.ChatID == "" {
		c.logger.With("name", "【飞书】").Error("发送消息失败：chatID不能为空", "error", errs.ErrSendFailed)
		return fmt.Errorf("chat ID is empty: %w", errs.ErrSendFailed)
	}

	// Build interactive card with markdown content
	cardContent, err := buildMarkdownCard(msg.Text)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：卡片构建失败", "error", err)
		return fmt.Errorf("feishu send: card build failed: %w", err)
	}
	return c.sendCard(ctx, msg.ChatID, cardContent)
}

// EditMessage implements channels.MessageEditor.
func (c *Channel) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	cardContent, err := buildMarkdownCard(content)
	if err != nil {
		return fmt.Errorf("feishu edit: card build failed: %w", err)
	}

	req := larkim.NewPatchMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewPatchMessageReqBodyBuilder().Content(cardContent).Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Patch(ctx, req)
	if err != nil {
		return fmt.Errorf("feishu edit: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("feishu edit api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}
	return nil
}

// SendPlaceholder implements channels.PlaceholderCapable.
func (c *Channel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	text := "Thinking..."

	cardContent, err := buildMarkdownCard(text)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder: card build failed: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("feishu placeholder send: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("feishu placeholder api error (code=%d msg=%s)", resp.Code, resp.Msg)
	}

	if resp.Data != nil && resp.Data.MessageId != nil {
		return *resp.Data.MessageId, nil
	}
	return "", nil
}

// ReactToMessage implements channels.ReactionCapable.
func (c *Channel) ReactToMessage(ctx context.Context, chatID, messageID string) (func(), error) {
	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().EmojiType("Pin").Build()).
			Build()).
		Build()

	resp, err := c.client.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：添加反应失败", "error", err)
		return func() {}, fmt.Errorf("feishu react: %w", err)
	}
	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("发送消息失败：添加反应失败", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return func() {}, fmt.Errorf("feishu react: %w", err)
	}

	var reactionID string
	if resp.Data != nil && resp.Data.ReactionId != nil {
		reactionID = *resp.Data.ReactionId
	}
	if reactionID == "" {
		return func() {}, nil
	}

	var undone atomic.Bool
	undo := func() {
		if !undone.CompareAndSwap(false, true) {
			return
		}
		delReq := larkim.NewDeleteMessageReactionReqBuilder().
			MessageId(messageID).
			ReactionId(reactionID).
			Build()
		_, _ = c.client.Im.V1.MessageReaction.Delete(context.Background(), delReq)
	}
	return undo, nil
}

// --- Inbound message handling ---

func (c *Channel) handleMessageReceive(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	message := event.Event.Message
	sender := event.Event.Sender

	chatID := stringValue(message.ChatId)
	if chatID == "" {
		return nil
	}

	senderID := extractSenderID(sender)
	if senderID == "" {
		senderID = "unknown"
	}

	messageType := stringValue(message.MessageType)
	messageID := stringValue(message.MessageId)
	rawContent := stringValue(message.Content)

	// Check allowlist early
	if !c.IsAllowed(senderID) {
		return nil
	}

	// Extract content based on message type
	content := extractContent(messageType, rawContent)

	// Handle media messages
	var mediaRefs []string
	if messageID != "" {
		mediaRefs = c.downloadInboundMedia(ctx, chatID, messageID, messageType, rawContent)
	}

	// Append media tags to content
	content = appendMediaTags(content, messageType, mediaRefs)

	if content == "" {
		content = "[empty message]"
	}

	metadata := map[string]any{}
	if messageID != "" {
		metadata["message_id"] = messageID
	}
	if messageType != "" {
		metadata["message_type"] = messageType
	}
	chatType := stringValue(message.ChatType)
	if chatType != "" {
		metadata["chat_type"] = chatType
	}
	if sender != nil && sender.TenantKey != nil {
		metadata["tenant_key"] = *sender.TenantKey
	}

	// Build inbound message
	inboundMsg := bus.InboundMessage{
		Channel:  c.Name(),
		ChatID:   chatID,
		Sender:   bus.SenderInfo{ID: senderID},
		Text:     content,
		Media:    mediaRefs,
		Metadata: metadata,
	}

	c.logger.With("name", "【飞书】").Info("收到消息",
		slog.String("sender_id", senderID),
		"chat_id", chatID,
		"message_id", messageID,
		"preview", truncate(content, 80),
	)

	// Publish to bus
	if err := c.bus.PublishInbound(ctx, inboundMsg); err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：发布消息失败", "error", err)
	}

	return nil
}

// --- Internal helpers ---

// fetchBotOpenID calls the Feishu bot info API to retrieve and store the bot's open_id.
func (c *Channel) fetchBotOpenID(ctx context.Context) error {
	resp, err := c.client.Do(ctx, &larkcore.ApiReq{
		HttpMethod:                http.MethodGet,
		ApiPath:                   "/open-apis/bot/v3/info",
		SupportedAccessTokenTypes: []larkcore.AccessTokenType{larkcore.AccessTokenTypeTenant},
	})
	if err != nil {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", err)
		return fmt.Errorf("bot info request: %w", err)
	}

	var result struct {
		Code int `json:"code"`
		Bot  struct {
			OpenID string `json:"open_id"`
		} `json:"bot"`
	}
	if err := json.Unmarshal(resp.RawBody, &result); err != nil {
		c.logger.With("name", "【飞书】").Error("解析机器人open_id失败", "error", err)
		return fmt.Errorf("机器人信息解析失败 %w", err)
	}
	if result.Code != 0 {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", slog.Any("code", result.Code))
		return fmt.Errorf("机器人信息解析失败 %w", err)
	}
	if result.Bot.OpenID == "" {
		c.logger.With("name", "【飞书】").Error("获取机器人open_id失败", "error", errs.ErrSendFailed)
		return fmt.Errorf("机器人open_id为空 %w", errs.ErrSendFailed)
	}

	c.botOpenID.Store(result.Bot.OpenID)
	c.logger.With("name", "【飞书】").Info("获取机器人open_id成功", slog.Any("open_id", result.Bot.OpenID))
	return nil
}

// sendCard sends an interactive card message to a chat.
func (c *Channel) sendCard(ctx context.Context, chatID, cardContent string) error {
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeInteractive).
			Content(cardContent).
			Build()).
		Build()

	resp, err := c.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("发送消息失败：发送卡片失败", "error", err)
		return fmt.Errorf("发送卡片失败 %w", err)
	}

	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("发送消息失败：发送卡片失败", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return fmt.Errorf("发送卡片失败 %w", err)
	}

	c.logger.With("name", "【飞书】").Info("发送卡片成功", slog.String("chat_id", chatID))
	return nil
}

// downloadInboundMedia downloads media from inbound messages.
func (c *Channel) downloadInboundMedia(
	ctx context.Context,
	chatID, messageID, messageType, rawContent string,
) []string {
	var refs []string

	switch messageType {
	case larkim.MsgTypeImage:
		imageKey := extractImageKey(rawContent)
		if imageKey == "" {
			return nil
		}
		ref := c.downloadResource(ctx, messageID, imageKey, "image", ".jpg")
		if ref != "" {
			refs = append(refs, ref)
		}

	case larkim.MsgTypeFile, larkim.MsgTypeAudio, larkim.MsgTypeMedia:
		fileKey := extractFileKey(rawContent)
		if fileKey == "" {
			return nil
		}
		var ext string
		switch messageType {
		case larkim.MsgTypeAudio:
			ext = ".ogg"
		case larkim.MsgTypeMedia:
			ext = ".mp4"
		default:
			ext = ""
		}
		ref := c.downloadResource(ctx, messageID, fileKey, "file", ext)
		if ref != "" {
			refs = append(refs, ref)
		}
	}

	return refs
}

// downloadResource downloads a message resource from Feishu.
func (c *Channel) downloadResource(
	ctx context.Context,
	messageID, fileKey, resourceType, fallbackExt string,
) string {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(resourceType).
		Build()

	resp, err := c.client.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("下载资源失败：", "error", err)
		return ""
	}
	if !resp.Success() {
		c.logger.With("name", "【飞书】").Error("下载资源失败：", slog.Any("code", resp.Code), slog.Any("msg", resp.Msg))
		return ""
	}

	if resp.File == nil {
		return ""
	}
	// Safely close the underlying reader if it implements io.Closer
	if closer, ok := resp.File.(io.Closer); ok {
		defer closer.Close()
	}

	filename := resp.FileName
	if filename == "" {
		filename = fileKey
	}
	if filepath.Ext(filename) == "" && fallbackExt != "" {
		filename += fallbackExt
	}

	// Write to temp directory
	mediaDir := filepath.Join(os.TempDir(), "icooclaw_media")
	if mkdirErr := os.MkdirAll(mediaDir, 0o700); mkdirErr != nil {
		c.logger.With("name", "【飞书】").Error("创建媒体目录失败", slog.String("目录", mediaDir), "error", mkdirErr.Error())
		return ""
	}
	ext := filepath.Ext(filename)
	localPath := filepath.Join(mediaDir, sanitizeFilename(messageID+"-"+fileKey+ext))

	out, err := os.Create(localPath)
	if err != nil {
		c.logger.With("name", "【飞书】").Error("创建媒体文件失败：", "error", err)
		return ""
	}

	if _, copyErr := io.Copy(out, resp.File); copyErr != nil {
		out.Close()
		os.Remove(localPath)
		c.logger.With("name", "【飞书】").Error("下载资源失败：", "error", copyErr.Error())
		return ""
	}
	out.Close()

	return localPath
}
