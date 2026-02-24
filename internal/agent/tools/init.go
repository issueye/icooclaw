package tools

import (
	"log/slog"

	"github.com/icooclaw/icooclaw/internal/channel"
	"github.com/icooclaw/icooclaw/internal/config"
)

// InitTools 初始化工具系统
func InitTools(cfg *config.Config, logger *slog.Logger, channelManager *channel.Manager) *Registry {
	registry := NewRegistry()

	// 创建文件工具配置
	toolConfig := &FileToolConfig{
		AllowedRead:   true,
		AllowedWrite:  true,
		AllowedEdit:   true,
		AllowedDelete: true,
		Workspace:     cfg.Workspace,
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
		toolConfig.Workspace = cfg.Tools.Workspace
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

	logger.Info("Tools initialized", "count", registry.Count())

	return registry
}
