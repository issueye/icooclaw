package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写超时
	writeWait = 10 * time.Second

	// Pong 超时
	pongWait = 60 * time.Second

	// Ping 发送间隔 (必须小于 pongWait)
	pingPeriod = (pongWait * 9) / 10

	// 发送缓冲区大小
	sendBufferSize = 256
)

// Connection WebSocket 连接
type Connection struct {
	id        string
	conn      *websocket.Conn
	manager   *Manager
	send      chan []byte
	sessionID string
	userID    string
	mu        sync.RWMutex
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewConnection 创建新的 WebSocket 连接
func NewConnection(id string, conn *websocket.Conn, manager *Manager, logger *slog.Logger) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		id:      id,
		conn:    conn,
		manager: manager,
		send:    make(chan []byte, sendBufferSize),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// ID 获取连接ID
func (c *Connection) ID() string {
	return c.id
}

// SessionID 获取会话ID
func (c *Connection) SessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// SetSessionID 设置会话ID
func (c *Connection) SetSessionID(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = sessionID
}

// UserID 获取用户ID
func (c *Connection) UserID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

// SetUserID 设置用户ID
func (c *Connection) SetUserID(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = userID
}

// ReadPump 读取消息循环
func (c *Connection) ReadPump() {
	defer func() {
		c.manager.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(10 * 1024 * 1024) // 10MB
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket 读取错误", "error", err, "connection_id", c.id)
			}
			break
		}

		c.handleMessage(message)
	}
}

// WritePump 写入消息循环
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send 发送消息
func (c *Connection) Send(message []byte) bool {
	select {
	case c.send <- message:
		return true
	default:
		c.logger.Warn("发送缓冲区已满", "connection_id", c.id)
		return false
	}
}

// SendMessage 发送结构化消息
func (c *Connection) SendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.Send(data)
	return nil
}

// Close 关闭连接
func (c *Connection) Close() {
	c.cancel()
	c.conn.Close()
}

// handleMessage 处理接收到的消息
func (c *Connection) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.logger.Error("解析消息失败", "error", err, "connection_id", c.id)
		c.SendMessage(NewErrorMessage(c.SessionID(), 400, "无效的消息格式"))
		return
	}

	msg.Timestamp = time.Now()

	// 将消息发送到管理器处理
	c.manager.handleClientMessage(c, &msg)
}
