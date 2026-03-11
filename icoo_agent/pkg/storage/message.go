package storage

import (
	"fmt"
	"icooclaw/pkg/consts"

	"gorm.io/gorm"
)

// Message represents a chat message.
type Message struct {
	Model
	SessionID string          `gorm:"column:session_id;type:char(36);not null;index;comment:会话ID" json:"session_id"`
	Role      consts.RoleType `gorm:"column:role;type:varchar(50);not null;comment:角色(user/assistant/system)" json:"role"`
	Content   string          `gorm:"column:content;type:text;not null;comment:消息内容" json:"content"`
	Metadata  string          `gorm:"column:metadata;type:text;comment:元数据(JSON格式)" json:"metadata"`
}

// TableName returns the table name for Message.
func (Message) TableName() string {
	return tableNamePrefix + "messages"
}

type QueryMessage struct {
	Page      Page   `json:"page"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	KeyWord   string `json:"key_word"`
}

type ResQueryMessage struct {
	Page    Page      `json:"page"`
	Records []Message `json:"records"`
}

type MessageStorage struct {
	db *gorm.DB
}

func NewMessageStorage(db *gorm.DB) *MessageStorage {
	return &MessageStorage{db: db}
}

// Save saves a message.
func (s *MessageStorage) Save(m *Message) error {
	return s.db.Create(m).Error
}

// Get gets messages by session ID.
func (s *MessageStorage) Get(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 100
	}
	var messages []*Message
	result := s.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get messages: %w", result.Error)
	}
	return messages, nil
}

// GetByID gets a message by ID.
func (s *MessageStorage) GetByID(id string) (*Message, error) {
	var m Message
	result := s.db.Where("id = ?", id).First(&m)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("message not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}
	return &m, nil
}

// Delete deletes a message by ID.
func (s *MessageStorage) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Message{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete message: %w", result.Error)
	}
	return nil
}

// Page gets messages with pagination.
func (s *MessageStorage) Page(query *QueryMessage) (*ResQueryMessage, error) {
	var res ResQueryMessage

	qry := s.db.Model(&Message{})

	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}

	if query.Role != "" {
		qry = qry.Where("role = ?", query.Role)
	}

	if query.KeyWord != "" {
		qry = qry.Where("content LIKE ?", "%"+query.KeyWord+"%")
	}

	qry = qry.Order("created_at DESC")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count messages: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get messages: %w", result.Error)
	}

	return &res, nil
}
