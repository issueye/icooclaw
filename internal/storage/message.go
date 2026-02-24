package storage

import (
	"time"
)

// Message 消息模型
type Message struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	SessionID        uint      `gorm:"index" json:"session_id"`
	Role             string    `gorm:"size:20;index" json:"role"` // user, assistant, system, tool
	Content          string    `gorm:"type:text" json:"content"`
	ToolCalls        string    `gorm:"type:text" json:"tool_calls"`        // JSON数组
	ToolCallID       string    `gorm:"size:100" json:"tool_call_id"`       // 工具调用ID
	ToolName         string    `gorm:"size:100" json:"tool_name"`          // 工具名称
	ReasoningContent string    `gorm:"type:text" json:"reasoning_content"` // 思考过程
	Timestamp        time.Time `gorm:"index" json:"timestamp"`
	CreatedAt        time.Time `json:"created_at"`
}

// TableName 表名
func (Message) TableName() string {
	return "messages"
}

// Create 创建消息
func (m *Message) Create() error {
	return DB.Create(m).Error
}

// GetByID 通过ID获取消息
func GetMessageByID(id uint) (*Message, error) {
	var msg Message
	err := DB.First(&msg, id).Error
	return &msg, err
}

// GetBySessionID 通过会话ID获取消息
func GetMessagesBySessionID(sessionID uint, limit, offset int) ([]Message, error) {
	var messages []Message
	err := DB.Where("session_id = ?", sessionID).Order("timestamp DESC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

// GetToolMessages 获取工具消息
func GetToolMessagesBySessionID(sessionID uint) ([]Message, error) {
	var messages []Message
	err := DB.Where("session_id = ? AND role = ?", sessionID, "tool").Order("timestamp ASC").Find(&messages).Error
	return messages, err
}

// DeleteBySessionID 删除会话的所有消息
func DeleteMessagesBySessionID(sessionID uint) error {
	return DB.Where("session_id = ?", sessionID).Delete(&Message{}).Error
}
