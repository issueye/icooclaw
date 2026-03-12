package file

import (
	"context"
	"fmt"
	"icooclaw/pkg/tools"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyFileTool 提供文件复制功能。
type CopyFileTool struct {
	WorkDir string
}

// NewCopyFileTool 创建一个新的文件复制工具。
func NewCopyFileTool(workDir string) *CopyFileTool {
	if workDir == "" {
		workDir = "./workspace"
	}
	os.MkdirAll(workDir, 0755)
	return &CopyFileTool{WorkDir: workDir}
}

// Name 返回工具名称。
func (t *CopyFileTool) Name() string {
	return "copy_file"
}

// Description 返回工具描述。
func (t *CopyFileTool) Description() string {
	return "复制文件到指定位置。"
}

// Parameters 返回工具参数。
func (t *CopyFileTool) Parameters() map[string]any {
	return map[string]any{
		"source": map[string]any{
			"type":        "string",
			"description": "源文件路径",
			"required":    true,
		},
		"destination": map[string]any{
			"type":        "string",
			"description": "目标文件路径",
			"required":    true,
		},
	}
}

// Execute 执行文件复制。
func (t *CopyFileTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	source, ok := args["source"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 source 参数")}
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 destination 参数")}
	}

	// 安全检查
	absWorkDir, _ := filepath.Abs(t.WorkDir)

	srcPath := filepath.Join(t.WorkDir, filepath.Clean(source))
	absSrcPath, _ := filepath.Abs(srcPath)
	if !strings.HasPrefix(absSrcPath, absWorkDir) {
		return &tools.Result{Success: false, Error: fmt.Errorf("源路径超出工作目录范围")}
	}

	dstPath := filepath.Join(t.WorkDir, filepath.Clean(destination))
	absDstPath, _ := filepath.Abs(dstPath)
	if !strings.HasPrefix(absDstPath, absWorkDir) {
		return &tools.Result{Success: false, Error: fmt.Errorf("目标路径超出工作目录范围")}
	}

	// 打开源文件
	srcFile, err := os.Open(absSrcPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("打开源文件失败: %w", err)}
	}
	defer srcFile.Close()

	// 确保目标目录存在
	os.MkdirAll(filepath.Dir(absDstPath), 0755)

	// 创建目标文件
	dstFile, err := os.Create(absDstPath)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("创建目标文件失败: %w", err)}
	}
	defer dstFile.Close()

	// 复制内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("复制文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("文件复制成功: %s -> %s (%d 字节)", source, destination, written),
	}
}
