package storage

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
