package file

import (
	"context"
	"fmt"
	"icooclaw/pkg/tools"
	"os"
	"path/filepath"
	"strings"
)

// ListDirTool 提供目录列表功能。
type ListDirTool struct {
	WorkDir string
}

// NewListDirTool 创建一个新的目录列表工具。
func NewListDirTool(workDir string) *ListDirTool {
	if workDir == "" {
		workDir = "./workspace"
	}
	os.MkdirAll(workDir, 0755)
	return &ListDirTool{WorkDir: workDir}
}

// Name 返回工具名称。
func (t *ListDirTool) Name() string {
	return "list_directory"
}

// Description 返回工具描述。
func (t *ListDirTool) Description() string {
	return "列出指定目录下的文件和子目录。"
}

// Parameters 返回工具参数。
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "要列出的目录路径（默认为工作目录）",
			"required":    true,
		},
	}
}

// Execute 执行目录列表。
func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, _ := args["path"].(string)

	// 安全检查
	fullPath := filepath.Join(t.WorkDir, filepath.Clean(path))
	absWorkDir, _ := filepath.Abs(t.WorkDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFullPath, absWorkDir) {
		return &tools.Result{Success: false, Error: fmt.Errorf("路径超出工作目录范围")}
	}

	entries, err := os.ReadDir(absFullPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取目录失败: %w", err)}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("目录: %s\n\n", path))

	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
		} else {
			info, _ := entry.Info()
			result.WriteString(fmt.Sprintf("📄 %s (%d 字节)\n", entry.Name(), info.Size()))
		}
	}

	return &tools.Result{Success: true, Content: result.String()}
}
