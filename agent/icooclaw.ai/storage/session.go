package storage

import (
	"fmt"
)

// Session 会话模型
type Session struct {
	Model
	Key              string `gorm:"uniqueIndex;size:255" json:"key"`    // channel:chat_id
	Channel          string `gorm:"size:50;index" json:"channel"`       // telegram, discord, feishu...
	ChatID           string `gorm:"size:255;index" json:"chat_id"`      // 用户/群组ID
	UserID           string `gorm:"size:255" json:"user_id"`            // 用户唯一标识
	LastConsolidated int    `gorm:"default:0" json:"last_consolidated"` // 已整合的消息数
	Metadata         string `gorm:"type:text" json:"metadata"`          // JSON元数据

	Messages []Message `gorm:"foreignKey:SessionID" json:"messages"`
}

// TableName 表名
func (Session) TableName() string {
	return tableNamePrefix + "sessions"
}

// GenerateSessionKey 生成会话 Key
func GenerateSessionKey(channel, chatID string) string {
	return fmt.Sprintf("%s:%s", channel, chatID)
}

// Create 创建会话
func (s *Session) Create(data *Session) error {
	return DB.Create(data).Error
}

type QuerySession struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
}

type ResQuerySession struct {
	Page    Page      `json:"page"`
	Records []Session `json:"records"`
}

// GetSessions 获取会话列表
func (s *Session) Page(q *QuerySession) (*ResQuerySession, error) {

	total := 0
	err := DB.
		Model(&Session{}).
		Where("key LIKE ?", fmt.Sprintf("%%%s%%", q.KeyWord)).
		Count(&total).Error
	if err != nil {
		return nil, err
	}

	var sessions []Session

	err = DB.
		Where("key LIKE ?", fmt.Sprintf("%%%s%%", q.KeyWord)).
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&sessions).Error

	res := &ResQuerySession{
		Page:    q.Page,
		Records: sessions,
	}

	return res, err
}

// GetByID 根据ID获取会话
func (s *Session) GetByID(id uint) (*Session, error) {
	var session Session
	err := DB.Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// CreateOrUpdate 创建或更新会话
func (s *Session) CreateOrUpdate(data *Session) error {
	if data.ID == 0 {
		return data.Create()
	}

	return DB.Save(data).Error
}

// Delete 删除会话
func (s *Session) Delete(id uint) error {
	return DB.Delete(&Session{}, id).Error
}
