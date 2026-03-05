package feishu

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"icooclaw.core/channel"
)

// ============ 配置接口 ============

// Config 飞书通道配置接口
type Config interface {
	Enabled() bool
	Host() string
	Port() int
	Path() string
	VerificationToken() string
	EncryptKey() string
	AppID() string
	AppSecret() string
}

// ============ Channel 实现 ============

// Channel 飞书通道
type Channel struct {
	config    Config
	bus       channel.MessageBus
	server    *http.Server
	logger    *slog.Logger
	dispatcher *EventDispatcher

	// Token 缓存
	accessToken    string
	tokenExpiresAt time.Time
	tokenMu        sync.RWMutex

	// 用户缓存
	userCache *UserCache

	// 状态
	running bool
	mu      sync.RWMutex
}

// NewChannel 创建飞书通道
func NewChannel(cfg Config, bus channel.MessageBus, logger channel.Logger) *Channel {
	slogLogger := toSlogLogger(logger)

	return &Channel{
		config:     cfg,
		bus:        bus,
		logger:     slogLogger,
		dispatcher: NewEventDispatcher(slogLogger),
		userCache:  NewUserCache(30 * time.Minute),
	}
}

// Name 获取通道名称
func (c *Channel) Name() string {
	return "feishu"
}

// Start 启动通道
func (c *Channel) Start(ctx context.Context) error {
	if !c.config.Enabled() {
		c.logger.Info("飞书通道已禁用")
		return nil
	}

	// 注册默认事件处理器
	c.registerDefaultHandlers()

	// 启动 Webhook 服务
	return c.startWebhookServer(ctx)
}

// Stop 停止通道
func (c *Channel) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("飞书通道关闭异常", "error", err)
			return err
		}
	}

	c.running = false
	c.logger.Info("飞书通道已停止")
	return nil
}

// IsRunning 检查是否运行中
func (c *Channel) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// SetRunning 设置运行状态
func (c *Channel) SetRunning(running bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = running
}

// registerDefaultHandlers 注册默认事件处理器
func (c *Channel) registerDefaultHandlers() {
	c.dispatcher.Register(EventIMMessageReceive, c.handleMessageEvent)
	c.dispatcher.Register(EventChatMemberBotAdded, c.handleBotAddedEvent)
	c.dispatcher.Register(EventChatMemberAdded, c.handleMemberAddedEvent)
	c.dispatcher.Register(EventChatDisbanded, c.handleChatDisbandedEvent)
}

// startWebhookServer 启动 Webhook 服务器
func (c *Channel) startWebhookServer(ctx context.Context) error {
	host := c.config.Host()
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port()
	if port == 0 {
		port = 8082
	}
	path := c.config.Path()
	if path == "" {
		path = "/feishu/webhook"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, c.handleWebhook)
	mux.HandleFunc("/feishu/health", c.handleHealth)

	c.server = &http.Server{
		Addr:         host + ":" + strconv.Itoa(port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		c.logger.Info("飞书通道 Webhook 服务启动", "addr", host+":"+strconv.Itoa(port), "path", path)
		if err := c.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.logger.Error("飞书通道 Webhook 服务异常", "error", err)
		}
	}()

	c.SetRunning(true)
	return nil
}

// ============ Webhook 处理 ============

// WebhookEventBody 飞书事件请求体
type WebhookEventBody struct {
	Encrypt string `json:"encrypt,omitempty"`

	Schema string        `json:"schema,omitempty"`
	Header *EventHeader  `json:"header,omitempty"`
	Event  interface{}   `json:"event,omitempty"`

	Challenge string `json:"challenge,omitempty"`
	Token     string `json:"token,omitempty"`
	Type      string `json:"type,omitempty"`
}

