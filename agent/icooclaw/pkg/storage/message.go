package storage

import (
	"fmt"
	"icooclaw/pkg/consts"

	"gorm.io/gorm"
)

// Message 消息模型
type Message struct {
	Model
	SessionID        string          `gorm:"column:session_id;index" json:"session_id"`
	Role             consts.RoleType `gorm:"column:role;size:20;index" json:"role"`                       // user, assistant, system, tool_call, tool_result
	Content          string          `gorm:"column:content;type:text" json:"content"`                     // 消息内容
	ToolCallID       string          `gorm:"column:tool_call_id;size:100" json:"tool_call_id"`            // 工具调用ID
	ToolName         string          `gorm:"column:tool_name;size:100" json:"tool_name"`                  // 工具名称
	ToolArguments    string          `gorm:"column:tool_arguments;type:text" json:"tool_arguments"`       // 工具参数
	ToolResult       string          `gorm:"column:tool_result;type:text" json:"tool_result"`             // 工具结果
	ToolResultError  string          `gorm:"column:tool_result_error;type:text" json:"tool_result_error"` // 工具结果错误
	Thinking         string          `gorm:"column:thinking;type:text" json:"thinking"`                   // 思考过程
	ReasoningContent string          `gorm:"column:reasoning_content;type:text" json:"reasoning_content"` // 思考过程
	Metadata         string          `gorm:"column:metadata;type:text" json:"metadata"`                   // 元数据
}

// TableName 表名
func (Message) TableName() string {
	return tableNamePrefix + "messages"
}

// QueryMessage 消息查询参数
type QueryMessage struct {
	Page      Page   `json:"page"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
}

// ResQueryMessage 消息查询结果
type ResQueryMessage struct {
	Page    Page      `json:"page"`
	Records []Message `json:"records"`
}

func NewUserMessage(sessionID string, content string) *Message {
	return &Message{
		SessionID: sessionID,
		Role:      consts.RoleUser,
		Content:   content,
	}
}

func NewAssistantMessage(sessionID string, content string, reasoningContent string) *Message {
	return &Message{
		SessionID:        sessionID,
		Role:             consts.RoleAssistant,
		Content:          content,
		ReasoningContent: reasoningContent,
	}
}

type MessageStorage struct {
	db *gorm.DB
}

func NewMessageStorage(db *gorm.DB) *MessageStorage {
	return &MessageStorage{db: db}
}

// Save saves a message entry.
func (s *MessageStorage) Save(m *Message) error {
	return s.db.Create(m).Error
}

// Get gets message entries for a session.
func (s *MessageStorage) Get(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 100
	}
	var memories []*Message
	result := s.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&memories)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}
	return memories, nil
}

func (s *MessageStorage) GetByID(id string) (*Message, error) {
	var m Message
	result := s.db.Where("id = ?", id).Find(&m)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}
	return &m, nil
}

// Delete deletes message entries for a session.
func (s *MessageStorage) Delete(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Message{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete message: %w", result.Error)
	}
	return nil
}

func (s *MessageStorage) Page(query *QueryMessage) (*ResQueryMessage, error) {
	var res ResQueryMessage

	qry := s.db.Model(&Message{})

	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}

	qry = qry.Order("created_at DESC")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count memories: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get message: %w", result.Error)
	}
	return &ResQueryMessage{
		Page:    query.Page,
		Records: res.Records,
	}, nil
}
