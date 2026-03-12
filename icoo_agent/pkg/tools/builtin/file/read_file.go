package file

import (
	"context"
	"fmt"
	"icooclaw/pkg/tools"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileTool 提供简单的文件读取功能。
type ReadFileTool struct {
	WorkDir string
}

// NewReadFileTool 创建一个新的文件读取工具。
func NewReadFileTool(workDir string) *ReadFileTool {
	if workDir == "" {
		workDir = "./workspace"
	}
	os.MkdirAll(workDir, 0755)
	return &ReadFileTool{WorkDir: workDir}
}

// Name 返回工具名称。
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description 返回工具描述。
func (t *ReadFileTool) Description() string {
	return "读取指定文件的内容。"
}

// Parameters 返回工具参数。
func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "要读取的文件路径",
			"required":    true,
		},
	}
}

// Execute 执行文件读取。
func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, ok := args["path"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 path 参数")}
	}

	// 安全检查
	fullPath := filepath.Join(t.WorkDir, filepath.Clean(path))
	absWorkDir, _ := filepath.Abs(t.WorkDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFullPath, absWorkDir) {
		return &tools.Result{Success: false, Error: fmt.Errorf("路径超出工作目录范围")}
	}

	content, err := os.ReadFile(absFullPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取文件失败: %w", err)}
	}

	return &tools.Result{Success: true, Content: string(content)}
}
