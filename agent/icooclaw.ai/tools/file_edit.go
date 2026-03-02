package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileEditTool 文件编辑工具
type FileEditTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFileEditTool 创建文件编辑工具
func NewFileEditTool(config *FileToolConfig) *FileEditTool {
	tool := NewBaseTool(
		"file_edit",
		"编辑文件内容（替换指定文本）。支持精确的文本替换和多行匹配。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"old_string": map[string]interface{}{
					"type":        "string",
					"description": "要替换的原始文本",
				},
				"new_string": map[string]interface{}{
					"type":        "string",
					"description": "替换后的新文本",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
		nil,
	)

	return &FileEditTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FileEditTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FileEditTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FileEditTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 编辑文件
func (t *FileEditTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedEdit {
		return "", fmt.Errorf("file edit is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	oldString, ok := params["old_string"].(string)
	if !ok {
		oldString = ""
	}

	newString, ok := params["new_string"].(string)
	if !ok {
		newString = ""
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 读取文件内容
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 检查oldString是否存在
	oldContent := string(content)
	if !strings.Contains(oldContent, oldString) {
		return "", fmt.Errorf("old_string not found in file")
	}

	// 替换文本
	newContent := strings.Replace(oldContent, oldString, newString, 1) // 只替换第一个匹配

	// 写回文件
	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"path":         absPath,
		"old_size":     len(oldContent),
		"new_size":     len(newContent),
		"bytes_diff":   len(newContent) - len(oldContent),
		"replacements": 1,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// resolvePath 解析路径
func (t *FileEditTool) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FileEditTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(path, workspace)
}

// ToDefinition 转换为工具定义
func (t *FileEditTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
