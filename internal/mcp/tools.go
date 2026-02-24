package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/icooclaw/icooclaw/internal/agent/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPToolAdapter MCP 工具适配器，将 MCP 工具转换为 Agent 工具
type MCPToolAdapter struct {
	serverName string
	tool       mcp.Tool
	client     *Client
	logger     *slog.Logger
}

// NewMCPToolAdapter 创建 MCP 工具适配器
func NewMCPToolAdapter(serverName string, tool mcp.Tool, client *Client, logger *slog.Logger) *MCPToolAdapter {
	return &MCPToolAdapter{
		serverName: serverName,
		tool:       tool,
		client:     client,
		logger:     logger,
	}
}

// Name 获取工具名称（包含服务器前缀以避免冲突）
func (a *MCPToolAdapter) Name() string {
	return fmt.Sprintf("mcp_%s_%s", a.serverName, a.tool.Name)
}

// Description 获取工具描述
func (a *MCPToolAdapter) Description() string {
	return a.tool.Description
}

// Parameters 获取参数定义
func (a *MCPToolAdapter) Parameters() map[string]interface{} {
	// ToolArgumentsSchema 包含 JSON schema 信息
	schema := a.tool.InputSchema
	result := make(map[string]interface{})

	if schema.Type != "" {
		result["type"] = schema.Type
	}
	if len(schema.Properties) > 0 {
		result["properties"] = schema.Properties
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}
	if len(schema.Defs) > 0 {
		result["$defs"] = schema.Defs
	}

	return result
}

// Execute 执行工具
func (a *MCPToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	a.logger.Debug("Executing MCP tool",
		"server", a.serverName,
		"tool", a.tool.Name,
		"params", params)

	result, err := a.client.CallTool(ctx, a.tool.Name, params)
	if err != nil {
		a.logger.Error("MCP tool call failed",
			"server", a.serverName,
			"tool", a.tool.Name,
			"error", err)
		return "", fmt.Errorf("MCP tool %s failed: %w", a.tool.Name, err)
	}

	a.logger.Debug("MCP tool executed",
		"server", a.serverName,
		"tool", a.tool.Name,
		"result_length", len(result))

	return result, nil
}

// ToDefinition 转换为工具定义
func (a *MCPToolAdapter) ToDefinition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Type: "function",
		Function: tools.FunctionDefinition{
			Name:        a.Name(),
			Description: a.Description(),
			Parameters:  a.Parameters(),
		},
	}
}

// MCPToolsManager MCP 工具管理器
type MCPToolsManager struct {
	registry    *tools.Registry
	clientMgr   *ClientManager
	logger      *slog.Logger
	mu          sync.RWMutex
	adapters    map[string]*MCPToolAdapter // toolName -> adapter
	serverNames map[string]string          // toolName -> serverName
}

// NewMCPToolsManager 创建 MCP 工具管理器
func NewMCPToolsManager(clientMgr *ClientManager, logger *slog.Logger) *MCPToolsManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &MCPToolsManager{
		registry:    tools.NewRegistry(),
		clientMgr:   clientMgr,
		logger:      logger,
		adapters:    make(map[string]*MCPToolAdapter),
		serverNames: make(map[string]string),
	}
}

// GetRegistry 获取工具注册表
func (m *MCPToolsManager) GetRegistry() *tools.Registry {
	return m.registry
}

// LoadTools 加载所有 MCP 工具
func (m *MCPToolsManager) LoadTools() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	allTools := m.clientMgr.GetAllTools()
	loadedCount := 0

	for serverName, toolsList := range allTools {
		client, ok := m.clientMgr.GetClient(serverName)
		if !ok {
			m.logger.Warn("Client not found for server", "server", serverName)
			continue
		}

		for _, tool := range toolsList {
			adapter := NewMCPToolAdapter(serverName, tool, client, m.logger)
			toolName := adapter.Name()

			m.adapters[toolName] = adapter
			m.serverNames[toolName] = serverName
			m.registry.Register(adapter)
			loadedCount++
		}
	}

	m.logger.Info("MCP tools loaded", "count", loadedCount)
	return nil
}

