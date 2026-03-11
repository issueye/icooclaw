package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection.
type Client struct {
	ID        string
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	sessionID string

	manager *Manager
	logger  *slog.Logger

	// State
	connected  atomic.Bool
	lastPing   time.Time
	lastPong   time.Time
	messageSeq atomic.Uint64

	// Configuration
	writeWait      time.Duration
	pongWait       time.Duration
	pingPeriod     time.Duration
	maxMessageSize int64

	mu sync.Mutex
}

// ClientConfig holds configuration for a Client.
type ClientConfig struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	SendBufferSize int
}

// DefaultClientConfig returns the default client configuration.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     (60 * time.Second * 9) / 10,
		MaxMessageSize: 512 * 1024, // 512KB
		SendBufferSize: 256,
	}
}

// NewClient creates a new WebSocket client.
func NewClient(conn *websocket.Conn, userID string, logger *slog.Logger) *Client {
	cfg := DefaultClientConfig()

	return &Client{
		ID:             uuid.New().String(),
		conn:           conn,
		send:           make(chan []byte, cfg.SendBufferSize),
		userID:         userID,
		logger:         logger,
		writeWait:      cfg.WriteWait,
		pongWait:       cfg.PongWait,
		pingPeriod:     cfg.PingPeriod,
		maxMessageSize: cfg.MaxMessageSize,
	}
}

// WithManager sets the manager for the client.
func (c *Client) WithManager(m *Manager) *Client {
	c.manager = m
	return c
}

// WithChatID sets the chat ID for the client.
func (c *Client) WithSessionID(sessionID string) *Client {
	c.sessionID = sessionID
	return c
}

// Run starts the client read/write loops.
func (c *Client) Run(ctx context.Context) {
	c.connected.Store(true)
	defer c.connected.Store(false)
	defer c.Close()

	// Start write pump
	go c.writePump(ctx)

	// Run read pump
	c.readPump(ctx)
}

// readPump pumps messages from the WebSocket connection to the hub.
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.connected.Store(false)
	}()

	// Set read limits and deadlines
	c.conn.SetReadLimit(c.maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))

	// Set pong handler
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		c.lastPong = time.Now()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("unexpected close error", "error", err, "client_id", c.ID)
				}
				return
			}

			c.lastPing = time.Now()
			c.messageSeq.Add(1)

			// Handle message
			c.handleMessage(ctx, messageType, message)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(c.pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			// Close connection gracefully
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles an incoming message.
func (c *Client) handleMessage(ctx context.Context, messageType int, message []byte) {
	if messageType != websocket.TextMessage {
		c.logger.With("name", "【WebSocket】").Warn("消息类型错误，仅支持文本消息", "type", messageType, "client_id", c.ID)
		c.SendError("消息类型错误")
		return
	}

	fmt.Println("收到消息", string(message))

	// Parse message
	var msg ChatMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.With("name", "【WebSocket】").Error("解析消息失败", "error", err, "client_id", c.ID)
		c.SendError("消息格式错误")
		return
	}

	// Set timestamp if not provided
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	// Use client's session ID if not provided in message
	if msg.SessionID == "" {
		msg.SessionID = c.sessionID
	}

	// Validate message
	if msg.Content == "" {
		c.SendError("消息内容不能为空")
		return
	}

	if msg.SessionID == "" {
		c.SendError("会话ID不能为空")
		return
	}

	// Process message based on type
	switch msg.Type {
	case "chat", "":
		if c.manager == nil {
			c.SendError("manager not configured")
			return
		}

		if msg.Stream {
			go c.manager.ProcessStreamMessage(ctx, c, &msg)
		} else {
			go c.manager.ProcessMessage(ctx, c, &msg)
		}

	case "ping":
		c.SendJSON(map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now().Unix(),
		})

	default:
		c.logger.Warn("unknown message type", "type", msg.Type, "client_id", c.ID)
		c.SendError("unknown message type: " + msg.Type)
	}
}

// Send queues a message to be sent to the client.
func (c *Client) Send(message []byte) bool {
	if !c.connected.Load() {
		return false
	}

	select {
	case c.send <- message:
		return true
	default:
		c.logger.With("name", "【WebSocket】").Warn("发送消息队列已满，丢弃消息", "client_id", c.ID)
		return false
	}
}

// SendJSON sends a JSON message to the client.
func (c *Client) SendJSON(v interface{}) bool {
	data, err := json.Marshal(v)
	if err != nil {
		c.logger.Error("failed to marshal json", "error", err, "client_id", c.ID)
		return false
	}
	return c.Send(data)
}

// SendError sends an error message to the client.
func (c *Client) SendError(message string) {
	c.SendJSON(map[string]interface{}{
		"type":      "error",
		"message":   message,
		"timestamp": time.Now().Unix(),
	})
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return nil
	}

	c.connected.Store(false)
	close(c.send)
	return c.conn.Close()
}

// IsConnected returns true if the client is connected.
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// GetStats returns client statistics.
func (c *Client) GetStats() *ClientStats {
	return &ClientStats{
		ID:         c.ID,
		UserID:     c.userID,
		SessionID:  c.sessionID,
		Connected:  c.connected.Load(),
		MessageSeq: c.messageSeq.Load(),
		LastPing:   c.lastPing,
		LastPong:   c.lastPong,
	}
}

// ClientStats represents client statistics.
type ClientStats struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	SessionID  string    `json:"session_id"`
	Connected  bool      `json:"connected"`
	MessageSeq uint64    `json:"message_seq"`
	LastPing   time.Time `json:"last_ping"`
	LastPong   time.Time `json:"last_pong"`
}
