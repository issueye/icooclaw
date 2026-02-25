package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSTool_Calculator(t *testing.T) {
	script := `
var tool = {
    name: "calculator",
    description: "执行数学计算",
    parameters: {
        type: "object",
        properties: {
            expression: {
                type: "string",
                description: "数学表达式"
            }
        },
        required: ["expression"]
    }
};

function execute(params) {
    var expr = params.expression;
    var result = eval(expr);
    return JSON.stringify({ result: result });
}
`

	def := JSToolDefinition{
		Name:        "calculator",
		Description: "执行数学计算",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "数学表达式",
				},
			},
		},
		Execute: script,
	}

	tool, err := NewJSTool(def, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "calculator", tool.Name())
	assert.Equal(t, "执行数学计算", tool.Description())
	assert.NotNil(t, tool.Parameters())

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"expression": "2 + 3",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "5")
}

func TestJSTool_TextProcessor(t *testing.T) {
	script := `
var tool = {
    name: "text_processor",
    description: "处理文本"
};

function execute(params) {
    var action = params.action;
    var text = params.text;
    
    if (action === "uppercase") {
        return text.toUpperCase();
    } else if (action === "lowercase") {
        return text.toLowerCase();
    }
    return text;
}
`

	def := JSToolDefinition{
		Name:        "text_processor",
		Description: "处理文本",
		Execute:     script,
	}

	tool, err := NewJSTool(def, nil, nil)
	require.NoError(t, err)

	// Test uppercase
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "uppercase",
		"text":   "hello",
	})
	require.NoError(t, err)
	assert.Equal(t, "HELLO", result)

	// Test lowercase
	result, err = tool.Execute(context.Background(), map[string]interface{}{
		"action": "lowercase",
		"text":   "WORLD",
	})
	require.NoError(t, err)
	assert.Equal(t, "world", result)
}

func TestJSToolLoader_LoadFromFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test_tool.js")

	script := `
var tool = {
    name: "test_tool",
    description: "A test tool",
    parameters: {
        type: "object",
        properties: {
            input: {
                type: "string",
                description: "Input text"
            }
        }
    }
};

function execute(params) {
    return "Processed: " + params.input;
}
`
	err := os.WriteFile(scriptPath, []byte(script), 0644)
	require.NoError(t, err)

	loader := NewJSToolLoader(nil, nil)
	tool, err := loader.LoadFromFile(scriptPath)
	require.NoError(t, err)

	assert.Equal(t, "test_tool", tool.Name())
	assert.Equal(t, "A test tool", tool.Description())

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"input": "hello",
	})
	require.NoError(t, err)
	assert.Equal(t, "Processed: hello", result)
}

func TestJSToolLoader_LoadFromDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple JS tool files
	scripts := map[string]string{
		"tool1.js": `
var tool = { name: "tool1", description: "Tool 1" };
function execute(params) { return "tool1: " + params.input; }
`,
		"tool2.js": `
var tool = { name: "tool2", description: "Tool 2" };
function execute(params) { return "tool2: " + params.input; }
`,
		"not_a_tool.txt": `This should be ignored`,
	}

	for name, content := range scripts {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	loader := NewJSToolLoader(nil, nil)
	tools, err := loader.LoadFromDirectory(tmpDir)
	require.NoError(t, err)

	// Should only load .js files
	assert.Len(t, tools, 2)

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}
	assert.True(t, toolNames["tool1"])
	assert.True(t, toolNames["tool2"])
}

func TestJSTool_JSONHandling(t *testing.T) {
	script := `
function execute(params) {
    var data = {
        name: params.name,
        count: params.count,
        nested: {
            key: "value"
        }
    };
    return JSON.stringify(data);
}
`

	def := JSToolDefinition{
		Name:        "json_test",
		Description: "Test JSON handling",
		Execute:     script,
	}

	tool, err := NewJSTool(def, nil, nil)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"name":  "test",
		"count": 42,
	})
	require.NoError(t, err)
	assert.Contains(t, result, `"name":"test"`)
	assert.Contains(t, result, `"count":42`)
}

func TestJSTool_Console(t *testing.T) {
	script := `
function execute(params) {
    console.log("Info message");
    console.debug("Debug message");
    console.warn("Warning message");
    console.error("Error message");
    return "done";
}
`

	def := JSToolDefinition{
		Name:        "console_test",
		Description: "Test console",
		Execute:     script,
	}

	tool, err := NewJSTool(def, nil, nil)
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, "done", result)
}

func TestJSTool_ErrorHandling(t *testing.T) {
	script := `
function execute(params) {
    throw new Error("Test error");
}
`

	def := JSToolDefinition{
		Name:        "error_test",
		Description: "Test error handling",
		Execute:     script,
	}

	tool, err := NewJSTool(def, nil, nil)
	require.NoError(t, err)

	_, err = tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Test error")
}
