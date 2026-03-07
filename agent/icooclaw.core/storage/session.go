package storage

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"icooclaw.core/consts"
)

// Session 会话模型
type Session struct {
	Model
	Key              string `gorm:"size:255" json:"key"`                // channel:chat_id
	Channel          string `gorm:"size:50;index" json:"channel"`       // telegram, discord, feishu...
	ChatID           string `gorm:"size:255;index" json:"chat_id"`      // 用户/群组ID
	UserID           string `gorm:"size:255" json:"user_id"`            // 用户唯一标识
	LastConsolidated int    `gorm:"default:0" json:"last_consolidated"` // 已整合的消息数
	Metadata         string `gorm:"type:text" json:"metadata"`          // JSON元数据

	Messages []Message `gorm:"foreignKey:SessionID" json:"messages"`
}

type QuerySession struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Channel string `json:"channel"`
	UserID  string `json:"user_id"`
}

type ResQuerySession struct {
	Page    Page      `json:"page"`
	Records []Session `json:"records"`
}

// TableName 表名
func (Session) TableName() string {
	return tableNamePrefix + "sessions"
}

// BeforeCreate 创建前回调
func (c *Session) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// SessionStorage 会话存储
type SessionStorage struct {
	db *gorm.DB
}

func NewSessionStorage(db *gorm.DB) *SessionStorage {
	return &SessionStorage{db: db}
}

func (s *SessionStorage) CreateOrUpdate(session *Session) error {
	return s.db.Save(session).Error
}

// Create 创建会话
func (s *SessionStorage) Create(session *Session) error {
	return s.db.Create(session).Error
}

// GetByID 通过ID获取会话
func (s *SessionStorage) GetByID(id string) (*Session, error) {
	var session Session
	err := s.db.Where("id = ?", id).First(&session).Error
	return &session, err
}

func (s *SessionStorage) GetOrCreateSession(channel, chatID, userID string) (*Session, error) {
	session := &Session{
		Key:     channel + ":" + chatID,
		Channel: channel,
		ChatID:  chatID,
		UserID:  userID,
	}
	err := s.CreateOrUpdate(session)
	return session, err
}

// GetByName 通过名称获取会话
func (s *SessionStorage) GetByName(name string) (*Session, error) {
	var session Session
	err := s.db.Where("key = ?", name).First(&session).Error
	return &session, err
}

// Delete 删除会话（按 uint ID）
func (s *SessionStorage) Delete(id uint) error {
	return s.db.Delete(&Session{}, id).Error
}

// DeleteByID 删除会话（按字符串 ID）
func (s *SessionStorage) DeleteByID(id string) error {
	return s.db.Where("id = ?", id).Delete(&Session{}).Error
}

// GetAll 获取所有会话
func (s *SessionStorage) GetAll() ([]Session, error) {
	var sessions []Session
	err := s.db.Find(&sessions).Error
	return sessions, err
}

// Page 分页获取会话
func (s *SessionStorage) Page(q *QuerySession) (*ResQuerySession, error) {
	var total int64
	query := s.db.Model(&Session{})
	if q.KeyWord != "" {
		query = query.Where("key LIKE ? OR channel LIKE ? OR user_id LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var sessions []Session
	err := query.
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&sessions).Error

	q.Page.Total = int(total)
	return &ResQuerySession{
		Page:    q.Page,
		Records: sessions,
	}, err
}

// AddMessage 添加消息到会话
func (s *SessionStorage) AddMessage(sessionID string, role consts.RoleType, content, reasoningContent, toolCallID, toolName, toolArguments, toolResult string) (*Message, error) {
	msg := Message{
		SessionID:        sessionID,
		Role:             role,
		Content:          content,
		ToolCallID:       toolCallID,
		ToolName:         toolName,
		ToolArguments:    toolArguments,
		ToolResult:       toolResult,
		ReasoningContent: reasoningContent,
	}
	err := s.db.Create(&msg).Error
	return &msg, err
}

func (s *SessionStorage) UpdateSessionMetadata(sessionID string, metadata string) error {
	return s.db.Model(&Session{}).Where("id = ?", sessionID).Update("metadata", metadata).Error
}

// GetMessages 获取会话消息
func (s *SessionStorage) GetMessages(id string, limit int) ([]Message, error) {
	var messages []Message
	err := s.db.Where("session_id = ?", id).Order("created_at ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

// UpdateLastConsolidated 更新已整合的消息数
func (s *SessionStorage) UpdateLastConsolidated(id string) error {
	var count int64
	err := s.db.Model(&Message{}).Where("session_id = ?", id).Count(&count).Error
	if err != nil {
		return err
	}
	return s.db.Model(&Session{}).Where("id = ?", id).Update("last_consolidated", count).Error
}