// handleWebhook 处理飞书 Webhook 事件
func (c *Channel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body failed"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 签名验证
	if c.config.VerificationToken() != "" {
		timestamp := r.Header.Get("X-Lark-Request-Timestamp")
		nonce := r.Header.Get("X-Lark-Request-Nonce")
		signature := r.Header.Get("X-Lark-Signature")

		if signature != "" && !c.verifySignature(timestamp, nonce, signature, body) {
			c.logger.Warn("飞书签名验证失败")
			http.Error(w, `{"error":"signature mismatch"}`, http.StatusUnauthorized)
			return
		}
	}

	// 解密（如配置了 EncryptKey）
	if c.config.EncryptKey() != "" {
		body, err = c.decryptBody(body, c.config.EncryptKey())
		if err != nil {
			c.logger.Error("飞书消息解密失败", "error", err)
			http.Error(w, `{"error":"decrypt failed"}`, http.StatusBadRequest)
			return
		}
	}

	var event WebhookEventBody
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("飞书消息 JSON 解析失败", "error", err)
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	// 1. URL Verification 挑战响应
	if event.Challenge != "" {
		if c.config.VerificationToken() != "" && event.Token != c.config.VerificationToken() {
			http.Error(w, `{"error":"token mismatch"}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"challenge": event.Challenge})
		return
	}

	// 2. 事件分发
	if event.Header != nil {
		feishuEvent := &Event{
			Schema: event.Schema,
			Header: event.Header,
			Event:  event.Event,
		}

		if err := c.dispatcher.Dispatch(r.Context(), feishuEvent); err != nil {
			c.logger.Error("飞书事件处理失败", "error", err, "event_type", event.Header.EventType)
			http.Error(w, `{"error":"handle event failed"}`, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"msg": "ok"})
}

// handleHealth 健康检查
func (c *Channel) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// verifySignature 验证飞书请求签名
func (c *Channel) verifySignature(timestamp, nonce, signature string, body []byte) bool {
	if timestamp == "" || signature == "" {
		return false
	}

	// 防重放攻击：检查时间戳（5分钟内有效）
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-ts > 300 {
		c.logger.Warn("飞书请求时间戳过期")
		return false
	}

	// 计算签名: SHA256(timestamp + nonce + token + body)
	h := sha256.Sum256([]byte(timestamp + nonce + c.config.VerificationToken() + string(body)))
	expected := hex.EncodeToString(h[:])

	return hmac.Equal([]byte(signature), []byte(expected))
}

// decryptBody 解密飞书 AES-256-CBC 加密的消息体
func (c *Channel) decryptBody(body []byte, key string) ([]byte, error) {
	var envelope struct {
		Encrypt string `json:"encrypt"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Encrypt == "" {
		return body, nil
	}

	h := sha256.Sum256([]byte(key))
	aesKey := h[:]

	cipherData, err := base64.StdEncoding.DecodeString(envelope.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	if len(cipherData) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := cipherData[:aes.BlockSize]
	cipherData = cipherData[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(cipherData, cipherData)

	length := len(cipherData)
	if length == 0 {
		return nil, errors.New("empty plaintext after decryption")
	}
	padding := int(cipherData[length-1])
	if padding > aes.BlockSize || padding == 0 {
		return nil, errors.New("invalid padding")
	}
	return cipherData[:length-padding], nil
}

// ============ 事件处理器 ============

// handleMessageEvent 处理消息事件
func (c *Channel) handleMessageEvent(ctx context.Context, event *Event) error {
	imEvent, err := ParseIMMessageEvent(event)
	if err != nil {
		return err
	}

	if imEvent.Message == nil {
		return errors.New("event.message is nil")
	}

	// 只处理文字类型消息
	if imEvent.Message.MessageType != "text" {
		c.logger.Debug("飞书忽略非文字消息", "message_type", imEvent.Message.MessageType)
		return nil
	}

	// 解析 content JSON: {"text": "..."}
	text, err := ParseTextContent(imEvent.Message.Content)
	if err != nil {
		return fmt.Errorf("parse text content: %w", err)
	}

	userID := ""
	if imEvent.Sender != nil && imEvent.Sender.SenderID != nil {
		userID = imEvent.Sender.SenderID.OpenID
	}

	inbound := channel.InboundMessage{
		ID:        imEvent.Message.MessageID,
		Channel:   "feishu",
		ChatID:    imEvent.Message.ChatID,
		UserID:    userID,
		Content:   text,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"chat_type":    imEvent.Message.ChatType,
			"message_type": imEvent.Message.MessageType,
			"event_id":     event.Header.EventID,
		},
	}

	return c.bus.PublishInbound(ctx, inbound)
}

// handleBotAddedEvent 处理机器人被添加到群聊事件
func (c *Channel) handleBotAddedEvent(ctx context.Context, event *Event) error {
	memberEvent, err := ParseChatMemberEvent(event)
	if err != nil {
		return err
	}

	c.logger.Info("机器人被添加到群聊", "chat_id", memberEvent.ChatID)

	// 发送欢迎消息
	return c.SendText(ctx, memberEvent.ChatID, "你好！我是 AI 助手，有什么可以帮助你的吗？")
}

// handleMemberAddedEvent 处理群成员加入事件
func (c *Channel) handleMemberAddedEvent(ctx context.Context, event *Event) error {
	memberEvent, err := ParseChatMemberEvent(event)
	if err != nil {
		return err
	}

	c.logger.Info("新成员加入群聊", "chat_id", memberEvent.ChatID)

	// 发布事件到 MessageBus
	return c.bus.Publish(NewChannelEvent("member_added", memberEvent.ChatID, "", memberEvent))
}

// handleChatDisbandedEvent 处理群解散事件
func (c *Channel) handleChatDisbandedEvent(ctx context.Context, event *Event) error {
	memberEvent, err := ParseChatMemberEvent(event)
	if err != nil {
		return err
	}

	c.logger.Info("群聊已解散", "chat_id", memberEvent.ChatID)

	// 发布事件到 MessageBus
	return c.bus.Publish(NewChannelEvent("chat_disbanded", memberEvent.ChatID, "", nil))
}

// ============ 发送消息 ============

// Send 发送消息（实现 Channel 接口）
func (c *Channel) Send(ctx context.Context, msg channel.OutboundMessage) error {
	return c.SendText(ctx, msg.ChatID, msg.Content)
}

// SendText 发送文本消息
func (c *Channel) SendText(ctx context.Context, chatID, text string) error {
	content := NewTextContent(text)
	return c.sendMessage(ctx, chatID, MsgTypeText, content)
}

// SendPost 发送富文本消息
func (c *Channel) SendPost(ctx context.Context, chatID string, post *PostContent) error {
	content, _ := json.Marshal(post)
	return c.sendMessage(ctx, chatID, MsgTypePost, string(content))
}

// SendImage 发送图片消息
func (c *Channel) SendImage(ctx context.Context, chatID, imageKey string) error {
	content := NewImageContent(imageKey)
	return c.sendMessage(ctx, chatID, MsgTypeImage, content)
}

// SendFile 发送文件消息
func (c *Channel) SendFile(ctx context.Context, chatID, fileKey string) error {
	content := NewFileContent(fileKey)
	return c.sendMessage(ctx, chatID, MsgTypeFile, content)
}

// SendCard 发送卡片消息
func (c *Channel) SendCard(ctx context.Context, chatID string, card *CardMessage) error {
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("序列化卡片失败: %w", err)
	}
	return c.sendMessage(ctx, chatID, MsgTypeInteractive, string(cardJSON))
}

// SendCardWithBuilder 使用构建器发送卡片
func (c *Channel) SendCardWithBuilder(ctx context.Context, chatID string, buildFn func(*CardBuilder)) error {
	builder := NewCardBuilder()
	buildFn(builder)
	return c.SendCard(ctx, chatID, builder.Build())
}

// sendMessage 通用消息发送
func (c *Channel) sendMessage(ctx context.Context, chatID string, msgType MessageType, content string) error {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("获取 token 失败: %w", err)
	}

	payload, _ := json.Marshal(&OutboundMessage{
		ReceiveID:     chatID,
		ReceiveIDType: ReceiveIDTypeChatID,
		MsgType:       msgType,
		Content:       content,
	})

	return doWithRetry(ctx, func() error {
		return c.sendMessageAPI(ctx, token, payload)
	}, DefaultRetryConfig, func(attempt int, err error) {
		c.logger.Warn("发送消息失败，准备重试", "attempt", attempt, "error", err)
	})
}

// sendMessageAPI 调用飞书消息发送 API
func (c *Channel) sendMessageAPI(ctx context.Context, token string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id",
		bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: send message: %w", err)
	}
	defer resp.Body.Close()

	var result MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("feishu: decode response: %w", err)
	}

	if result.Code != 0 {
		err := NewFeishuError(result.Code, result.Msg)
		// Token 过期时清除缓存
		if err.IsTokenError() {
			c.tokenMu.Lock()
			c.accessToken = ""
			c.tokenExpiresAt = time.Time{}
			c.tokenMu.Unlock()
		}
		return err
	}

	return nil
}

