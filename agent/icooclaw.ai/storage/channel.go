package storage

import "time"

// ChannelConfig 通道配置模型
type ChannelConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex" json:"name"` // telegram, discord...
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"` // JSON配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (ChannelConfig) TableName() string {
	return tableNamePrefix + "channel_configs"
}

// Create 创建通道配置
func (c *ChannelConfig) Create() error {
	return DB.Create(c).Error
}

// Update 更新通道配置
func (c *ChannelConfig) Update() error {
	return DB.Save(c).Error
}

// GetByName 通过名称获取通道配置
func GetChannelConfigByName(name string) (*ChannelConfig, error) {
	var config ChannelConfig
	err := DB.Where("name = ?", name).First(&config).Error
	return &config, err
}
