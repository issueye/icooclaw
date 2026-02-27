package builtons

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/icooclaw/icooclaw/internal/script/config"
)

// === fs 文件系统对象 ===

type FileSystem struct {
	cfg    *config.Config
	logger *slog.Logger
}

func NewFileSystem(cfg *config.Config, logger *slog.Logger) *FileSystem {
	return &FileSystem{
		cfg:    cfg,
		logger: logger,
	}
}

func (fs *FileSystem) GetCfg() *config.Config {
	return fs.cfg
}

func (fs *FileSystem) SetCfg(cfg *config.Config) {
	fs.cfg = cfg
}

// ReadFile 读取文件
func (fs *FileSystem) ReadFile(path string) (string, error) {
	if !fs.cfg.AllowFileRead {
		return "", fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}

// ReadFileBytes 以字节数组形式读取文件
func (fs *FileSystem) ReadFileBytes(path string) ([]byte, error) {
	if !fs.cfg.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.ReadFile(absPath)
}

// WriteFile 写入文件
func (fs *FileSystem) WriteFile(path string, content string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)

	// 确保目录存在
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(absPath, []byte(content), 0644)
}

// WriteFileBytes 写入字节数组到文件
func (fs *FileSystem) WriteFileBytes(path string, content []byte) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(absPath, content, 0644)
}

// AppendFile 追加内容到文件
func (fs *FileSystem) AppendFile(path string, content string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// DeleteFile 删除文件
func (fs *FileSystem) DeleteFile(path string) error {
	if !fs.cfg.AllowFileDelete {
		return fmt.Errorf("file delete is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.Remove(absPath)
}

// Exists 检查文件是否存在
func (fs *FileSystem) Exists(path string) bool {
	absPath := fs.resolvePath(path)
	_, err := os.Stat(absPath)
	return err == nil
}

// IsDir 检查是否是目录
func (fs *FileSystem) IsDir(path string) bool {
	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile 检查是否是文件
func (fs *FileSystem) IsFile(path string) bool {
	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ListDir 列出目录内容
func (fs *FileSystem) ListDir(path string) ([]map[string]interface{}, error) {
	if !fs.cfg.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		result = append(result, map[string]interface{}{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"modified": info.ModTime().Format(time.RFC3339),
		})
	}

	return result, nil
}

// Mkdir 创建目录
func (fs *FileSystem) Mkdir(path string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file write is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.MkdirAll(absPath, 0755)
}

// MkdirAll 递归创建目录
func (fs *FileSystem) MkdirAll(path string) error {
	return fs.Mkdir(path)
}

// Rmdir 删除目录
func (fs *FileSystem) Rmdir(path string) error {
	if !fs.cfg.AllowFileDelete {
		return fmt.Errorf("file delete is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.RemoveAll(absPath)
}

// CopyFile 复制文件
func (fs *FileSystem) CopyFile(src, dst string) error {
	if !fs.cfg.AllowFileRead || !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file operation not allowed")
	}

	absSrc := fs.resolvePath(src)
	absDst := fs.resolvePath(dst)

	srcFile, err := os.Open(absSrc)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// 确保目标目录存在
	dir := filepath.Dir(absDst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dstFile, err := os.Create(absDst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MoveFile 移动文件
func (fs *FileSystem) MoveFile(src, dst string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file operation not allowed")
	}

	absSrc := fs.resolvePath(src)
	absDst := fs.resolvePath(dst)

	// 确保目标目录存在
	dir := filepath.Dir(absDst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.Rename(absSrc, absDst)
}

// GetInfo 获取文件信息
func (fs *FileSystem) GetInfo(path string) (map[string]interface{}, error) {
	if !fs.cfg.AllowFileRead {
		return nil, fmt.Errorf("file read is not allowed")
	}

	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return map[string]interface{}{
		"name":     info.Name(),
		"size":     info.Size(),
		"is_dir":   info.IsDir(),
		"modified": info.ModTime().Format(time.RFC3339),
		"mode":     info.Mode().String(),
	}, nil
}

func (fs *FileSystem) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fs.cfg.Workspace, path)
}
