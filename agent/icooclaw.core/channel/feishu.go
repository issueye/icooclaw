package channel

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ============ 飞书通道实现 ============

// FeishuChannel 飞书通道
type FeishuChannel struct {
	*BaseChannel
	config FeishuConfig
	bus    MessageBus
	server *http.Server
	logger *slog.Logger

	// 飞书 API 访问 Token 缓存
	accessToken    string
	tokenExpiresAt time.Time
}

// NewFeishuChannel 创建飞书通道
func NewFeishuChannel(cfg FeishuConfig, messageBus MessageBus, logger Logger) *FeishuChannel {
	base := NewBaseChannel("feishu", toSlogLogger(logger))
	return &FeishuChannel{
		BaseChannel: base,
		config:      cfg,
		bus:         messageBus,
		logger:      toSlogLogger(logger),
	}
}

// Start 启动飞书 Webhook 监听服务
func (c *FeishuChannel) Start(ctx context.Context) error {
	if c.config == nil || !c.config.Enabled() {
		c.logger.Info("飞书通道已禁用")
		return nil
	}

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

// Stop 停止飞书通道服务
func (c *FeishuChannel) Stop() error {
	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("飞书通道关闭异常", "error", err)
			return err
		}
	}
	c.SetRunning(false)
	c.logger.Info("飞书通道已停止")
	return nil
}

// ============ 飞书 Webhook 消息结构 ============

// FeishuEventBody 飞书事件请求体
type FeishuEventBody struct {
	// 加密模式下仅有此字段
	Encrypt string `json:"encrypt,omitempty"`

	// 明文模式
	Schema string             `json:"schema,omitempty"`
	Header *FeishuEventHeader `json:"header,omitempty"`
	Event  *FeishuIMEvent     `json:"event,omitempty"`

	// URL Verification（飞书回调验证）
	Challenge string `json:"challenge,omitempty"`
	Token     string `json:"token,omitempty"`
	Type      string `json:"type,omitempty"`
}

// FeishuEventHeader 飞书事件头
type FeishuEventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
	CreateTime string `json:"create_time"`
}

// FeishuIMEvent IM 消息事件
type FeishuIMEvent struct {
	Sender  *FeishuSender  `json:"sender,omitempty"`
	Message *FeishuMessage `json:"message,omitempty"`
}

// FeishuSender 发送者
type FeishuSender struct {
	SenderID *FeishuID `json:"sender_id,omitempty"`
}

