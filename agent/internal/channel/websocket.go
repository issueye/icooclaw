package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/internal/storage"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketMessage WebSocket 消息格式
type WebSocketMessage struct {
	Type     string          `json:"type"`
	Content  string          `json:"content"`
	Thinking string          `json:"thinking,omitempty"`
	ChatID   string          `json:"chat_id,omitempty"`
	UserID   string          `json:"user_id,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// WebSocketClient WebSocket 客户端连接
type WebSocketClient struct {
	ID        string
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *WebSocketHub
	ChatID    string
	UserID    string
	CreatedAt time.Time
}

// WebSocketHub WebSocket 连接管理器
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	logger     *slog.Logger
	mu         sync.RWMutex
}

// NewWebSocketHub 创建 WebSocket Hub
func NewWebSocketHub(logger *slog.Logger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		logger:     logger,
	}
}

// Run 运行 Hub
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("Client connected", "id", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.logger.Info("Client disconnected", "id", client.ID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.clients {
				client.Conn.Close()
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// Register 注册客户端
func (h *WebSocketHub) Register(client *WebSocketClient) {
	h.register <- client
}

// Unregister 注销客户端
func (h *WebSocketHub) Unregister(client *WebSocketClient) {
	h.unregister <- client
}

// Broadcast 广播消息
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// SendToClient 发送消息给指定客户端
func (h *WebSocketHub) SendToClient(clientID, chatID, msgType, content, thinking string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(WebSocketMessage{
		Type:     msgType,
		Content:  content,
		Thinking: thinking,
		ChatID:   chatID,
	})
	if err != nil {
		return err
	}

	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.Send <- data:
				return nil
			default:
				return errors.New("client buffer full")
			}
		}
	}
	return errors.New("client not found")
}

// SendToUser 发送消息给指定用户
func (h *WebSocketHub) SendToUser(userID, message string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(WebSocketMessage{
		Type:    "message",
		Content: message,
		UserID:  userID,
	})
	if err != nil {
		return err
	}

	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
				return nil
			default:
				return errors.New("client buffer full")
			}
		}
	}
	return errors.New("user not found")
}

// Count 获取连接数
func (h *WebSocketHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

type WebSocketChannel struct {
	*BaseChannel
	config  config.ChannelSettings
	hub     *WebSocketHub
	server  *http.Server
	bus     *bus.MessageBus
	storage *storage.Storage
	agent   interface {
		ProcessMessage(ctx context.Context, content string) (string, error)
		Storage() *storage.Storage
		Provider() interface {
			GetName() string
			GetDefaultModel() string
		}
	}
	logger *slog.Logger
}

// NewWebSocketChannel 创建 WebSocket 通道
func NewWebSocketChannel(cfg config.ChannelSettings, messageBus *bus.MessageBus, storage *storage.Storage, logger *slog.Logger) *WebSocketChannel {
	base := NewBaseChannel("websocket", logger)
	hub := NewWebSocketHub(logger)

	return &WebSocketChannel{
		BaseChannel: base,
		config:      cfg,
		hub:         hub,
		bus:         messageBus,
		storage:     storage,
		logger:      logger,
	}
}

// SetAgent 设置 Agent 引用（用于 REST API 直接调用）
func (c *WebSocketChannel) SetAgent(agent interface {
	ProcessMessage(ctx context.Context, content string) (string, error)
	Storage() *storage.Storage
	Provider() interface {
		GetName() string
		GetDefaultModel() string
	}
}) {
	c.agent = agent
}

// Start 启动 WebSocket 服务
func (c *WebSocketChannel) Start(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("WebSocket channel is disabled")
		return nil
	}

	host := c.config.Host
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port
	if port == 0 {
		port = 8080
	}

	addr := host + ":"

	// 创建 chi 路由器
	r := chi.NewRouter()

	// 全局中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS 中间件
	r.Use(c.corsMiddleware)

	// WebSocket 端点
	r.Get("/ws", c.handleWebSocket)

	// REST API v1
	r.Post("/api/v1/chat", c.handleRestChat)
	r.Post("/api/v1/chat/stream", c.handleRestChatStream)
	r.Get("/api/v1/sessions", c.handleRestSessions)
	r.Post("/api/v1/sessions", c.handleRestCreateSession)
	r.Route("/api/v1/sessions/{id}", func(r chi.Router) {
		r.Get("/", c.handleRestSessionDetail)
		r.Get("/messages", c.handleRestSessionDetail)
		r.Delete("/", c.handleRestSessionDetail)
	})
	r.Get("/api/v1/providers", c.handleRestProviders)
	r.Get("/api/v1/skills", c.handleRestSkills)
	r.Route("/api/v1/skills/{id}", func(r chi.Router) {
		r.Get("/", c.handleRestSkillDetail)
		r.Put("/", c.handleRestSkillDetail)
		r.Delete("/", c.handleRestSkillDetail)
	})
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	c.server = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 启动 Hub
	go c.hub.Run(ctx)

	// 启动总线监听
	go c.processOutbound(ctx)

	// 启动 HTTP 服务器
	go func() {
		serveAddr := host + ":" + strconv.Itoa(port)
		c.server.Addr = serveAddr
		c.logger.Info("WebSocket server starting", "host", host, "port", port)
		if err := c.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.logger.Error("WebSocket server error", "error", err)
		}
	}()

	c.SetRunning(true)
	c.logger.Info("WebSocket channel started", "host", host, "port", port)
	return nil
}

// handleWebSocket 处理 WebSocket 连接
func (c *WebSocketChannel) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	clientID := generateClientID()

	client := &WebSocketClient{
		ID:        clientID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       c.hub,
		CreatedAt: time.Now(),
	}

	c.hub.Register(client)

	go c.writePump(client)
	go c.readPump(client)
}

// readPump 读取客户端消息
func (c *WebSocketChannel) readPump(client *WebSocketClient) {
	defer func() {
		c.hub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(512 * 1024)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket read error", "error", err)
			}
			break
		}

		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.logger.Error("Failed to parse message", "error", err)
			continue
		}

		if wsMsg.ChatID != "" {
			client.ChatID = wsMsg.ChatID
		}
		if wsMsg.UserID != "" {
			client.UserID = wsMsg.UserID
		}

		// 发送到消息总线
		inboundMsg := bus.InboundMessage{
			Channel:  "websocket",
			ChatID:   client.ChatID,
			UserID:   client.UserID,
			Content:  wsMsg.Content,
			Metadata: map[string]interface{}{"client_id": client.ID},
		}

		ctx := context.Background()
		if err := c.bus.PublishInbound(ctx, inboundMsg); err != nil {
			c.logger.Error("Failed to publish message", "error", err)
		}
	}
}

// writePump 写入客户端消息
func (c *WebSocketChannel) writePump(client *WebSocketClient) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processOutbound 监听总线并将消息路由到客户端
func (c *WebSocketChannel) processOutbound(ctx context.Context) {
	outbound := c.bus.SubscribeOutbound("websocket")
	defer c.bus.UnsubscribeOutbound("websocket")

	for {
		select {
		case msg := <-outbound:
			clientID, _ := msg.Metadata["client_id"].(string)
			if clientID != "" {
				// 精准推送
				c.hub.SendToClient(clientID, msg.ChatID, msg.Type, msg.Content, msg.Thinking)
			} else {
				// 广播（回退逻辑）
				wsMsg := WebSocketMessage{
					Type:     msg.Type,
					Content:  msg.Content,
					Thinking: msg.Thinking,
					ChatID:   msg.ChatID,
				}
				data, _ := json.Marshal(wsMsg)
				c.hub.Broadcast(data)
			}
		case <-ctx.Done():
			return
		}
	}
}

// corsWrapper CORS 包装器
// corsMiddleware CORS 中间件
func (c *WebSocketChannel) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// REST API Handlers

func (c *WebSocketChannel) handleRestChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if c.agent == nil {
		http.Error(w, "Agent not configured", http.StatusInternalServerError)
		return
	}

	// 这种模式下我们直接调用 ProcessMessage
	// 注意：ProcessMessage 内部应该已经处理了持久化
	resp, err := c.agent.ProcessMessage(context.Background(), req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}

func (c *WebSocketChannel) handleRestChatStream(w http.ResponseWriter, r *http.Request) {
	// SSE 需要特殊的 Header
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
		UserID  string `json:"user_id"`
	}
	// 注意：如果是通过 URL 参数传参，需要特殊处理。这里假设是 POST 请求
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// 创建一个临时的 channel 来接收总线消息
	// 或者直接在 handleMessage 逻辑里注入这个 w 作为输出
	// 为了简单，我们发布一个特殊的 InboundMessage，Metadata 带上 rest_stream: true
	inboundMsg := bus.InboundMessage{
		Channel: "rest",
		ChatID:  req.ChatID,
		UserID:  req.UserID,
		Content: req.Content,
		Metadata: map[string]interface{}{
			"rest_stream": true,
			"stream_id":   generateClientID(),
		},
	}

	// 订阅这个 stream_id 的 outbound 消息
	streamID := inboundMsg.Metadata["stream_id"].(string)
	outbound := c.bus.SubscribeOutbound("rest_" + streamID)
	defer c.bus.UnsubscribeOutbound("rest_" + streamID)

	c.bus.PublishInbound(r.Context(), inboundMsg)

	for {
		select {
		case msg := <-outbound:
			if msg.Type == "chunk" {
				fmt.Fprintf(w, "data: %s\n\n", msg.Content)
				flusher.Flush()
			} else if msg.Type == "chunk_end" {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
		case <-r.Context().Done():
			return
		case <-time.After(60 * time.Second):
			return
		}
	}
}

func (c *WebSocketChannel) handleRestSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := c.storage.GetSessions("", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (c *WebSocketChannel) handleRestCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Metadata string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chatID := fmt.Sprintf("chat_%d", time.Now().UnixNano())
	session, err := c.storage.GetOrCreateSession("websocket", chatID, req.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Metadata != "" {
		c.storage.UpdateSessionMetadata(session.ID, req.Metadata)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (c *WebSocketChannel) handleRestSessionDetail(w http.ResponseWriter, r *http.Request) {
	// 简单的路径解析 /api/v1/sessions/:id/messages
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/"), "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	sessionID := parts[0]

	if r.Method == http.MethodDelete {
		if err := c.storage.DeleteSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	if len(parts) > 1 && parts[1] == "messages" {
		messages, err := c.storage.GetMessages(sessionID, 100, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func (c *WebSocketChannel) handleRestProviders(w http.ResponseWriter, r *http.Request) {
	if c.agent == nil {
		http.Error(w, "Agent not configured", http.StatusInternalServerError)
		return
	}
	info := map[string]interface{}{
		"provider":      c.agent.Provider().GetName(),
		"default_model": c.agent.Provider().GetDefaultModel(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleRestSkills 处理技能列表
func (c *WebSocketChannel) handleRestSkills(w http.ResponseWriter, r *http.Request) {
	if c.storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		// 获取所有技能
		skills, err := c.storage.GetAllSkills()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"skills": skills,
		})
	case "POST":
		// 创建新技能
		var skill storage.Skill
		if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		skill.Source = "user"
		if err := c.storage.CreateSkill(&skill); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skill)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRestSkillDetail 处理单个技能
func (c *WebSocketChannel) handleRestSkillDetail(w http.ResponseWriter, r *http.Request) {
	if c.storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	// 解析 ID
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/skills/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid skill ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		skill, err := c.storage.GetSkillByID(uint(id))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skill)
	case "PUT":
		var skill storage.Skill
		if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		skill.ID = uint(id)
		if err := c.storage.UpdateSkill(&skill); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skill)
	case "DELETE":
		if err := c.storage.DeleteSkill(uint(id)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"deleted"}`))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *WebSocketChannel) handleMessage(w http.ResponseWriter, r *http.Request) {
	// 保持对旧接口的兼容或按需修改
	http.Error(w, "Please use /api/v1/chat", http.StatusMovedPermanently)
}

// Stop 停止 WebSocket 服务
func (c *WebSocketChannel) Stop() error {
	if c.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("WebSocket server shutdown error", "error", err)
			return err
		}
	}

	c.SetRunning(false)
	c.logger.Info("WebSocket channel stopped")
	return nil
}

// Send 发送消息
func (c *WebSocketChannel) Send(ctx context.Context, msg OutboundMessage) error {
	wsMsg := WebSocketMessage{
		Type:    "message",
		Content: msg.Content,
		ChatID:  msg.ChatID,
	}

	data, err := json.Marshal(wsMsg)
	if err != nil {
		return err
	}

	c.hub.Broadcast(data)
	return nil
}

// generateClientID 生成客户端 ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[(now+int64(i))%int64(len(letters))]
	}
	return string(b)
}
