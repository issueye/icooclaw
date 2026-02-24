package channel

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"

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
	Type    string          `json:"type"`
	Content string          `json:"content"`
	ChatID  string          `json:"chat_id,omitempty"`
	UserID  string          `json:"user_id,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
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

// WebSocketChannel WebSocket 通道实现
type WebSocketChannel struct {
	*BaseChannel
	config config.ChannelSettings
	hub    *WebSocketHub
	server *http.Server
	bus    *bus.MessageBus
	logger *slog.Logger
}

// NewWebSocketChannel 创建 WebSocket 通道
func NewWebSocketChannel(cfg config.ChannelSettings, messageBus *bus.MessageBus, logger *slog.Logger) *WebSocketChannel {
	base := NewBaseChannel("websocket", logger)
	hub := NewWebSocketHub(logger)

	return &WebSocketChannel{
		BaseChannel: base,
		config:      cfg,
		hub:         hub,
		bus:         messageBus,
		logger:      logger,
	}
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

	// 创建 HTTP 服务器
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc("/ws", c.handleWebSocket)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API 端点
	mux.HandleFunc("/api/message", c.handleMessage)

	c.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 启动 Hub
	go c.hub.Run(ctx)

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

// handleMessage 处理 HTTP 消息发送
func (c *WebSocketChannel) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var msg WebSocketMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, _ := json.Marshal(msg)
		c.hub.Broadcast(data)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
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
