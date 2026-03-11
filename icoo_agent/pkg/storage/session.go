package storage

import (
	"fmt"
	"time"

	icooclawErrors "icooclaw/pkg/errors"

	"gorm.io/gorm"
)

// Session represents a chat session.
type Session struct {
	Model
	Channel    string    `gorm:"column:channel;type:varchar(50);not null;comment:渠道" json:"channel"`    // 渠道
	UserID     string    `gorm:"column:user_id;type:varchar(100);not null;comment:用户ID" json:"user_id"` // 用户ID
	Summary    string    `gorm:"column:summary;type:text;comment:会话摘要" json:"summary"`                  // 会话摘要
	Title      string    `gorm:"column:title;type:varchar(100);comment:会话标题" json:"title"`              // 会话标题
	LastActive time.Time `gorm:"column:last_active;type:datetime;comment:最后活跃时间" json:"last_active"`    // 最后活跃时间
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
	Page    Page       `json:"page"`
	Records []*Session `json:"records"`
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
	result := s.db.Save(sess)
	return result.Error
}

// Get gets a session by ID.
func (s *SessionStorage) Get(id string) (*Session, error) {
	var sess Session
	result := s.db.Where("id = ?", id).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// GetBySessionID gets a session by channel and session ID.
func (s *SessionStorage) GetBySessionID(channel, sessionID string) (*Session, error) {
	var sess Session
	result := s.db.Where("channel = ? AND id = ?", channel, sessionID).First(&sess)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get session: %w", result.Error)
	}
	return &sess, nil
}

// Delete deletes a session.
func (s *SessionStorage) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Session{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	return nil
}

// Page gets sessions with pagination.
func (s *SessionStorage) Page(query *QuerySession) (*ResQuerySession, error) {
	var res ResQuerySession

	qry := s.db.Model(&Session{}).
		Where("channel = ? AND (title LIKE ?)",
			query.Channel, "%"+query.KeyWord+"%").
		Order("last_active DESC")

	var count int64
	result := qry.Count(&count)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count sessions: %w", result.Error)
	}

	// 分页查询
	var sessions []*Session
	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&sessions)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&sessions)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", result.Error)
	}

	res.Page.Total = count
	res.Records = sessions

	return &res, nil
}
