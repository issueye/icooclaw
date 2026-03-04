package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"icooclaw.core/bus"
	"icooclaw.core/storage"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(ctx context.Context, sessionID uint, content string, clientID string) error
}

// HandlerFunc 消息处理函数
type HandlerFunc func(ctx context.Context, sessionID uint, content string, clientID string) error

// HandleMessage 实现 MessageHandler 接口
func (hf HandlerFunc) HandleMessage(ctx context.Context, sessionID uint, content string, clientID string) error {
	return hf(ctx, sessionID, content, clientID)
}

// Manager WebSocket 连接管理器
type Manager struct {
	connections map[string]*Connection
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.RWMutex
	logger      *slog.Logger
	storage     *storage.Storage
	bus         *bus.MessageBus
	handler     MessageHandler
	ctx         context.Context
	cancel      context.CancelFunc
}

// ManagerOption 管理器选项
type ManagerOption func(*Manager)

// NewManager 创建 WebSocket 管理器
func NewManager(storage *storage.Storage, logger *slog.Logger, opts ...ManagerOption) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		connections: make(map[string]*Connection),
		register:    make(chan *Connection, 100),
		unregister:  make(chan *Connection, 100),
		logger:      logger,
		storage:     storage,
		bus:         bus.NewMessageBus(1000),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 应用选项
	for _, opt := range opts {
		opt(m)
	}

	go m.run()
	go m.runOutboundHandler()

	return m
}

// WithHandler 设置消息处理器
func WithHandler(handler MessageHandler) ManagerOption {
	return func(m *Manager) {
		m.handler = handler
	}
}

// WithBus 设置消息总线
func WithBus(b *bus.MessageBus) ManagerOption {
	return func(m *Manager) {
		m.bus = b
	}
}

// SetHandler 设置消息处理器
func (m *Manager) SetHandler(handler MessageHandler) {
	m.handler = handler
}

// run 运行管理器
func (m *Manager) run() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case conn := <-m.register:
			m.mu.Lock()
			m.connections[conn.ID()] = conn
			m.mu.Unlock()
			m.logger.Info("WebSocket 连接已注册", "connection_id", conn.ID())

		case conn := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.connections[conn.ID()]; ok {
				delete(m.connections, conn.ID())
				conn.Close()
			}
			m.mu.Unlock()
			m.logger.Info("WebSocket 连接已注销", "connection_id", conn.ID())
		}
	}
}

// runOutboundHandler 处理出站消息
func (m *Manager) runOutboundHandler() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case msg := <-m.bus.OutboundChannel():
			m.broadcastOutbound(msg)
		}
	}
}

// broadcastOutbound 广播出站消息到对应的 WebSocket 连接
func (m *Manager) broadcastOutbound(msg bus.OutboundMessage) {
	clientID, ok := msg.Metadata["client_id"].(string)
	if !ok {
		return
	}

	m.mu.RLock()
	conn, ok := m.connections[clientID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	var wsMsg *Message
	switch msg.Type {
	case bus.MessageTypeChunk:
		wsMsg = &Message{
			Type:      MessageTypeChunk,
			SessionID: conn.SessionID(),
			Data: &ChunkData{
				SessionID: conn.SessionID(),
				Content:   msg.Content,
			},
			Timestamp: time.Now(),
		}
	case bus.MessageTypeThinking:
		wsMsg = &Message{
			Type:      MessageTypeThinking,
			SessionID: conn.SessionID(),
			Data: &ThinkingData{
				SessionID: conn.SessionID(),
				Content:   msg.Thinking,
			},
			Timestamp: time.Now(),
		}
	case bus.MessageTypeToolCall:
		wsMsg = &Message{
			Type:      MessageTypeToolCall,
			SessionID: conn.SessionID(),
			Data: &ToolCallData{
				SessionID:  conn.SessionID(),
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				Arguments:  msg.Arguments,
			},
			Timestamp: time.Now(),
		}
	case bus.MessageTypeToolResult:
		wsMsg = &Message{
			Type:      MessageTypeToolResult,
			SessionID: conn.SessionID(),
			Data: &ToolResultData{
				SessionID:  conn.SessionID(),
				ToolCallID: msg.ToolCallID,
				Result:     msg.Content,
			},
			Timestamp: time.Now(),
		}
	case bus.MessageTypeEnd:
		wsMsg = &Message{
			Type:      MessageTypeEnd,
			SessionID: conn.SessionID(),
			Data: &EndData{
				SessionID: conn.SessionID(),
			},
			Timestamp: time.Now(),
		}
	case bus.MessageTypeError:
		wsMsg = &Message{
			Type:      MessageTypeError,
			SessionID: conn.SessionID(),
			Error: &ErrorInfo{
				Code:    500,
				Message: msg.Error,
			},
			Timestamp: time.Now(),
		}
	}

	if wsMsg != nil {
		conn.SendMessage(wsMsg)
	}
}

// ServeHTTP 处理 WebSocket 升级请求
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("WebSocket 升级失败", "error", err)
		return
	}

	connectionID := uuid.New().String()
	connection := NewConnection(connectionID, conn, m, m.logger)

	m.register <- connection

	// 启动读写协程
	go connection.WritePump()
	connection.ReadPump()
}

