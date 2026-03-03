package storage

import "strings"

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
	Args        []string `gorm:"size:255" json:"args"`             // MCP 参数
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

// Create 创建MCP配置
func (table *MCPConfig) Create() error {
	return DB.Create(table).Error
}

// Update 更新MCP配置
func (table *MCPConfig) Update() error {
	return DB.Save(table).Error
}

// GetByName 通过名称获取MCP配置
func GetMCPConfigByName(name string) (*MCPConfig, error) {
	var config MCPConfig
	err := DB.Where("name = ?", name).First(&config).Error
	return &config, err
}

// Delete 删除MCP配置
func (table *MCPConfig) Delete() error {
	return DB.Delete(table).Error
}

// GetAll 获取所有MCP配置
func GetAllMCPConfigs() ([]MCPConfig, error) {
	var configs []MCPConfig
	err := DB.Find(&configs).Error
	return configs, err
}
