package storage

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChannelConfig 通道配置模型
type ChannelConfig struct {
	Model
	Name    string `gorm:"size:50;uniqueIndex" json:"name"` // telegram, discord...
	Enabled bool   `gorm:"default:false" json:"enabled"`
	Config  string `gorm:"type:text" json:"config"` // JSON配置
}

// TableName 表名
func (ChannelConfig) TableName() string {
	return tableNamePrefix + "channel_configs"
}

// BeforeCreate 创建前回调
func (c *ChannelConfig) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// ChannelConfigStorage 通道配置存储
type ChannelConfigStorage struct {
	db *gorm.DB
}

// NewChannelConfigStorage 创建通道配置存储
func NewChannelConfigStorage(db *gorm.DB) *ChannelConfigStorage {
	return &ChannelConfigStorage{db: db}
}

// Create 创建通道配置
func (s *ChannelConfigStorage) Create(config *ChannelConfig) error {
	return s.db.Create(config).Error
}

// Update 更新通道配置
func (s *ChannelConfigStorage) Update(config *ChannelConfig) error {
	return s.db.Save(config).Error
}

// GetByID 通过ID获取通道配置
func (s *ChannelConfigStorage) GetByID(id string) (*ChannelConfig, error) {
	var config ChannelConfig
	err := s.db.First(&config, id).Error
	return &config, err
}

// GetByName 通过名称获取通道配置
func (s *ChannelConfigStorage) GetByName(name string) (*ChannelConfig, error) {
	var config ChannelConfig
	err := s.db.Where("name = ?", name).First(&config).Error
	return &config, err
}

// Delete 删除通道配置
func (s *ChannelConfigStorage) Delete(id string) error {
	return s.db.Delete(&ChannelConfig{}, id).Error
}

// GetAll 获取所有通道配置
func (s *ChannelConfigStorage) GetAll() ([]ChannelConfig, error) {
	var configs []ChannelConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// GetEnabled 获取启用的通道配置
func (s *ChannelConfigStorage) GetEnabled() ([]ChannelConfig, error) {
	var configs []ChannelConfig
	err := s.db.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

// Page 分页获取通道配置
func (s *ChannelConfigStorage) Page(q *QueryChannelConfig) (*ResQueryChannelConfig, error) {
	var total int64
	query := s.db.Model(&ChannelConfig{})
	if q.KeyWord != "" {
		query = query.Where("name LIKE ?", "%"+q.KeyWord+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var configs []ChannelConfig
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&configs).Error

	q.Page.Total = int(total)
	return &ResQueryChannelConfig{
		Page:    q.Page,
		Records: configs,
	}, err
}

// QueryChannelConfig 通道配置查询参数
type QueryChannelConfig struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

// ResQueryChannelConfig 通道配置查询结果
type ResQueryChannelConfig struct {
	Page    Page            `json:"page"`
	Records []ChannelConfig `json:"records"`
}