// HandleWebSocket 处理 WebSocket 连接 (用于路由注册)
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	m.ServeHTTP(w, r)
}

// handleClientMessage 处理客户端消息
func (m *Manager) handleClientMessage(conn *Connection, msg *Message) {
	switch msg.Type {
	case MessageTypeCreateSession:
		m.handleCreateSession(conn, msg)
	case MessageTypeChat:
		m.handleChat(conn, msg)
	case MessageTypePing:
		m.handlePing(conn, msg)
	default:
		conn.SendMessage(NewErrorMessage(conn.SessionID(), 400, fmt.Sprintf("未知的消息类型: %s", msg.Type)))
	}
}

// handleCreateSession 处理创建会话请求
func (m *Manager) handleCreateSession(conn *Connection, msg *Message) {
	data, err := json.Marshal(msg.Data)
	if err != nil {
		conn.SendMessage(NewErrorMessage(0, 400, "无效的请求数据"))
		return
	}

	var req CreateSessionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		conn.SendMessage(NewErrorMessage(0, 400, "无效的请求数据格式"))
		return
	}

	// 设置默认渠道
	if req.Channel == "" {
		req.Channel = "websocket"
	}

	// 生成唯一的 ChatID (使用连接ID)
	chatID := fmt.Sprintf("ws-%s", conn.ID())

	// 创建会话
	session, err := m.storage.Session().GetOrCreateSession(req.Channel, chatID, req.UserID)
	if err != nil {
		conn.SendMessage(NewErrorMessage(0, 500, "创建会话失败"))
		return
	}

	// 设置连接的会话ID和用户ID
	conn.SetSessionID(session.ID)
	conn.SetUserID(req.UserID)

	// 发送响应
	conn.SendMessage(&Message{
		Type:      MessageTypeSessionCreated,
		SessionID: session.ID,
		Data: &CreateSessionResponse{
			SessionID: session.ID,
			Channel:   session.Channel,
			ChatID:    session.ChatID,
			UserID:    session.UserID,
		},
		Timestamp: time.Now(),
	})

	m.logger.Info("会话创建成功", "session_id", session.ID, "connection_id", conn.ID())
}

// handleChat 处理聊天消息
func (m *Manager) handleChat(conn *Connection, msg *Message) {
	sessionID := conn.SessionID()
	if sessionID == 0 {
		conn.SendMessage(NewErrorMessage(0, 400, "请先创建会话"))
		return
	}

	data, err := json.Marshal(msg.Data)
	if err != nil {
		conn.SendMessage(NewErrorMessage(sessionID, 400, "无效的请求数据"))
		return
	}

	var req ChatRequest
	if err := json.Unmarshal(data, &req); err != nil {
		conn.SendMessage(NewErrorMessage(sessionID, 400, "无效的请求数据格式"))
		return
	}

	// 验证会话ID
	if req.SessionID != sessionID {
		conn.SendMessage(NewErrorMessage(sessionID, 400, "会话ID不匹配"))
		return
	}

	// 如果设置了消息处理器，使用处理器处理消息
	if m.handler != nil {
		err = m.handler.HandleMessage(m.ctx, sessionID, req.Content, conn.ID())
		if err != nil {
			conn.SendMessage(NewErrorMessage(sessionID, 500, fmt.Sprintf("处理消息失败: %v", err)))
			return
		}
	} else {
		// 获取会话
		session, err := m.storage.Session().GetByID(sessionID)
		if err != nil {
			conn.SendMessage(NewErrorMessage(sessionID, 500, "获取会话失败"))
			return
		}

		// 发布入站消息到消息总线
		err = m.bus.PublishInbound(m.ctx, bus.InboundMessage{
			Channel: session.Channel,
			ChatID:  session.ChatID,
			UserID:  session.UserID,
			Content: req.Content,
			Metadata: map[string]any{
				"client_id": conn.ID(),
			},
		})
		if err != nil {
			conn.SendMessage(NewErrorMessage(sessionID, 500, "发送消息失败"))
			return
		}
	}

	m.logger.Debug("聊天消息已发送", "session_id", sessionID, "content_length", len(req.Content))
}

// handlePing 处理心跳
func (m *Manager) handlePing(conn *Connection, msg *Message) {
	conn.SendMessage(&Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	})
}

// Close 关闭管理器
func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	for _, conn := range m.connections {
		conn.Close()
	}
	m.connections = make(map[string]*Connection)
	m.mu.Unlock()
	m.bus.Close()
}

// GetConnectionCount 获取连接数
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// Broadcast 广播消息给所有连接
func (m *Manager) Broadcast(msg *Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		m.logger.Error("序列化广播消息失败", "error", err)
		return
	}

	for _, conn := range m.connections {
		conn.Send(data)
	}
}

// SendToConnection 发送消息给指定连接
func (m *Manager) SendToConnection(connectionID string, msg *Message) error {
	m.mu.RLock()
	conn, ok := m.connections[connectionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("连接不存在: %s", connectionID)
	}

	return conn.SendMessage(msg)
}

// Bus 获取消息总线
func (m *Manager) Bus() *bus.MessageBus {
	return m.bus
}

// Storage 获取存储
func (m *Manager) Storage() *storage.Storage {
	return m.storage
}