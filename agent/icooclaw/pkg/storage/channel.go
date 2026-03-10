package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	icooclawErrors "icooclaw/pkg/errors"
)

// Channel represents a channel configuration.
type Channel struct {
	Model
	Name        string `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:渠道名称" json:"name"` // 渠道名称
	Type        string `gorm:"column:type;type:varchar(50);not null;comment:渠道类型" json:"type"`             // 渠道类型
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`    // 是否启用
	Config      string `gorm:"column:config;type:text;comment:配置(JSON格式)" json:"config"`                   // JSON object
	Permissions string `gorm:"column:permissions;type:text;comment:权限(JSON数组)" json:"permissions"`          // JSON array
}

// TableName returns the table name for Channel.
func (Channel) TableName() string {
	return tableNamePrefix + "channels"
}

// SaveChannel saves a channel configuration.
func (s *Storage) SaveChannel(c *Channel) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "enabled", "config", "permissions", "updated_at"}),
	}).Create(c)
	return result.Error
}

// GetChannel gets a channel by name.
func (s *Storage) GetChannel(name string) (*Channel, error) {
	var c Channel
	result := s.db.Where("name = ?", name).First(&c)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get channel: %w", result.Error)
	}
	return &c, nil
}

// ListChannels lists all channels.
func (s *Storage) ListChannels() ([]*Channel, error) {
	var channels []*Channel
	result := s.db.Order("name").Find(&channels)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list channels: %w", result.Error)
	}
	return channels, nil
}

// ListEnabledChannels lists all enabled channels.
func (s *Storage) ListEnabledChannels() ([]*Channel, error) {
	var channels []*Channel
	result := s.db.Where("enabled = ?", true).Order("name").Find(&channels)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list enabled channels: %w", result.Error)
	}
	return channels, nil
}

// DeleteChannel deletes a channel by name.
func (s *Storage) DeleteChannel(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Channel{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete channel: %w", result.Error)
	}
	return nil
}
