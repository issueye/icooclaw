package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// DefaultAPEndpoint 默认 AP 接入点
	DefaultAPEndpoint = "wss://ap.agentunion.cn"
	// PingInterval 心跳间隔
	PingInterval = 30 * time.Second
	// WriteWait 写入超时
	WriteWait = 10 * time.Second
	// ReadLimit 读取限制
	ReadLimit = 512 * 1024
)

// Client ACP 客户端
type Client struct {
	config     *APConfig
	conn       *websocket.Conn
	mu         sync.RWMutex
	sessions   map[string]*Session
	agents     map[string]*AgentInfo
	ctx        context.Context
	cancel     context.CancelFunc
	handler    EventHandler
	agentHandler AgentEventHandler
	done       chan struct{}
}

// NewClient 创建 ACP 客户端
func NewClient(config *APConfig) *Client {
	if config.Endpoint == "" {
		config.Endpoint = DefaultAPEndpoint
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:   config,
		sessions: make(map[string]*Session),
		agents:   make(map[string]*AgentInfo),
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// SetEventHandler 设置消息处理器
func (c *Client) SetEventHandler(handler EventHandler) {
	c.handler = handler
}

// SetAgentEventHandler 设置 Agent 事件处理器
func (c *Client) SetAgentEventHandler(handler AgentEventHandler) {
	c.agentHandler = handler
}

// Connect 连接到 AP 接入点
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	u, err := url.Parse(c.config.Endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	// 构建 WebSocket URL
	wsURL := fmt.Sprintf("%s/ws?aid=%s", u.String(), c.config.AID)
	if c.config.APIKey != "" {
		wsURL += "&api_key=" + c.config.APIKey
	}

	log.Printf("Connecting to ACP AP: %s", wsURL)

	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.config.APIKey)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	conn.SetReadLimit(ReadLimit)
	conn.SetReadDeadline(time.Now().Add(PingInterval * 2))
	conn.SetWriteDeadline(time.Now().Add(WriteWait))

	c.conn = conn

	// 启动读取和心跳协程
	go c.readLoop()
	go c.pingLoop()

	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	close(c.done)
	c.cancel()

	err := c.conn.Close()
	c.conn = nil
	c.sessions = make(map[string]*Session)

	return err
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

// CreateSession 创建会话
func (c *Client) CreateSession(agentAID string) (*Session, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	session := &Session{
		ID:        generateSessionID(),
		AgentAID:  agentAID,
		CreatedAt: time.Now(),
		Channel:   "websocket",
	}

	msg := &ACPMessage{
		Type: "create_session",
		Data: SessionRequest{
			SessionID: session.ID,
			Channel:   "websocket",
		},
	}

	if err := c.sendMessage(msg); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.sessions[session.ID] = session
	c.mu.Unlock()

	return session, nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(sessionID, content string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	msg := &ACPMessage{
		Type:       "send_message",
		SessionID: sessionID,
		Data: ChatRequest{
			SessionID: sessionID,
			Content:   content,
		},
	}

	return c.sendMessage(msg)
}

// CloseSession 关闭会话
func (c *Client) CloseSession(sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg := &ACPMessage{
		Type:       "close_session",
		SessionID: sessionID,
	}

	if err := c.sendMessage(msg); err != nil {
		return err
	}

	delete(c.sessions, sessionID)
	return nil
}

// GetSession 获取会话
func (c *Client) GetSession(sessionID string) *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessions[sessionID]
}

// ListSessions 列出所有会话
func (c *Client) ListSessions() []*Session {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sessions := make([]*Session, 0, len(c.sessions))
	for _, s := range c.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// RegisterAgent 注册 Agent
func (c *Client) RegisterAgent(agent *AgentInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.agents[agent.AID] = agent
}

// GetAgent 获取 Agent 信息
func (c *Client) GetAgent(aid string) *AgentInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agents[aid]
}

// ListAgents 列出所有已连接 Agent
func (c *Client) ListAgents() []*AgentInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	agents := make([]*AgentInfo, 0, len(c.agents))
	for _, a := range c.agents {
		agents = append(agents, a)
	}
	return agents
}

// UpdateAgentStatus 更新 Agent 状态
func (c *Client) UpdateAgentStatus(aid string, status AgentStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if agent, ok := c.agents[aid]; ok {
		agent.Status = status
	}
}

// QueryAgent 查询 Agent 信息
func (c *Client) QueryAgent(aid string) (*AgentInfo, error) {
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, err
		}
	}

	msg := &ACPMessage{
		Type: "query_agent",
		Data: map[string]string{
			"aid": aid,
		},
	}

	if err := c.sendMessage(msg); err != nil {
		return nil, err
	}

	// 等待响应（简化实现，实际应该使用响应通道）
	time.Sleep(100 * time.Millisecond)

	return c.GetAgent(aid), nil
}

func (c *Client) sendMessage(msg *ACPMessage) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (c *Client) readLoop() {
	defer func() {
		c.Disconnect()
	}()

	for {
		select {
		case <-c.done:
			return
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			conn.SetReadDeadline(time.Now().Add(PingInterval * 2))
			_, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			var msg ACPMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// 处理消息
			c.handleMessage(&msg)
		}
	}
}

func (c *Client) handleMessage(msg *ACPMessage) {
	// 调用全局处理器
	if c.handler != nil {
		c.handler(msg)
	}

	// 调用 Agent 特定处理器
	if msg.SessionID != "" && c.agentHandler != nil {
		c.mu.RLock()
		session := c.sessions[msg.SessionID]
		c.mu.RUnlock()

		if session != nil && c.agentHandler != nil {
			c.agentHandler(session.AgentAID, msg)
		}
	}

	// 处理特定消息类型
	switch msg.Type {
	case "session_created":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if sessionID, ok := data["session_id"].(string); ok {
				c.mu.Lock()
				if _, ok := c.sessions[sessionID]; ok {
					// 会话创建成功
					log.Printf("Session created: %s", sessionID)
				}
				c.mu.Unlock()
			}
		}
	case "agent_status":
		if data, ok := msg.Data.(map[string]interface{}); ok {
			if aid, ok := data["aid"].(string); ok {
				if status, ok := data["status"].(string); ok {
					c.UpdateAgentStatus(aid, AgentStatus(status))
				}
			}
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %v", err)
				return
			}
		}
	}
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}