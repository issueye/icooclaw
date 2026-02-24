package storage

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库连接实例
var DB *gorm.DB

// InitDB 初始化数据库
func InitDB(dsn string) (*gorm.DB, error) {
	var err error

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// 自动迁移
	err = DB.AutoMigrate(
		&Session{},
		&Message{},
		&Task{},
		&Skill{},
		&Memory{},
		&ChannelConfig{},
		&ProviderConfig{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return DB, nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// Session 会话模型
type Session struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Key              string    `gorm:"uniqueIndex;size:255" json:"key"`    // channel:chat_id
	Channel          string    `gorm:"size:50;index" json:"channel"`       // telegram, discord, feishu...
	ChatID           string    `gorm:"size:255;index" json:"chat_id"`      // 用户/群组ID
	UserID           string    `gorm:"size:255" json:"user_id"`            // 用户唯一标识
	LastConsolidated int       `gorm:"default:0" json:"last_consolidated"` // 已整合的消息数
	Metadata         string    `gorm:"type:text" json:"metadata"`          // JSON元数据
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	Messages []Message `gorm:"foreignKey:SessionID" json:"messages"`
}

// TableName 表名
func (Session) TableName() string {
	return "sessions"
}

// Create 创建会话
func (s *Session) Create() error {
	return DB.Create(s).Error
}

// GetByKey 通过Key获取会话
func GetSessionByKey(key string) (*Session, error) {
	var session Session
	err := DB.Where("key = ?", key).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetOrCreateByChannelChatID 通过通道和聊天ID获取或创建会话
func GetOrCreateSession(channel, chatID, userID string) (*Session, error) {
	key := fmt.Sprintf("%s:%s", channel, chatID)
	session, err := GetSessionByKey(key)
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
		err = session.Create()
		return session, err
	}

	return nil, err
}

// AddMessage 添加消息到会话
func (s *Session) AddMessage(role, content, toolCalls, toolCallID, toolName, reasoningContent string) (*Message, error) {
	msg := Message{
		SessionID:        s.ID,
		Role:             role,
		Content:          content,
		ToolCalls:        toolCalls,
		ToolCallID:       toolCallID,
		ToolName:         toolName,
		ReasoningContent: reasoningContent,
		Timestamp:        time.Now(),
	}
	err := DB.Create(&msg).Error
	return &msg, err
}

// GetMessages 获取会话消息
func (s *Session) GetMessages(limit int) ([]Message, error) {
	var messages []Message
	err := DB.Where("session_id = ?", s.ID).Order("timestamp ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

// UpdateLastConsolidated 更新已整合的消息数
func (s *Session) UpdateLastConsolidated() error {
	var count int64
	DB.Model(&Message{}).Where("session_id = ?", s.ID).Count(&count)
	return DB.Model(s).Update("last_consolidated", count).Error
}
