// Package mcp provides MCP (Model Context Protocol) support for icooclaw.
package mcp

import (
	"context"
	"fmt"
	"icooclaw/pkg/tools"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ConnectionState represents the connection state of an MCP client.
type ConnectionState int

const (
	// ConnectionStateDisconnected indicates the client is disconnected.
	ConnectionStateDisconnected ConnectionState = iota
	// ConnectionStateConnecting indicates the client is connecting.
	ConnectionStateConnecting
	// ConnectionStateConnected indicates the client is connected.
	ConnectionStateConnected
	// ConnectionStateError indicates the client encountered an error.
	ConnectionStateError
)

func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateDisconnected:
		return "disconnected"
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateConnected:
		return "connected"
	case ConnectionStateError:
		return "error"
	default:
		return "unknown"
	}
}

// ClientConfig holds configuration for MCP client connection.
type ClientConfig struct {
	// Command is the command to execute for stdio connection.
	Command string `json:"command,omitempty"`
	// Args are the arguments for the command.
	Args []string `json:"args,omitempty"`
	// Env is the environment variables for the command.
	Env map[string]string `json:"env,omitempty"`
	// URL is the SSE endpoint URL.
	URL string `json:"url,omitempty"`
	// RetryCount is the number of retry attempts.
	RetryCount int `json:"retry_count,omitempty"`
	// RetryDelay is the delay between retries.
	RetryDelay time.Duration `json:"retry_delay,omitempty"`
	// Timeout is the connection timeout.
	Timeout time.Duration `json:"timeout,omitempty"`
}

