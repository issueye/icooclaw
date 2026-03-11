// Package websocket provides WebSocket connection management for the gateway.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"

	"github.com/gorilla/websocket"
)

// Manager manages WebSocket connections and message routing.
type Manager struct {
	hub           *Hub
	bus           *bus.MessageBus
	agentLoop     *agent.Loop
	agentRegistry *agent.AgentRegistry

	// Configuration
	maxConcurrent int
	authenticate  func(r *http.Request) (string, bool)

	// State
	connections atomic.Int64
	running     atomic.Bool

	logger *slog.Logger

	// Upgrader
	upgrader websocket.Upgrader

	mu sync.RWMutex
}

// ManagerConfig holds configuration for the Manager.
type ManagerConfig struct {
	MaxConcurrent   int
	Authenticate    func(r *http.Request) (string, bool)
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultManagerConfig returns the default manager configuration.
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxConcurrent:   100,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// NewManager creates a new WebSocket manager.
func NewManager(cfg *ManagerConfig, logger *slog.Logger) *Manager {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		hub:           NewHub(logger),
		maxConcurrent: cfg.MaxConcurrent,
		authenticate:  cfg.Authenticate,
		logger:        logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	return m
}

// WithBus sets the message bus.
func (m *Manager) WithBus(b *bus.MessageBus) *Manager {
	m.bus = b
	return m
}

// WithAgentLoop sets the agent loop.
func (m *Manager) WithAgentLoop(l *agent.Loop) *Manager {
	m.agentLoop = l
	return m
}

// WithAgentRegistry sets the agent registry.
func (m *Manager) WithAgentRegistry(r *agent.AgentRegistry) *Manager {
	m.agentRegistry = r
	return m
}

// HandleWebSocket handles WebSocket connection upgrade and management.
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Check concurrent limit
	if int(m.connections.Load()) >= m.maxConcurrent {
		m.logger.With("name", "【网关服务】").Error("已达到最大并发连接数",
			"max_concurrent", m.maxConcurrent,
			"current_connections", int(m.connections.Load()))
		http.Error(w, "【网关服务】已达到最大并发连接数", http.StatusServiceUnavailable)
		return
	}

	// Authenticate if required
	var userID string
	if m.authenticate != nil {
		var ok bool
		userID, ok = m.authenticate(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	} else {
		userID = "anonymous"
	}

	// Upgrade connection
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("failed to upgrade websocket", "error", err)
		return
	}

	m.connections.Add(1)
	defer m.connections.Add(-1)

	// Create client
	client := NewClient(conn, userID, m.logger)
	client.WithManager(m)

	// Register with hub
	m.hub.Register(client)
	defer m.hub.Unregister(client)

	m.logger.With("name", "【网关服务】").Info("WebSocket客户端连接成功",
		"user_id", userID,
		"client_id", client.ID,
		"total_connections", m.connections.Load())

	// Run client
	client.Run(r.Context())
}

