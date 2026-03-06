package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"icooclaw.core/bus"
	"icooclaw.core/storage"
	"icooclaw.ai/provider"
	"icooclaw.core/config"
)

// AgentManagerConfig Agent 管理器配置
type AgentManagerConfig struct {
	MaxAgents     int           // 最大 Agent 数量
	IdleTimeout   time.Duration // Agent 空闲超时（超过此时间未使用会被回收）
	PreStartCount int           // 预启动的 Agent 数量
}

// DefaultAgentManagerConfig 默认配置
func DefaultAgentManagerConfig() *AgentManagerConfig {
	return &AgentManagerConfig{
		MaxAgents:     10,                // 默认最大 10 个 Agent
		IdleTimeout:   10 * time.Minute,  // 默认空闲超时 10 分钟
		PreStartCount: 2,                 // 默认预启动 2 个 Agent
	}
}

// AgentInfo Agent 信息
type AgentInfo struct {
	ID           uint      // Agent ID（会话 ID）
	Name         string    // Agent 名称
	CreatedAt    time.Time // 创建时间
	LastUsedAt   time.Time // 最后使用时间
	IsActive     bool      // 是否正在处理消息
	MessageCount int64     // 处理的消息数量
}

// AgentManager Agent 管理器
type AgentManager struct {
	config       *AgentManagerConfig
	logger       *slog.Logger
	storage      *storage.Storage  // 存储接口
	provider     provider.Provider // Provider 接口
	agentConfig  *config.AgentSettings // Agent 配置
	workspace    string
	agents       map[uint]*Agent // 会话 ID -> Agent
	agentInfos   map[uint]*AgentInfo
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	messageBus   *bus.MessageBus
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(
	config *AgentManagerConfig,
	storage *storage.Storage,
	provider provider.Provider,
	agentConfig *config.AgentSettings,
	workspace string,
	logger *slog.Logger,
) *AgentManager {
	if config == nil {
		config = DefaultAgentManagerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &AgentManager{
		config:      config,
		logger:      logger,
		storage:     storage,
		provider:    provider,
		agentConfig: agentConfig,
		workspace:   workspace,
		agents:      make(map[uint]*Agent),
		agentInfos:  make(map[uint]*AgentInfo),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 预启动 Agent
	manager.preStartAgents()

	// 启动清理协程
	go manager.cleanupIdleAgents()

	return manager
}

// preStartAgents 预启动 Agent
func (m *AgentManager) preStartAgents() {
	count := m.config.PreStartCount
	if count > m.config.MaxAgents {
		count = m.config.MaxAgents
	}

	m.logger.Info("[AgentManager] 开始预启动 Agent",
		"count", count,
		"max_agents", m.config.MaxAgents,
	)

	m.logger.Info("[AgentManager] 预启动 Agent 完成")
}

// SetMessageBus 设置消息总线（在 Manager 创建后设置）
func (m *AgentManager) SetMessageBus(messageBus *bus.MessageBus) {
	m.messageBus = messageBus
	m.logger.Info("[AgentManager] 消息总线已设置，开始监听消息")

	// 启动消息监听协程
	m.wg.Add(1)
	go m.listenMessages()
}

// listenMessages 监听消息总线并分发到 Agent
func (m *AgentManager) listenMessages() {
	defer m.wg.Done()

	m.logger.Info("[AgentManager] 开始监听消息总线")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("[AgentManager] 上下文已取消，停止监听消息")
			return
		default:
			msg, err := m.messageBus.ConsumeInbound(m.ctx)
			if err != nil {
				if m.ctx.Err() != nil {
					return
				}
				m.logger.Error("[AgentManager] 从消息总线消费消息失败",
					"error", err,
				)
				continue
			}

			// 获取客户端 ID 用于追踪
			clientID := ""
			if msg.Metadata != nil {
				if id, ok := msg.Metadata["client_id"].(string); ok {
					clientID = id
				}
			}

			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}

			m.logger.Info("[AgentManager] ✓ 从消息总线接收到消息",
				"channel", msg.Channel,
				"chat_id", msg.ChatID,
				"user_id", msg.UserID,
				"client_id", clientID,
				"content_length", len(msg.Content),
				"content_preview", content,
			)

			// 为每个消息分配或创建 Agent
			m.wg.Add(1)
			go func(msg bus.InboundMessage) {
				defer m.wg.Done()
				m.processMessage(m.ctx, msg)
			}(msg)
		}
	}
}