// Client represents an MCP client connection.
type Client struct {
	name          string
	client        *client.Client
	config        ClientConfig
	tools         map[string]mcp.Tool
	logger        *slog.Logger
	mu            sync.RWMutex
	state         ConnectionState
	stateMu       sync.RWMutex
	cancelFunc    context.CancelFunc
	cancelCtx     context.Context
	lastError     error
	lastErrorAt   time.Time
	retryCount    int
	onStateChange func(string, ConnectionState)
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithLogger sets the logger for the client.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithRetryConfig sets the retry configuration.
func WithRetryConfig(retryCount int, retryDelay time.Duration) ClientOption {
	return func(c *Client) {
		c.retryCount = retryCount
	}
}

// WithStateChangeHandler sets the state change handler.
func WithStateChangeHandler(handler func(string, ConnectionState)) ClientOption {
	return func(c *Client) {
		c.onStateChange = handler
	}
}

// NewClient creates a new MCP client.
func NewClient(name string, opts ...ClientOption) *Client {
	c := &Client{
		name:       name,
		tools:      make(map[string]mcp.Tool),
		logger:     slog.Default(),
		retryCount: 3,
		state:      ConnectionStateDisconnected,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ConnectStdio connects to an MCP server via stdio.
func (c *Client) ConnectStdio(ctx context.Context, command string, args []string, env map[string]string) error {
	c.config = ClientConfig{
		Command:    command,
		Args:       args,
		Env:        env,
		RetryCount: c.retryCount,
		RetryDelay: 1 * time.Second,
		Timeout:    30 * time.Second,
	}

	return c.connect(ctx, func(ctx context.Context) error {
		cli, err := client.NewStdioMCPClient(command, args)
		if err != nil {
			return fmt.Errorf("failed to create stdio client: %w", err)
		}
		c.client = cli
		return nil
	})
}

// ConnectSSE connects to an MCP server via SSE.
func (c *Client) ConnectSSE(ctx context.Context, url string) error {
	c.config = ClientConfig{
		URL:        url,
		RetryCount: c.retryCount,
		RetryDelay: 1 * time.Second,
		Timeout:    30 * time.Second,
	}

	return c.connect(ctx, func(ctx context.Context) error {
		cli, err := client.NewSSEMCPClient(url)
		if err != nil {
			return fmt.Errorf("failed to create SSE client: %w", err)
		}
		c.client = cli

		// Start the client
		if err := cli.Start(ctx); err != nil {
			return fmt.Errorf("failed to start SSE client: %w", err)
		}
		return nil
	})
}

// connect establishes the connection with retry logic.
func (c *Client) connect(ctx context.Context, connectFunc func(context.Context) error) error {
	// Create cancelable context
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())

	var lastErr error
	attempts := 0
	maxAttempts := c.config.RetryCount + 1

	for attempts < maxAttempts {
		c.setState(ConnectionStateConnecting)
		c.logger.Info("connecting to MCP server", "name", c.name, "attempt", attempts+1, "max_attempts", maxAttempts)

		connectCtx := ctx
		if c.config.Timeout > 0 {
			var cancel context.CancelFunc
			connectCtx, cancel = context.WithTimeout(ctx, c.config.Timeout)
			defer cancel()
		}

		err := connectFunc(connectCtx)
		if err == nil {
			// Initialize
			if err := c.initialize(connectCtx); err != nil {
				lastErr = err
				c.logger.Error("failed to initialize MCP connection", "error", err)
				attempts++
				if attempts < maxAttempts {
					time.Sleep(c.config.RetryDelay)
				}
				continue
			}

			// List tools
			if err := c.listTools(connectCtx); err != nil {
				lastErr = err
				c.logger.Error("failed to list tools", "error", err)
				attempts++
				if attempts < maxAttempts {
					time.Sleep(c.config.RetryDelay)
				}
				continue
			}

			c.setState(ConnectionStateConnected)
			c.logger.Info("MCP connection established", "name", c.name, "tools_count", len(c.tools))
			return nil
		}

		lastErr = err
		c.logger.Error("connection attempt failed", "name", c.name, "attempt", attempts+1, "error", err)
		attempts++

		if attempts < maxAttempts {
			time.Sleep(c.config.RetryDelay)
		}
	}

	c.setState(ConnectionStateError)
	c.lastError = lastErr
	c.lastErrorAt = time.Now()
	return fmt.Errorf("failed to connect after %d attempts: %w", attempts, lastErr)
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

	c.logger.Debug("MCP initialized", "name", c.name, "protocol_version", mcp.LATEST_PROTOCOL_VERSION)
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

	c.tools = make(map[string]mcp.Tool, len(result.Tools))
	for _, tool := range result.Tools {
		c.tools[tool.Name] = tool
		c.logger.Debug("discovered MCP tool", "name", tool.Name, "description", tool.Description)
	}

	return nil
}

// Close closes the MCP connection.
func (c *Client) Close() error {
	// Cancel any pending operations
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	if c.client != nil {
		err := c.client.Close()
		c.setState(ConnectionStateDisconnected)
		return err
	}
	c.setState(ConnectionStateDisconnected)
	return nil
}

// Reconnect attempts to reconnect to the MCP server.
func (c *Client) Reconnect(ctx context.Context) error {
	c.logger.Info("reconnecting to MCP server", "name", c.name)

	// Close existing connection
	if err := c.Close(); err != nil {
		c.logger.Warn("failed to close existing connection", "error", err)
	}

	// Reconnect based on config
	if c.config.Command != "" {
		return c.ConnectStdio(ctx, c.config.Command, c.config.Args, c.config.Env)
	} else if c.config.URL != "" {
		return c.ConnectSSE(ctx, c.config.URL)
	}

	return fmt.Errorf("no connection configuration available")
}

// GetState returns the current connection state.
func (c *Client) GetState() ConnectionState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// IsConnected returns true if the client is connected.
func (c *Client) IsConnected() bool {
	return c.GetState() == ConnectionStateConnected
}

// GetLastError returns the last error and when it occurred.
func (c *Client) GetLastError() (error, time.Time) {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.lastError, c.lastErrorAt
}

// setState sets the connection state and notifies listeners.
func (c *Client) setState(state ConnectionState) {
	c.stateMu.Lock()
	oldState := c.state
	c.state = state
	c.stateMu.Unlock()

	if oldState != state && c.onStateChange != nil {
		c.onStateChange(c.name, state)
	}
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

// GetToolNames returns the names of all discovered tools.
func (c *Client) GetToolNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.tools))
	for name := range c.tools {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// HasTool checks if a tool is available.
func (c *Client) HasTool(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.tools[name]
	return ok
}

// SearchTools searches for tools by keyword.
func (c *Client) SearchTools(keyword string) []mcp.Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keyword = strings.ToLower(keyword)
	var results []mcp.Tool

	for _, tool := range c.tools {
		if strings.Contains(strings.ToLower(tool.Name), keyword) ||
			strings.Contains(strings.ToLower(tool.Description), keyword) {
			results = append(results, tool)
		}
	}

	return results
}

// ExecuteTool executes an MCP tool.
func (c *Client) ExecuteTool(ctx context.Context, name string, args map[string]any) (*tools.Result, error) {
	c.mu.RLock()
	tool, ok := c.tools[name]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}

	// Validate arguments against schema if available
	if err := c.validateArgs(tool, args); err != nil {
		return &tools.Result{
			Success: false,
			Error:   err,
		}, nil
	}

	c.logger.Debug("executing MCP tool", "name", name, "args_count", len(args))

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}

	result, err := c.client.CallTool(ctx, req)
	if err != nil {
		c.logger.Error("MCP tool execution failed", "name", name, "error", err)
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("tool execution failed: %w", err),
		}, nil
	}

	// Extract content
	var content strings.Builder
	for _, item := range result.Content {
		switch v := item.(type) {
		case mcp.TextContent:
			content.WriteString(v.Text)
		case mcp.ImageContent:
			content.WriteString(fmt.Sprintf("[Image: %s]", v.MIMEType))
		}
	}

	if result.IsError {
		return &tools.Result{
			Success: false,
			Error:   fmt.Errorf("tool returned error: %s", content.String()),
		}, nil
	}

	return &tools.Result{
		Success: true,
		Content: content.String(),
	}, nil
}

