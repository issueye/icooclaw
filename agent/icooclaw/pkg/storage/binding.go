package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
)

// Binding represents an agent binding.
type Binding struct {
	Model
	Channel   string `gorm:"column:channel;type:varchar(50);not null;uniqueIndex:idx_binding;comment:渠道" json:"channel"`
	ChatID    string `gorm:"column:chat_id;type:varchar(100);not null;uniqueIndex:idx_binding;comment:聊天ID" json:"chat_id"`
	AgentName string `gorm:"column:agent_name;type:varchar(100);not null;comment:代理名称" json:"agent_name"`
	Enabled   bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
}

// TableName returns the table name for Binding.
func (Binding) TableName() string {
	return tableNamePrefix + "bindings"
}

type BindingStorage struct {
	db *gorm.DB
}

func NewBindingStorage(db *gorm.DB) *BindingStorage {
	return &BindingStorage{db: db}
}

// SaveBinding saves an agent binding.
func (s *BindingStorage) SaveBinding(b *Binding) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel"}, {Name: "chat_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"agent_name", "enabled"}),
	}).Create(b)
	return result.Error
}

// GetBinding gets a binding by channel and chat ID.
func (s *BindingStorage) GetBinding(channel, chatID string) (*Binding, error) {
	var b Binding
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).First(&b)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get binding: %w", result.Error)
	}
	return &b, nil
}

// ListBindings lists all bindings.
func (s *BindingStorage) ListBindings() ([]*Binding, error) {
	var bindings []*Binding
	result := s.db.Order("channel, chat_id").Find(&bindings)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list bindings: %w", result.Error)
	}
	return bindings, nil
}

// DeleteBinding deletes a binding.
func (s *BindingStorage) DeleteBinding(channel, chatID string) error {
	result := s.db.Where("channel = ? AND chat_id = ?", channel, chatID).Delete(&Binding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete binding: %w", result.Error)
	}
	return nil
}
