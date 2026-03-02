package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icooclaw/icooclaw/pkg/utils"
)

// FileToolConfig 文件工具配置
type FileToolConfig struct {
	AllowedRead   bool
	AllowedWrite  bool
	AllowedEdit   bool
	AllowedDelete bool
	Workspace     string
}

// resolveToolPath 解析工具路径，处理 /workspace 前缀和 ~ 展开
func resolveToolPath(path, workspace string) string {
	path = strings.TrimPrefix(path, "/workspace")
	path = strings.TrimPrefix(path, "workspace")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")

	if filepath.IsAbs(path) {
		absPath, err := utils.ExpandPath(path)
		if err != nil {
			return path
		}
		return absPath
	}

	expandedWorkspace, err := utils.ExpandPath(workspace)
	if err != nil {
		expandedWorkspace = workspace
	}
	return filepath.Join(expandedWorkspace, path)
}

// normalizePath 规范化路径，处理 Windows 上的路径大小写和分隔符问题
func normalizePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ToLower(path)
	return path
}

// isPathInWorkspace 检查路径是否在工作区内
func isPathInWorkspace(path, workspace string) bool {
	workspaceAbs, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		workspaceAbs = workspace
	}
	workspaceAbs = normalizePath(workspaceAbs)

	pathAbs, err := filepath.EvalSymlinks(path)
	if err != nil {
		if path == "" || normalizePath(path) == workspaceAbs || normalizePath(path) == normalizePath(workspace) {
			return true
		}
		return false
	}
	pathAbs = normalizePath(pathAbs)

	return strings.HasPrefix(pathAbs, workspaceAbs) || pathAbs == workspaceAbs
}