// validateArgs validates arguments against the tool's input schema.
func (c *Client) validateArgs(tool mcp.Tool, args map[string]any) error {
	schema := tool.InputSchema
	
	// Check required parameters
	if len(schema.Required) > 0 {
		for _, req := range schema.Required {
			if _, exists := args[req]; !exists {
				return fmt.Errorf("missing required parameter: %s", req)
			}
		}
	}

	// Check parameter types
	if len(schema.Properties) > 0 {
		for paramName, prop := range schema.Properties {
			propMap, ok := prop.(map[string]any)
			if !ok {
				continue
			}

			argValue, exists := args[paramName]
			if !exists {
				continue
			}

			expectedType, ok := propMap["type"].(string)
			if !ok {
				continue
			}

			if err := c.checkType(argValue, expectedType); err != nil {
				return fmt.Errorf("invalid type for parameter %s: %w", paramName, err)
			}
		}
	}

	return nil
}

// checkType checks if a value matches the expected type.
func (c *Client) checkType(value any, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
	case "number", "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return nil
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("expected array, got %T", value)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", value)
		}
	}
	return nil
}

// ExecuteToolWithRetry executes an MCP tool with retry logic.
func (c *Client) ExecuteToolWithRetry(ctx context.Context, name string, args map[string]any, maxRetries int) (*tools.Result, error) {
	var lastResult *tools.Result
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := c.ExecuteTool(ctx, name, args)
		if err == nil && result.Success {
			return result, nil
		}

		lastResult = result
		lastErr = err

		c.logger.Warn("tool execution failed, retrying",
			"name", name,
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"error", err)

		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond * time.Duration(attempt+1)):
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return lastResult, nil
}

// BatchExecute executes multiple tools in sequence.
func (c *Client) BatchExecute(ctx context.Context, requests []BatchRequest) []BatchResult {
	results := make([]BatchResult, len(requests))

	for i, req := range requests {
		select {
		case <-ctx.Done():
			results[i] = BatchResult{
				Name:  req.Name,
				Error: ctx.Err(),
			}
			continue
		default:
		}

		result, err := c.ExecuteTool(ctx, req.Name, req.Args)
		results[i] = BatchResult{
			Name:   req.Name,
			Result: result,
			Error:  err,
		}
	}

	return results
}

// BatchExecuteParallel executes multiple tools in parallel.
func (c *Client) BatchExecuteParallel(ctx context.Context, requests []BatchRequest) []BatchResult {
	results := make([]BatchResult, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request BatchRequest) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				results[index] = BatchResult{
					Name:  request.Name,
					Error: ctx.Err(),
				}
				return
			default:
			}

			result, err := c.ExecuteTool(ctx, request.Name, request.Args)
			results[index] = BatchResult{
				Name:   request.Name,
				Result: result,
				Error:  err,
			}
		}(i, req)
	}

	wg.Wait()
	return results
}

