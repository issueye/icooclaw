package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellExecTool tests for ShellExecTool
func TestShellExecTool_NewShellExecTool(t *testing.T) {
	config := &ShellExecConfig{
		Allowed:   true,
		Timeout:   30,
		Workspace: t.TempDir(),
	}

	tool := NewShellExecTool(config)
	require.NotNil(t, tool)
	assert.Equal(t, "exec", tool.Name())
}

func TestShellExecTool_Execute_Allowed(t *testing.T) {
	config := &ShellExecConfig{
		Allowed:   true,
		Timeout:   30,
		Workspace: t.TempDir(),
	}

	tool := NewShellExecTool(config)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo hello",
	})

	require.NoError(t, err)
	assert.Contains(t, result, "hello")
}

func TestShellExecTool_Execute_NotAllowed(t *testing.T) {
	config := &ShellExecConfig{
		Allowed:   false,
		Timeout:   30,
		Workspace: t.TempDir(),
	}

	tool := NewShellExecTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo hello",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestShellExecTool_Execute_InvalidCommand(t *testing.T) {
	config := &ShellExecConfig{
		Allowed:   true,
		Timeout:   30,
		Workspace: t.TempDir(),
	}

	tool := NewShellExecTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "",
	})

	assert.Error(t, err)
}
