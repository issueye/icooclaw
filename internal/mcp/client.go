package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client MCP 客户端
type Client struct {
	name      string
	serverCfg ServerConfig
	logger    *slog.Logger
	mcpClient *client.Client
	tools     []mcp.Tool
	resources []mcp.Resource
	prompts   []mcp.Prompt
	mu        sync.RWMutex
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// ServerConfig MCP 服务器配置
type ServerConfig struct {
	Name        string            `mapstructure:"name"`
	Command     string            `mapstructure:"command"`
	Args        []string          `mapstructure:"args"`
	Env         map[string]string `mapstructure:"env"`
	Transport   string            `mapstructure:"transport"` // stdio or http
	URL         string            `mapstructure:"url"`       // for http transport
	AuthHeaders map[string]string `mapstructure:"auth_headers"`
	Timeout     time.Duration     `mapstructure:"timeout"`
}

// NewClient 创建 MCP 客户端
func NewClient(name string, cfg ServerConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		name:      name,
		serverCfg: cfg,
		logger:    logger,
	}
}

// Name 获取名称
func (c *Client) Name() string {
	return c.name
}

// Connect 连接到 MCP 服务器
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("client already connected")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	var err error
	switch c.serverCfg.Transport {
	case "http", "https", "sse":
		err = c.connectHTTP(c.ctx)
	case "stdio", "":
		err = c.connectStdio(c.ctx)
	default:
		err = fmt.Errorf("unsupported transport: %s", c.serverCfg.Transport)
	}

	if err != nil {
		c.cancel()
		return fmt.Errorf("failed to connect to MCP server %s: %w", c.name, err)
	}

	c.running = true
	c.logger.Info("MCP client connected", "name", c.name, "transport", c.serverCfg.Transport)
	return nil
}

// connectStdio 使用 stdio 模式连接
func (c *Client) connectStdio(ctx context.Context) error {
	// 准备环境变量
	env := os.Environ()
	for k, v := range c.serverCfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 创建 stdio MCP 客户端
	mcpClient, err := client.NewStdioMCPClient(c.serverCfg.Command, env, c.serverCfg.Args...)
	if err != nil {
		return fmt.Errorf("failed to create stdio client: %w", err)
	}

	c.mcpClient = mcpClient

	// 初始化
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// 列出可用工具
	if err := c.listTools(ctx); err != nil {
		c.logger.Warn("Failed to list tools", "error", err)
	}

	// 列出可用资源
	if err := c.listResources(ctx); err != nil {
		c.logger.Warn("Failed to list resources", "error", err)
	}

	// 列出可用提示
	if err := c.listPrompts(ctx); err != nil {
		c.logger.Warn("Failed to list prompts", "error", err)
	}

	return nil
}

// connectHTTP 使用 HTTP/SSE 模式连接
func (c *Client) connectHTTP(ctx context.Context) error {
	url := c.serverCfg.URL
	if url == "" {
		return fmt.Errorf("URL is required for HTTP transport")
	}

	// 创建 SSE 客户端
	mcpClient, err := client.NewSSEMCPClient(url)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	c.mcpClient = mcpClient

	// 初始化
	if err := c.initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// 列出可用工具
	if err := c.listTools(ctx); err != nil {
		c.logger.Warn("Failed to list tools", "error", err)
	}

	// 列出可用资源
	if err := c.listResources(ctx); err != nil {
		c.logger.Warn("Failed to list resources", "error", err)
	}

	// 列出可用提示
	if err := c.listPrompts(ctx); err != nil {
		c.logger.Warn("Failed to list prompts", "error", err)
	}

	return nil
}

// initialize 初始化 MCP 会话
func (c *Client) initialize(ctx context.Context) error {
	if c.mcpClient == nil {
		return fmt.Errorf("client not initialized")
	}

	// 调用 Initialize 方法
	result, err := c.mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "icooclaw",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.logger.Debug("MCP server initialized",
		"name", result.ServerInfo.Name,
		"version", result.ServerInfo.Version,
		"protocol", result.ProtocolVersion)

	return nil
}

// listTools 列出可用工具
func (c *Client) listTools(ctx context.Context) error {
	result, err := c.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("list tools failed: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	c.logger.Debug("Listed tools", "count", len(c.tools), "server", c.name)
	return nil
}

// listResources 列出可用资源
func (c *Client) listResources(ctx context.Context) error {
	result, err := c.mcpClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return fmt.Errorf("list resources failed: %w", err)
	}

	c.mu.Lock()
	c.resources = result.Resources
	c.mu.Unlock()

	c.logger.Debug("Listed resources", "count", len(c.resources), "server", c.name)
	return nil
}