// BatchRequest represents a batch tool execution request.
type BatchRequest struct {
	Name string
	Args map[string]any
}

// BatchResult represents a batch tool execution result.
type BatchResult struct {
	Name   string
	Result *tools.Result
	Error  error
}

// MCPTool wraps an MCP tool as a tools.Tool.
type MCPTool struct {
	name        string
	description string
	parameters  map[string]any
	client      *Client
}

// NewMCPTool creates a new MCP tool wrapper.
func NewMCPTool(tool mcp.Tool, client *Client) *MCPTool {
	params := make(map[string]any)

	// Parse input schema
	schema := tool.InputSchema

	// Parse properties
	for name, prop := range schema.Properties {
		if p, ok := prop.(map[string]any); ok {
			param := map[string]any{
				"type":        getString(p, "type"),
				"description": getString(p, "description"),
			}
			if enum, ok := p["enum"].([]any); ok {
				param["enum"] = enum
			}
			if def, ok := p["default"]; ok {
				param["default"] = def
			}
			params[name] = param
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
func (t *MCPTool) Parameters() map[string]any {
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
	clients       map[string]*Client
	tools         *tools.Registry
	logger        *slog.Logger
	mu            sync.RWMutex
	stateHandlers []func(string, ConnectionState)
}

// ManagerOption is a function that configures a Manager.
type ManagerOption func(*Manager)

// WithManagerLogger sets the logger for the manager.
func WithManagerLogger(logger *slog.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// WithManagerStateChangeHandler adds a state change handler to all clients.
func WithManagerStateChangeHandler(handler func(string, ConnectionState)) ManagerOption {
	return func(m *Manager) {
		m.stateHandlers = append(m.stateHandlers, handler)
	}
}

// NewManager creates a new MCP manager.
func NewManager(registry *tools.Registry, opts ...ManagerOption) *Manager {
	m := &Manager{
		clients: make(map[string]*Client),
		tools:   registry,
		logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// CreateAndAddClient creates a new client and adds it to the manager.
func (m *Manager) CreateAndAddClient(name string, opts ...ClientOption) *Client {
	// Create client with base options
	client := NewClient(name, opts...)
	
	// Set state change handler
	client.onStateChange = func(n string, s ConnectionState) {
		m.handleStateChange(n, s)
	}
	
	m.AddClient(name, client)
	return client
}

// handleStateChange handles state changes from clients.
func (m *Manager) handleStateChange(name string, state ConnectionState) {
	m.logger.Debug("MCP client state changed", "name", name, "state", state)

	for _, handler := range m.stateHandlers {
		handler(name, state)
	}
}

// AddClient adds an MCP client.
func (m *Manager) AddClient(name string, client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set state change handler
	client.onStateChange = func(n string, s ConnectionState) {
		m.handleStateChange(n, s)
	}

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
		m.logger.Info("unregistered MCP tool", "client", name, "tool", tool.Name)
	}

	// Close client
	if err := client.Close(); err != nil {
		m.logger.Error("failed to close MCP client", "name", name, "error", err)
	}

	delete(m.clients, name)
	return nil
}

// GetClient gets a client by name.
func (m *Manager) GetClient(name string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[name]
}

// ListClients returns all client names.
func (m *Manager) ListClients() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Close closes all MCP clients.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("client %s: %w", name, err))
			m.logger.Error("failed to close MCP client", "name", name, "error", err)
		}
	}

	m.clients = make(map[string]*Client)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}
	return nil
}

// GetConnectionStatus returns the connection status of all clients.
func (m *Manager) GetConnectionStatus() map[string]ConnectionState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]ConnectionState, len(m.clients))
	for name, client := range m.clients {
		status[name] = client.GetState()
	}
	return status
}

// ReconnectAll attempts to reconnect all disconnected clients.
func (m *Manager) ReconnectAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error
	for name, client := range m.clients {
		if client.GetState() != ConnectionStateConnected {
			m.logger.Info("reconnecting client", "name", name)
			if err := client.Reconnect(ctx); err != nil {
				errs = append(errs, fmt.Errorf("client %s: %w", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors reconnecting: %v", errs)
	}
	return nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
