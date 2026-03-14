// Package websocket provides WebSocket connection management for the gateway.
package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"icooclaw/pkg/agent"
	"icooclaw/pkg/agent/react"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Manager manages WebSocket connections and message routing.
type Manager struct {
	hub          *Hub
	bus          *bus.MessageBus
	agentManager *agent.AgentManager

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

func (m *Manager) WithAgentManager(am *agent.AgentManager) *Manager {
	m.agentManager = am
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

	// Create client with auto-generated session ID
	client := NewClient(conn, userID, m.logger)
	client.WithManager(m)
	client.WithSessionID(uuid.New().String()) // 自动生成 SessionID

	// Register with hub
	m.hub.Register(client)
	defer m.hub.Unregister(client)

	m.logger.With("name", "【网关服务】").Info("WebSocket客户端连接成功",
		"user_id", userID,
		"client_id", client.ID,
		"session_id", client.sessionID,
		"total_connections", m.connections.Load())

	// Run client
	client.Run(r.Context())
}

// HandleWebSocketWithSessionID handles WebSocket connection with a specific session ID.
func (m *Manager) HandleWebSocketWithSessionID(w http.ResponseWriter, r *http.Request, sessionID string) {
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
	client.WithSessionID(sessionID)

	// Register with hub
	m.hub.Register(client)
	defer m.hub.Unregister(client)

	m.logger.With("name", "【网关服务】").Info("WebSocket客户端连接成功",
		"user_id", userID,
		"client_id", client.ID,
		"session_id", sessionID,
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
	m.logger.With("name", "【网关服务】").Debug("【WebSocket】处理消息",
		"client_id", client.ID,
		"session_id", msg.SessionID,
		"content_length", len(msg.Content))

	// 发送错误响应消息
	sendErrorResponse := func(errMsg string) {
		client.SendJSON(map[string]any{
			"type":      "error",
			"error":     map[string]string{"message": errMsg},
			"timestamp": time.Now().Unix(),
		})
	}

	// 如果有智能体管理器，直接处理消息
	if m.agentManager != nil {
		// 直接处理消息
		inbound := bus.InboundMessage{
			Channel:   consts.WEBSOCKET,
			SessionID: msg.SessionID,
			Sender:    bus.SenderInfo{ID: client.userID, Name: client.userID},
			Text:      msg.Content,
			Timestamp: time.Now(),
		}
		// 运行智能体
		finallyContent, err := m.agentManager.RunAgent(inbound)
		if err != nil {
			return err
		}

		// 发送 chunk 消息
		data := map[string]any{
			"content": finallyContent,
		}
		client.SendJSON(map[string]any{
			"type":      "chunk",
			"data":      data,
			"timestamp": time.Now().Unix(),
		})

		// 发送结束消息
		client.SendJSON(map[string]any{
			"type":      "end",
			"timestamp": time.Now().Unix(),
		})
		return nil
	}

	// 没有智能体管理器，发送错误响应消息
	sendErrorResponse("服务未配置：缺少智能体或消息总线")
	return nil
}

// ProcessStreamMessage 处理流式消息
func (m *Manager) ProcessStreamMessage(ctx context.Context, client *Client, msg *ChatMessage) error {
	m.logger.With("name", "【网关服务】").Debug("【WebSocket】处理流式消息",
		"client_id", client.ID,
		"session_id", msg.SessionID)

	// Helper function to send stream error event
	sendStreamError := func(errMsg string) {
		client.SendJSON(map[string]any{
			"type":      "error",
			"error":     map[string]string{"message": errMsg},
			"timestamp": time.Now().Unix(),
		})
	}

	if m.agentManager == nil {
		sendStreamError("服务未配置：缺少智能体管理器")
		return nil
	}

	inbound := bus.InboundMessage{
		Channel:   consts.WEBSOCKET,
		SessionID: msg.SessionID,
		Sender:    bus.SenderInfo{ID: client.userID, Name: client.userID},
		Text:      msg.Content,
		Timestamp: time.Now(),
	}

	// 运行智能体流式处理
	err := m.agentManager.RunAgentStream(inbound, func(chunk react.StreamChunk) error {
		if chunk.Content != "" || chunk.Reasoning != "" {
			data := map[string]interface{}{
				"content": chunk.Content,
			}
			if chunk.Reasoning != "" {
				data["reasoning"] = chunk.Reasoning
			}
			client.SendJSON(map[string]interface{}{
				"type":      "chunk",
				"data":      data,
				"timestamp": time.Now().Unix(),
			})
		}

		// Send end message when done
		if chunk.Done {
			client.SendJSON(map[string]interface{}{
				"type":      "end",
				"timestamp": time.Now().Unix(),
			})
		}

		return nil
	})

	if err != nil {
		m.logger.With("name", "【网关服务】").Error("流式处理消息失败",
			"error", err,
			"client_id", client.ID,
			"session_id", msg.SessionID)
		sendStreamError("处理消息失败: " + err.Error())
		return err
	}

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
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Stream    bool   `json:"stream,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// StreamEvent represents a streaming event.
type StreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}