// UnloadTools 卸载所有 MCP 工具
func (m *MCPToolsManager) UnloadTools() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.adapters {
		delete(m.adapters, name)
	}
	for name := range m.serverNames {
		delete(m.serverNames, name)
	}

	// 重新创建注册表
	m.registry = tools.NewRegistry()

	m.logger.Info("MCP tools unloaded")
	return nil
}

// ReloadTools 重新加载工具
func (m *MCPToolsManager) ReloadTools() error {
	m.UnloadTools()
	return m.LoadTools()
}

// GetToolDefinitions 获取工具定义列表
func (m *MCPToolsManager) GetToolDefinitions() []tools.ToolDefinition {
	return m.registry.ToDefinitions()
}

// ToolCount 工具数量
func (m *MCPToolsManager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.adapters)
}

// GetServerForTool 获取工具对应的服务器名称
func (m *MCPToolsManager) GetServerForTool(toolName string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	serverName, ok := m.serverNames[toolName]
	return serverName, ok
}

// MCPResourceAdapter MCP 资源适配器
type MCPResourceAdapter struct {
	serverName string
	resource   mcp.Resource
	client     *Client
	logger     *slog.Logger
}

// NewMCPResourceAdapter 创建资源适配器
func NewMCPResourceAdapter(serverName string, resource mcp.Resource, client *Client, logger *slog.Logger) *MCPResourceAdapter {
	return &MCPResourceAdapter{
		serverName: serverName,
		resource:   resource,
		client:     client,
		logger:     logger,
	}
}

// URI 获取资源 URI
func (a *MCPResourceAdapter) URI() string {
	return a.resource.URI
}

// Name 获取资源名称
func (a *MCPResourceAdapter) Name() string {
	return a.resource.Name
}

// Description 获取资源描述
func (a *MCPResourceAdapter) Description() string {
	return a.resource.Description
}

// MIMEType 获取 MIME 类型
func (a *MCPResourceAdapter) MIMEType() string {
	return a.resource.MIMEType
}

// Read 读取资源内容
func (a *MCPResourceAdapter) Read(ctx context.Context) (string, error) {
	a.logger.Debug("Reading MCP resource",
		"server", a.serverName,
		"uri", a.resource.URI)

	content, err := a.client.ReadResource(ctx, a.resource.URI)
	if err != nil {
		a.logger.Error("Failed to read MCP resource",
			"server", a.serverName,
			"uri", a.resource.URI,
			"error", err)
		return "", fmt.Errorf("read resource %s failed: %w", a.resource.URI, err)
	}

	return content, nil
}

// MCPResourceManager MCP 资源管理器
type MCPResourceManager struct {
	clientMgr *ClientManager
	logger    *slog.Logger
	mu        sync.RWMutex
	resources map[string]*MCPResourceAdapter // uri -> adapter
}

// NewMCPResourceManager 创建资源管理器
func NewMCPResourceManager(clientMgr *ClientManager, logger *slog.Logger) *MCPResourceManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &MCPResourceManager{
		clientMgr: clientMgr,
		logger:    logger,
		resources: make(map[string]*MCPResourceAdapter),
	}
}

// LoadResources 加载所有资源
func (m *MCPResourceManager) LoadResources() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := m.clientMgr.ListClients()
	for _, serverName := range clients {
		client, ok := m.clientMgr.GetClient(serverName)
		if !ok {
			continue
		}

		for _, resource := range client.Resources() {
			adapter := NewMCPResourceAdapter(serverName, resource, client, m.logger)
			m.resources[resource.URI] = adapter
		}
	}

	m.logger.Info("MCP resources loaded", "count", len(m.resources))
	return nil
}

// Get 获取资源
func (m *MCPResourceManager) Get(uri string) (*MCPResourceAdapter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, ok := m.resources[uri]
	return adapter, ok
}

// List 列出所有资源
func (m *MCPResourceManager) List() []*MCPResourceAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*MCPResourceAdapter, 0, len(m.resources))
	for _, adapter := range m.resources {
		result = append(result, adapter)
	}
	return result
}

// Read 读取资源
func (m *MCPResourceManager) Read(ctx context.Context, uri string) (string, error) {
	adapter, ok := m.Get(uri)
	if !ok {
		return "", fmt.Errorf("resource not found: %s", uri)
	}
	return adapter.Read(ctx)
}

// ResourceCount 资源数量
func (m *MCPResourceManager) ResourceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.resources)
}