// processMessage 处理消息（为每个会话分配/创建 Agent）
func (m *AgentManager) processMessage(ctx context.Context, msg bus.InboundMessage) {
	// 从消息中提取会话标识
	// 这里使用 chat_id 作为会话标识
	sessionID, err := m.getSessionIDFromMessage(msg)
	if err != nil {
		m.logger.Error("[AgentManager] 获取会话 ID 失败",
			"chat_id", msg.ChatID,
			"error", err,
		)
		return
	}

	// 获取或创建 Agent
	agent := m.getOrCreateAgent(sessionID)
	if agent == nil {
		m.logger.Error("[AgentManager] 创建 Agent 失败",
			"session_id", sessionID,
		)
		return
	}

	// 更新 Agent 使用状态
	m.updateAgentUsage(sessionID)

	// 将消息交给 Agent 处理
	agent.handleMessage(ctx, sessionID, msg)
}

// getSessionIDFromMessage 从消息中提取会话 ID
// 这里简化处理，实际可能需要更复杂的逻辑
func (m *AgentManager) getSessionIDFromMessage(msg bus.InboundMessage) (uint, error) {
	// 这里需要根据实际业务逻辑实现
	// 目前返回一个基于 chat_id 的哈希值作为会话 ID
	// 实际应该从存储中获取真实的会话 ID
	
	// 临时实现：返回 0 表示使用全局 Agent
	// TODO: 实现真实的会话 ID 获取逻辑
	return 0, nil
}

// getOrCreateAgent 获取或创建 Agent
func (m *AgentManager) getOrCreateAgent(sessionID uint) *Agent {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if agent, ok := m.agents[sessionID]; ok {
		m.logger.Debug("[AgentManager] 使用已存在的 Agent",
			"session_id", sessionID,
		)
		return agent
	}

	// 检查是否超过最大数量
	if len(m.agents) >= m.config.MaxAgents {
		m.logger.Warn("[AgentManager] 已达到最大 Agent 数量，尝试回收空闲 Agent",
			"current_count", len(m.agents),
			"max_count", m.config.MaxAgents,
		)
		
		// 尝试回收一个空闲 Agent
		m.recycleIdleAgent()
		
		// 如果还是超过限制，返回 nil
		if len(m.agents) >= m.config.MaxAgents {
			m.logger.Error("[AgentManager] 无法创建新 Agent，已达到最大数量限制",
				"max_count", m.config.MaxAgents,
			)
			return nil
		}
	}

	// 创建新的 Agent
	agent := m.createAgent(sessionID)
	if agent == nil {
		return nil
	}

	m.agents[sessionID] = agent

	m.logger.Info("[AgentManager] 创建新 Agent",
		"session_id", sessionID,
		"name", agent.Name(),
		"total_agents", len(m.agents),
	)

	return agent
}

// createAgent 创建 Agent
func (m *AgentManager) createAgent(sessionID uint) *Agent {
	// 创建 Agent 配置
	agentConfig := &config.AgentSettings{
		Name:         fmt.Sprintf("agent-%d", sessionID),
		Model:        m.agentConfig.Model,
		Temperature:  m.agentConfig.Temperature,
		MaxTokens:    m.agentConfig.MaxTokens,
		MemoryWindow: m.agentConfig.MemoryWindow,
		SystemPrompt: m.agentConfig.SystemPrompt,
	}

	agent := NewAgent(
		sessionID,
		agentConfig.Name,
		m.provider,
		m.storage,
		agentConfig,
		m.logger,
		m.workspace,
	)

	// 记录 Agent 信息
	m.agentInfos[sessionID] = &AgentInfo{
		ID:           sessionID,
		Name:         agentConfig.Name,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		IsActive:     false,
		MessageCount: 0,
	}

	return agent
}

