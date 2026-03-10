package storage

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
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

// SaveSession saves a session.
func (s *Storage) SaveSession(sess *Session) error {
	sess.LastActive = time.Now()
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"channel", "chat_id", "agent_name", "context", "summary", "last_active"}),
	}).Create(sess)
	return result.Error
}

// GetSession gets a session by session ID.
func (s *Storage) GetSession(sessionID string) (*Session, error) {
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

// GetSessionByChat gets a session by channel and chat ID.
func (s *Storage) GetSessionByChat(channel, chatID string) (*Session, error) {
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

// DeleteSession deletes a session.
func (s *Storage) DeleteSession(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	return nil
}
