package storage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
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
	Model                   // 嵌入 Model 结构体
	Name        string      `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:MCP名称" json:"name"`            // MCP 名称
	Description string      `gorm:"column:description;type:varchar(255);comment:MCP描述" json:"description"`                   // MCP 描述
	Type        MCPType     `gorm:"column:type;type:varchar(100);not null;comment:MCP类型(stdio/Streamable HTTP)" json:"type"` // MCP 类型
	Args        StringArray `gorm:"column:args;type:text;serializer:json;comment:MCP参数(JSON数组)" json:"args"`                 // MCP 参数
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

type MCPStorage struct {
	db *gorm.DB
}

func NewMCPStorage(db *gorm.DB) *MCPStorage {
	return &MCPStorage{db: db}
}

// BeforeCreate 创建前回调
func (c *MCPConfig) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New().String()
	return nil
}

// Save saves a MCP configuration.
func (s *MCPStorage) Save(m *MCPConfig) error {
	return s.db.Create(m).Error
}

// Get gets a MCP by name.
func (s *MCPStorage) Get(name string) (*MCPConfig, error) {
	var m MCPConfig
	result := s.db.Where("name = ?", name).First(&m)
	if result.Error != nil {
		return nil, result.Error
	}
	return &m, nil
}

// List lists all MCP configurations.
func (s *MCPStorage) List() ([]*MCPConfig, error) {
	var mcpConfigs []*MCPConfig
	result := s.db.Order("name").Find(&mcpConfigs)
	if result.Error != nil {
		return nil, result.Error
	}
	return mcpConfigs, nil
}

// Delete deletes a MCP by name.
func (s *MCPStorage) Delete(name string) error {
	result := s.db.Where("name = ?", name).Delete(&MCPConfig{})
	return result.Error
}

// DeleteByID deletes a MCP by ID.
func (s *MCPStorage) DeleteByID(id string) error {
	result := s.db.Where("id = ?", id).Delete(&MCPConfig{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete mcp config: %w", result.Error)
	}
	return nil
}

// GetByID gets a MCP by ID.
func (s *MCPStorage) GetByID(id string) (*MCPConfig, error) {
	var m MCPConfig
	result := s.db.Where("id = ?", id).First(&m)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("mcp config not found")
		}
		return nil, fmt.Errorf("failed to get mcp config: %w", result.Error)
	}
	return &m, nil
}

// Update updates a MCP configuration.
func (s *MCPStorage) Update(m *MCPConfig) error {
	result := s.db.Save(m)
	if result.Error != nil {
		return fmt.Errorf("failed to update mcp config: %w", result.Error)
	}
	return nil
}

// Create creates a new MCP configuration.
func (s *MCPStorage) Create(m *MCPConfig) error {
	return s.db.Create(m).Error
}

type QueryMCP struct {
	Page    Page    `json:"page"`
	KeyWord string  `json:"key_word"`
	Type    MCPType `json:"type"`
}

type ResQueryMCP struct {
	Page    Page        `json:"page"`
	Records []MCPConfig `json:"records"`
}

// Page gets MCP configurations with pagination.
func (s *MCPStorage) Page(query *QueryMCP) (*ResQueryMCP, error) {
	var res ResQueryMCP

	qry := s.db.Model(&MCPConfig{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Type != "" {
		qry = qry.Where("type = ?", query.Type)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count mcp configs: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get mcp configs: %w", result.Error)
	}

	return &res, nil
}
