package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// Memory represents a memory entry.
type Memory struct {
	Model
	SessionID string `gorm:"column:session_id;type:char(36);not null;index;comment:会话ID" json:"session_id"`
	Role      string `gorm:"column:role;type:varchar(50);not null;comment:角色(user/assistant/system)" json:"role"`
	Content   string `gorm:"column:content;type:text;not null;comment:消息内容" json:"content"`
	Metadata  string `gorm:"column:metadata;type:text;comment:元数据(JSON格式)" json:"metadata"` // JSON object
}

// TableName returns the table name for Memory.
func (Memory) TableName() string {
	return tableNamePrefix + "memory"
}

type QueryMemory struct {
	Page      Page   `json:"page"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Query     string `json:"query"`
}

type ResQueryMemory struct {
	Page    Page     `json:"page"`
	Records []Memory `json:"records"`
}

type MemoryStorage struct {
	db *gorm.DB
}

func NewMemoryStorage(db *gorm.DB) *MemoryStorage {
	return &MemoryStorage{db: db}
}

// Save saves a memory entry.
func (s *MemoryStorage) Save(m *Memory) error {
	return s.db.Create(m).Error
}

// Get gets memory entries for a session.
func (s *MemoryStorage) Get(sessionID string, limit int) ([]*Memory, error) {
	if limit <= 0 {
		limit = 100
	}
	var memories []*Memory
	result := s.db.Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&memories)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get memory: %w", result.Error)
	}
	return memories, nil
}

// Delete deletes memory entries for a session.
func (s *MemoryStorage) Delete(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Memory{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete memory: %w", result.Error)
	}
	return nil
}

// Page gets memories with pagination.
func (s *MemoryStorage) Page(query *QueryMemory) (*ResQueryMemory, error) {
	var res ResQueryMemory

	qry := s.db.Model(&Memory{})

	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}

	if query.Role != "" {
		qry = qry.Where("role = ?", query.Role)
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
		return nil, fmt.Errorf("failed to get memories: %w", result.Error)
	}

	return &res, nil
}

// Search searches for memories.
func (s *MemoryStorage) Search(query *QueryMemory) ([]*Memory, error) {
	var memories []*Memory

	qry := s.db.Model(&Memory{})

	if query.SessionID != "" {
		qry = qry.Where("session_id = ?", query.SessionID)
	}

	if query.Role != "" {
		qry = qry.Where("role = ?", query.Role)
	}

	if query.Query != "" {
		qry = qry.Where("content LIKE ?", "%"+query.Query+"%")
	}

	var result *gorm.DB
	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&memories)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&memories)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to search memories: %w", result.Error)
	}

	return memories, nil
}
