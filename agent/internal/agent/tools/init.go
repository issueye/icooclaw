package tools

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/icooclaw/icooclaw/internal/channel"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/pkg/utils"
)

// InitTools 初始化工具系统
func InitTools(cfg *config.Config, logger *slog.Logger, channelManager *channel.Manager) *Registry {
	registry := NewRegistry()

	expandedWorkspace, _ := utils.ExpandPath(cfg.Workspace)

	// 创建文件工具配置
	toolConfig := &FileToolConfig{
		AllowedRead:   true,
		AllowedWrite:  true,
		AllowedEdit:   true,
		AllowedDelete: true,
		Workspace:     expandedWorkspace,
	}

	// 从配置读取工具权限
	if !cfg.Tools.Enabled {
		logger.Info("Tools are disabled in config")
		return registry
	}

	if cfg.Tools.AllowFileRead {
		toolConfig.AllowedRead = true
	} else {
		toolConfig.AllowedRead = false
	}

	if cfg.Tools.AllowFileWrite {
		toolConfig.AllowedWrite = true
	} else {
		toolConfig.AllowedWrite = false
	}

	if cfg.Tools.AllowFileEdit {
		toolConfig.AllowedEdit = true
	} else {
		toolConfig.AllowedEdit = false
	}

	if cfg.Tools.AllowFileDelete {
		toolConfig.AllowedDelete = true
	} else {
		toolConfig.AllowedDelete = false
	}

	if cfg.Tools.Workspace != "" {
		toolConfig.Workspace, _ = utils.ExpandPath(cfg.Tools.Workspace)
	}

	// 注册 HTTP 请求工具
	registry.Register(NewHTTPRequestTool())
	logger.Debug("Registered tool: http_request")

	// 注册文件工具
	if toolConfig.AllowedRead {
		registry.Register(NewFileReadTool(toolConfig))
		logger.Debug("Registered tool: file_read")
	}
	if toolConfig.AllowedWrite {
		registry.Register(NewFileWriteTool(toolConfig))
		logger.Debug("Registered tool: file_write")
	}
	if toolConfig.AllowedEdit {
		registry.Register(NewFileEditTool(toolConfig))
		logger.Debug("Registered tool: file_edit")
	}
	if toolConfig.AllowedDelete {
		registry.Register(NewFileDeleteTool(toolConfig))
		logger.Debug("Registered tool: file_delete")
	}
	registry.Register(NewFileListTool(toolConfig))
	logger.Debug("Registered tool: file_list")

	// 注册搜索工具
	registry.Register(NewWebSearchTool())
	logger.Debug("Registered tool: web_search")
	registry.Register(NewWebFetchTool())
	logger.Debug("Registered tool: web_fetch")

	// 注册计算器工具
	registry.Register(NewCalculatorTool())
	logger.Debug("Registered tool: calculator")

	// 注册Shell执行工具
	if cfg.Tools.AllowExec {
		execConfig := &ShellExecConfig{
			Allowed:   true,
			Timeout:   cfg.Tools.ExecTimeout,
			Workspace: cfg.Workspace,
		}
		registry.Register(NewShellExecTool(execConfig))
		logger.Debug("Registered tool: exec")
	}

	// 注册消息工具
	if channelManager != nil {
		messageConfig := &MessageConfig{
			ChannelManager: channelManager,
		}
		registry.Register(NewMessageTool(messageConfig))
		logger.Debug("Registered tool: message")
	}

	// 注册 Grep 搜索工具
	if toolConfig.AllowedRead {
		registry.Register(NewGrepTool(toolConfig))
		logger.Debug("Registered tool: grep")
		registry.Register(NewFindTool(toolConfig))
		logger.Debug("Registered tool: find")
		registry.Register(NewTreeTool(toolConfig))
		logger.Debug("Registered tool: tree")
		registry.Register(NewReadPartTool(toolConfig))
		logger.Debug("Registered tool: read_part")
		registry.Register(NewLineCountTool(toolConfig))
		logger.Debug("Registered tool: wc")
	}

	// 注册 JS 工具管理工具
	var mu sync.RWMutex
	createToolConfig := &CreateJSToolConfig{
		Workspace: cfg.Workspace,
		ToolsDir:  cfg.Tools.JS.ToolsDir,
		Registry:  registry,
	}
	registry.Register(NewCreateJSTool(createToolConfig))
	logger.Debug("Registered tool: create_tool")

	listToolConfig := &ListJSToolsConfig{
		Workspace: cfg.Workspace,
		ToolsDir:  cfg.Tools.JS.ToolsDir,
	}
	registry.Register(NewListJSTools(listToolConfig))
	logger.Debug("Registered tool: list_tools")

	deleteToolConfig := &DeleteJSToolConfig{
		Workspace: cfg.Workspace,
		ToolsDir:  cfg.Tools.JS.ToolsDir,
		Registry:  registry,
		mu:        &mu,
	}
	registry.Register(NewDeleteJSTool(deleteToolConfig))
	logger.Debug("Registered tool: delete_tool")

	// 加载 JavaScript 工具
	if cfg.Tools.JS.Enabled {
		toolsDir := filepath.Join(cfg.Workspace, cfg.Tools.JS.ToolsDir)
		jsConfig := &JSToolConfig{
			Workspace:       cfg.Workspace,
			MaxMemory:       cfg.Tools.JS.MaxMemory,
			Timeout:         cfg.Tools.JS.Timeout,
			AllowFileRead:   cfg.Tools.JS.Permissions.FileRead,
			AllowFileWrite:  cfg.Tools.JS.Permissions.FileWrite,
			AllowFileDelete: cfg.Tools.JS.Permissions.FileDelete,
			AllowNetwork:    cfg.Tools.JS.Permissions.Network,
			AllowExec:       cfg.Tools.JS.Permissions.Exec,
			ExecTimeout:     cfg.Tools.JS.Permissions.ExecTimeout,
			HTTPTimeout:     cfg.Tools.JS.Permissions.HTTPTimeout,
			AllowedDomains:  cfg.Tools.JS.Permissions.AllowedDomains,
		}
		if err := RegisterJSTools(registry, toolsDir, jsConfig, logger); err != nil {
			logger.Warn("Failed to load JS tools", "error", err)
		}
	} else {
		logger.Info("JavaScript tools are disabled")
	}

	logger.Info("Tools initialized", "count", registry.Count())

	return registry
}
