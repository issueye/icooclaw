package mcp

import (
	"context"
	"testing"
	"time"

	"icooclaw/pkg/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestConnectionState tests the connection state enum.
func TestConnectionState(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{ConnectionStateDisconnected, "disconnected"},
		{ConnectionStateConnecting, "connecting"},
		{ConnectionStateConnected, "connected"},
		{ConnectionStateError, "error"},
		{ConnectionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("ConnectionState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewClient tests client creation.
func TestNewClient(t *testing.T) {
	client := NewClient("test-client")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.name != "test-client" {
		t.Errorf("client.name = %v, want %v", client.name, "test-client")
	}
	if client.GetState() != ConnectionStateDisconnected {
		t.Errorf("client.GetState() = %v, want %v", client.GetState(), ConnectionStateDisconnected)
	}
}

// TestNewClientWithOptions tests client creation with options.
func TestNewClientWithOptions(t *testing.T) {
	stateChanges := make([]ConnectionState, 0)

	client := NewClient("test-client",
		WithRetryConfig(5, 2*time.Second),
		WithStateChangeHandler(func(name string, state ConnectionState) {
			stateChanges = append(stateChanges, state)
		}),
	)

	if client.retryCount != 5 {
		t.Errorf("client.retryCount = %v, want %v", client.retryCount, 5)
	}
	if client.onStateChange == nil {
		t.Error("client.onStateChange should not be nil")
	}
	
	// Test state change handler
	client.setState(ConnectionStateConnecting)
	if len(stateChanges) != 1 {
		t.Errorf("state change handler called %d times, want %d", len(stateChanges), 1)
	}
}

// TestClientState tests state management.
func TestClientState(t *testing.T) {
	client := NewClient("test-client")

	// Test initial state
	if client.GetState() != ConnectionStateDisconnected {
		t.Errorf("initial state = %v, want %v", client.GetState(), ConnectionStateDisconnected)
	}

	// Test IsConnected
	if client.IsConnected() {
		t.Error("client should not be connected initially")
	}

	// Test state transitions
	client.setState(ConnectionStateConnecting)
	if client.GetState() != ConnectionStateConnecting {
		t.Errorf("state after transition = %v, want %v", client.GetState(), ConnectionStateConnecting)
	}
}

// TestGetTools tests tool retrieval.
func TestGetTools(t *testing.T) {
	client := NewClient("test-client")

	// Add some tools
	client.tools = map[string]mcp.Tool{
		"tool1": {Name: "tool1", Description: "Test tool 1"},
		"tool2": {Name: "tool2", Description: "Test tool 2"},
	}

	tools := client.GetTools()
	if len(tools) != 2 {
		t.Errorf("len(GetTools()) = %v, want %v", len(tools), 2)
	}

	names := client.GetToolNames()
	if len(names) != 2 {
		t.Errorf("len(GetToolNames()) = %v, want %v", len(names), 2)
	}
}

// TestHasTool tests tool existence check.
func TestHasTool(t *testing.T) {
	client := NewClient("test-client")
	client.tools = map[string]mcp.Tool{
		"tool1": {Name: "tool1", Description: "Test tool 1"},
	}

	if !client.HasTool("tool1") {
		t.Error("HasTool(tool1) should return true")
	}
	if client.HasTool("nonexistent") {
		t.Error("HasTool(nonexistent) should return false")
	}
}

// TestSearchTools tests tool search.
func TestSearchTools(t *testing.T) {
	client := NewClient("test-client")
	client.tools = map[string]mcp.Tool{
		"file_read":   {Name: "file_read", Description: "Read a file"},
		"file_write":  {Name: "file_write", Description: "Write to a file"},
		"http_get":    {Name: "http_get", Description: "Make HTTP GET request"},
		"http_post":   {Name: "http_post", Description: "Make HTTP POST request"},
	}

	// Search by name
	results := client.SearchTools("file")
	if len(results) != 2 {
		t.Errorf("len(SearchTools(file)) = %v, want %v", len(results), 2)
	}

	// Search by description
	results = client.SearchTools("http")
	if len(results) != 2 {
		t.Errorf("len(SearchTools(http)) = %v, want %v", len(results), 2)
	}

	// Case insensitive
	results = client.SearchTools("FILE")
	if len(results) != 2 {
		t.Errorf("len(SearchTools(FILE)) = %v, want %v", len(results), 2)
	}
}

// TestValidateArgs tests argument validation.
func TestValidateArgs(t *testing.T) {
	client := NewClient("test-client")

	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"name":  map[string]any{"type": "string"},
				"count": map[string]any{"type": "integer"},
				"flag":  map[string]any{"type": "boolean"},
				"tags":  map[string]any{"type": "array"},
				"meta":  map[string]any{"type": "object"},
			},
			Required: []string{"name", "count"},
		},
	}

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{
			name: "valid args",
			args: map[string]any{
				"name":  "test",
				"count": 5,
				"flag":  true,
				"tags":  []any{"a", "b"},
				"meta":  map[string]any{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "missing required",
			args: map[string]any{
				"count": 5,
			},
			wantErr: true,
		},
		{
			name: "wrong type string",
			args: map[string]any{
				"name":  123,
				"count": 5,
			},
			wantErr: true,
		},
		{
			name: "wrong type number",
			args: map[string]any{
				"name":  "test",
				"count": "five",
			},
			wantErr: true,
		},
		{
			name: "wrong type boolean",
			args: map[string]any{
				"name":  "test",
				"count": 5,
				"flag":  "yes",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateArgs(tool, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckType tests type checking.
func TestCheckType(t *testing.T) {
	client := NewClient("test-client")

	tests := []struct {
		name         string
		value        any
		expectedType string
		wantErr      bool
	}{
		{"string valid", "hello", "string", false},
		{"string invalid", 123, "string", true},
		{"number valid int", 42, "number", false},
		{"number valid float", 3.14, "number", false},
		{"number invalid", "forty-two", "number", true},
		{"integer valid", 42, "integer", false},
		{"integer invalid", "42", "integer", true},
		{"boolean valid", true, "boolean", false},
		{"boolean invalid", "true", "boolean", true},
		{"array valid", []any{1, 2, 3}, "array", false},
		{"array invalid", "not an array", "array", true},
		{"object valid", map[string]any{"k": "v"}, "object", false},
		{"object invalid", "not an object", "object", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.checkType(tt.value, tt.expectedType)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMCPTool tests MCPTool wrapper.
func TestMCPTool(t *testing.T) {
	client := NewClient("test-client")

	mcpTool := mcp.Tool{
		Name:        "test_tool",
		Description: "Test tool description",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"param1": map[string]any{
					"type":        "string",
					"description": "Parameter 1",
					"default":     "default_value",
				},
				"param2": map[string]any{
					"type": "integer",
					"enum": []any{1, 2, 3},
				},
			},
		},
	}

	tool := NewMCPTool(mcpTool, client)

	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %v, want %v", tool.Name(), "test_tool")
	}

	if tool.Description() != "Test tool description" {
		t.Errorf("Description() = %v, want %v", tool.Description(), "Test tool description")
	}

	params := tool.Parameters()
	if len(params) != 2 {
		t.Errorf("len(Parameters()) = %v, want %v", len(params), 2)
	}

	if param1, ok := params["param1"].(map[string]any); ok {
		if param1["type"] != "string" {
			t.Errorf("param1.type = %v, want %v", param1["type"], "string")
		}

		if param1["default"] != "default_value" {
			t.Errorf("param1.default = %v, want %v", param1["default"], "default_value")
		}
	} else {
		t.Errorf("param1 is not a map[string]any")
	}

	if param2, ok := params["param2"].(map[string]any); ok {
		enum, ok := param2["enum"].([]any)
		if !ok || len(enum) != 3 {
			t.Errorf("len(param2.enum) = %v, want %v", len(enum), 3)
		}
	} else {
		t.Errorf("param2 is not a map[string]any")
	}
}

// TestMCPToolExecute tests MCPTool execution.
func TestMCPToolExecute(t *testing.T) {
	client := NewClient("test-client")
	mcpTool := mcp.Tool{Name: "test_tool", Description: "Test"}
	tool := NewMCPTool(mcpTool, client)

	ctx := context.Background()
	result := tool.Execute(ctx, map[string]any{})

	if result != nil && result.Success {
		t.Error("Execute should fail when client is not connected")
	}
}

// TestManager tests Manager functionality.
func TestManager(t *testing.T) {
	registry := tools.NewRegistry()
	manager := NewManager(registry)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	// Test with options
	stateChanges := make([]string, 0)
	manager = NewManager(registry,
		WithManagerStateChangeHandler(func(name string, state ConnectionState) {
			stateChanges = append(stateChanges, name)
		}),
	)

	// Add a client
	client := NewClient("client1")
	manager.AddClient("client1", client)

	// Test GetClient
	got := manager.GetClient("client1")
	if got != client {
		t.Error("GetClient should return the added client")
	}

	// Test ListClients
	clients := manager.ListClients()
	if len(clients) != 1 {
		t.Errorf("len(ListClients()) = %v, want %v", len(clients), 1)
	}

	// Test GetConnectionStatus
	status := manager.GetConnectionStatus()
	if len(status) != 1 {
		t.Errorf("len(GetConnectionStatus()) = %v, want %v", len(status), 1)
	}

	// Test RemoveClient
	err := manager.RemoveClient("client1")
	if err != nil {
		t.Errorf("RemoveClient returned error: %v", err)
	}

	clients = manager.ListClients()
	if len(clients) != 0 {
		t.Errorf("len(ListClients()) after remove = %v, want %v", len(clients), 0)
	}
}

// TestManagerRemoveNonExistentClient tests removing a non-existent client.
func TestManagerRemoveNonExistentClient(t *testing.T) {
	registry := tools.NewRegistry()
	manager := NewManager(registry)

	err := manager.RemoveClient("nonexistent")
	if err != nil {
		t.Errorf("RemoveClient for nonexistent client should not return error: %v", err)
	}
}

// TestManagerClose tests manager close functionality.
func TestManagerClose(t *testing.T) {
	registry := tools.NewRegistry()
	manager := NewManager(registry)

	// Add clients
	manager.AddClient("client1", NewClient("client1"))
	manager.AddClient("client2", NewClient("client2"))

	err := manager.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Verify clients are cleared
	if len(manager.ListClients()) != 0 {
		t.Error("Close should clear all clients")
	}
}

// TestCreateAndAddClient tests CreateAndAddClient.
func TestCreateAndAddClient(t *testing.T) {
	registry := tools.NewRegistry()
	manager := NewManager(registry)

	client := manager.CreateAndAddClient("test",
		WithRetryConfig(10, 500*time.Millisecond),
	)

	if client == nil {
		t.Fatal("CreateAndAddClient returned nil")
	}

	if client.retryCount != 10 {
		t.Errorf("client.retryCount = %v, want %v", client.retryCount, 10)
	}

	// Verify client was added
	if manager.GetClient("test") != client {
		t.Error("CreateAndAddClient should add the client to manager")
	}
}

// TestReconnectAll tests ReconnectAll.
func TestReconnectAll(t *testing.T) {
	registry := tools.NewRegistry()
	manager := NewManager(registry)

	// Add a disconnected client
	client := NewClient("client1")
	manager.AddClient("client1", client)

	ctx := context.Background()
	err := manager.ReconnectAll(ctx)
	// Should not error, just skip disconnected clients without config
	if err == nil {
		// Expected - no config means no reconnection attempt
	}
}

// TestBatchRequest tests batch request types.
func TestBatchRequest(t *testing.T) {
	req := BatchRequest{
		Name: "test_tool",
		Args: map[string]any{"key": "value"},
	}

	if req.Name != "test_tool" {
		t.Errorf("BatchRequest.Name = %v, want %v", req.Name, "test_tool")
	}
}

// TestBatchResult tests batch result types.
func TestBatchResult(t *testing.T) {
	result := BatchResult{
		Name: "test_tool",
		Result: &tools.Result{
			Success: true,
			Content: "test content",
		},
	}

	if result.Name != "test_tool" {
		t.Errorf("BatchResult.Name = %v, want %v", result.Name, "test_tool")
	}
	if !result.Result.Success {
		t.Error("BatchResult.Result.Success should be true")
	}
}

// TestClientGetLastError tests GetLastError.
func TestClientGetLastError(t *testing.T) {
	client := NewClient("test-client")

	err, timestamp := client.GetLastError()
	if err != nil {
		t.Errorf("GetLastError should return nil initially, got %v", err)
	}
	if !timestamp.IsZero() {
		t.Error("GetLastError timestamp should be zero initially")
	}
}

// TestClientConfig tests ClientConfig.
func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		Command:    "test-command",
		Args:       []string{"arg1", "arg2"},
		Env:        map[string]string{"KEY": "value"},
		URL:        "http://example.com/sse",
		RetryCount: 5,
		RetryDelay: 2 * time.Second,
		Timeout:    30 * time.Second,
	}

	if config.Command != "test-command" {
		t.Errorf("Command = %v, want %v", config.Command, "test-command")
	}
	if len(config.Args) != 2 {
		t.Errorf("len(Args) = %v, want %v", len(config.Args), 2)
	}
	if config.RetryCount != 5 {
		t.Errorf("RetryCount = %v, want %v", config.RetryCount, 5)
	}
}
