package storage

import "gorm.io/gorm"

// ProviderConfig Provider配置模型
type ProviderConfig struct {
	Model
	Name    string `gorm:"size:50;uniqueIndex" json:"name"` // openai, anthropic...
	Enabled bool   `gorm:"default:false" json:"enabled"`
	Config  string `gorm:"type:text" json:"config"` // JSON配置
}

// TableName 表名
func (ProviderConfig) TableName() string {
	return tableNamePrefix + "provider_configs"
}

// ProviderConfigStorage Provider配置存储
type ProviderConfigStorage struct {
	db *gorm.DB
}

// NewProviderConfigStorage 创建Provider配置存储
func NewProviderConfigStorage(db *gorm.DB) *ProviderConfigStorage {
	return &ProviderConfigStorage{db: db}
}

// CreateOrUpdate 创建或更新Provider配置
func (s *ProviderConfigStorage) CreateOrUpdate(config *ProviderConfig) error {
	return s.db.Save(config).Error
}

// Create 创建Provider配置
func (s *ProviderConfigStorage) Create(config *ProviderConfig) error {
	return s.db.Create(config).Error
}

// Update 更新Provider配置
func (s *ProviderConfigStorage) Update(config *ProviderConfig) error {
	return s.db.Save(config).Error
}

// GetByID 通过ID获取Provider配置
func (s *ProviderConfigStorage) GetByID(id uint) (*ProviderConfig, error) {
	var config ProviderConfig
	err := s.db.First(&config, id).Error
	return &config, err
}

// GetByName 通过名称获取Provider配置
func (s *ProviderConfigStorage) GetByName(name string) (*ProviderConfig, error) {
	var config ProviderConfig
	err := s.db.Where("name = ?", name).First(&config).Error
	return &config, err
}

// Delete 删除Provider配置
func (s *ProviderConfigStorage) Delete(id uint) error {
	return s.db.Delete(&ProviderConfig{}, id).Error
}

// GetAll 获取所有Provider配置
func (s *ProviderConfigStorage) GetAll() ([]ProviderConfig, error) {
	var configs []ProviderConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// GetEnabled 获取启用的Provider配置
func (s *ProviderConfigStorage) GetEnabled() ([]ProviderConfig, error) {
	var configs []ProviderConfig
	err := s.db.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

// Page 分页获取Provider配置
func (s *ProviderConfigStorage) Page(q *QueryProviderConfig) (*ResQueryProviderConfig, error) {
	var total int64
	query := s.db.Model(&ProviderConfig{})
	if q.KeyWord != "" {
		query = query.Where("name LIKE ?", "%"+q.KeyWord+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var configs []ProviderConfig
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&configs).Error

	q.Page.Total = int(total)
	return &ResQueryProviderConfig{
		Page:    q.Page,
		Records: configs,
	}, err
}

// QueryProviderConfig Provider配置查询参数
type QueryProviderConfig struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

// ResQueryProviderConfig Provider配置查询结果
type ResQueryProviderConfig struct {
	Page    Page             `json:"page"`
	Records []ProviderConfig `json:"records"`
}
