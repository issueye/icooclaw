package channel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// WebhookChannel Webhook 通道实现
type WebhookChannel struct {
	*BaseChannel
	config WebhookConfig
	bus    MessageBus
	server *http.Server
	logger Logger
}

// WebhookMessage Webhook 消息格式
type WebhookMessage struct {
	Content   string                 `json:"content"`
	ChatID    string                 `json:"chat_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	MessageID string                 `json:"message_id,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// WebhookResponse Webhook 响应格式
type WebhookResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewWebhookChannel 创建 Webhook 通道
func NewWebhookChannel(cfg WebhookConfig, messageBus MessageBus, logger Logger) *WebhookChannel {
	base := NewBaseChannel("webhook", toSlogLogger(logger))

	return &WebhookChannel{
		BaseChannel: base,
		config:      cfg,
		bus:         messageBus,
		logger:      logger,
	}
}

// toSlogLogger 将 Logger 接口转换为 *slog.Logger
func toSlogLogger(logger Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	if l, ok := logger.(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Start 启动 Webhook 服务
func (c *WebhookChannel) Start(ctx context.Context) error {
	if c.config == nil || !c.config.Enabled() {
		c.logger.Info("Webhook channel is disabled")
		return nil
	}

	host := c.config.Host()
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port()
	if port == 0 {
		port = 8081
	}

	path := c.config.Path()
	if path == "" {
		path = "/webhook"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	mux := http.NewServeMux()

	mux.HandleFunc(path, c.handleWebhook)
	mux.HandleFunc("/health", c.handleHealth)
	mux.HandleFunc("/status", c.handleStatus)

	c.server = &http.Server{
		Addr:         host + ":" + strconv.Itoa(port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		c.logger.Info("Webhook server starting", "host", host, "port", port, "path", path)
		if err := c.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.logger.Error("Webhook server error", "error", err)
		}
	}()

	c.SetRunning(true)
	c.logger.Info("Webhook channel started", "host", host, "port", port, "path", path)
	return nil
}

// handleWebhook 处理 Webhook 请求
func (c *WebhookChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		c.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	secret := c.config.Secret()
	if secret != "" {
		if !c.verifySignature(r, secret) {
			c.sendError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.sendError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var msg WebhookMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		msg.Content = string(body)
	}

	if msg.Content == "" {
		content := r.FormValue("content")
		if content != "" {
			msg.Content = content
		}
	}

	if msg.Content == "" {
		c.sendError(w, http.StatusBadRequest, "missing content")
		return
	}

	if msg.ChatID == "" {
		msg.ChatID = r.FormValue("chat_id")
	}
	if msg.UserID == "" {
		msg.UserID = r.FormValue("user_id")
	}
	if msg.MessageID == "" {
		msg.MessageID = r.FormValue("message_id")
	}

	inboundMsg := InboundMessage{
		Channel:   "webhook",
		ChatID:    msg.ChatID,
		UserID:    msg.UserID,
		Content:   msg.Content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"extra":      msg.Extra,
			"message_id": msg.MessageID,
		},
	}

	if err := c.bus.PublishInbound(r.Context(), inboundMsg); err != nil {
		c.logger.Error("Failed to publish message", "error", err)
		c.sendError(w, http.StatusInternalServerError, "failed to process message")
		return
	}

	c.sendSuccess(w, WebhookResponse{
		Success: true,
		Message: "Message received",
		Data: map[string]interface{}{
			"message_id": msg.MessageID,
		},
	})
}

// verifySignature 验证请求签名
func (c *WebhookChannel) verifySignature(r *http.Request, secret string) bool {
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		signature = r.URL.Query().Get("signature")
		if signature == "" {
			return false
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	defer r.Body.Close()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// handleHealth 健康检查
func (c *WebhookChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleStatus 状态检查
func (c *WebhookChannel) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "running",
		"channel": "webhook",
		"running": c.IsRunning(),
	})
}

// sendSuccess 发送成功响应
func (c *WebhookChannel) sendSuccess(w http.ResponseWriter, resp WebhookResponse) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// sendError 发送错误响应
func (c *WebhookChannel) sendError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(WebhookResponse{
		Success: false,
		Message: message,
	})
}

// Stop 停止 Webhook 服务
func (c *WebhookChannel) Stop() error {
	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("Webhook server shutdown error", "error", err)
			return err
		}
	}

	c.SetRunning(false)
	c.logger.Info("Webhook channel stopped")
	return nil
}

// Send 发送消息
func (c *WebhookChannel) Send(ctx context.Context, msg OutboundMessage) error {
	return errors.New("webhook channel does not support sending messages directly, use callback URLs")
}
