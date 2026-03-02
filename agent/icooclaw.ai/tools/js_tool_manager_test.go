package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateJSTool_Execute(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	config := &CreateJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        mu,
	}

	tool := NewCreateJSTool(config)

	// Test creating a new tool
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"name":        "test_calculator",
		"description": "A simple calculator",
		"code":        "function execute(params) { return JSON.stringify({result: params.a + params.b}); }",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number"},
				"b": map[string]interface{}{"type": "number"},
			},
		},
	})

	require.NoError(t, err)
	assert.Contains(t, result, "success")
	assert.Contains(t, result, "test_calculator")

	// Verify file was created
	scriptPath := filepath.Join(tmpDir, "tools", "test_calculator.js")
	_, err = os.Stat(scriptPath)
	require.NoError(t, err)

	// Verify tool was registered
	_, err = registry.Get("test_calculator")
	require.NoError(t, err)
}

func TestCreateJSTool_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	config := &CreateJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        mu,
	}

	tool := NewCreateJSTool(config)

	// Test with reserved name
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"name":        "file_read",
		"description": "Test",
		"code":        "function execute(params) { return 'test'; }",
	})
	assert.Error(t, err)
	// 错误消息可能是中文 "保留名称" 或英文 "reserved"
	assert.True(t, strings.Contains(err.Error(), "reserved") || strings.Contains(err.Error(), "保留"),
		"expected error to contain 'reserved' or '保留', got: %s", err.Error())

	// Test with invalid characters
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"name":        "test-tool!",
		"description": "Test",
		"code":        "function execute(params) { return 'test'; }",
	})
	assert.Error(t, err)
	// 错误消息可能是中文 "只能包含字母" 或英文 "can only contain"
	assert.True(t, strings.Contains(err.Error(), "can only contain") || strings.Contains(err.Error(), "只能包含"),
		"expected error to contain 'can only contain' or '只能包含', got: %s", err.Error())
}

func TestCreateJSTool_UnsafeCode(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	config := &CreateJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        mu,
	}

	tool := NewCreateJSTool(config)

	// Test with unsafe code patterns
	unsafeCodes := []string{
		"require('fs')",
		"eval('code')",
		"process.exit()",
		"child_process.spawn()",
	}

	for _, code := range unsafeCodes {
		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"name":        "unsafe_tool",
			"description": "Test",
			"code":        "function execute(params) { " + code + " }",
		})
		assert.Error(t, err)
	}
}

func TestCreateJSTool_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	config := &CreateJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        mu,
	}

	tool := NewCreateJSTool(config)

	// Create first tool
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"name":        "overwrite_test",
		"description": "First version",
		"code":        "function execute(params) { return 'v1'; }",
	})
	require.NoError(t, err)

	// Try to create again without overwrite
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"name":        "overwrite_test",
		"description": "Second version",
		"code":        "function execute(params) { return 'v2'; }",
	})
	assert.Error(t, err)
	// 错误消息可能是中文 "已存在" 或英文 "already exists"
	assert.True(t, strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "已存在"),
		"expected error to contain 'already exists' or '已存在', got: %s", err.Error())

	// Create with overwrite
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"name":        "overwrite_test",
		"description": "Second version",
		"code":        "function execute(params) { return 'v2'; }",
		"overwrite":   true,
	})
	require.NoError(t, err)
}

func TestListJSTools_Execute(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some tool files
	toolsDir := filepath.Join(tmpDir, "tools")
	require.NoError(t, os.MkdirAll(toolsDir, 0755))

	toolContent := `
var tool = { name: "test_tool", description: "Test" };
function execute(params) { return "test"; }
`
	require.NoError(t, os.WriteFile(filepath.Join(toolsDir, "test_tool.js"), []byte(toolContent), 0644))

	config := &ListJSToolsConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
	}

	tool := NewListJSTools(config)
	result, err := tool.Execute(context.Background(), map[string]interface{}{})

	require.NoError(t, err)
	assert.Contains(t, result, "builtin_tools")
	assert.Contains(t, result, "dynamic_tools")
	assert.Contains(t, result, "test_tool")
}

func TestDeleteJSTool_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	// First create a tool
	createConfig := &CreateJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        mu,
	}
	createTool := NewCreateJSTool(createConfig)

	_, err := createTool.Execute(context.Background(), map[string]interface{}{
		"name":        "tool_to_delete",
		"description": "Test tool",
		"code":        "function execute(params) { return 'test'; }",
	})
	require.NoError(t, err)

	// Verify file exists
	scriptPath := filepath.Join(tmpDir, "tools", "tool_to_delete.js")
	_, err = os.Stat(scriptPath)
	require.NoError(t, err)

	// Delete the tool
	deleteConfig := &DeleteJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        &mu,
	}
	deleteTool := NewDeleteJSTool(deleteConfig)

	result, err := deleteTool.Execute(context.Background(), map[string]interface{}{
		"name": "tool_to_delete",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "success")

	// Verify file was deleted
	_, err = os.Stat(scriptPath)
	assert.True(t, os.IsNotExist(err))
}

func TestDeleteJSTool_CannotDeleteBuiltin(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()
	var mu sync.RWMutex

	config := &DeleteJSToolConfig{
		Workspace: tmpDir,
		ToolsDir:  "tools",
		Registry:  registry,
		mu:        &mu,
	}

	tool := NewDeleteJSTool(config)

	// Try to delete builtin tool
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"name": "file_read",
	})
	assert.Error(t, err)
	// 错误消息可能是中文 "不能删除内置工具" 或英文 "cannot delete built-in tool"
	assert.True(t, strings.Contains(err.Error(), "cannot delete built-in tool") || strings.Contains(err.Error(), "不能删除内置工具") || strings.Contains(err.Error(), "内置工具"),
		"expected error to contain 'cannot delete built-in tool' or '内置工具', got: %s", err.Error())
}

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "my_tool", false},
		{"valid with numbers", "tool123", false},
		{"valid camelCase", "myTool", false},
		{"too short", "a", true},
		{"starts with number", "123tool", true},
		{"has special char", "my-tool", true},
		{"has space", "my tool", true},
		{"reserved name", "file_read", true},
		{"reserved create_tool", "create_tool", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateJSCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid code", "function execute(params) { return 'ok'; }", false},
		{"missing execute", "function run(params) { return 'ok'; }", true},
		{"has require", "function execute(params) { require('fs'); }", true},
		{"has eval", "function execute(params) { eval('code'); }", true},
		{"has process", "function execute(params) { process.exit(); }", true},
		{"has os.Exit", "function execute(params) { os.Exit(1); }", true},
		{"has import", "function execute(params) { import fs from 'fs'; }", true},
		{"has async", "async function execute(params) { return 'ok'; }", true},
		{"has await", "function execute(params) { await something(); }", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSCode(tt.code)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateToolScript(t *testing.T) {
	script := generateToolScript(
		"test_tool",
		"A test tool",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type": "string",
				},
			},
		},
		"function execute(params) { return params.input; }",
		toolPermissions{FileRead: true, Network: true},
	)

	assert.Contains(t, script, "test_tool")
	assert.Contains(t, script, "A test tool")
	assert.Contains(t, script, "function execute")
	assert.Contains(t, script, "var tool =")
	assert.Contains(t, script, "fileRead: true")
	assert.Contains(t, script, "network: true")
}
