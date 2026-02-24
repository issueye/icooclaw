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
	MaxSessionMemories     int  `mapstructure:"max_session_memories"`
	MaxUserMemories        int  `mapstructure:"max_user_memories"`
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

// SessionMemory 会话级别记忆
type SessionMemory struct {
	SessionID uint
	Key       string
	Content   string
	Tags      []string
}

// UserMemory 用户级别记忆
type UserMemory struct {
	UserID    string
	Key       string
	Content   string
	Tags      []string
}

// MemoryStats 记忆统计
type MemoryStats struct {
	Total       int64 `json:"total"`
	Memory      int64 `json:"memory"`
	History     int64 `json:"history"`
	Session     int64 `json:"session"`
	User        int64 `json:"user"`
	Pinned      int64 `json:"pinned"`
	Expired     int64 `json:"expired"`
}

// NewMemoryStore 创建记忆存储
func NewMemoryStoreWithConfig(storage *storage.Storage, logger *slog.Logger, cfg MemoryConfig) *MemoryStore {
	if cfg.ConsolidationThreshold == 0 {
		cfg.ConsolidationThreshold = 50
	}
	if cfg.MaxMemoryAge == 0 {
		cfg.MaxMemoryAge = 30
	}
	if cfg.MaxSessionMemories == 0 {
		cfg.MaxSessionMemories = 100
	}
	if cfg.MaxUserMemories == 0 {
		cfg.MaxUserMemories = 500
	}

	return &MemoryStore{
		storage:              storage,
		logger:               logger,
		config:               cfg,
		sessionMessageCounts: make(map[uint]int),
		lastConsolidation:    make(map[uint]time.Time),
	}
}

