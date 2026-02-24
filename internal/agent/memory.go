package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/icooclaw/icooclaw/internal/storage"
)

// MemoryConfig 内存配置
type MemoryConfig struct {
	ConsolidationThreshold int  `mapstructure:"consolidation_threshold"`
	SummaryEnabled         bool `mapstructure:"summary_enabled"`
	AutoSave               bool `mapstructure:"auto_save"`
	MaxMemoryAge           int  `mapstructure:"max_memory_age"`
}

// MemoryStore 长期记忆系统
type MemoryStore struct {
	storage *storage.Storage
	logger  *slog.Logger
	config  MemoryConfig
	mu      sync.RWMutex

	// 内存缓存
	sessionMessageCounts map[uint]int       // session ID -> message count
	lastConsolidation    map[uint]time.Time // session ID -> last consolidation time
}

// NewMemoryStore 创建记忆存储
func NewMemoryStoreWithConfig(storage *storage.Storage, logger *slog.Logger, cfg MemoryConfig) *MemoryStore {
	if cfg.ConsolidationThreshold == 0 {
		cfg.ConsolidationThreshold = 50
	}
	if cfg.MaxMemoryAge == 0 {
		cfg.MaxMemoryAge = 30
	}

	return &MemoryStore{
		storage:              storage,
		logger:               logger,
		config:               cfg,
		sessionMessageCounts: make(map[uint]int),
		lastConsolidation:    make(map[uint]time.Time),
	}
}

// Load 加载记忆
func (m *MemoryStore) Load(ctx context.Context) error {
	m.logger.Info("Loading memory")
	// 从数据库加载记忆
	return nil
}

// GetAll 获取所有记忆
func (m *MemoryStore) GetAll() ([]storage.Memory, error) {
	return m.storage.GetAllMemories()
}

// Get 获取记忆
func (m *MemoryStore) Get(key string) (*storage.Memory, error) {
	return m.storage.GetMemoryByKey(key)
}

// Add 添加记忆
func (m *MemoryStore) Add(memType, key, content string) error {
	memory := &storage.Memory{
		Type:    memType,
		Key:     key,
		Content: content,
	}
	return m.storage.CreateMemory(memory)
}

// Update 更新记忆
func (m *MemoryStore) Update(key, content string) error {
	memory, err := m.storage.GetMemoryByKey(key)
	if err != nil {
		return err
	}
	memory.Content = content
	return m.storage.UpdateMemory(memory)
}

// Delete 删除记忆
func (m *MemoryStore) Delete(key string) error {
	memory, err := m.storage.GetMemoryByKey(key)
	if err != nil {
		return err
	}
	return m.storage.DeleteMemory(memory.ID)
}

// Search 搜索记忆
func (m *MemoryStore) Search(query string) ([]storage.Memory, error) {
	return m.storage.SearchMemories(query)
}

