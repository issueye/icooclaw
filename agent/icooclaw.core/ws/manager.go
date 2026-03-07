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
	HandleMessage(ctx context.Context, sessionID string, content string, clientID string) error
}

// HandlerFunc 消息处理函数
type HandlerFunc func(ctx context.Context, sessionID string, content string, clientID string) error

// HandleMessage 实现 MessageHandler 接口
func (hf HandlerFunc) HandleMessage(ctx context.Context, sessionID string, content string, clientID string) error {
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
	queue       *ConversationQueue
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
		queue:       NewConversationQueue(DefaultMaxConcurrent, logger),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 应用选项
	for _, opt := range opts {
		opt(m)
	}

	// 设置队列通知器
	m.queue.SetNotifier(m)

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

// WithMaxConcurrent 设置最大并发数
func WithMaxConcurrent(max int) ManagerOption {
	return func(m *Manager) {
		m.queue = NewConversationQueue(max, m.logger)
	}
}

// SetHandler 设置消息处理器
func (m *Manager) SetHandler(handler MessageHandler) {
	m.handler = handler
}

// NotifyQueueStatus 实现 QueueNotifier 接口
func (m *Manager) NotifyQueueStatus(sessionID string, status *QueueStatus) {
	m.mu.RLock()
	for _, conn := range m.connections {
		if conn.SessionID() == sessionID {
			conn.SendMessage(&Message{
				Type:      MessageTypeQueueStatus,
				SessionID: sessionID,
				Data: &QueueStatusData{
					ActiveCount:   status.ActiveCount,
					WaitingCount:  status.WaitingCount,
					MaxConcurrent: status.MaxConcurrent,
				},
				CreatedAt: time.Now(),
			})
			break
		}
	}
	m.mu.RUnlock()
}

// NotifyQueuePosition 实现 QueueNotifier 接口
func (m *Manager) NotifyQueuePosition(sessionID string, position int) {
	m.mu.RLock()
	for _, conn := range m.connections {
		if conn.SessionID() == sessionID {
			status := m.queue.GetStatus()
			conn.SendMessage(&Message{
				Type:      MessageTypeQueueStatus,
				SessionID: sessionID,
				Data: &QueueStatusData{
					ActiveCount:   status.ActiveCount,
					WaitingCount:  status.WaitingCount,
					MaxConcurrent: status.MaxConcurrent,
					Position:      position,
				},
				CreatedAt: time.Now(),
			})
			break
		}
	}
	m.mu.RUnlock()
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
				// 取消该连接的队列任务
				if conn.SessionID() != "" {
					m.queue.Cancel(conn.SessionID())
				}
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
			CreatedAt: time.Now(),
		}
	case bus.MessageTypeThinking:
		wsMsg = &Message{
			Type:      MessageTypeThinking,
			SessionID: conn.SessionID(),
			Data: &ThinkingData{
				SessionID: conn.SessionID(),
				Content:   msg.Thinking,
			},
			CreatedAt: time.Now(),
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
			CreatedAt: time.Now(),
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
			CreatedAt: time.Now(),
		}
	case bus.MessageTypeEnd:
		wsMsg = &Message{
			Type:      MessageTypeEnd,
			SessionID: conn.SessionID(),
			Data: &EndData{
				SessionID: conn.SessionID(),
			},
			CreatedAt: time.Now(),
		}
	case bus.MessageTypeError:
		wsMsg = &Message{
			Type:      MessageTypeError,
			SessionID: conn.SessionID(),
			Error: &ErrorInfo{
				Code:    500,
				Message: msg.Error,
			},
			CreatedAt: time.Now(),
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
	case MessageTypeQueueStatus:
		m.handleQueueStatus(conn, msg)
	default:
		conn.SendMessage(NewErrorMessage(conn.SessionID(), 400, fmt.Sprintf("未知的消息类型：%s", msg.Type)))
	}
}

// handleCreateSession 处理创建会话请求
func (m *Manager) handleCreateSession(conn *Connection, msg *Message) {
	data, err := json.Marshal(msg.Data)
	if err != nil {
		conn.SendMessage(NewErrorMessage(conn.SessionID(), 400, "无效的请求数据"))
		return
	}

	var req CreateSessionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		conn.SendMessage(NewErrorMessage(conn.SessionID(), 400, "无效的请求数据格式"))
		return
	}

	// 设置默认渠道
	if req.Channel == "" {
		req.Channel = "websocket"
	}

	var session *storage.Session

	// 如果提供了 session_id，尝试复用已有会话
	if req.SessionID != "" {
		session, err = m.storage.Session().GetByID(req.SessionID)
		if err == nil && session != nil && session.ID != "" {
			m.logger.Info("复用已有会话", "session_id", session.ID, "connection_id", conn.ID())
		} else {
			session = nil
		}
	}

	// 如果没有复用成功，创建新会话
	if session == nil {
		chatID := fmt.Sprintf("ws-%s", conn.ID())
		session, err = m.storage.Session().GetOrCreateSession(req.Channel, chatID, req.UserID)
		if err != nil {
			conn.SendMessage(NewErrorMessage(conn.SessionID(), 500, "创建会话失败"))
			return
		}
	}

	// 设置连接的会话 ID 和用户 ID
	conn.SetSessionID(session.ID)
	if req.UserID != "" {
		conn.SetUserID(req.UserID)
	}

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
		CreatedAt: time.Now(),
	})

	m.logger.Info("会话创建成功", "session_id", session.ID, "connection_id", conn.ID())
}

