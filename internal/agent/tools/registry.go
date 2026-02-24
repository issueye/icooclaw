package tools

import (
	"github.com/icooclaw/icooclaw/internal/config"
)

// ToolRegistry 工具注册表管理器
type ToolRegistry struct {
	registry   *Registry
	config     *config.ToolsConfig
	toolConfig *FileToolConfig
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry(cfg *config.Config) *ToolRegistry {
	registry := NewRegistry()

	// 创建文件工具配置
	toolConfig := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: true,
		Workspace:    cfg.Workspace,
	}

	// 注册 HTTP 请求工具
	registry.Register(NewHTTPRequestTool())

	// 注册文件工具
	registry.Register(NewFileReadTool(toolConfig))
	registry.Register(NewFileWriteTool(toolConfig))
	registry.Register(NewFileListTool(toolConfig))
	registry.Register(NewFileDeleteTool(toolConfig))

	// 注册搜索工具
	registry.Register(NewWebSearchTool())
	registry.Register(NewWebFetchTool())

	// 注册计算器工具
	registry.Register(NewCalculatorTool())

	return &ToolRegistry{
		registry:   registry,
		config:     &cfg.Tools,
		toolConfig: toolConfig,
	}
}

// GetRegistry 获取工具注册表
func (t *ToolRegistry) GetRegistry() *Registry {
	return t.registry
}

// GetToolDefinitions 获取工具定义列表
func (t *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	return t.registry.ToDefinitions()
}

// UpdateFileConfig 更新文件工具配置
func (t *ToolRegistry) UpdateFileConfig(allowedRead, allowedWrite bool, workspace string) {
	t.toolConfig.AllowedRead = allowedRead
	t.toolConfig.AllowedWrite = allowedWrite
	t.toolConfig.Workspace = workspace

	// 重新注册文件工具
	// 注意：这里简化处理，实际应该更新现有工具的配置
}
