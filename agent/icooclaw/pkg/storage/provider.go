package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"icooclaw/pkg/consts"
	icooclawErrors "icooclaw/pkg/errors"
)

// Provider represents a provider configuration.
type Provider struct {
	Model
	Name         string              `gorm:"column:name;type:varchar(100);not null;comment:提供商名称" json:"name"`
	Type         consts.ProviderType `gorm:"column:type;type:varchar(50);not null;comment:提供商类型" json:"type"`
	APIKey       string              `gorm:"column:api_key;type:varchar(255);comment:API密钥" json:"api_key"`
	APIBase      string              `gorm:"column:api_base;type:varchar(255);comment:API基础URL" json:"api_base"`
	DefaultModel string              `gorm:"column:default_model;type:varchar(100);comment:默认模型" json:"default_model"`
	Models       []string            `gorm:"column:models;type:text;serializer:json;comment:支持的模型列表(JSON数组)" json:"models"` // JSON array
	Config       string              `gorm:"column:config;type:text;comment:配置(JSON格式)" json:"config"`                      // JSON object
	Metadata     map[string]any      `gorm:"column:metadata;type:text;serializer:json;comment:元数据(JSON格式)" json:"metadata"` // JSON object
}

// TableName returns the table name for Provider.
func (Provider) TableName() string {
	return tableNamePrefix + "providers"
}

type ProviderStorage struct {
	db *gorm.DB
}

func NewProviderStorage(db *gorm.DB) *ProviderStorage {
	return &ProviderStorage{db: db}
}

// Save saves a provider configuration.
func (s *ProviderStorage) Save(p *Provider) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "api_key", "api_base", "default_model", "models", "config", "updated_at"}),
	}).Create(p)
	return result.Error
}

// GetByName gets a provider by name.
func (s *ProviderStorage) GetByName(name string) (*Provider, error) {
	var p Provider
	result := s.db.Where("name = ?", name).First(&p)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, icooclawErrors.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get provider: %w", result.Error)
	}
	return &p, nil
}

// List lists all providers.
func (s *ProviderStorage) List() ([]*Provider, error) {
	var providers []*Provider
	result := s.db.Order("name").Find(&providers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list providers: %w", result.Error)
	}
	return providers, nil
}

// Delete deletes a provider by name.
func (s *ProviderStorage) Delete(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Provider{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete provider: %w", result.Error)
	}
	return nil
}

type QueryProvider struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Type    string `json:"type"`
}

type ResQueryProvider struct {
	Page    Page       `json:"page"`
	Records []Provider `json:"records"`
}

// Page gets providers with pagination.
func (s *ProviderStorage) Page(query *QueryProvider) (*ResQueryProvider, error) {
	var res ResQueryProvider

	qry := s.db.Model(&Provider{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR type LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count providers: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get providers: %w", result.Error)
	}

	return &res, nil
}