// FeishuID 飞书通用 ID
type FeishuID struct {
	OpenID  string `json:"open_id,omitempty"`
	UnionID string `json:"union_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

// FeishuMessage 飞书消息体
type FeishuMessage struct {
	MessageID   string `json:"message_id"`
	ChatID      string `json:"chat_id"`
	ChatType    string `json:"chat_type"`
	MessageType string `json:"message_type"`
	Content     string `json:"content"` // JSON 字符串，如 {"text":"hello"}
}

// feishuTextContent 飞书文本消息内容
type feishuTextContent struct {
	Text string `json:"text"`
}

// handleWebhook 处理飞书 Webhook 事件
func (c *FeishuChannel) handleWebhook(w http.ResponseWriter, r *http.Request) {
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

	// 解密（如配置了 EncryptKey）
	if c.config.EncryptKey() != "" {
		body, err = c.decryptBody(body, c.config.EncryptKey())
		if err != nil {
			c.logger.Error("飞书消息解密失败", "error", err)
			http.Error(w, `{"error":"decrypt failed"}`, http.StatusBadRequest)
			return
		}
	}

	var event FeishuEventBody
	if err := json.Unmarshal(body, &event); err != nil {
		c.logger.Error("飞书消息 JSON 解析失败", "error", err)
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	// 1. URL Verification 挑战响应
	if event.Challenge != "" {
		// 旧版事件格式
		if c.config.VerificationToken() != "" && event.Token != c.config.VerificationToken() {
			http.Error(w, `{"error":"token mismatch"}`, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"challenge": event.Challenge})
		return
	}

	// 2. 处理 IM 消息事件
	if event.Header != nil && event.Header.EventType == "im.message.receive_v1" && event.Event != nil {
		if err := c.handleIMMessage(r.Context(), event); err != nil {
			c.logger.Error("飞书消息处理失败", "error", err)
			http.Error(w, `{"error":"handle message failed"}`, http.StatusInternalServerError)
			return
		}
	} else {
		c.logger.Debug("飞书收到非 IM 消息事件", "event_type", func() string {
			if event.Header != nil {
				return event.Header.EventType
			}
			return "unknown"
		}())
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"msg": "ok"})
}

// handleIMMessage 处理 IM 消息内容并发布到 MessageBus
func (c *FeishuChannel) handleIMMessage(ctx context.Context, event FeishuEventBody) error {
	msg := event.Event.Message
	sender := event.Event.Sender

	if msg == nil {
		return errors.New("event.message is nil")
	}

	// 只处理文字类型消息
	if msg.MessageType != "text" {
		c.logger.Debug("飞书忽略非文字消息", "message_type", msg.MessageType)
		return nil
	}

	// 解析 content JSON: {"text": "..."}
	var textContent feishuTextContent
	if err := json.Unmarshal([]byte(msg.Content), &textContent); err != nil {
		return fmt.Errorf("parse text content: %w", err)
	}

	userID := ""
	if sender != nil && sender.SenderID != nil {
		userID = sender.SenderID.OpenID
	}

	inbound := InboundMessage{
		ID:        msg.MessageID,
		Channel:   "feishu",
		ChatID:    msg.ChatID,
		UserID:    userID,
		Content:   textContent.Text,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"chat_type":    msg.ChatType,
			"message_type": msg.MessageType,
			"event_id":     event.Header.EventID,
		},
	}

	return c.bus.PublishInbound(ctx, inbound)
}

// handleHealth 健康检查
func (c *FeishuChannel) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// ============ 发送消息 ============

// Send 发送消息到飞书
func (c *FeishuChannel) Send(ctx context.Context, msg OutboundMessage) error {
	if msg.ChatID == "" {
		return errors.New("feishu: ChatID is required")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("feishu: get access token: %w", err)
	}

	// 构建文本消息
	content, _ := json.Marshal(map[string]string{"text": msg.Content})
	payload, _ := json.Marshal(map[string]interface{}{
		"receive_id": msg.ChatID,
		"msg_type":   "text",
		"content":    string(content),
	})

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu: send API error: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// ============ 飞书 tenant_access_token 获取 ============

// feishuTokenResponse 飞书 token 响应
type feishuTokenResponse struct {
	Code              int    `json:"code"`
	Msg               string `json:"msg"`
	TenantAccessToken string `json:"tenant_access_token"`
	Expire            int    `json:"expire"`
}

// getAccessToken 获取（带缓存的）飞书 tenant_access_token
func (c *FeishuChannel) getAccessToken(ctx context.Context) (string, error) {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiresAt) {
		return c.accessToken, nil
	}

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

	var result feishuTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("feishu: decode token: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("feishu: token API error: %d %s", result.Code, result.Msg)
	}

	c.accessToken = result.TenantAccessToken
	// 提前 60 秒过期，预留刷新余量
	c.tokenExpiresAt = time.Now().Add(time.Duration(result.Expire-60) * time.Second)

	return c.accessToken, nil
}

// ============ AES 解密 ============

// decryptBody 解密飞书 AES-256-CBC 加密的消息体
// 参考：https://open.feishu.cn/document/server-docs/event-subscription-guide/event-subscription-configure-/encrypt-key-encryption-configuration-case
func (c *FeishuChannel) decryptBody(body []byte, key string) ([]byte, error) {
	// 解包 {"encrypt":"..."}
	var envelope struct {
		Encrypt string `json:"encrypt"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Encrypt == "" {
		// 非加密格式，直接返回
		return body, nil
	}

	// SHA256 生成 32 字节密钥
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

	// 去除 PKCS#7 padding
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
