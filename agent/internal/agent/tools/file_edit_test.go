package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileEditTool tests for FileEditTool
func TestFileEditTool_NewFileEditTool(t *testing.T) {
	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   "/tmp/test",
	}

	tool := NewFileEditTool(config)
	require.NotNil(t, tool)
	assert.Equal(t, "file_edit", tool.Name())
	assert.Equal(t, "编辑文件内容（替换指定文本）。支持精确的文本替换和多行匹配。", tool.Description())
}

func TestFileEditTool_Execute_EditAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   tmpDir,
	}

	tool := NewFileEditTool(config)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "test.txt",
		"old_string": "World",
		"new_string": "Go",
	})

	require.NoError(t, err)
	assert.Contains(t, result, `"replacements":1`)
	assert.Contains(t, result, `"bytes_diff":-3`)

	// Verify file content was updated
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Hello, Go!", string(content))
}

func TestFileEditTool_Execute_EditNotAllowed(t *testing.T) {
	config := &FileToolConfig{
		AllowedEdit: false,
		Workspace:   "/tmp/test",
	}

	tool := NewFileEditTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "test.txt",
		"old_string": "old",
		"new_string": "new",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestFileEditTool_Execute_OldStringNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   tmpDir,
	}

	tool := NewFileEditTool(config)
	_, err = tool.Execute(context.Background(), map[string]interface{}{
		"path":       "test.txt",
		"old_string": "NonExistent",
		"new_string": "New",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFileEditTool_Execute_InvalidPath(t *testing.T) {
	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   "/tmp/test",
	}

	tool := NewFileEditTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "",
		"old_string": "old",
		"new_string": "new",
	})

	assert.Error(t, err)
}

func TestFileEditTool_Execute_PathOutsideWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   tmpDir,
	}

	tool := NewFileEditTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "/etc/passwd",
		"old_string": "old",
		"new_string": "new",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside workspace")
}

func TestFileEditTool_Execute_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   tmpDir,
	}

	tool := NewFileEditTool(config)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "nonexistent.txt",
		"old_string": "old",
		"new_string": "new",
	})

	assert.Error(t, err)
}

func TestFileEditTool_Execute_MultipleReplacements(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello World, Hello World!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	config := &FileToolConfig{
		AllowedEdit: true,
		Workspace:   tmpDir,
	}

	tool := NewFileEditTool(config)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":       "test.txt",
		"old_string": "Hello",
		"new_string": "Hi",
	})

	require.NoError(t, err)
	// Should only replace first occurrence
	assert.Contains(t, result, `"replacements":1`)

	// Verify only first was replaced
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Hi World, Hello World!", string(content))
}
