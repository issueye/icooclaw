package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// ParamConfig 运行时参数配置模型
type ParamConfig struct {
	Model
	Key         string `gorm:"column:key;type:varchar(100);not null;comment:参数键" json:"key"`                       // 参数键
	Value       string `gorm:"column:value;type:text;comment:参数值(JSON格式)" json:"value"`                           // 参数值（JSON 格式）
	Description string `gorm:"column:description;type:varchar(500);comment:参数描述" json:"description"`               // 参数描述
	Group       string `gorm:"column:group;type:varchar(50);default:'general';comment:参数分组" json:"group"`          // 参数分组
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`           // 是否启用
}

// TableName returns the table name for ParamConfig.
func (ParamConfig) TableName() string {
	return tableNamePrefix + "param_config"
}

type ParamStorage struct {
	db *gorm.DB
}

func NewParamStorage(db *gorm.DB) *ParamStorage {
	return &ParamStorage{db: db}
}

// Save saves a param configuration.
func (s *ParamStorage) Save(p *ParamConfig) error {
	return s.db.Create(p).Error
}

// Get gets a param by key.
func (s *ParamStorage) Get(key string) (*ParamConfig, error) {
	var p ParamConfig
	result := s.db.Where("key = ?", key).First(&p)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get param: %w", result.Error)
	}
	return &p, nil
}

// List lists all param configurations.
func (s *ParamStorage) List() ([]*ParamConfig, error) {
	var params []*ParamConfig
	result := s.db.Order("key").Find(&params)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list params: %w", result.Error)
	}
	return params, nil
}

// ListByGroup lists all param configurations by group.
func (s *ParamStorage) ListByGroup(group string) ([]*ParamConfig, error) {
	var params []*ParamConfig
	result := s.db.Where("group = ?", group).Order("key").Find(&params)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list params by group: %w", result.Error)
	}
	return params, nil
}

// Delete deletes a param by key.
func (s *ParamStorage) Delete(key string) error {
	result := s.db.Where("key = ?", key).Delete(&ParamConfig{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete param: %w", result.Error)
	}
	return nil
}

type QueryParam struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Group   string `json:"group"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryParam struct {
	Page    Page         `json:"page"`
	Records []ParamConfig `json:"records"`
}

// Page gets param configurations with pagination.
func (s *ParamStorage) Page(query *QueryParam) (*ResQueryParam, error) {
	var res ResQueryParam

	qry := s.db.Model(&ParamConfig{})

	if query.KeyWord != "" {
		qry = qry.Where("key LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Group != "" {
		qry = qry.Where("group = ?", query.Group)
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("key")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count params: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get params: %w", result.Error)
	}

	return &res, nil
}
