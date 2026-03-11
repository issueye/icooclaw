package storage

import (
	"fmt"
	"time"

	icooclawErrors "icooclaw/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Session represents a chat session.
type Session struct {
	Model
	SessionID  string    `gorm:"column:session_id;type:char(36);uniqueIndex;not null;comment:会话ID" json:"session_id"`
	Channel    string    `gorm:"column:channel;type:varchar(50);not null;index:idx_session_channel_chat;comment:渠道" json:"channel"`
	ChatID     string    `gorm:"column:chat_id;type:varchar(100);not null;index:idx_session_channel_chat;comment:聊天ID" json:"chat_id"`
	AgentName  string    `gorm:"column:agent_name;type:varchar(100);comment:代理名称" json:"agent_name"`
	Context    string    `gorm:"column:context;type:text;comment:上下文(JSON格式)" json:"context"`
	Summary    string    `gorm:"column:summary;type:text;comment:会话摘要" json:"summary"`
	LastActive time.Time `gorm:"column:last_active;type:datetime;comment:最后活跃时间" json:"last_active"`
}

// TableName returns the table name for Session.
func (Session) TableName() string {
	return tableNamePrefix + "sessions"
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

type SessionStorage struct {
	db *gorm.DB
}

func NewSessionStorage(db *gorm.DB) *SessionStorage {
	return &SessionStorage{db: db}
}

// Save saves a session.
func (s *SessionStorage) Save(sess *Session) error {
	sess.LastActive = time.Now()
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"channel", "chat_id", "agent_name", "context", "summary", "last_active"}),
	}).Create(sess)
	return result.Error
}

// Get gets a session by session ID.
func (s *SessionStorage) Get(sessionID string) (*Session, error) {
	var sess Session
	result := s.db.Where("session_id = ?", sessionID).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// GetByChat gets a session by channel and chat ID.
func (s *SessionStorage) GetByChat(channel, chatID string) (*Session, error) {
	var sess Session
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// Delete deletes a session.
func (s *SessionStorage) Delete(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	return nil
}

// Page gets sessions with pagination.
func (s *SessionStorage) Page(query *QuerySession) (*ResQuerySession, error) {
	var res ResQuerySession

	qry := s.db.Model(&Session{}).
		Where("channel = ? AND (agent_name LIKE ? OR chat_id LIKE ?)",
			query.Channel, "%"+query.KeyWord+"%", "%"+query.KeyWord+"%").
		Order("last_active DESC")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", result.Error)
	}

	// 分页查询
	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", result.Error)
	}

	return &res, nil
}
