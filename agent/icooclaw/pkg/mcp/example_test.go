package mcp_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"icooclaw/pkg/mcp"
	"icooclaw/pkg/tools"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// ExampleNewClient demonstrates creating a new MCP client.
func ExampleNewClient() {
	// Create a new client with default settings
	client := mcp.NewClient("my-client")
	fmt.Println("Created client:", client != nil)

	// Create a client with custom logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	clientWithLogger := mcp.NewClient("my-client", mcp.WithLogger(logger))
	fmt.Println("Created client with logger:", clientWithLogger != nil)

	// Create a client with retry configuration
	clientWithRetry := mcp.NewClient("my-client",
		mcp.WithRetryConfig(5, 2*time.Second),
	)
	fmt.Println("Created client with retry:", clientWithRetry != nil)
}

// ExampleClient_ConnectStdio demonstrates connecting to an MCP server via stdio.
func ExampleClient_ConnectStdio() {
	ctx := context.Background()
	client := mcp.NewClient("stdio-client")

	// Connect to an MCP server via stdio
	// Note: This is a demonstration - replace with actual command
	err := client.ConnectStdio(ctx, "npx", []string{"-y", "@modelcontextprotocol/server-example"}, nil)
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer client.Close()

	fmt.Println("Connected successfully")
}

// ExampleClient_ConnectSSE demonstrates connecting to an MCP server via SSE.
func ExampleClient_ConnectSSE() {
	ctx := context.Background()
	client := mcp.NewClient("sse-client")

	// Connect to an MCP server via SSE
	err := client.ConnectSSE(ctx, "http://localhost:8080/sse")
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer client.Close()

	fmt.Println("Connected successfully")
}

// ExampleClient_GetTools demonstrates retrieving available tools.
func ExampleClient_GetTools() {
	client := mcp.NewClient("example-client")

	// Get all available tools
	tools := client.GetTools()
	fmt.Println("Available tools:", len(tools))

	// Get tool names
	names := client.GetToolNames()
	fmt.Println("Tool names:", names)

	// Check if a specific tool exists
	hasTool := client.HasTool("file_read")
	fmt.Println("Has file_read tool:", hasTool)
}

// ExampleClient_SearchTools demonstrates searching for tools.
func ExampleClient_SearchTools() {
	client := mcp.NewClient("example-client")

	// Search for tools by keyword
	fileTools := client.SearchTools("file")
	fmt.Println("File-related tools:", len(fileTools))

	httpTools := client.SearchTools("http")
	fmt.Println("HTTP-related tools:", len(httpTools))
}

// ExampleClient_ExecuteTool demonstrates executing a tool.
func ExampleClient_ExecuteTool() {
	ctx := context.Background()
	client := mcp.NewClient("example-client")

	// Execute a tool with arguments
	result, err := client.ExecuteTool(ctx, "file_read", map[string]any{
		"path": "/path/to/file.txt",
	})
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}

	if result.Success {
		fmt.Println("Tool executed successfully:", result.Content)
	} else {
		fmt.Println("Tool execution failed:", result.Error)
	}
}

// ExampleClient_ExecuteToolWithRetry demonstrates executing a tool with retry.
func ExampleClient_ExecuteToolWithRetry() {
	ctx := context.Background()
	client := mcp.NewClient("example-client")

	// Execute a tool with retry logic
	result, err := client.ExecuteToolWithRetry(ctx, "http_get", map[string]any{
		"url": "https://api.example.com/data",
	}, 3) // max 3 retries
	if err != nil {
		fmt.Println("Execution error:", err)
		return
	}

	fmt.Println("Result:", result.Success)
}

// ExampleClient_BatchExecute demonstrates batch tool execution.
func ExampleClient_BatchExecute() {
	ctx := context.Background()
	client := mcp.NewClient("example-client")

	// Prepare batch requests
	requests := []mcp.BatchRequest{
		{Name: "file_read", Args: map[string]any{"path": "/file1.txt"}},
		{Name: "file_read", Args: map[string]any{"path": "/file2.txt"}},
		{Name: "file_write", Args: map[string]any{"path": "/output.txt", "content": "data"}},
	}

	// Execute sequentially
	results := client.BatchExecute(ctx, requests)
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Tool %s failed: %v\n", result.Name, result.Error)
		} else {
			fmt.Printf("Tool %s succeeded\n", result.Name)
		}
	}
}

// ExampleClient_BatchExecuteParallel demonstrates parallel batch tool execution.
func ExampleClient_BatchExecuteParallel() {
	ctx := context.Background()
	client := mcp.NewClient("example-client")

	// Prepare batch requests
	requests := []mcp.BatchRequest{
		{Name: "http_get", Args: map[string]any{"url": "https://api1.example.com"}},
		{Name: "http_get", Args: map[string]any{"url": "https://api2.example.com"}},
		{Name: "http_get", Args: map[string]any{"url": "https://api3.example.com"}},
	}

	// Execute in parallel
	results := client.BatchExecuteParallel(ctx, requests)
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Tool %s failed: %v\n", result.Name, result.Error)
		} else {
			fmt.Printf("Tool %s succeeded\n", result.Name)
		}
	}
}

