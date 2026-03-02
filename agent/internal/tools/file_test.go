package tools

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveToolPath(t *testing.T) {
	workspace := filepath.Join("home", "user", "workspace")

	tests := []struct {
		name      string
		path      string
		workspace string
		want      string
	}{
		{
			name:      "relative path",
			path:      "test.txt",
			workspace: workspace,
			want:      filepath.Join(workspace, "test.txt"),
		},
		{
			name:      "workspace prefix with slash",
			path:      "/workspace/test.txt",
			workspace: workspace,
			want:      filepath.Join(workspace, "test.txt"),
		},
		{
			name:      "workspace prefix without slash",
			path:      "workspace/test.txt",
			workspace: workspace,
			want:      filepath.Join(workspace, "test.txt"),
		},
		{
			name:      "empty path returns workspace",
			path:      "",
			workspace: workspace,
			want:      workspace,
		},
		{
			name:      "workspace root",
			path:      "/workspace",
			workspace: workspace,
			want:      workspace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveToolPath(tt.path, tt.workspace)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveToolPath_AbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping absolute path test on Windows")
	}

	workspace := "/home/user/workspace"
	tests := []struct {
		name      string
		path      string
		workspace string
		want      string
	}{
		{
			name:      "absolute path outside workspace",
			path:      "/other/path/test.txt",
			workspace: workspace,
			want:      "/other/path/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveToolPath(tt.path, tt.workspace)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPathInWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	nestedFile := filepath.Join(subDir, "nested.txt")
	require.NoError(t, os.WriteFile(nestedFile, []byte("nested"), 0644))

	tests := []struct {
		name      string
		path      string
		workspace string
		want      bool
	}{
		{
			name:      "path in workspace",
			path:      testFile,
			workspace: tmpDir,
			want:      true,
		},
		{
			name:      "workspace root",
			path:      tmpDir,
			workspace: tmpDir,
			want:      true,
		},
		{
			name:      "subdirectory in workspace",
			path:      nestedFile,
			workspace: tmpDir,
			want:      true,
		},
		{
			name:      "subdirectory itself",
			path:      subDir,
			workspace: tmpDir,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPathInWorkspace(tt.path, tt.workspace)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPathInWorkspace_OutsidePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping outside workspace test on Windows due to path format differences")
	}

	tests := []struct {
		name      string
		path      string
		workspace string
		want      bool
	}{
		{
			name:      "path outside workspace",
			path:      "/other/path/test.txt",
			workspace: "/home/user/workspace",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPathInWorkspace(tt.path, tt.workspace)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileReadTool_WorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: true,
		Workspace:    tmpDir,
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Hello World"), 0644)
	require.NoError(t, err)

	tool := NewFileReadTool(config)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "relative path",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "workspace prefix",
			path:    "/workspace/test.txt",
			wantErr: false,
		},
		{
			name:    "absolute path in workspace",
			path:    testFile,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), map[string]interface{}{
				"path": tt.path,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileListTool_WorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: true,
		Workspace:    tmpDir,
	}

	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	tool := NewFileListTool(config)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "workspace prefix",
			path:    "/workspace",
			wantErr: false,
		},
		{
			name:    "absolute path in workspace",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    ".",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), map[string]interface{}{
				"path": tt.path,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindTool_WorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()
	config := &FileToolConfig{
		AllowedRead:  true,
		AllowedWrite: true,
		Workspace:    tmpDir,
	}

	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)
	require.NoError(t, err)

	tool := NewFindTool(config)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "workspace prefix",
			path:    "/workspace",
			wantErr: false,
		},
		{
			name:    "absolute path in workspace",
			path:    tmpDir,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), map[string]interface{}{
				"path": tt.path,
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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
