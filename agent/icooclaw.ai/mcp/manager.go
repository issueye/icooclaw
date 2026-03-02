package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/icooclaw/icooclaw/internal/config"
)

// Manager MCP 管理器（整合客户端、工具、资源、提示管理器）
type Manager struct {
	clientMgr   *ClientManager
	toolsMgr    *MCPToolsManager
	resourceMgr *MCPResourceManager
	promptsMgr  *MCPPromptManager
	config      config.MCPConfig
	logger      *slog.Logger
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewManager 创建 MCP 管理器
func NewManager(cfg config.MCPConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Init 初始化 MCP 管理器
func (m *Manager) Init(ctx context.Context) error {
	if !m.config.Enabled {
		m.logger.Info("MCP is disabled")
		return nil
	}

	// 创建客户端管理器
	m.clientMgr = NewClientManager(m.logger)

	// 添加所有配置的服务器
	for name, serverCfg := range m.config.Servers {
		cfg := ServerConfig{
			Name:        name,
			Command:     serverCfg.Command,
			Args:        serverCfg.Args,
			Env:         serverCfg.Env,
			Transport:   serverCfg.Transport,
			URL:         serverCfg.URL,
			AuthHeaders: serverCfg.AuthHeaders,
			Timeout:     serverCfg.Timeout,
		}
		m.clientMgr.AddClient(name, cfg)
		m.logger.Debug("Added MCP server", "name", name, "transport", serverCfg.Transport)
	}

	// 连接所有客户端
	if err := m.clientMgr.ConnectAll(ctx); err != nil {
		return fmt.Errorf("failed to connect MCP clients: %w", err)
	}

	// 创建工具管理器
	m.toolsMgr = NewMCPToolsManager(m.clientMgr, m.logger)
	if err := m.toolsMgr.LoadTools(); err != nil {
		m.logger.Warn("Failed to load MCP tools", "error", err)
	}

	// 创建资源管理器
	m.resourceMgr = NewMCPResourceManager(m.clientMgr, m.logger)
	if err := m.resourceMgr.LoadResources(); err != nil {
		m.logger.Warn("Failed to load MCP resources", "error", err)
	}

	// 创建提示管理器
	m.promptsMgr = NewMCPPromptManager(m.clientMgr, m.logger)
	if err := m.promptsMgr.LoadPrompts(); err != nil {
		m.logger.Warn("Failed to load MCP prompts", "error", err)
	}

	m.logger.Info("MCP initialized",
		"servers", m.clientMgr.Count(),
		"tools", m.toolsMgr.ToolCount(),
		"resources", m.resourceMgr.ResourceCount(),
		"prompts", m.promptsMgr.PromptCount())

	return nil
}

// GetClientManager 获取客户端管理器
func (m *Manager) GetClientManager() *ClientManager {
	return m.clientMgr
}

// GetToolsManager 获取工具管理器
func (m *Manager) GetToolsManager() *MCPToolsManager {
	return m.toolsMgr
}

// GetResourceManager 获取资源管理器
func (m *Manager) GetResourceManager() *MCPResourceManager {
	return m.resourceMgr
}

// GetPromptsManager 获取提示管理器
func (m *Manager) GetPromptsManager() *MCPPromptManager {
	return m.promptsMgr
}

// GetToolRegistry 获取工具注册表
func (m *Manager) GetToolRegistry() interface{} {
	if m.toolsMgr == nil {
		return nil
	}
	return m.toolsMgr.GetRegistry()
}

// Reload 重新加载
func (m *Manager) Reload(ctx context.Context) error {
	m.logger.Info("Reloading MCP...")

	// 断开所有客户端
	if m.clientMgr != nil {
		_ = m.clientMgr.DisconnectAll()
	}

	// 重新初始化
	return m.Init(ctx)
}

// Close 关闭
func (m *Manager) Close() error {
	m.cancel()

	if m.clientMgr != nil {
		return m.clientMgr.Close()
	}
	return nil
}

// Status 获取状态
func (m *Manager) Status() *ManagerStatus {
	status := &ManagerStatus{
		Enabled: m.config.Enabled,
	}

	if m.clientMgr != nil {
		status.Servers = m.clientMgr.Count()
		status.ConnectedServers = m.countConnected()
	}

	if m.toolsMgr != nil {
		status.Tools = m.toolsMgr.ToolCount()
	}

	if m.resourceMgr != nil {
		status.Resources = m.resourceMgr.ResourceCount()
	}

	if m.promptsMgr != nil {
		status.Prompts = m.promptsMgr.PromptCount()
	}

	return status
}

// countConnected 计算已连接的服务器数量
func (m *Manager) countConnected() int {
	if m.clientMgr == nil {
		return 0
	}

	count := 0
	for _, name := range m.clientMgr.ListClients() {
		client, _ := m.clientMgr.GetClient(name)
		if client != nil && client.IsConnected() {
			count++
		}
	}
	return count
}

// ManagerStatus 管理器状态
type ManagerStatus struct {
	Enabled          bool `json:"enabled"`
	Servers          int  `json:"servers"`
	ConnectedServers int  `json:"connected_servers"`
	Tools            int  `json:"tools"`
	Resources        int  `json:"resources"`
	Prompts          int  `json:"prompts"`
}

// CreateManagerFromConfig 从配置创建管理器
func CreateManagerFromConfig(cfg *config.Config, logger *slog.Logger) (*Manager, error) {
	mgr := NewManager(cfg.MCP, logger)
	if err := mgr.Init(context.Background()); err != nil {
		return nil, err
	}
	return mgr, nil
}

// Wait 等待（保持运行）
func (m *Manager) Wait() {
	<-m.ctx.Done()
}