// ExampleClient_Reconnect demonstrates reconnecting to an MCP server.
func ExampleClient_Reconnect() {
	ctx := context.Background()
	client := mcp.NewClient("example-client")

	// ... initial connection and usage ...

	// Reconnect when connection is lost
	err := client.Reconnect(ctx)
	if err != nil {
		fmt.Println("Reconnection error:", err)
		return
	}

	fmt.Println("Reconnected successfully")
}

// ExampleClient_GetState demonstrates checking connection state.
func ExampleClient_GetState() {
	client := mcp.NewClient("example-client")

	// Check connection state
	state := client.GetState()
	fmt.Println("Connection state:", state)

	// Check if connected
	isConnected := client.IsConnected()
	fmt.Println("Is connected:", isConnected)

	// Get last error
	err, timestamp := client.GetLastError()
	if err != nil {
		fmt.Printf("Last error: %v at %v\n", err, timestamp)
	}
}

// ExampleNewManager demonstrates creating an MCP manager.
func ExampleNewManager() {
	// Create a tool registry
	registry := tools.NewRegistry()

	// Create a manager
	manager := mcp.NewManager(registry)
	fmt.Println("Created manager:", manager != nil)

	// Create a manager with options
	managerWithOptions := mcp.NewManager(registry,
		mcp.WithManagerStateChangeHandler(func(name string, state mcp.ConnectionState) {
			fmt.Printf("Client %s state changed to %s\n", name, state)
		}),
	)
	fmt.Println("Created manager with options:", managerWithOptions != nil)
}

// ExampleManager_AddClient demonstrates adding clients to a manager.
func ExampleManager_AddClient() {
	registry := tools.NewRegistry()
	manager := mcp.NewManager(registry)

	// Create and add a client
	client := mcp.NewClient("client1")
	manager.AddClient("client1", client)

	// Or use CreateAndAddClient
	client2 := manager.CreateAndAddClient("client2",
		mcp.WithRetryConfig(5, time.Second),
	)

	fmt.Println("Added clients:", len(manager.ListClients()))
	fmt.Println("Client2 created:", client2 != nil)
}

// ExampleManager_GetConnectionStatus demonstrates getting connection status.
func ExampleManager_GetConnectionStatus() {
	registry := tools.NewRegistry()
	manager := mcp.NewManager(registry)

	manager.AddClient("client1", mcp.NewClient("client1"))
	manager.AddClient("client2", mcp.NewClient("client2"))

	// Get status of all clients
	status := manager.GetConnectionStatus()
	for name, state := range status {
		fmt.Printf("Client %s: %s\n", name, state)
	}
}

// ExampleManager_ReconnectAll demonstrates reconnecting all clients.
func ExampleManager_ReconnectAll() {
	registry := tools.NewRegistry()
	manager := mcp.NewManager(registry)

	manager.AddClient("client1", mcp.NewClient("client1"))
	manager.AddClient("client2", mcp.NewClient("client2"))

	ctx := context.Background()
	err := manager.ReconnectAll(ctx)
	if err != nil {
		fmt.Println("Reconnection error:", err)
		return
	}

	fmt.Println("All clients reconnected")
}

// ExampleMCPTool demonstrates using MCPTool wrapper.
func ExampleMCPTool() {
	client := mcp.NewClient("example-client")

	// Create an MCP tool wrapper from an MCP tool
	mcpToolDef := mcpgo.Tool{
		Name:        "test_tool",
		Description: "A test tool",
	}
	tool := mcp.NewMCPTool(mcpToolDef, client)

	// Get tool information
	fmt.Println("Tool name:", tool.Name())
	fmt.Println("Tool description:", tool.Description())
}

// ExampleConnectionState demonstrates connection states.
func ExampleConnectionState() {
	states := []mcp.ConnectionState{
		mcp.ConnectionStateDisconnected,
		mcp.ConnectionStateConnecting,
		mcp.ConnectionStateConnected,
		mcp.ConnectionStateError,
	}

	for _, state := range states {
		fmt.Printf("State %d: %s\n", state, state.String())
	}
}

// ExampleClientConfig demonstrates client configuration.
func ExampleClientConfig() {
	config := mcp.ClientConfig{
		Command:    "npx",
		Args:       []string{"-y", "@modelcontextprotocol/server"},
		Env:        map[string]string{"API_KEY": "secret"},
		URL:        "http://localhost:8080/sse",
		RetryCount: 3,
		RetryDelay: time.Second,
		Timeout:    30 * time.Second,
	}

	fmt.Println("Command:", config.Command)
	fmt.Println("Args:", len(config.Args))
	fmt.Println("Retry count:", config.RetryCount)
	fmt.Println("Timeout:", config.Timeout)
}
