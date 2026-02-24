package storage

import (
	"time"

	"gorm.io/gorm"
)

// Memory 长期记忆模型
type Memory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"size:20;index" json:"type"` // memory, history
	Key       string    `gorm:"size:255;index" json:"key"` // 记忆键
	Content   string    `gorm:"type:text" json:"content"`  // 记忆内容
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (Memory) TableName() string {
	return "memories"
}

// Create 创建记忆
func (m *Memory) Create() error {
	return DB.Create(m).Error
}

// Update 更新记忆
func (m *Memory) Update() error {
	return DB.Save(m).Error
}

// Delete 删除记忆
func (m *Memory) Delete() error {
	return DB.Delete(m).Error
}

// GetByID 通过ID获取记忆
func GetMemoryByID(id uint) (*Memory, error) {
	var memory Memory
	err := DB.First(&memory, id).Error
	return &memory, err
}

// GetByKey 通过Key获取记忆
func GetMemoryByKey(key string) (*Memory, error) {
	var memory Memory
	err := DB.Where("key = ?", key).First(&memory).Error
	return &memory, err
}

// GetByType 通过类型获取记忆
func GetMemoriesByType(memType string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("type = ?", memType).Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// GetAll 获取所有记忆
func GetAllMemories() ([]Memory, error) {
	var memories []Memory
	err := DB.Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

// Upsert 创建或更新记忆
func (m *Memory) Upsert() error {
	var existing Memory
	err := DB.Where("key = ? AND type = ?", m.Key, m.Type).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return m.Create()
	}
	m.ID = existing.ID
	return m.Update()
}

// SearchMemories 搜索记忆
func SearchMemories(query string) ([]Memory, error) {
	var memories []Memory
	err := DB.Where("content LIKE ?", "%"+query+"%").Order("updated_at DESC").Find(&memories).Error
	return memories, err
}

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
	return "channel_configs"
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

// ProviderConfig Provider配置模型
type ProviderConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex" json:"name"` // openai, anthropic...
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"` // JSON配置
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 表名
func (ProviderConfig) TableName() string {
	return "provider_configs"
}

// Create 创建Provider配置
func (p *ProviderConfig) Create() error {
	return DB.Create(p).Error
}

// Update 更新Provider配置
func (p *ProviderConfig) Update() error {
	return DB.Save(p).Error
}

// GetByName 通过名称获取Provider配置
func GetProviderConfigByName(name string) (*ProviderConfig, error) {
	var config ProviderConfig
	err := DB.Where("name = ?", name).First(&config).Error
	return &config, err
}
