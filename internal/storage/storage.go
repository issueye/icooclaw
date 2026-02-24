package storage

import (
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
	err := s.db.Where("content LIKE ?", "%"+query+"%").Find(&memories).Error
	return memories, err
}
