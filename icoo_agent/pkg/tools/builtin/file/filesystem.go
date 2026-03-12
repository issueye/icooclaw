package file

import (
	"context"
	"encoding/json"
	"fmt"
	"icooclaw/pkg/tools"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FilesystemTool 提供文件系统操作功能。
type FilesystemTool struct {
	// WorkDir 工作目录，所有文件操作都限制在此目录内
	WorkDir string
}

// NewFilesystemTool 创建一个新的文件系统工具。
func NewFilesystemTool(workDir string) *FilesystemTool {
	if workDir == "" {
		// 默认使用当前目录下的 workspace 文件夹
		workDir = "./workspace"
	}
	// 确保工作目录存在
	os.MkdirAll(workDir, 0755)
	return &FilesystemTool{WorkDir: workDir}
}

// Name 返回工具名称。
func (t *FilesystemTool) Name() string {
	return "filesystem"
}

// Description 返回工具描述。
func (t *FilesystemTool) Description() string {
	return "文件系统操作工具，支持读取、写入、列出目录、创建目录、删除文件等操作。"
}

// Parameters 返回工具参数定义。
func (t *FilesystemTool) Parameters() map[string]any {
	return map[string]any{
		"operation": map[string]any{
			"type":        "string",
			"description": "操作类型: read(读取文件), write(写入文件), list(列出目录), mkdir(创建目录), delete(删除文件/目录), exists(检查是否存在), info(获取文件信息)",
			"enum":        []string{"read", "write", "list", "mkdir", "delete", "exists", "info"},
		},
		"path": map[string]any{
			"type":        "string",
			"description": "文件或目录路径（相对于工作目录）",
		},
		"content": map[string]any{
			"type":        "string",
			"description": "要写入的内容（仅用于 write 操作）",
		},
		"recursive": map[string]any{
			"type":        "boolean",
			"description": "是否递归操作（用于 list 和 delete）",
		},
	}
}

// Execute 执行文件系统操作。
func (t *FilesystemTool) Execute(ctx context.Context, args map[string]any) *tools.Result {
	operation, _ := args["operation"].(string)
	if operation == "" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 operation 参数")}
	}

	path, _ := args["path"].(string)
	if path == "" && operation != "list" {
		return &tools.Result{Success: false, Error: fmt.Errorf("需要提供 path 参数")}
	}

	// 安全检查：确保路径在工作目录内
	fullPath, err := t.safePath(path)
	if err != nil {
		return &tools.Result{Success: false, Error: err}
	}

	switch operation {
	case "read":
		return t.readFile(fullPath)
	case "write":
		content, _ := args["content"].(string)
		return t.writeFile(fullPath, content)
	case "list":
		return t.listDir(fullPath, args)
	case "mkdir":
		return t.mkdir(fullPath)
	case "delete":
		return t.delete(fullPath, args)
	case "exists":
		return t.exists(fullPath)
	case "info":
		return t.info(fullPath)
	default:
		return &tools.Result{Success: false, Error: fmt.Errorf("不支持的操作类型: %s", operation)}
	}
}

// safePath 确保路径在工作目录内，防止路径遍历攻击。
func (t *FilesystemTool) safePath(path string) (string, error) {
	if path == "" {
		return t.WorkDir, nil
	}

	// 清理路径
	cleanPath := filepath.Clean(path)

	// 构建完整路径
	fullPath := filepath.Join(t.WorkDir, cleanPath)

	// 获取绝对路径进行比较
	absWorkDir, err := filepath.Abs(t.WorkDir)
	if err != nil {
		return "", fmt.Errorf("获取工作目录绝对路径失败: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("获取文件绝对路径失败: %w", err)
	}

	// 检查路径是否在工作目录内
	if !strings.HasPrefix(absFullPath, absWorkDir) {
		return "", fmt.Errorf("路径超出工作目录范围: %s", path)
	}

	return absFullPath, nil
}

// readFile 读取文件内容。
func (t *FilesystemTool) readFile(path string) *tools.Result {
	content, err := os.ReadFile(path)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: string(content),
	}
}

// writeFile 写入文件内容。
func (t *FilesystemTool) writeFile(path, content string) *tools.Result {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("创建目录失败: %w", err)}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("写入文件失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("文件写入成功: %s (%d 字节)", filepath.Base(path), len(content)),
	}
}

// listDir 列出目录内容。
func (t *FilesystemTool) listDir(path string, args map[string]any) *tools.Result {
	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("读取目录失败: %w", err)}
	}

	type FileInfo struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"is_dir"`
		Size    int64  `json:"size,omitempty"`
		ModTime string `json:"mod_time,omitempty"`
	}

	var files []FileInfo

	for _, entry := range entries {
		info := FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
		}

		if !entry.IsDir() {
			if fi, err := entry.Info(); err == nil {
				info.Size = fi.Size()
				info.ModTime = fi.ModTime().Format(time.RFC3339)
			}
		}

		files = append(files, info)

		// 递归列出子目录
		if recursive && entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			subResult := t.listDir(subPath, args)
			if subResult.Success {
				// 解析子目录结果并添加前缀
				var subFiles []FileInfo
				if err := json.Unmarshal([]byte(subResult.Content), &subFiles); err == nil {
					for _, sf := range subFiles {
						sf.Name = entry.Name() + "/" + sf.Name
						files = append(files, sf)
					}
				}
			}
		}
	}

	resultJSON, _ := json.MarshalIndent(files, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}

// mkdir 创建目录。
func (t *FilesystemTool) mkdir(path string) *tools.Result {
	if err := os.MkdirAll(path, 0755); err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("创建目录失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("目录创建成功: %s", filepath.Base(path)),
	}
}

// delete 删除文件或目录。
func (t *FilesystemTool) delete(path string, args map[string]any) *tools.Result {
	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	info, err := os.Stat(path)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("文件或目录不存在: %w", err)}
	}

	if info.IsDir() {
		if recursive {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("删除失败: %w", err)}
	}

	return &tools.Result{
		Success: true,
		Content: fmt.Sprintf("删除成功: %s", filepath.Base(path)),
	}
}

// exists 检查文件或目录是否存在。
func (t *FilesystemTool) exists(path string) *tools.Result {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result := map[string]any{
				"exists": false,
				"path":   path,
			}
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			return &tools.Result{Success: true, Content: string(resultJSON)}
		}
		return &tools.Result{Success: false, Error: fmt.Errorf("检查文件失败: %w", err)}
	}

	result := map[string]any{
		"exists": true,
		"path":   path,
		"is_dir": info.IsDir(),
		"size":   info.Size(),
	}
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}

// info 获取文件或目录详细信息。
func (t *FilesystemTool) info(path string) *tools.Result {
	info, err := os.Stat(path)
	if err != nil {
		return &tools.Result{Success: false, Error: fmt.Errorf("获取文件信息失败: %w", err)}
	}

	result := map[string]any{
		"name":     info.Name(),
		"path":     path,
		"is_dir":   info.IsDir(),
		"size":     info.Size(),
		"mode":     info.Mode().String(),
		"mod_time": info.ModTime().Format(time.RFC3339),
	}

	// 如果是文件，尝试检测内容类型
	if !info.IsDir() {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			buf := make([]byte, 512)
			n, _ := file.Read(buf)
			if n > 0 {
				// 简单的内容类型检测
				content := string(buf[:n])
				if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
					result["type"] = "json"
				} else if strings.HasPrefix(content, "<") {
					result["type"] = "html/xml"
				} else {
					result["type"] = "text"
				}
			}
		}
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &tools.Result{Success: true, Content: string(resultJSON)}
}
