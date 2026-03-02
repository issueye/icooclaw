package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaseTool tests for BaseTool
func TestBaseTool_Name(t *testing.T) {
	tool := NewBaseTool(
		"test_tool",
		"Test description",
		map[string]interface{}{},
		nil,
	)

	assert.Equal(t, "test_tool", tool.Name())
}

func TestBaseTool_Description(t *testing.T) {
	tool := NewBaseTool(
		"test_tool",
		"Test description",
		map[string]interface{}{},
		nil,
	)

	assert.Equal(t, "Test description", tool.Description())
}

func TestBaseTool_Parameters(t *testing.T) {
	params := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}

	tool := NewBaseTool(
		"test_tool",
		"Test description",
		params,
		nil,
	)

	assert.Equal(t, params, tool.Parameters())
}

func TestBaseTool_ToDefinition(t *testing.T) {
	tool := NewBaseTool(
		"test_tool",
		"Test description",
		map[string]interface{}{
			"type": "object",
		},
		nil,
	)

	def := tool.ToDefinition()
	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "test_tool", def.Function.Name)
	assert.Equal(t, "Test description", def.Function.Description)
}

// TestRegistry tests for Registry
func TestRegistry_NewRegistry(t *testing.T) {
	registry := NewRegistry()
	require.NotNil(t, registry)
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		nil,
	)

	registry.Register(tool)
	assert.Equal(t, 1, registry.Count())
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		nil,
	)

	registry.Register(tool)

	// Test getting existing tool
	retrieved, err := registry.Get("test_tool")
	require.NoError(t, err)
	assert.Equal(t, "test_tool", retrieved.Name())

	// Test getting non-existing tool
	_, err = registry.Get("non_existing")
	assert.Error(t, err)
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	tool1 := NewBaseTool("tool1", "Tool 1", map[string]interface{}{}, nil)
	tool2 := NewBaseTool("tool2", "Tool 2", map[string]interface{}{}, nil)

	registry.Register(tool1)
	registry.Register(tool2)

	tools := registry.List()
	assert.Len(t, tools, 2)
}

func TestRegistry_ToDefinitions(t *testing.T) {
	registry := NewRegistry()

	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		nil,
	)

	registry.Register(tool)

	defs := registry.ToDefinitions()
	assert.Len(t, defs, 1)
	assert.Equal(t, "test_tool", defs[0].Function.Name)
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	executed := false
	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			executed = true
			return "result", nil
		},
	)

	registry.Register(tool)

	// Create a valid ToolCall
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_tool",
			Arguments: `{}`,
		},
	}

	result := registry.Execute(context.Background(), toolCall)
	assert.True(t, executed)
	assert.NoError(t, result.Error)
	assert.Equal(t, "result", result.Content)
}

func TestRegistry_Execute_WithParams(t *testing.T) {
	registry := NewRegistry()

	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			name, ok := params["name"].(string)
			if !ok {
				return "", assert.AnError
			}
			return "Hello, " + name, nil
		},
	)

	registry.Register(tool)

	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_tool",
			Arguments: `{"name":"World"}`,
		},
	}

	result := registry.Execute(context.Background(), toolCall)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Hello, World", result.Content)
}

func TestRegistry_Execute_InvalidTool(t *testing.T) {
	registry := NewRegistry()

	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "non_existing",
			Arguments: `{}`,
		},
	}

	result := registry.Execute(context.Background(), toolCall)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not found")
}

func TestRegistry_Execute_InvalidArgs(t *testing.T) {
	registry := NewRegistry()

	tool := NewBaseTool(
		"test_tool",
		"Test tool",
		map[string]interface{}{},
		func(ctx context.Context, params map[string]interface{}) (string, error) {
			return "result", nil
		},
	)

	registry.Register(tool)

	// Invalid JSON in arguments
	toolCall := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_tool",
			Arguments: `invalid json`,
		},
	}

	result := registry.Execute(context.Background(), toolCall)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "parse")
}

// TestToolDefinition tests for ToolDefinition struct
func TestToolDefinition_Structure(t *testing.T) {
	def := ToolDefinition{
		Type: "function",
		Function: FunctionDefinition{
			Name:        "test",
			Description: "Test function",
			Parameters: map[string]interface{}{
				"type": "object",
			},
		},
	}

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "test", def.Function.Name)
	assert.Equal(t, "Test function", def.Function.Description)
}

// TestToolCall tests for ToolCall struct
func TestToolCall_Structure(t *testing.T) {
	call := ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: ToolCallFunction{
			Name:      "test_func",
			Arguments: `{"key":"value"}`,
		},
	}

	assert.Equal(t, "call_123", call.ID)
	assert.Equal(t, "function", call.Type)
	assert.Equal(t, "test_func", call.Function.Name)
	assert.Equal(t, `{"key":"value"}`, call.Function.Arguments)
}

// TestToolResult tests for ToolResult struct
func TestToolResult_Structure(t *testing.T) {
	result := ToolResult{
		ToolCallID: "call_123",
		Content:    "result content",
		Error:      nil,
	}

	assert.Equal(t, "call_123", result.ToolCallID)
	assert.Equal(t, "result content", result.Content)
	assert.NoError(t, result.Error)
}

func TestToolResult_WithError(t *testing.T) {
	result := ToolResult{
		ToolCallID: "call_123",
		Content:    "",
		Error:      assert.AnError,
	}

	assert.Equal(t, "call_123", result.ToolCallID)
	assert.Error(t, result.Error)
}