// listPrompts 列出可用提示
func (c *Client) listPrompts(ctx context.Context) error {
	result, err := c.mcpClient.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		return fmt.Errorf("list prompts failed: %w", err)
	}

	c.mu.Lock()
	c.prompts = result.Prompts
	c.mu.Unlock()

	c.logger.Debug("Listed prompts", "count", len(c.prompts), "server", c.name)
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
	}

	if c.mcpClient != nil {
		c.mcpClient.Close()
		c.mcpClient = nil
	}

	c.running = false
	c.logger.Info("MCP client disconnected", "name", c.name)
	return nil
}

// IsConnected 检查是否连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// Tools 获取工具列表
func (c *Client) Tools() []mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// Resources 获取资源列表
func (c *Client) Resources() []mcp.Resource {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.resources
}

// Prompts 获取提示列表
func (c *Client) Prompts() []mcp.Prompt {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.prompts
}

// CallTool 调用工具
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (string, error) {
	c.mu.RLock()
	if !c.running || c.mcpClient == nil {
		c.mu.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	// 添加超时
	ctx, cancel := context.WithTimeout(ctx, c.serverCfg.Timeout)
	defer cancel()

	result, err := c.mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", fmt.Errorf("tool call failed: %w", err)
	}

	// 处理结果
	if len(result.Content) == 0 {
		return "", nil
	}

	// 转换为文本
	var content string
	for _, item := range result.Content {
		if textContent, ok := item.(mcp.TextContent); ok {
			content += textContent.Text
		}
	}

	return content, nil
}

// ReadResource 读取资源
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	c.mu.RLock()
	if !c.running || c.mcpClient == nil {
		c.mu.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, c.serverCfg.Timeout)
	defer cancel()

	result, err := c.mcpClient.ReadResource(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	})
	if err != nil {
		return "", fmt.Errorf("read resource failed: %w", err)
	}

	if len(result.Contents) == 0 {
		return "", nil
	}

	// 处理资源内容
	var content string
	for _, item := range result.Contents {
		if textResource, ok := item.(mcp.TextResourceContents); ok {
			content += textResource.Text
		}
	}

	return content, nil
}

// GetPrompt 获取提示
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (string, error) {
	c.mu.RLock()
	if !c.running || c.mcpClient == nil {
		c.mu.RUnlock()
		return "", fmt.Errorf("client not connected")
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, c.serverCfg.Timeout)
	defer cancel()

	result, err := c.mcpClient.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      name,
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", fmt.Errorf("get prompt failed: %w", err)
	}

	// 组装提示消息
	var messages string
	for _, msg := range result.Messages {
		if msg.Role == "user" {
			// 处理 Content，可能是各种类型
			content := mcp.GetTextFromContent(msg.Content)
			if content != "" {
				messages += content + "\n"
			}
		}
	}

	return messages, nil
}

// ClientManager MCP 客户端管理器
type ClientManager struct {
	clients map[string]*Client
	logger  *slog.Logger
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewClientManager 创建客户端管理器
func NewClientManager(logger *slog.Logger) *ClientManager {
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ClientManager{
		clients: make(map[string]*Client),
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddClient 添加客户端
func (m *ClientManager) AddClient(name string, cfg ServerConfig) *Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := NewClient(name, cfg, m.logger)
	m.clients[name] = client
	m.logger.Info("MCP client added", "name", name)
	return client
}

// GetClient 获取客户端
func (m *ClientManager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[name]
	return client, ok
}

// RemoveClient 移除客户端
func (m *ClientManager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return fmt.Errorf("client not found: %s", name)
	}

	if err := client.Disconnect(); err != nil {
		m.logger.Warn("Failed to disconnect client", "name", name, "error", err)
	}

	delete(m.clients, name)
	m.logger.Info("MCP client removed", "name", name)
	return nil
}

// ConnectAll 连接所有客户端
func (m *ClientManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	var errs []error
	for _, c := range clients {
		if err := c.Connect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
			m.logger.Error("Failed to connect MCP client", "name", c.Name(), "error", err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to connect some clients: %v", errs)
	}

	return nil
}

// DisconnectAll 断开所有客户端
func (m *ClientManager) DisconnectAll() error {
	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	for _, c := range clients {
		if err := c.Disconnect(); err != nil {
			m.logger.Warn("Failed to disconnect client", "name", c.Name(), "error", err)
		}
	}

	return nil
}

// ListClients 列出所有客户端
func (m *ClientManager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// Count 客户端数量
func (m *ClientManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetAllTools 获取所有工具
func (m *ClientManager) GetAllTools() map[string][]mcp.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]mcp.Tool)
	for name, c := range m.clients {
		result[name] = c.Tools()
	}
	return result
}

// Close 关闭管理器
func (m *ClientManager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return m.DisconnectAll()
}
