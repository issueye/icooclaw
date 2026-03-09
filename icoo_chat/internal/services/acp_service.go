package services

import (
	"context"
	"encoding/json"
	"fmt"
	"icoo_chat/internal/acp"
	"log"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ACPService ACP 服务
type ACPService struct {
	client      *acp.Client
	config      *acp.APConfig
	mu          sync.RWMutex
	messageChan chan *acp.ACPMessage
	app         *App
	ctx         context.Context
}

// NewACPService 创建 ACP 服务
func NewACPService(app *App) *ACPService {
	return &ACPService{
		messageChan: make(chan *acp.ACPMessage, 100),
		app:         app,
	}
}

// SetContext 设置上下文
func (s *ACPService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// InitACP 初始化 ACP 客户端
func (s *ACPService) InitACP(endpoint, apiKey, aid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Disconnect()
	}

	s.config = &acp.APConfig{
		Endpoint: endpoint,
		APIKey:   apiKey,
		AID:      aid,
	}

	s.client = acp.NewClient(s.config)

	// 设置消息处理器
	s.client.SetEventHandler(func(msg *acp.ACPMessage) {
		s.messageChan <- msg
	})

	s.client.SetAgentEventHandler(func(aid string, msg *acp.ACPMessage) {
		// 发送到前端
		if s.ctx != nil {
			runtime.EventsEmit(s.ctx, "acp:message", map[string]interface{}{
				"aid":  aid,
				"type": msg.Type,
				"data": msg.Data,
			})
		}
	})

	log.Printf("ACP client initialized with endpoint: %s, aid: %s", endpoint, aid)
	return nil
}

// Connect 连接到 AP
func (s *ACPService) Connect() error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("ACP client not initialized")
	}

	return client.Connect()
}

// Disconnect 断开连接
func (s *ACPService) Disconnect() error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil
	}

	return client.Disconnect()
}

// IsConnected 检查连接状态
func (s *ACPService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.client == nil {
		return false
	}
	return s.client.IsConnected()
}

// ConnectAgent 连接 Agent
func (s *ACPService) ConnectAgent(aid string) (*acp.AgentInfo, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("ACP client not initialized")
	}

	// 查询 Agent 信息
	agent, err := client.QueryAgent(aid)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	if agent != nil {
		client.RegisterAgent(agent)
	}

	return agent, nil
}

// DisconnectAgent 断开 Agent 连接
func (s *ACPService) DisconnectAgent(aid string) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("ACP client not initialized")
	}

	client.UpdateAgentStatus(aid, acp.AgentStatusDisconnected)
	return nil
}

// GetAgentInfo 获取 Agent 信息
func (s *ACPService) GetAgentInfo(aid string) (*acp.AgentInfo, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("ACP client not initialized")
	}

	return client.GetAgent(aid), nil
}

// ListConnectedAgents 列出已连接 Agent
func (s *ACPService) ListConnectedAgents() []*acp.AgentInfo {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil
	}

	return client.ListAgents()
}

// CreateSession 创建会话
func (s *ACPService) CreateSession(agentAID string) (string, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("ACP client not initialized")
	}

	session, err := client.CreateSession(agentAID)
	if err != nil {
		return "", err
	}

	return session.ID, nil
}

// SendMessage 发送消息
func (s *ACPService) SendMessage(sessionID, content string) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("ACP client not initialized")
	}

	return client.SendMessage(sessionID, content)
}

// CloseSession 关闭会话
func (s *ACPService) CloseSession(sessionID string) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("ACP client not initialized")
	}

	return client.CloseSession(sessionID)
}

// GetConfig 获取当前配置
func (s *ACPService) GetConfig() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	return map[string]string{
		"endpoint": s.config.Endpoint,
		"aid":      s.config.AID,
		// 不返回 API Key
	}
}

// StartMessageListener 启动消息监听
func (s *ACPService) StartMessageListener(ctx context.Context) {
	s.ctx = ctx
	go func() {
		for {
			select {
			case msg := <-s.messageChan:
				if msg == nil {
					return
				}
				// 发送消息到前端
				data, _ := json.Marshal(msg)
				runtime.EventsEmit(ctx, "acp:message", string(data))
			}
		}
	}()
}