package file

import (
	"context"
	"fmt"
	"icooclaw/pkg/tools"
	"os"
	"path/filepath"
	"strings"
)

// WriteFileTool 提供简单的文件写入功能。
type WriteFileTool struct {
	WorkDir string
}

// NewWriteFileTool 创建一个新的文件写入工具。
func NewWriteFileTool(workDir string) *WriteFileTool {
	if workDir == "" {
		workDir = "./workspace"
	}
	os.MkdirAll(workDir, 0755)
	return &WriteFileTool{WorkDir: workDir}
}

// Name 返回工具名称。
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description 返回工具描述。
func (t *WriteFileTool) Description() string {
	return "将内容写入指定文件。"
}

// Parameters 返回工具参数。
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"path": map[string]any{
			"type":        "string",
			"description": "要写入的文件路径",
			"required":    true,
		},
		"content": map[string]any{
			"type":        "string",
			"description": "要写入的内容",
			"required":    true,
		},
	}
}

// Execute 执行文件写入。
func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	path, ok := args["path"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 path 参数")}
	}

	content, ok := args["content"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 content 参数")}
	}

	// 安全检查
	fullPath := filepath.Join(t.WorkDir, filepath.Clean(path))
	absWorkDir, _ := filepath.Abs(t.WorkDir)
	absFullPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFullPath, absWorkDir) {
		return &tools.Result{Success: false, Error: fmt.Errorf("路径超出工作目录范围")}
	}

	// 确保目录存在
	os.MkdirAll(filepath.Dir(absFullPath), 0755)

	if err := os.WriteFile(absFullPath, []byte(content), 0644); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("写入文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("文件写入成功: %s (%d 字节)", path, len(content)),
	}
}