// MCPPromptAdapter MCP 提示适配器
type MCPPromptAdapter struct {
	serverName string
	prompt     mcp.Prompt
	client     *Client
	logger     *slog.Logger
}

// NewMCPPromptAdapter 创建提示适配器
func NewMCPPromptAdapter(serverName string, prompt mcp.Prompt, client *Client, logger *slog.Logger) *MCPPromptAdapter {
	return &MCPPromptAdapter{
		serverName: serverName,
		prompt:     prompt,
		client:     client,
		logger:     logger,
	}
}

// Name 获取提示名称
func (a *MCPPromptAdapter) Name() string {
	return fmt.Sprintf("mcp_%s_%s", a.serverName, a.prompt.Name)
}

// Description 获取提示描述
func (a *MCPPromptAdapter) Description() string {
	return a.prompt.Description
}

// Get 获取提示内容
func (a *MCPPromptAdapter) Get(ctx context.Context, arguments map[string]string) (string, error) {
	a.logger.Debug("Getting MCP prompt",
		"server", a.serverName,
		"prompt", a.prompt.Name,
		"args", arguments)

	content, err := a.client.GetPrompt(ctx, a.prompt.Name, arguments)
	if err != nil {
		a.logger.Error("Failed to get MCP prompt",
			"server", a.serverName,
			"prompt", a.prompt.Name,
			"error", err)
		return "", fmt.Errorf("get prompt %s failed: %w", a.prompt.Name, err)
	}

	return content, nil
}

// MCPPromptManager MCP 提示管理器
type MCPPromptManager struct {
	clientMgr *ClientManager
	logger    *slog.Logger
	mu        sync.RWMutex
	prompts   map[string]*MCPPromptAdapter // name -> adapter
}

// NewMCPPromptManager 创建提示管理器
func NewMCPPromptManager(clientMgr *ClientManager, logger *slog.Logger) *MCPPromptManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &MCPPromptManager{
		clientMgr: clientMgr,
		logger:    logger,
		prompts:   make(map[string]*MCPPromptAdapter),
	}
}

// LoadPrompts 加载所有提示
func (m *MCPPromptManager) LoadPrompts() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := m.clientMgr.ListClients()
	for _, serverName := range clients {
		client, ok := m.clientMgr.GetClient(serverName)
		if !ok {
			continue
		}

		for _, prompt := range client.Prompts() {
			adapter := NewMCPPromptAdapter(serverName, prompt, client, m.logger)
			m.prompts[adapter.Name()] = adapter
		}
	}

	m.logger.Info("MCP prompts loaded", "count", len(m.prompts))
	return nil
}

// Get 获取提示
func (m *MCPPromptManager) Get(name string) (*MCPPromptAdapter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, ok := m.prompts[name]
	return adapter, ok
}

// List 列出所有提示
func (m *MCPPromptManager) List() []*MCPPromptAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*MCPPromptAdapter, 0, len(m.prompts))
	for _, adapter := range m.prompts {
		result = append(result, adapter)
	}
	return result
}

// PromptCount 提示数量
func (m *MCPPromptManager) PromptCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.prompts)
}

// ConvertToolsToJSONSchema 将 MCP 工具参数转换为 JSON Schema
func ConvertToolsToJSONSchema(tool mcp.Tool) map[string]interface{} {
	schema := tool.InputSchema
	result := make(map[string]interface{})

	if schema.Type != "" {
		result["type"] = schema.Type
	}
	if len(schema.Properties) > 0 {
		result["properties"] = schema.Properties
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}
	if len(schema.Defs) > 0 {
		result["$defs"] = schema.Defs
	}

	return result
}

// BuildToolDescription 构建工具描述字符串
func BuildToolDescription(tool mcp.Tool) string {
	var parts []string

	if tool.Description != "" {
		parts = append(parts, tool.Description)
	}

	// 显示输入 schema 信息
	schema := tool.InputSchema
	if len(schema.Required) > 0 {
		parts = append(parts, "\nRequired arguments:", strings.Join(schema.Required, ", "))
	}
	if len(schema.Properties) > 0 {
		parts = append(parts, "\nAvailable properties:", strings.Join(getKeys(schema.Properties), ", "))
	}

	return strings.Join(parts, "\n")
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