// ============ Token 管理 ============

// tokenResponse 飞书 token 响应
type tokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

// getAccessToken 获取（带缓存的）飞书 tenant_access_token
func (c *Channel) getAccessToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	payload, _ := json.Marshal(map[string]string{
		"app_id":     c.config.AppID(),
		"app_secret": c.config.AppSecret(),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("feishu: request token: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("feishu: decode token: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu: token API error: %d %s", result.Code, result.Msg)
	}

	c.tokenMu.Lock()
	c.accessToken = result.TenantAccessToken
	// 提前 60 秒过期，预留刷新余量
	c.tokenExpiresAt = time.Now().Add(time.Duration(result.Expire-60) * time.Second)
	c.tokenMu.Unlock()

	return c.accessToken, nil
}

// ============ 辅助函数 ============

// toSlogLogger 转换 Logger 接口
func toSlogLogger(logger channel.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}

	// 如果已经是 slog.Logger，直接返回
	if slogLogger, ok := logger.(*slog.Logger); ok {
		return slogLogger
	}

	// 否则创建适配器
	return slog.New(&loggerHandler{logger: logger})
}

// loggerHandler slog.Handler 适配器
type loggerHandler struct {
	logger channel.Logger
}

func (h *loggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *loggerHandler) Handle(ctx context.Context, r slog.Record) error {
	msg := r.Message
	args := make([]interface{}, 0, r.NumAttrs()*2)
	r.Attrs(func(a slog.Attr) bool {
		args = append(args, a.Key, a.Value.Any())
		return true
	})

	switch r.Level {
	case slog.LevelDebug:
		h.logger.Debug(msg, args...)
	case slog.LevelInfo:
		h.logger.Info(msg, args...)
	case slog.LevelWarn:
		h.logger.Warn(msg, args...)
	case slog.LevelError:
		h.logger.Error(msg, args...)
	}
	return nil
}

func (h *loggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *loggerHandler) WithGroup(name string) slog.Handler {
	return h
}