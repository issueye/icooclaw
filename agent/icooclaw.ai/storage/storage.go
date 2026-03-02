package storage

import (
	"time"

	"gorm.io/gorm"
)

// Storage 存储封装
type Storage struct {
	db *gorm.DB
}

// NewStorage 创建存储实例
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{db: db}
}

// DB 返回原生数据库实例
func (s *Storage) DB() *gorm.DB {
	return s.db
}

// Session operations

// CreateSession 创建会话
func (s *Storage) CreateSession(session *Session) error {
	return s.db.Create(session).Error
}

// GetSessionByKey 通过Key获取会话
func (s *Storage) GetSessionByKey(key string) (*Session, error) {
	var session Session
	err := s.db.Where("key = ?", key).First(&session).Error
	return &session, err
}

// GetOrCreateSession 获取或创建会话
func (s *Storage) GetOrCreateSession(channel, chatID, userID string) (*Session, error) {
	key := channel + ":" + chatID
	session, err := s.GetSessionByKey(key)
	if err == nil {
		return session, nil
	}

	if err == gorm.ErrRecordNotFound {
		session = &Session{
			Key:     key,
			Channel: channel,
			ChatID:  chatID,
			UserID:  userID,
		}
		err = s.db.Create(session).Error
		return session, err
	}

	return nil, err
}

// AddMessage 添加消息
func (s *Storage) AddMessage(sessionID uint, role, content, toolCalls, toolCallID, toolName, reasoningContent string) (*Message, error) {
	msg := Message{
		SessionID:        sessionID,
		Role:             role,
		Content:          content,
		ToolCalls:        toolCalls,
		ToolCallID:       toolCallID,
		ToolName:         toolName,
		ReasoningContent: reasoningContent,
	}
	err := s.db.Create(&msg).Error
	return &msg, err
}

