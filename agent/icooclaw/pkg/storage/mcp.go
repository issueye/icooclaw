package storage

import (
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
	Name        string      `gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:MCP名称" json:"name"`                                      // MCP 名称
	Description string      `gorm:"column:description;type:varchar(255);comment:MCP描述" json:"description"`                                            // MCP 描述
	Type        MCPType     `gorm:"column:type;type:varchar(100);not null;comment:MCP类型(stdio/Streamable HTTP)" json:"type"`                          // MCP 类型
	Args        StringArray `gorm:"column:args;type:text;serializer:json;comment:MCP参数(JSON数组)" json:"args"`                                          // MCP 参数
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
