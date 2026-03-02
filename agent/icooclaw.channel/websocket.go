package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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
	logger     Logger
	mu         sync.RWMutex
}

// NewWebSocketHub 创建 WebSocket Hub
func NewWebSocketHub(logger Logger) *WebSocketHub {
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

// Count 获取连接数
func (h *WebSocketHub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WebSocketChannel WebSocket 通道实现
type WebSocketChannel struct {
	*BaseChannel
	config  WebSocketConfig
	hub     *WebSocketHub
	server  *http.Server
	bus     MessageBus
	storage StorageReader
	agent   AgentInterface
	logger  Logger
}

// AgentInterface Agent 接口
type AgentInterface interface {
	ProcessMessage(ctx context.Context, content string) (string, error)
	GetProvider() ProviderInterface
}

// ProviderInterface Provider 接口
type ProviderInterface interface {
	GetName() string
	GetDefaultModel() string
}

// NewWebSocketChannel 创建 WebSocket 通道
func NewWebSocketChannel(cfg WebSocketConfig, messageBus MessageBus, storage interface{}, logger Logger) *WebSocketChannel {
	base := NewBaseChannel("websocket", toSlogLogger(logger))
	hub := NewWebSocketHub(logger)

	var storageReader StorageReader
	if storage != nil {
		if sr, ok := storage.(StorageReader); ok {
			storageReader = sr
		}
	}

	return &WebSocketChannel{
		BaseChannel: base,
		config:      cfg,
		hub:         hub,
		bus:         messageBus,
		storage:     storageReader,
		logger:      logger,
	}
}

// SetAgent 设置 Agent 引用
func (c *WebSocketChannel) SetAgent(agent AgentInterface) {
	c.agent = agent
}

// Start 启动 WebSocket 服务
func (c *WebSocketChannel) Start(ctx context.Context) error {
	if c.config == nil || !c.config.Enabled() {
		c.logger.Info("WebSocket channel is disabled")
		return nil
	}

	host := c.config.Host()
	if host == "" {
		host = "0.0.0.0"
	}
	port := c.config.Port()
	if port == 0 {
		port = 8080
	}

	addr := host + ":" + strconv.Itoa(port)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(c.corsMiddleware)

	r.Get("/ws", c.handleWebSocket)

	r.Post("/api/v1/chat", c.handleRestChat)
	r.Post("/api/v1/chat/stream", c.handleRestChatStream)
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

	go c.hub.Run(ctx)

	go c.processOutbound(ctx)

	go func() {
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

		inboundMsg := InboundMessage{
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
	_ = ctx
}

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

	resp, err := c.agent.ProcessMessage(context.Background(), req.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}

func (c *WebSocketChannel) handleRestChatStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var req struct {
		Content string `json:"content"`
		ChatID  string `json:"chat_id"`
		UserID  string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	_ = flusher
}

func (c *WebSocketChannel) handleRestSessions(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (c *WebSocketChannel) handleRestCreateSession(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (c *WebSocketChannel) handleRestSessionDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (c *WebSocketChannel) handleRestProviders(w http.ResponseWriter, r *http.Request) {
	if c.agent == nil {
		http.Error(w, "Agent not configured", http.StatusInternalServerError)
		return
	}
	info := map[string]interface{}{
		"provider":      c.agent.GetProvider().GetName(),
		"default_model": c.agent.GetProvider().GetDefaultModel(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (c *WebSocketChannel) handleRestSkills(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func (c *WebSocketChannel) handleRestSkillDetail(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
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
	c.hub = nil
	c.logger.Info("WebSocket channel stopped")
	return nil
}

// Send 发送消息
func (c *WebSocketChannel) Send(ctx context.Context, msg OutboundMessage) error {
	data, err := json.Marshal(WebSocketMessage{
		Type:    "message",
		Content: msg.Content,
		ChatID:  msg.ChatID,
	})
	if err != nil {
		return err
	}

	c.hub.Broadcast(data)
	return nil
}

func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