// handleChat 处理聊天消息
func (m *Manager) handleChat(conn *Connection, msg *Message) {
	sessionID := conn.SessionID()
	if sessionID == "" {
		conn.SendMessage(NewErrorMessage(conn.SessionID(), 400, "请先创建会话"))
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

	// 验证会话 ID
	if req.SessionID != sessionID {
		conn.SendMessage(NewErrorMessage(sessionID, 400, "会话 ID 不匹配"))
		return
	}

	// 检查是否已在队列中
	if m.queue.IsQueued(sessionID) {
		position := m.queue.GetPosition(sessionID)
		conn.SendMessage(NewErrorMessage(sessionID, 429, fmt.Sprintf("请求正在处理中，队列位置：%d", position)))
		return
	}

	// 设置队列处理器
	m.queue.SetHandler(QueueHandlerFunc(func(ctx context.Context, item *QueueItem) error {
		return m.processChatMessage(ctx, item.SessionID, item.Content, item.ConnectionID)
	}))

	// 加入队列
	item, accepted := m.queue.Enqueue(m.ctx, sessionID, conn.ID(), req.Content)
	if !accepted {
		conn.SendMessage(NewErrorMessage(sessionID, 500, "无法加入处理队列"))
		return
	}

	// 如果正在等待，发送队列状态
	if m.queue.IsWaiting(sessionID) {
		position := m.queue.GetPosition(sessionID)
		status := m.queue.GetStatus()
		conn.SendMessage(&Message{
			Type:      MessageTypeQueueStatus,
			SessionID: sessionID,
			Data: &QueueStatusData{
				ActiveCount:   status.ActiveCount,
				WaitingCount:  status.WaitingCount,
				MaxConcurrent: status.MaxConcurrent,
				Position:      position,
			},
			CreatedAt: time.Now(),
		})
	}

	m.logger.Debug("聊天消息已加入队列", "session_id", sessionID, "content_length", len(req.Content), "item_accepted", accepted && item != nil)
}

// processChatMessage 处理聊天消息
func (m *Manager) processChatMessage(ctx context.Context, sessionID string, content string, clientID string) error {
	// 如果设置了消息处理器，使用处理器处理消息
	if m.handler != nil {
		m.logger.Info("[WebSocket] 使用自定义消息处理器",
			"session_id", sessionID,
			"client_id", clientID,
			"content_length", len(content),
		)
		return m.handler.HandleMessage(ctx, sessionID, content, clientID)
	}

	// 获取会话
	session, err := m.storage.Session().GetByID(sessionID)
	if err != nil {
		m.logger.Error("[WebSocket] 获取会话失败",
			"session_id", sessionID,
			"error", err,
		)
		return fmt.Errorf("获取会话失败：%w", err)
	}

	contentPreview := content
	if len(content) > 100 {
		contentPreview = content[:100] + "..."
	}

	m.logger.Info("[WebSocket] 发布消息到总线",
		"session_id", sessionID,
		"channel", session.Channel,
		"chat_id", session.ChatID,
		"client_id", clientID,
		"content_length", len(content),
		"content_preview", contentPreview,
	)

	// 发布入站消息到消息总线
	err = m.bus.PublishInbound(ctx, bus.InboundMessage{
		SessionID: sessionID,
		Channel:   session.Channel,
		ChatID:    session.ChatID,
		UserID:    session.UserID,
		Content:   content,
		Metadata: map[string]any{
			"client_id": clientID,
		},
	})
	if err != nil {
		m.logger.Error("[WebSocket] 发布消息到总线失败",
			"session_id", sessionID,
			"error", err,
		)
		return fmt.Errorf("发送消息失败：%w", err)
	}

	m.logger.Debug("[WebSocket] 消息已成功发布到总线",
		"session_id", sessionID,
		"client_id", clientID,
	)

	return nil
}

// handlePing 处理心跳
func (m *Manager) handlePing(conn *Connection, msg *Message) {
	conn.SendMessage(&Message{
		Type:      MessageTypePong,
		CreatedAt: time.Now(),
	})
}

// handleQueueStatus 处理队列状态查询
func (m *Manager) handleQueueStatus(conn *Connection, msg *Message) {
	sessionID := conn.SessionID()
	status := m.queue.GetStatus()
	position := -1
	if sessionID != "" {
		position = m.queue.GetPosition(sessionID)
	}

	conn.SendMessage(&Message{
		Type:      MessageTypeQueueStatus,
		SessionID: sessionID,
		Data: &QueueStatusData{
			ActiveCount:   status.ActiveCount,
			WaitingCount:  status.WaitingCount,
			MaxConcurrent: status.MaxConcurrent,
			Position:      position,
		},
		CreatedAt: time.Now(),
	})
}

// Close 关闭管理器
func (m *Manager) Close() {
	m.cancel()
	m.queue.Close()
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
		return fmt.Errorf("连接不存在：%s", connectionID)
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

// Queue 获取对话队列
func (m *Manager) Queue() *ConversationQueue {
	return m.queue
}

// SetMaxConcurrent 设置最大并发数
func (m *Manager) SetMaxConcurrent(max int) {
	m.queue.SetMaxConcurrent(max)
}

// GetQueueStatus 获取队列状态
func (m *Manager) GetQueueStatus() *QueueStatus {
	return m.queue.GetStatus()
}