// HandleWebSocketWithChatID handles WebSocket connection with a specific chat ID.
func (m *Manager) HandleWebSocketWithChatID(w http.ResponseWriter, r *http.Request, chatID string) {
	// Check concurrent limit
	if int(m.connections.Load()) >= m.maxConcurrent {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	// Authenticate if required
	var userID string
	if m.authenticate != nil {
		var ok bool
		userID, ok = m.authenticate(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	} else {
		userID = "anonymous"
	}

	// Upgrade connection
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("failed to upgrade websocket", "error", err)
		return
	}

	m.connections.Add(1)
	defer m.connections.Add(-1)

	// Create client with chat ID
	client := NewClient(conn, userID, m.logger)
	client.WithManager(m)
	client.chatID = chatID

	// Register with hub
	m.hub.Register(client)
	defer m.hub.Unregister(client)

	m.logger.With("name", "【网关服务】").Info("WebSocket客户端连接成功",
		"user_id", userID,
		"client_id", client.ID,
		"chat_id", chatID,
		"total_connections", m.connections.Load())

	// Run client
	client.Run(r.Context())
}

// Broadcast sends a message to all connected clients.
func (m *Manager) Broadcast(message []byte) {
	m.hub.Broadcast(message)
}

// BroadcastTo sends a message to a specific client.
func (m *Manager) BroadcastTo(clientID string, message []byte) {
	m.hub.BroadcastTo(clientID, message)
}

// GetQueueStatus returns the current queue status.
func (m *Manager) GetQueueStatus() *QueueStatus {
	return &QueueStatus{
		Connections:   int(m.connections.Load()),
		MaxConcurrent: m.maxConcurrent,
	}
}

// SetMaxConcurrent sets the maximum concurrent connections.
func (m *Manager) SetMaxConcurrent(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxConcurrent = max
}

// GetConnectionCount returns the current connection count.
func (m *Manager) GetConnectionCount() int {
	return int(m.connections.Load())
}

// Run starts the manager (starts the hub).
func (m *Manager) Run(ctx context.Context) error {
	m.running.Store(true)
	defer m.running.Store(false)

	m.logger.Info("【WebSocket】管理器已启动")

	// Start hub
	go m.hub.Run(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	m.logger.Info("【WebSocket】管理器已停止")
	return ctx.Err()
}

// Stop stops the manager.
func (m *Manager) Stop() {
	m.running.Store(false)
}

// IsRunning returns true if the manager is running.
func (m *Manager) IsRunning() bool {
	return m.running.Load()
}

// ProcessMessage processes an incoming chat message.
func (m *Manager) ProcessMessage(ctx context.Context, client *Client, msg *ChatMessage) error {
	m.logger.Debug("processing message",
		"client_id", client.ID,
		"chat_id", msg.ChatID,
		"content_length", len(msg.Content))

	// If we have an agent loop, process the message
	if m.agentLoop != nil {
		// Process directly and get response
		response, err := m.agentLoop.ProcessDirectWithChannel(
			ctx,
			msg.Content,
			msg.ChatID,
			consts.WEBSOCKET,
			msg.ChatID,
		)
		if err != nil {
			return err
		}

		// Send response back to client
		resp := &ChatResponse{
			Type:      "response",
			ChatID:    msg.ChatID,
			Content:   response,
			Timestamp: time.Now().Unix(),
		}

		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		client.Send(data)
		return nil
	}

	// If we have a bus, publish the message
	if m.bus != nil {
		inbound := bus.InboundMessage{
			Channel:   consts.WEBSOCKET,
			ChatID:    msg.ChatID,
			Sender:    bus.SenderInfo{ID: client.userID, Name: client.userID},
			Text:      msg.Content,
			Timestamp: time.Now(),
		}

		return m.bus.PublishInbound(ctx, inbound)
	}

	return nil
}

// ProcessStreamMessage processes a message with streaming response.
func (m *Manager) ProcessStreamMessage(ctx context.Context, client *Client, msg *ChatMessage) error {
	m.logger.Debug("processing stream message",
		"client_id", client.ID,
		"chat_id", msg.ChatID)

	// Send start event
	client.SendJSON(&StreamEvent{
		Type:   "stream_start",
		ChatID: msg.ChatID,
	})

	// For now, use the same logic as ProcessMessage
	// In a real implementation, this would use streaming LLM responses
	err := m.ProcessMessage(ctx, client, msg)
	if err != nil {
		// Send error event
		client.SendJSON(&StreamEvent{
			Type:  "stream_error",
			Error: err.Error(),
		})
		return err
	}

	// Send end event
	client.SendJSON(&StreamEvent{
		Type:   "stream_end",
		ChatID: msg.ChatID,
	})

	return nil
}

// QueueStatus represents the queue status.
type QueueStatus struct {
	Connections   int `json:"connections"`
	MaxConcurrent int `json:"max_concurrent"`
}

// ChatMessage represents an incoming chat message.
type ChatMessage struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	Content   string `json:"content"`
	Stream    bool   `json:"stream,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Type      string `json:"type"`
	ChatID    string `json:"chat_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}