// GetSessionMessages 获取会话消息
func (s *Storage) GetSessionMessages(sessionID uint, limit int) ([]Message, error) {
	var messages []Message
	err := s.db.Where("session_id = ?", sessionID).Order("timestamp ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

// GetMessages 获取分页消息
func (s *Storage) GetMessages(sessionID string, limit, offset int) ([]Message, error) {
	var messages []Message
	err := s.db.Where("session_id = ?", sessionID).Order("timestamp ASC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

// GetSessions 获取会话列表
func (s *Storage) GetSessions(userID, channel string) ([]Session, error) {
	var sessions []Session
	db := s.db
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if channel != "" {
		db = db.Where("channel = ?", channel)
	}
	err := db.Order("updated_at DESC").Find(&sessions).Error
	return sessions, err
}

// UpdateSessionMetadata 更新会话元数据
func (s *Storage) UpdateSessionMetadata(sessionID uint, metadata string) error {
	return s.db.Model(&Session{}).Where("id = ?", sessionID).Update("metadata", metadata).Error
}

// GetSession 获取会话
func (s *Storage) GetSession(sessionID uint) (*Session, error) {
	var session Session
	err := s.db.First(&session, sessionID).Error
	return &session, err
}

// DeleteSession 删除会话
func (s *Storage) DeleteSession(sessionID string) error {
	// 删除相关的消息
	err := s.db.Where("session_id = ?", sessionID).Delete(&Message{}).Error
	if err != nil {
		return err
	}
	// 删除会话本身
	return s.db.Where("id = ?", sessionID).Delete(&Session{}).Error
}

// Task operations

// CreateTask 创建任务
func (s *Storage) CreateTask(task *Task) error {
	return s.db.Create(task).Error
}

// GetTaskByName 通过名称获取任务
func (s *Storage) GetTaskByName(name string) (*Task, error) {
	var task Task
	err := s.db.Where("name = ?", name).First(&task).Error
	return &task, err
}

// GetAllTasks 获取所有任务
func (s *Storage) GetAllTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Find(&tasks).Error
	return tasks, err
}

// GetEnabledTasks 获取启用任务
func (s *Storage) GetEnabledTasks() ([]Task, error) {
	var tasks []Task
	err := s.db.Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// UpdateTask 更新任务
func (s *Storage) UpdateTask(task *Task) error {
	return s.db.Save(task).Error
}

// DeleteTask 删除任务
func (s *Storage) DeleteTask(id uint) error {
	return s.db.Delete(&Task{}, id).Error
}

// Skill operations

// CreateSkill 创建技能
func (s *Storage) CreateSkill(skill *Skill) error {
	return s.db.Create(skill).Error
}

// GetSkillByName 通过名称获取技能
func (s *Storage) GetSkillByName(name string) (*Skill, error) {
	var skill Skill
	err := s.db.Where("name = ?", name).First(&skill).Error
	return &skill, err
}

// GetAllSkills 获取所有技能
func (s *Storage) GetAllSkills() ([]Skill, error) {
	var skills []Skill
	err := s.db.Find(&skills).Error
	return skills, err
}

// GetEnabledSkills 获取启用技能
func (s *Storage) GetEnabledSkills() ([]Skill, error) {
	var skills []Skill
	err := s.db.Where("enabled = ?", true).Find(&skills).Error
	return skills, err
}

// UpdateSkill 更新技能
func (s *Storage) UpdateSkill(skill *Skill) error {
	return s.db.Save(skill).Error
}

// DeleteSkill 删除技能
func (s *Storage) DeleteSkill(id uint) error {
	return s.db.Delete(&Skill{}, id).Error
}

// GetSkillByID 通过ID获取技能
func (s *Storage) GetSkillByID(id uint) (*Skill, error) {
	var skill Skill
	err := s.db.First(&skill, id).Error
	return &skill, err
}

// Memory operations

// CreateMemory 创建记忆
func (s *Storage) CreateMemory(memory *Memory) error {
	return s.db.Create(memory).Error
}

// GetMemoryByKey 通过Key获取记忆
func (s *Storage) GetMemoryByKey(key string) (*Memory, error) {
	var memory Memory
	err := s.db.Where("key = ?", key).First(&memory).Error
	return &memory, err
}

// GetAllMemories 获取所有记忆
func (s *Storage) GetAllMemories() ([]Memory, error) {
	var memories []Memory
	err := s.db.Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// UpdateMemory 更新记忆
func (s *Storage) UpdateMemory(memory *Memory) error {
	return s.db.Save(memory).Error
}

// DeleteMemory 删除记忆
func (s *Storage) DeleteMemory(id uint) error {
	return s.db.Delete(&Memory{}, id).Error
}

// SearchMemories 搜索记忆
func (s *Storage) SearchMemories(query string) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("is_deleted = ? AND (content LIKE ? OR tags LIKE ?)", false, "%"+query+"%", "%"+query+"%").Order("importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetMemoriesByTypeAndSession 按类型和会话获取记忆
func (s *Storage) GetMemoriesByTypeAndSession(memType string, sessionID uint) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("type = ? AND session_id = ? AND is_deleted = ?", memType, sessionID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetMemoriesByUserID 按用户ID获取记忆
func (s *Storage) GetMemoriesByUserID(userID string) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("user_id = ? AND is_deleted = ?", userID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetMemoriesBySessionID 按会话ID获取记忆
func (s *Storage) GetMemoriesBySessionID(sessionID uint) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("session_id = ? AND is_deleted = ?", sessionID, false).Order("is_pinned DESC, importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetPinnedMemories 获取置顶记忆
func (s *Storage) GetPinnedMemories() ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("is_pinned = ? AND is_deleted = ?", true, false).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetExpiredMemories 获取过期记忆
func (s *Storage) GetExpiredMemories() ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("expires_at IS NOT NULL AND expires_at < ? AND is_deleted = ?", time.Now(), false).Find(&memories).Error
	return memories, err
}

// SoftDeleteMemory 软删除记忆
func (s *Storage) SoftDeleteMemory(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_deleted", true).Error
}

// RestoreMemory 恢复记忆
func (s *Storage) RestoreMemory(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_deleted", false).Error
}

// PinMemory 置顶记忆
func (s *Storage) PinMemory(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_pinned", true).Error
}

// UnpinMemory 取消置顶
func (s *Storage) UnpinMemory(id uint) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("is_pinned", false).Error
}

// UpdateMemoryImportance 更新记忆重要性
func (s *Storage) UpdateMemoryImportance(id uint, importance int) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("importance", importance).Error
}

// SetMemoryTags 设置记忆标签
func (s *Storage) SetMemoryTags(id uint, tags string) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("tags", tags).Error
}

// SetMemoryExpiration 设置记忆过期时间
func (s *Storage) SetMemoryExpiration(id uint, expiresAt *time.Time) error {
	return s.db.Model(&Memory{}).Where("id = ?", id).Update("expires_at", expiresAt).Error
}

// BatchCreateMemories 批量创建记忆
func (s *Storage) BatchCreateMemories(memories []*Memory) error {
	if len(memories) == 0 {
		return nil
	}
	return s.db.Create(memories).Error
}

// BatchDeleteMemories 批量删除记忆
func (s *Storage) BatchDeleteMemories(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.Model(&Memory{}).Where("id IN ?", ids).Update("is_deleted", true).Error
}

// CountMemoriesByType 按类型统计记忆数量
func (s *Storage) CountMemoriesByType(memType string) (int64, error) {
	var count int64
	err := s.db.Model(&Memory{}).Where("type = ? AND is_deleted = ?", memType, false).Count(&count).Error
	return count, err
}

// GetMemoriesByTags 按标签获取记忆
func (s *Storage) GetMemoriesByTags(tag string) ([]Memory, error) {
	var memories []Memory
	err := s.db.Where("tags LIKE ? AND is_deleted = ?", "%,"+tag+",%", false).Order("importance DESC, updated_at DESC").Find(&memories).Error
	return memories, err
}

// ClearSessionMemories 清除会话记忆
func (s *Storage) ClearSessionMemories(sessionID uint) error {
	return s.db.Where("session_id = ? AND type = ?", sessionID, "session").Delete(&Memory{}).Error
}

// ClearUserMemories 清除用户记忆
func (s *Storage) ClearUserMemories(userID string) error {
	return s.db.Where("user_id = ? AND type = ?", userID, "user").Delete(&Memory{}).Error
}

// GetMemoriesPaginated 分页获取记忆
func (s *Storage) GetMemoriesPaginated(memType string, page, pageSize int) ([]Memory, int64, error) {
	var memories []Memory
	var total int64

	query := s.db.Model(&Memory{}).Where("type = ? AND is_deleted = ?", memType, false)
	query.Count(&total)

	err := query.Order("is_pinned DESC, importance DESC, updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&memories).Error
	return memories, total, err
}