// NewMemoryStore 创建记忆存储（兼容旧接口）
func NewMemoryStore(storage *storage.Storage, logger *slog.Logger) *MemoryStore {
	return NewMemoryStoreWithConfig(storage, logger, MemoryConfig{
		ConsolidationThreshold: 50,
		MaxMemoryAge:           30,
		MaxSessionMemories:     100,
		MaxUserMemories:        500,
	})
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

// SaveSessionMemory 保存会话级别记忆
func (m *MemoryStore) SaveSessionMemory(sessionID uint, key, content string, tags ...string) error {
	sessionIDPtr := &sessionID
	memory := &storage.Memory{
		Type:      "session",
		Key:       fmt.Sprintf("session_%d_%s", sessionID, key),
		Content:   content,
		SessionID: sessionIDPtr,
		Tags:      "," + strings.Join(tags, ",") + ",",
	}
	return m.storage.CreateMemory(memory)
}

// GetSessionMemories 获取会话级别记忆
func (m *MemoryStore) GetSessionMemories(sessionID uint) ([]storage.Memory, error) {
	return m.storage.GetMemoriesBySessionID(sessionID)
}

// SaveUserMemory 保存用户级别记忆
func (m *MemoryStore) SaveUserMemory(userID, key, content string, tags ...string) error {
	memory := &storage.Memory{
		Type:     "user",
		Key:      fmt.Sprintf("user_%s_%s", userID, key),
		Content:  content,
		UserID:   userID,
		Tags:     "," + strings.Join(tags, ",") + ",",
	}
	return m.storage.CreateMemory(memory)
}

// GetUserMemories 获取用户级别记忆
func (m *MemoryStore) GetUserMemories(userID string) ([]storage.Memory, error) {
	return m.storage.GetMemoriesByUserID(userID)
}

// SavePinnedMemory 保存置顶记忆
func (m *MemoryStore) SavePinnedMemory(key, content string) error {
	memory := &storage.Memory{
		Type:       "memory",
		Key:        key,
		Content:    content,
		IsPinned:   true,
		Importance: 10,
	}
	return m.storage.CreateMemory(memory)
}

// GetPinnedMemories 获取所有置顶记忆
func (m *MemoryStore) GetPinnedMemories() ([]storage.Memory, error) {
	return m.storage.GetPinnedMemories()
}

// DeleteSessionMemory 删除会话记忆
func (m *MemoryStore) DeleteSessionMemory(sessionID uint, key string) error {
	fullKey := fmt.Sprintf("session_%d_%s", sessionID, key)
	memory, err := m.storage.GetMemoryByKey(fullKey)
	if err != nil {
		return err
	}
	return m.storage.SoftDeleteMemory(memory.ID)
}

// DeleteUserMemory 删除用户记忆
func (m *MemoryStore) DeleteUserMemory(userID, key string) error {
	fullKey := fmt.Sprintf("user_%s_%s", userID, key)
	memory, err := m.storage.GetMemoryByKey(fullKey)
	if err != nil {
		return err
	}
	return m.storage.SoftDeleteMemory(memory.ID)
}

// ClearSessionMemories 清除会话所有记忆
func (m *MemoryStore) ClearSessionMemories(sessionID uint) error {
	return m.storage.ClearSessionMemories(sessionID)
}

// ClearUserMemories 清除用户所有记忆
func (m *MemoryStore) ClearUserMemories(userID string) error {
	return m.storage.ClearUserMemories(userID)
}

// GetDetailedMemoryStats 获取详细记忆统计信息
func (m *MemoryStore) GetDetailedMemoryStats() (*MemoryStats, error) {
	stats := &MemoryStats{}

	// 统计总数
	memories, err := m.storage.GetAllMemories()
	if err != nil {
		return nil, err
	}

	// 过滤未删除的记忆
	var validMemories []storage.Memory
	for _, mem := range memories {
		if !mem.IsDeleted {
			validMemories = append(validMemories, mem)
		}
	}

	stats.Total = int64(len(validMemories))

	// 按类型统计
	for _, mem := range validMemories {
		switch mem.Type {
		case "memory":
			stats.Memory++
		case "history":
			stats.History++
		case "session":
			stats.Session++
		case "user":
			stats.User++
		}
		if mem.IsPinned {
			stats.Pinned++
		}
	}

	// 统计过期记忆
	expired, err := m.storage.GetExpiredMemories()
	if err == nil {
		stats.Expired = int64(len(expired))
	}

	return stats, nil
}

// SearchWithFilters 带过滤条件的搜索
func (m *MemoryStore) SearchWithFilters(query string, memType string, sessionID *uint, userID string, tags []string, limit int) ([]storage.Memory, error) {
	// 基础搜索
	memories, err := m.storage.SearchMemories(query)
	if err != nil {
		return nil, err
	}

	// 过滤结果
	var filtered []storage.Memory
	for _, mem := range memories {
		// 类型过滤
		if memType != "" && mem.Type != memType {
			continue
		}
		// 会话过滤
		if sessionID != nil && (mem.SessionID == nil || *mem.SessionID != *sessionID) {
			continue
		}
		// 用户过滤
		if userID != "" && mem.UserID != userID {
			continue
		}
		// 标签过滤
		if len(tags) > 0 {
			memTags := mem.GetTags()
			hasTag := false
			for _, tag := range tags {
				for _, memTag := range memTags {
					if tag == memTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		filtered = append(filtered, mem)
	}

	// 限制数量
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

// CleanupExpiredMemories 清理过期记忆
func (m *MemoryStore) CleanupExpiredMemories() (int, error) {
	expired, err := m.storage.GetExpiredMemories()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, mem := range expired {
		if err := m.storage.SoftDeleteMemory(mem.ID); err != nil {
			m.logger.Warn("Failed to delete expired memory", "id", mem.ID, "error", err)
			continue
		}
		count++
	}

	m.logger.Info("Cleaned up expired memories", "count", count)
	return count, nil
}

// EnforceSessionLimit 强制会话记忆数量限制
func (m *MemoryStore) EnforceSessionLimit(sessionID uint) error {
	memories, err := m.storage.GetMemoriesBySessionID(sessionID)
	if err != nil {
		return err
	}

	if len(memories) > m.config.MaxSessionMemories {
		// 删除最旧的非置顶记忆
		var toDelete []uint
		for _, mem := range memories {
			if !mem.IsPinned {
				toDelete = append(toDelete, mem.ID)
				if len(memories)-len(toDelete) <= m.config.MaxSessionMemories {
					break
				}
			}
		}
		if len(toDelete) > 0 {
			return m.storage.BatchDeleteMemories(toDelete)
		}
	}
	return nil
}

// EnforceUserLimit 强制用户记忆数量限制
func (m *MemoryStore) EnforceUserLimit(userID string) error {
	memories, err := m.storage.GetMemoriesByUserID(userID)
	if err != nil {
		return err
	}

	if len(memories) > m.config.MaxUserMemories {
		// 删除最旧的非置顶记忆
		var toDelete []uint
		for _, mem := range memories {
			if !mem.IsPinned {
				toDelete = append(toDelete, mem.ID)
				if len(memories)-len(toDelete) <= m.config.MaxUserMemories {
					break
				}
			}
		}
		if len(toDelete) > 0 {
			return m.storage.BatchDeleteMemories(toDelete)
		}
	}
	return nil
}

// StartCleanupScheduler 启动定期清理调度器
func (m *MemoryStore) StartCleanupScheduler(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 清理过期记忆
				if _, err := m.CleanupExpiredMemories(); err != nil {
					m.logger.Error("Failed to cleanup expired memories", "error", err)
				}
				// 清理旧历史
				before := time.Now().AddDate(0, 0, -m.config.MaxMemoryAge)
				if err := m.CleanOldHistory(before); err != nil {
					m.logger.Error("Failed to clean old history", "error", err)
				}
			}
		}
	}()
}

// GetRelevantMemoriesForSession 获取会话相关记忆
func (m *MemoryStore) GetRelevantMemoriesForSession(sessionID uint, query string, limit int) ([]string, error) {
	// 1. 获取会话级别记忆
	sessionMemories, err := m.storage.GetMemoriesBySessionID(sessionID)
	if err != nil {
		return nil, err
	}

	// 2. 搜索相关内容
	searchResults, err := m.storage.SearchMemories(query)
	if err != nil {
		return nil, err
	}

	// 3. 合并结果，优先使用会话级别记忆
	var results []string

	// 添加会话记忆
	for _, mem := range sessionMemories {
		if !mem.IsDeleted {
			results = append(results, fmt.Sprintf("[会话记忆] %s", mem.Content))
		}
	}

	// 添加搜索结果
	for _, mem := range searchResults {
		if !mem.IsDeleted && mem.Type != "session" {
			results = append(results, fmt.Sprintf("[%s] %s", mem.Type, mem.Content))
		}
	}

	// 限制数量
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
