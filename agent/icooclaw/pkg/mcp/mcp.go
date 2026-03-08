// Package mcp provides MCP (Model Context Protocol) support for icooclaw.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client represents an MCP client connection.
type Client struct {
	name   string
	client *client.Client
	tools  map[string]mcp.Tool
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewClient creates a new MCP client.
func NewClient(name string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		name:   name,
		tools:  make(map[string]mcp.Tool),
		logger: logger,
	}
}

// ConnectStdio connects to an MCP server via stdio.
func (c *Client) ConnectStdio(ctx context.Context, command string, args []string, env map[string]string) error {
	cli, err := client.NewStdioMCPClient(command, args)
	if err != nil {
		return fmt.Errorf("failed to create stdio client: %w", err)
	}

	c.client = cli

	// Initialize
	if err := c.initialize(ctx); err != nil {
		return err
	}

	// List tools
	if err := c.listTools(ctx); err != nil {
		return err
	}

	return nil
}

// ConnectSSE connects to an MCP server via SSE.
func (c *Client) ConnectSSE(ctx context.Context, url string) error {
	cli, err := client.NewSSEMCPClient(url)
	if err != nil {
		return fmt.Errorf("failed to create SSE client: %w", err)
	}

	c.client = cli

	// Start the client
	if err := cli.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SSE client: %w", err)
	}

	// Initialize
	if err := c.initialize(ctx); err != nil {
		return err
	}

	// List tools
	if err := c.listTools(ctx); err != nil {
		return err
	}

	return nil
}

// initialize initializes the MCP connection.
func (c *Client) initialize(ctx context.Context) error {
	req := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "icooclaw",
				Version: "1.0.0",
			},
			Capabilities: mcp.ClientCapabilities{},
		},
	}

	_, err := c.client.Initialize(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	return nil
}

// listTools lists available tools from the MCP server.
func (c *Client) listTools(ctx context.Context) error {
	req := mcp.ListToolsRequest{}

	result, err := c.client.ListTools(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, tool := range result.Tools {
		c.tools[tool.Name] = tool
		c.logger.Debug("discovered MCP tool", "name", tool.Name, "description", tool.Description)
	}

	return nil
}

// Close closes the MCP connection.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetTools returns all discovered MCP tools.
func (c *Client) GetTools() map[string]mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tools := make(map[string]mcp.Tool, len(c.tools))
	for k, v := range c.tools {
		tools[k] = v
	}
	return tools
}

// ExecuteTool executes an MCP tool.
func (c *Client) ExecuteTool(ctx context.Context, name string, args map[string]any) (*tools.Result, error) {
	c.mu.RLock()
	_, ok := c.tools[name]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	// Convert args to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal args: %w", err)
	}

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: argsJSON,
		},
	}

	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}, nil
	}

	// Extract content
	var content string
	for _, item := range result.Content {
		if text, ok := item.(mcp.TextContent); ok {
			content += text.Text
		}
	}

	return &tools.Result{
		Success: true,
		Content: content,
	}, nil
}

// MCPTool wraps an MCP tool as a tools.Tool.
type MCPTool struct {
	name        string
	description string
	parameters  map[string]tools.Parameter
	client      *Client
}

// NewMCPTool creates a new MCP tool wrapper.
func NewMCPTool(tool mcp.Tool, client *Client) *MCPTool {
	params := make(map[string]tools.Parameter)

	// Parse input schema
	if tool.InputSchema != nil {
		if props, ok := tool.InputSchema["properties"].(map[string]any); ok {
			for name, prop := range props {
				if p, ok := prop.(map[string]any); ok {
					param := tools.Parameter{
						Type:        getString(p, "type"),
						Description: getString(p, "description"),
					}
					if enum, ok := p["enum"].([]any); ok {
						for _, e := range enum {
							param.Enum = append(param.Enum, fmt.Sprintf("%v", e))
						}
					}
					params[name] = param
				}
			}
		}
	}

	return &MCPTool{
		name:        tool.Name,
		description: tool.Description,
		parameters:  params,
		client:      client,
	}
}

// Name returns the tool name.
func (t *MCPTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *MCPTool) Description() string {
	return t.description
}

// Parameters returns the tool parameters.
func (t *MCPTool) Parameters() map[string]tools.Parameter {
	return t.parameters
}

// Execute executes the tool.
func (t *MCPTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	result, err := t.client.ExecuteTool(ctx, t.name, args)
	if err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}
	}
	return result
}

// Manager manages multiple MCP clients.
type Manager struct {
	clients map[string]*Client
	tools   *tools.Registry
	logger  *slog.Logger
	mu      sync.RWMutex
}

// NewManager creates a new MCP manager.
func NewManager(registry *tools.Registry, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		clients: make(map[string]*Client),
		tools:   registry,
		logger:  logger,
	}
}

// AddClient adds an MCP client.
func (m *Manager) AddClient(name string, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[name] = client

	// Register tools
	for _, tool := range client.GetTools() {
		m.tools.Register(NewMCPTool(tool, client))
		m.logger.Info("registered MCP tool", "client", name, "tool", tool.Name)
	}
}

// RemoveClient removes an MCP client.
func (m *Manager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[name]
	if !ok {
		return nil
	}

	// Unregister tools
	for _, tool := range client.GetTools() {
		m.tools.Unregister(tool.Name)
	}

	// Close client
	client.Close()
	delete(m.clients, name)

	return nil
}

// Close closes all MCP clients.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			m.logger.Error("failed to close MCP client", "name", name, "error", err)
		}
	}

	m.clients = make(map[string]*Client)
	return nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
