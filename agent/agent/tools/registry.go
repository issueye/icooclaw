package tools

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/icooclaw/icooclaw/internal/config"
)

// ToolRegistry 工具注册表管理器
type ToolRegistry struct {
	registry   *Registry
	config     *config.ToolsConfig
	toolConfig *FileToolConfig
	logger     *slog.Logger
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry(cfg *config.Config) *ToolRegistry {
	return NewToolRegistryWithLogger(cfg, slog.Default())
}

// NewToolRegistryWithLogger 创建带日志的工具注册表
func NewToolRegistryWithLogger(cfg *config.Config, logger *slog.Logger) *ToolRegistry {
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
		logger:     logger,
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

// LoadJSTools 从目录加载 JavaScript 工具
func (t *ToolRegistry) LoadJSTools(dir string) error {
	// 确保目录存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		t.logger.Info("Created JS tools directory", "path", dir)
		return nil
	}

	jsConfig := &JSToolConfig{
		Workspace: t.toolConfig.Workspace,
		MaxMemory: 10 * 1024 * 1024,
		Timeout:   30,
	}

	return RegisterJSTools(t.registry, dir, jsConfig, t.logger)
}

// LoadJSToolsFromWorkspace 从工作区加载 JavaScript 工具
func (t *ToolRegistry) LoadJSToolsFromWorkspace() error {
	toolsDir := filepath.Join(t.toolConfig.Workspace, "tools")
	return t.LoadJSTools(toolsDir)
}