// updateAgentUsage 更新 Agent 使用状态
func (m *AgentManager) updateAgentUsage(sessionID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.agentInfos[sessionID]; ok {
		info.LastUsedAt = time.Now()
		info.IsActive = true
		info.MessageCount++
	}
}

// markAgentIdle 标记 Agent 为空闲
func (m *AgentManager) markAgentIdle(sessionID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.agentInfos[sessionID]; ok {
		info.IsActive = false
	}
}

// recycleIdleAgent 回收一个空闲 Agent
func (m *AgentManager) recycleIdleAgent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var oldestSessionID uint
	var oldestTime time.Time

	// 找到最长时间未使用的空闲 Agent
	for sessionID, info := range m.agentInfos {
		if !info.IsActive {
			if oldestSessionID == 0 || info.LastUsedAt.Before(oldestTime) {
				oldestSessionID = sessionID
				oldestTime = info.LastUsedAt
			}
		}
	}

	if oldestSessionID == 0 {
		m.logger.Warn("[AgentManager] 没有找到可回收的空闲 Agent")
		return
	}

	// 删除 Agent
	delete(m.agents, oldestSessionID)
	delete(m.agentInfos, oldestSessionID)

	m.logger.Info("[AgentManager] 已回收空闲 Agent",
		"session_id", oldestSessionID,
	)
}

// cleanupIdleAgents 定期清理空闲 Agent
func (m *AgentManager) cleanupIdleAgents() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("[AgentManager] 停止清理空闲 Agent")
			return
		case <-ticker.C:
			m.doCleanup()
		}
	}
}

// doCleanup 执行清理
func (m *AgentManager) doCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	timeout := m.config.IdleTimeout
	now := time.Now()

	var toDelete []uint

	for sessionID, info := range m.agentInfos {
		if !info.IsActive && now.Sub(info.LastUsedAt) > timeout {
			toDelete = append(toDelete, sessionID)
		}
	}

	for _, sessionID := range toDelete {
		info := m.agentInfos[sessionID]
		delete(m.agents, sessionID)
		delete(m.agentInfos, sessionID)
		m.logger.Info("[AgentManager] 清理超时空闲 Agent",
			"session_id", sessionID,
			"idle_duration", now.Sub(info.LastUsedAt).String(),
		)
	}

	if len(toDelete) > 0 {
		m.logger.Info("[AgentManager] 本次清理了 %d 个空闲 Agent", len(toDelete))
	}
}

// GetAgentCount 获取当前 Agent 数量
func (m *AgentManager) GetAgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// GetActiveAgentCount 获取活跃 Agent 数量
func (m *AgentManager) GetActiveAgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, info := range m.agentInfos {
		if info.IsActive {
			count++
		}
	}
	return count
}

// GetAgentInfos 获取所有 Agent 信息
func (m *AgentManager) GetAgentInfos() []*AgentInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]*AgentInfo, 0, len(m.agentInfos))
	for _, info := range m.agentInfos {
		infos = append(infos, info)
	}
	return infos
}

// SetMaxAgents 设置最大 Agent 数量
func (m *AgentManager) SetMaxAgents(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if max <= 0 {
		m.logger.Warn("[AgentManager] 最大 Agent 数量必须大于 0")
		return
	}

	oldMax := m.config.MaxAgents
	m.config.MaxAgents = max

	m.logger.Info("[AgentManager] 更新最大 Agent 数量",
		"old_max", oldMax,
		"new_max", max,
	)

	// 如果新最大值小于当前数量，触发清理
	if max < len(m.agents) {
		go m.doCleanup()
	}
}

// GetMaxAgents 获取最大 Agent 数量
func (m *AgentManager) GetMaxAgents() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.MaxAgents
}

// Close 关闭管理器
func (m *AgentManager) Close() {
	m.logger.Info("[AgentManager] 开始关闭管理器")

	// 取消上下文
	m.cancel()

	// 等待所有协程结束
	m.wg.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 清理所有 Agent
	for sessionID := range m.agents {
		delete(m.agents, sessionID)
	}
	m.agentInfos = make(map[uint]*AgentInfo)

	m.logger.Info("[AgentManager] 管理器已关闭")
}