// Consolidate 整合记忆（根据会话历史）
func (m *MemoryStore) Consolidate(session *storage.Session) error {
	// 获取会话消息
	messages, err := session.GetMessages(100)
	if err != nil {
		return fmt.Errorf("failed to get messages: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	// 检查是否达到阈值
	m.mu.Lock()
	count := m.sessionMessageCounts[session.ID]
	if count < m.config.ConsolidationThreshold {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// 执行摘要
	if m.config.SummaryEnabled {
		summary, err := m.summarizeMessages(messages)
		if err != nil {
			m.logger.Warn("Failed to summarize messages", "error", err)
		} else {
			// 保存摘要
			summaryKey := fmt.Sprintf("summary_%d", session.ID)
			if err := m.RememberHistory(summaryKey, summary); err != nil {
				m.logger.Warn("Failed to save summary", "error", err)
			}

			// 更新最后整合时间
			m.mu.Lock()
			m.lastConsolidation[session.ID] = time.Now()
			m.sessionMessageCounts[session.ID] = 0 // 重置计数
			m.mu.Unlock()

			m.logger.Info("Memory consolidated", "session_id", session.ID, "summary_length", len(summary))
		}
	}

	return nil
}

// summarizeMessages 对消息进行摘要
func (m *MemoryStore) summarizeMessages(messages []storage.Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// 简单摘要策略：提取关键信息
	// 1. 用户问题和助手回答的模式
	// 2. 工具调用记录
	// 3. 关键决策

	var summaryParts []string
	summaryParts = append(summaryParts, fmt.Sprintf("会话包含 %d 条消息", len(messages)))

	// 统计工具调用
	toolCallCount := 0
	userMsgCount := 0
	assistantMsgCount := 0

	for _, msg := range messages {
		if msg.Role == "user" {
			userMsgCount++
		} else if msg.Role == "assistant" {
			assistantMsgCount++
			if msg.ToolCalls != "" {
				toolCallCount++
			}
		}
	}

	summaryParts = append(summaryParts, fmt.Sprintf("用户消息: %d, 助手消息: %d, 工具调用: %d",
		userMsgCount, assistantMsgCount, toolCallCount))

	// 提取最近的5对对话作为样例
	if len(messages) >= 2 {
		summaryParts = append(summaryParts, "\n最近对话样例:")
		sampleCount := 0
		for i := len(messages) - 1; i >= 0 && sampleCount < 5; i-- {
			if messages[i].Role == "user" && i+1 < len(messages) && messages[i+1].Role == "assistant" {
				userContent := truncate(messages[i].Content, 100)
				assistantContent := truncate(messages[i+1].Content, 100)
				summaryParts = append(summaryParts, fmt.Sprintf("Q: %s\nA: %s", userContent, assistantContent))
				sampleCount++
			}
		}
	}

	return strings.Join(summaryParts, "\n"), nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RememberImportant 记住重要信息
func (m *MemoryStore) RememberImportant(key, content string) error {
	return m.Add("memory", key, content)
}

// RememberHistory 记住历史
func (m *MemoryStore) RememberHistory(key, content string) error {
	return m.Add("history", key, content)
}

// AutoSave 自动保存重要信息到记忆
func (m *MemoryStore) AutoSave(sessionID uint, content string) error {
	if !m.config.AutoSave {
		return nil
	}

	// 更新消息计数
	m.mu.Lock()
	m.sessionMessageCounts[sessionID]++
	m.mu.Unlock()

	m.logger.Debug("Auto-saving to memory", "session_id", sessionID, "content_length", len(content))
	return nil
}

// IncrementMessageCount 增加消息计数
func (m *MemoryStore) IncrementMessageCount(sessionID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionMessageCounts[sessionID]++
}

// GetMessageCount 获取消息计数
func (m *MemoryStore) GetMessageCount(sessionID uint) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionMessageCounts[sessionID]
}

// ShouldConsolidate 检查是否应该整合
func (m *MemoryStore) ShouldConsolidate(sessionID uint) bool {
	m.mu.RLock()
	count := m.sessionMessageCounts[sessionID]
	m.mu.RUnlock()
	return count >= m.config.ConsolidationThreshold
}

// GetRecentHistory 获取最近历史
func (m *MemoryStore) GetRecentHistory(limit int) ([]storage.Memory, error) {
	all, err := m.storage.GetAllMemories()
	if err != nil {
		return nil, err
	}
	if len(all) > limit {
		return all[:limit], nil
	}
	return all, nil
}

// GetSessionSummary 获取会话摘要
func (m *MemoryStore) GetSessionSummary(sessionID uint) (string, error) {
	key := fmt.Sprintf("summary_%d", sessionID)
	memory, err := m.storage.GetMemoryByKey(key)
	if err != nil {
		return "", err
	}
	return memory.Content, nil
}

// CleanOldHistory 清理旧历史
func (m *MemoryStore) CleanOldHistory(before time.Time) error {
	m.logger.Info("Cleaning old history", "before", before)

	// 获取所有记忆
	memories, err := m.storage.GetAllMemories()
	if err != nil {
		return err
	}

	var toDelete []uint
	for _, mem := range memories {
		if mem.Type == "history" && mem.UpdatedAt.Before(before) {
			toDelete = append(toDelete, mem.ID)
		}
	}

	for _, id := range toDelete {
		if err := m.storage.DeleteMemory(id); err != nil {
			m.logger.Warn("Failed to delete old memory", "id", id, "error", err)
		}
	}

	m.logger.Info("Old history cleaned", "deleted_count", len(toDelete))
	return nil
}

// ScheduleCleanup 安排定期清理
func (m *MemoryStore) ScheduleCleanup(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				before := time.Now().AddDate(0, 0, -m.config.MaxMemoryAge)
				if err := m.CleanOldHistory(before); err != nil {
					m.logger.Error("Failed to clean old history", "error", err)
				}
			}
		}
	}()
}

