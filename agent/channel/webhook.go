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

	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
)

// WebhookChannel Webhook 通道实现
type WebhookChannel struct {
	*BaseChannel
	config config.ChannelSettings
	bus    *bus.MessageBus
	server *http.Server
	logger *slog.Logger
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
func NewWebhookChannel(cfg config.ChannelSettings, messageBus *bus.MessageBus, logger *slog.Logger) *WebhookChannel {
	base := NewBaseChannel("webhook", logger)

	return &WebhookChannel{
		BaseChannel: base,
		config:      cfg,
		bus:         messageBus,
		logger:      logger,
	}
}

// Start 启动 Webhook 服务
func (c *WebhookChannel) Start(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("Webhook channel is disabled")
		return nil
	}

	host := c.config.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port
	if port == 0 {
		port = 8081
	}

	path := c.config.Extra["path"]
	if path == "" {
		path = "/webhook"
	}

	// 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 创建 HTTP 服务器
	mux := http.NewServeMux()

	// Webhook 端点
	mux.HandleFunc(path, c.handleWebhook)

	// 健康检查端点
	mux.HandleFunc("/health", c.handleHealth)

	// 状态端点
	mux.HandleFunc("/status", c.handleStatus)

	c.server = &http.Server{
		Addr:         host + ":" + strconv.Itoa(port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 启动 HTTP 服务器
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
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 只接受 POST 方法
	if r.Method != http.MethodPost {
		c.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 验证签名（如果配置了 secret）
	secret := c.config.Extra["secret"]
	if secret != "" {
		if !c.verifySignature(r, secret) {
			c.sendError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		c.sendError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// 解析 JSON
	var msg WebhookMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		// 尝试作为纯文本处理
		msg.Content = string(body)
	}

	// 如果没有 content 字段，尝试从其他字段获取
	if msg.Content == "" {
		// 尝试从表单数据获取
		content := r.FormValue("content")
		if content != "" {
			msg.Content = content
		}
	}

	// 如果仍然没有内容
	if msg.Content == "" {
		c.sendError(w, http.StatusBadRequest, "missing content")
		return
	}

	// 提取 ChatID 和 UserID
	if msg.ChatID == "" {
		msg.ChatID = r.FormValue("chat_id")
	}
	if msg.UserID == "" {
		msg.UserID = r.FormValue("user_id")
	}
	if msg.MessageID == "" {
		msg.MessageID = r.FormValue("message_id")
	}

	// 发送到消息总线
	inboundMsg := bus.InboundMessage{
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

	// 发送成功响应
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
	// 获取签名
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		// 也尝试从查询参数获取
		signature = r.URL.Query().Get("signature")
		if signature == "" {
			return false
		}
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	defer r.Body.Close()

	// 计算 HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// 比较签名
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
	// Webhook 通道的发送功能需要额外配置回调 URL
	// 这里暂时返回不支持的错误
	return errors.New("webhook channel does not support sending messages directly, use callback URLs")
}
