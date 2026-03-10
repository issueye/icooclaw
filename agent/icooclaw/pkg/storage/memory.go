package storage

import (
	"fmt"
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

// SaveMemory saves a memory entry.
func (s *Storage) SaveMemory(m *Memory) error {
	return s.db.Create(m).Error
}

// GetMemory gets memory entries for a session.
func (s *Storage) GetMemory(sessionID string, limit int) ([]*Memory, error) {
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

// DeleteMemory deletes memory entries for a session.
func (s *Storage) DeleteMemory(sessionID string) error {
	result := s.db.Where("session_id = ?", sessionID).Delete(&Memory{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete memory: %w", result.Error)
	}
	return nil
}
