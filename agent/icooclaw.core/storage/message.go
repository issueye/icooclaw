package storage

import "gorm.io/gorm"

// Message 消息模型
type Message struct {
	Model            `gorm:"embedded"`
	SessionID        uint   `gorm:"index" json:"session_id"`
	Role             string `gorm:"size:20;index" json:"role"` // user, assistant, system, tool
	Content          string `gorm:"type:text" json:"content"`
	ToolCalls        string `gorm:"type:text" json:"tool_calls"`        // JSON数组
	ToolCallID       string `gorm:"size:100" json:"tool_call_id"`       // 工具调用ID
	ToolName         string `gorm:"size:100" json:"tool_name"`          // 工具名称
	ReasoningContent string `gorm:"type:text" json:"reasoning_content"` // 思考过程
}

// TableName 表名
func (Message) TableName() string {
	return tableNamePrefix + "messages"
}

// MessageStorage 消息存储
type MessageStorage struct {
	db *gorm.DB
}

// NewMessageStorage 创建消息存储
func NewMessageStorage(db *gorm.DB) *MessageStorage {
	return &MessageStorage{db: db}
}

// CreateOrUpdate 创建或更新消息
func (s *MessageStorage) CreateOrUpdate(msg *Message) error {
	return s.db.Save(msg).Error
}

// Create 创建消息
func (s *MessageStorage) Create(msg *Message) error {
	return s.db.Create(msg).Error
}

// GetByID 通过ID获取消息
func (s *MessageStorage) GetByID(id uint) (*Message, error) {
	var msg Message
	err := s.db.First(&msg, id).Error
	return &msg, err
}

// GetBySessionID 通过会话ID获取消息
func (s *MessageStorage) GetBySessionID(sessionID uint, limit, offset int) ([]Message, error) {
	var messages []Message
	err := s.db.Where("session_id = ?", sessionID).Order("created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error
	return messages, err
}

// GetToolMessages 获取工具消息
func (s *MessageStorage) GetToolMessages(sessionID uint) ([]Message, error) {
	var messages []Message
	err := s.db.Where("session_id = ? AND role = ?", sessionID, "tool").Order("created_at ASC").Find(&messages).Error
	return messages, err
}

// DeleteBySessionID 删除会话的所有消息
func (s *MessageStorage) DeleteBySessionID(sessionID uint) error {
	return s.db.Where("session_id = ?", sessionID).Delete(&Message{}).Error
}

// Delete 删除消息
func (s *MessageStorage) Delete(id uint) error {
	return s.db.Delete(&Message{}, id).Error
}

// Update 更新消息
func (s *MessageStorage) Update(msg *Message) error {
	return s.db.Save(msg).Error
}

// Page 分页获取消息
func (s *MessageStorage) Page(q *QueryMessage) (*ResQueryMessage, error) {
	var total int64
	query := s.db.Model(&Message{})
	if q.SessionID > 0 {
		query = query.Where("session_id = ?", q.SessionID)
	}
	if q.Role != "" {
		query = query.Where("role = ?", q.Role)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var messages []Message
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&messages).Error

	q.Page.Total = int(total)
	return &ResQueryMessage{
		Page:    q.Page,
		Records: messages,
	}, err
}

// QueryMessage 消息查询参数
type QueryMessage struct {
	Page      Page   `json:"page"`
	SessionID uint   `json:"session_id"`
	Role      string `json:"role"`
}

// ResQueryMessage 消息查询结果
type ResQueryMessage struct {
	Page    Page      `json:"page"`
	Records []Message `json:"records"`
}