// FileReadTool 文件读取工具
type FileReadTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFileReadTool 创建文件读取工具
func NewFileReadTool(config *FileToolConfig) *FileReadTool {
	tool := NewBaseTool(
		"file_read",
		"读取文件内容。支持读取文本文件和 JSON 文件。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &FileReadTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FileReadTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FileReadTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FileReadTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 读取文件
func (t *FileReadTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedRead {
		return "", fmt.Errorf("file read is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 读取文件
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"path":    absPath,
		"size":    len(content),
		"content": string(content),
	}

	// 尝试解析 JSON
	if strings.HasSuffix(strings.ToLower(path), ".json") {
		var jsonData interface{}
		if err := json.Unmarshal(content, &jsonData); err == nil {
			result["json"] = jsonData
		}
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// resolvePath 解析路径
func (t *FileReadTool) resolvePath(path string) string {
	// 处理 /workspace 前缀，将其视为 workspace 根目录
	path = strings.TrimPrefix(path, "/workspace")
	path = strings.TrimPrefix(path, "workspace")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FileReadTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		if path == "" || path == workspace {
			return true
		}
		return false
	}

	return strings.HasPrefix(path, workspace) || path == workspace
}

// ToDefinition 转换为工具定义
func (t *FileReadTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// FileWriteTool 文件写入工具
type FileWriteTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFileWriteTool 创建文件写入工具
func NewFileWriteTool(config *FileToolConfig) *FileWriteTool {
	tool := NewBaseTool(
		"file_write",
		"写入内容到文件。如果文件不存在则创建，如果存在则覆盖。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文件内容",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "是否追加模式（默认 false，覆盖）",
					"default":     false,
				},
			},
			"required": []string{"path", "content"},
		},
		nil,
	)

	return &FileWriteTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FileWriteTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FileWriteTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FileWriteTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 写入文件
func (t *FileWriteTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedWrite {
		return "", fmt.Errorf("file write is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	content, ok := params["content"].(string)
	if !ok {
		content = ""
	}

	appendMode := false
	if a, ok := params["append"].(bool); ok {
		appendMode = a
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 确保目录存在
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 写入文件
	var err error
	if appendMode {
		f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()
		_, err = f.WriteString(content)
	} else {
		err = os.WriteFile(absPath, []byte(content), 0644)
	}

	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"path": absPath,
		"size": len(content),
		"mode": map[bool]string{true: "append", false: "overwrite"}[appendMode],
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// resolvePath 解析路径
func (t *FileWriteTool) resolvePath(path string) string {
	path = strings.TrimPrefix(path, "/workspace")
	path = strings.TrimPrefix(path, "workspace")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")

	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FileWriteTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		if path == "" || path == workspace {
			return true
		}
		return false
	}

	return strings.HasPrefix(path, workspace) || path == workspace
}

// ToDefinition 转换为工具定义
func (t *FileWriteTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// FileListTool 文件列表工具
type FileListTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFileListTool 创建文件列表工具
func NewFileListTool(config *FileToolConfig) *FileListTool {
	tool := NewBaseTool(
		"file_list",
		"列出目录中的文件和子目录。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归列出子目录",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &FileListTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FileListTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FileListTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FileListTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 列出文件
func (t *FileListTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	recursive := false
	if r, ok := params["recursive"].(bool); ok {
		recursive = r
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 列出文件
	var entries []map[string]interface{}
	var walkErr error

	if recursive {
		walkErr = filepath.Walk(absPath, func(walkPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(absPath, walkPath)
			entries = append(entries, map[string]interface{}{
				"name":    info.Name(),
				"path":    relPath,
				"is_dir":  info.IsDir(),
				"size":    info.Size(),
				"modTime": info.ModTime(),
			})
			return nil
		})
	} else {
		infos, err := os.ReadDir(absPath)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}
		for _, info := range infos {
			entries = append(entries, map[string]interface{}{
				"name":   info.Name(),
				"is_dir": info.IsDir(),
				"size":   0, // ReadDir 不返回大小
			})
		}
	}

	if walkErr != nil {
		return "", fmt.Errorf("failed to list files: %w", walkErr)
	}

	// 构建结果
	result := map[string]interface{}{
		"path":    absPath,
		"entries": entries,
		"count":   len(entries),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// resolvePath 解析路径
func (t *FileListTool) resolvePath(path string) string {
	path = strings.TrimPrefix(path, "/workspace")
	path = strings.TrimPrefix(path, "workspace")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")

	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FileListTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		if path == "" || path == workspace {
			return true
		}
		return false
	}

	return strings.HasPrefix(path, workspace) || path == workspace
}

// ToDefinition 转换为工具定义
func (t *FileListTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}

// FileDeleteTool 文件删除工具
type FileDeleteTool struct {
	baseTool *BaseTool
	config   *FileToolConfig
}

// NewFileDeleteTool 创建文件删除工具
func NewFileDeleteTool(config *FileToolConfig) *FileDeleteTool {
	tool := NewBaseTool(
		"file_delete",
		"删除文件或空目录。注意：无法删除非空目录。",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件或目录路径",
				},
			},
			"required": []string{"path"},
		},
		nil,
	)

	return &FileDeleteTool{
		baseTool: tool,
		config:   config,
	}
}

// Name 获取名称
func (t *FileDeleteTool) Name() string {
	return t.baseTool.Name()
}

// Description 获取描述
func (t *FileDeleteTool) Description() string {
	return t.baseTool.Description()
}

// Parameters 获取参数定义
func (t *FileDeleteTool) Parameters() map[string]interface{} {
	return t.baseTool.Parameters()
}

// Execute 删除文件
func (t *FileDeleteTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	if !t.config.AllowedWrite {
		return "", fmt.Errorf("file delete is not allowed")
	}

	path, ok := params["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("invalid or missing path")
	}

	// 解析路径
	absPath := t.resolvePath(path)

	// 检查路径是否在工作区内
	if !t.isInWorkspace(absPath) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	// 获取文件信息
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", path)
		}
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// 删除
	if info.IsDir() {
		err = os.Remove(absPath)
	} else {
		err = os.Remove(absPath)
	}

	if err != nil {
		return "", fmt.Errorf("failed to delete: %w", err)
	}

	// 构建结果
	result := map[string]interface{}{
		"path": absPath,
		"type": map[bool]string{true: "directory", false: "file"}[info.IsDir()],
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// resolvePath 解析路径
func (t *FileDeleteTool) resolvePath(path string) string {
	path = strings.TrimPrefix(path, "/workspace")
	path = strings.TrimPrefix(path, "workspace")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "\\")

	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(t.config.Workspace, path)
}

// isInWorkspace 检查路径是否在工作区内
func (t *FileDeleteTool) isInWorkspace(path string) bool {
	workspace, err := filepath.EvalSymlinks(t.config.Workspace)
	if err != nil {
		workspace = t.config.Workspace
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		if path == "" || path == workspace {
			return true
		}
		return false
	}

	return strings.HasPrefix(path, workspace) || path == workspace
}

// ToDefinition 转换为工具定义
func (t *FileDeleteTool) ToDefinition() ToolDefinition {
	return t.baseTool.ToDefinition()
}
