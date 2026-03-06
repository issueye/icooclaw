package storage

import (
	"encoding/json"

	"gorm.io/gorm"
)

// unmarshalJSON 辅助函数
func unmarshalJSON(data string, v interface{}) error {
	if data == "" {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal([]byte(data), v)
}

// ParamConfig 运行时参数配置模型
type ParamConfig struct {
	Model
	Key         string `gorm:"size:100;uniqueIndex" json:"key"`          // 参数键
	Value       string `gorm:"type:text" json:"value"`                   // 参数值（JSON 格式）
	Description string `gorm:"size:500" json:"description"`              // 参数描述
	Group       string `gorm:"size:50;default:'general'" json:"group"`   // 参数分组
	Enabled     bool   `gorm:"default:true" json:"enabled"`              // 是否启用
}

// TableName 表名
func (ParamConfig) TableName() string {
	return tableNamePrefix + "param_configs"
}

// ParamConfigStorage 运行时参数配置存储
type ParamConfigStorage struct {
	db *gorm.DB
}

// NewParamConfigStorage 创建运行时参数配置存储
func NewParamConfigStorage(db *gorm.DB) *ParamConfigStorage {
	return &ParamConfigStorage{db: db}
}

// CreateOrUpdate 创建或更新参数配置
func (s *ParamConfigStorage) CreateOrUpdate(config *ParamConfig) error {
	return s.db.Save(config).Error
}

// Create 创建参数配置
func (s *ParamConfigStorage) Create(config *ParamConfig) error {
	return s.db.Create(config).Error
}

// Update 更新参数配置
func (s *ParamConfigStorage) Update(config *ParamConfig) error {
	return s.db.Save(config).Error
}

// GetByID 通过 ID 获取参数配置
func (s *ParamConfigStorage) GetByID(id uint) (*ParamConfig, error) {
	var config ParamConfig
	err := s.db.First(&config, id).Error
	return &config, err
}

// GetByKey 通过键获取参数配置
func (s *ParamConfigStorage) GetByKey(key string) (*ParamConfig, error) {
	var config ParamConfig
	err := s.db.Where("key = ?", key).First(&config).Error
	return &config, err
}

// Delete 删除参数配置
func (s *ParamConfigStorage) Delete(id uint) error {
	return s.db.Delete(&ParamConfig{}, id).Error
}

// DeleteByKey 通过键删除参数配置
func (s *ParamConfigStorage) DeleteByKey(key string) error {
	return s.db.Where("key = ?", key).Delete(&ParamConfig{}).Error
}

// GetAll 获取所有参数配置
func (s *ParamConfigStorage) GetAll() ([]ParamConfig, error) {
	var configs []ParamConfig
	err := s.db.Order("created_at DESC").Find(&configs).Error
	return configs, err
}

// GetByGroup 按分组获取参数配置
func (s *ParamConfigStorage) GetByGroup(group string) ([]ParamConfig, error) {
	var configs []ParamConfig
	err := s.db.Where("enabled = ? AND group = ?", true, group).
		Order("created_at DESC").
		Find(&configs).Error
	return configs, err
}

// GetEnabled 获取所有启用的参数配置
func (s *ParamConfigStorage) GetEnabled() ([]ParamConfig, error) {
	var configs []ParamConfig
	err := s.db.Where("enabled = ?", true).
		Order("group, key").
		Find(&configs).Error
	return configs, err
}

// Page 分页获取参数配置
func (s *ParamConfigStorage) Page(q *QueryParamConfig) (*ResQueryParamConfig, error) {
	var total int64
	query := s.db.Model(&ParamConfig{})

	if q.KeyWord != "" {
		query = query.Where("key LIKE ? OR description LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if q.Group != "" {
		query = query.Where("group = ?", q.Group)
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var configs []ParamConfig
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&configs).Error

	q.Page.Total = int(total)
	return &ResQueryParamConfig{
		Page:    q.Page,
		Records: configs,
	}, err
}

// GetStringValue 获取字符串类型的参数值
func (s *ParamConfigStorage) GetStringValue(key string, defaultValue string) string {
	config, err := s.GetByKey(key)
	if err != nil || config == nil {
		return defaultValue
	}
	return config.Value
}

// GetIntValue 获取整数类型的参数值
func (s *ParamConfigStorage) GetIntValue(key string, defaultValue int) int {
	config, err := s.GetByKey(key)
	if err != nil || config == nil {
		return defaultValue
	}
	var value int
	if err := unmarshalJSON(config.Value, &value); err != nil {
		return defaultValue
	}
	return value
}

// GetBoolValue 获取布尔类型的参数值
func (s *ParamConfigStorage) GetBoolValue(key string, defaultValue bool) bool {
	config, err := s.GetByKey(key)
	if err != nil || config == nil {
		return defaultValue
	}
	var value bool
	if err := unmarshalJSON(config.Value, &value); err != nil {
		return defaultValue
	}
	return value
}

// SetStringValue 设置字符串类型的参数值
func (s *ParamConfigStorage) SetStringValue(key, value, description, group string) error {
	config, err := s.GetByKey(key)
	if err != nil {
		// 创建新配置
		config = &ParamConfig{
			Key:         key,
			Value:       value,
			Description: description,
			Group:       group,
			Enabled:     true,
		}
		return s.Create(config)
	}

	config.Value = value
	if description != "" {
		config.Description = description
	}
	if group != "" {
		config.Group = group
	}
	return s.Update(config)
}

// QueryParamConfig 参数配置查询参数
type QueryParamConfig struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Group   string `json:"group"`
	Enabled *bool  `json:"enabled"`
}

// ResQueryParamConfig 参数配置查询结果
type ResQueryParamConfig struct {
	Page    Page           `json:"page"`
	Records []ParamConfig  `json:"records"`
}
