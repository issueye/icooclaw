package storage

import (
	"strings"

	"gorm.io/gorm"
)

type MCPType string

const (
	MCPTypeStdio MCPType = "stdio"           // stdio 类型的 MCP
	MCPTypeSSE   MCPType = "Streamable HTTP" // sse 类型的 MCP
)

func (mcpType MCPType) String() string {
	return string(mcpType)
}

type MCPConfig struct {
	Model                // 嵌入 Model 结构体
	Name        string   `gorm:"size:100;uniqueIndex" json:"name"` // MCP 名称
	Description string   `gorm:"size:255" json:"description"`      // MCP 描述
	Type        MCPType  `gorm:"size:100" json:"type"`             // MCP 类型
	Args        []string `gorm:"type:text;serializer:json" json:"args"` // MCP 参数
}

func (table *MCPConfig) IsStdio() bool {
	return table.Type == MCPTypeStdio
}

func (table *MCPConfig) IsSSE() bool {
	return table.Type == MCPTypeSSE
}

func (table *MCPConfig) ArgsString() string {
	return strings.Join(table.Args, " ")
}

func (table *MCPConfig) TableName() string {
	return tableNamePrefix + "mcp"
}

// MCPConfigStorage MCP配置存储
type MCPConfigStorage struct {
	db *gorm.DB
}

// NewMCPConfigStorage 创建MCP配置存储
func NewMCPConfigStorage(db *gorm.DB) *MCPConfigStorage {
	return &MCPConfigStorage{db: db}
}

// CreateOrUpdate 创建或更新MCP配置
func (s *MCPConfigStorage) CreateOrUpdate(config *MCPConfig) error {
	return s.db.Save(config).Error
}

// Create 创建MCP配置
func (s *MCPConfigStorage) Create(config *MCPConfig) error {
	return s.db.Create(config).Error
}

// Update 更新MCP配置
func (s *MCPConfigStorage) Update(config *MCPConfig) error {
	return s.db.Save(config).Error
}

// GetByID 通过ID获取MCP配置
func (s *MCPConfigStorage) GetByID(id uint) (*MCPConfig, error) {
	var config MCPConfig
	err := s.db.First(&config, id).Error
	return &config, err
}

// GetByName 通过名称获取MCP配置
func (s *MCPConfigStorage) GetByName(name string) (*MCPConfig, error) {
	var config MCPConfig
	err := s.db.Where("name = ?", name).First(&config).Error
	return &config, err
}

// Delete 删除MCP配置
func (s *MCPConfigStorage) Delete(id uint) error {
	return s.db.Delete(&MCPConfig{}, id).Error
}

// GetAll 获取所有MCP配置
func (s *MCPConfigStorage) GetAll() ([]MCPConfig, error) {
	var configs []MCPConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// Page 分页获取MCP配置
func (s *MCPConfigStorage) Page(q *QueryMCPConfig) (*ResQueryMCPConfig, error) {
	var total int64
	query := s.db.Model(&MCPConfig{})
	if q.KeyWord != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+q.KeyWord+"%", "%"+q.KeyWord+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var configs []MCPConfig
	err := query.Order("created_at DESC").
		Offset((q.Page.Page - 1) * q.Page.Size).
		Limit(q.Page.Size).
		Find(&configs).Error

	q.Page.Total = int(total)
	return &ResQueryMCPConfig{
		Page:    q.Page,
		Records: configs,
	}, err
}

// QueryMCPConfig MCP配置查询参数
type QueryMCPConfig struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
}

// ResQueryMCPConfig MCP配置查询结果
type ResQueryMCPConfig struct {
	Page    Page        `json:"page"`
	Records []MCPConfig `json:"records"`
}
