package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileWriteTool tests for FileWriteTool
func TestFileWriteTool_NewFileWriteTool(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: true,
		Workspace:    tmpDir,
	}

	tool := NewFileWriteTool(config)
	require.NotNil(t, tool)
	assert.Equal(t, "file_write", tool.Name())
}

func TestFileWriteTool_Execute_WriteNotAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: false,
		Workspace:    tmpDir,
	}

	tool := NewFileWriteTool(config)
	_, err := tool.Execute(nil, map[string]interface{}{
		"path":    "test.txt",
		"content": "Hello",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}