// GetMemoryStats 获取记忆统计
func (m *MemoryStore) GetMemoryStats() (map[string]int, error) {
	memories, err := m.storage.GetAllMemories()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	stats["total"] = len(memories)
	stats["memory"] = 0
	stats["history"] = 0

	for _, mem := range memories {
		if mem.Type == "memory" {
			stats["memory"]++
		} else if mem.Type == "history" {
			stats["history"]++
		}
	}

	return stats, nil
}

// RetrieveRelevantMemories 检索相关记忆
func (m *MemoryStore) RetrieveRelevantMemories(query string, limit int) ([]string, error) {
	memories, err := m.storage.SearchMemories(query)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(memories) > limit {
		memories = memories[:limit]
	}

	result := make([]string, len(memories))
	for i, mem := range memories {
		result[i] = fmt.Sprintf("[%s] %s: %s", mem.Type, mem.Key, mem.Content)
	}

	return result, nil
}

// MergeSessionMemories 合并会话记忆
func (m *MemoryStore) MergeSessionMemories(sessionID uint, summary string) error {
	key := fmt.Sprintf("session_%d_merged", sessionID)
	return m.RememberHistory(key, summary)
}

// ExtractKeyPoints 提取关键点
func (m *MemoryStore) ExtractKeyPoints(messages []storage.Message) []string {
	var keyPoints []string

	for _, msg := range messages {
		// 检测关键信息模式
		if strings.Contains(strings.ToLower(msg.Content), "important") ||
			strings.Contains(strings.ToLower(msg.Content), "remember") ||
			strings.Contains(strings.ToLower(msg.Content), "note") {
			keyPoints = append(keyPoints, msg.Content)
		}

		// 提取工具调用结果中的重要信息
		if msg.Role == "tool" && len(msg.Content) > 50 {
			keyPoints = append(keyPoints, truncate(msg.Content, 200))
		}
	}

	return keyPoints
}

// ContextualMemory 添加上下文记忆
type ContextualMemory struct {
	SessionID    uint
	UserID       string
	Preferences  map[string]string
	LastTopic    string
	LastActivity time.Time
}

// SaveContextualMemory 保存上下文记忆
func (m *MemoryStore) SaveContextualMemory(ctx *ContextualMemory) error {
	data, err := json.Marshal(ctx)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("context_%d_%s", ctx.SessionID, ctx.UserID)
	return m.RememberImportant(key, string(data))
}

// GetContextualMemory 获取上下文记忆
func (m *MemoryStore) GetContextualMemory(sessionID uint, userID string) (*ContextualMemory, error) {
	key := fmt.Sprintf("context_%d_%s", sessionID, userID)
	memory, err := m.storage.GetMemoryByKey(key)
	if err != nil {
		return nil, err
	}

	var ctx ContextualMemory
	if err := json.Unmarshal([]byte(memory.Content), &ctx); err != nil {
		return nil, err
	}

	return &ctx, nil
}
