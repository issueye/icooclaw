// Package script provides JavaScript scripting engine for icooclaw.
package script

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem provides file system operations.
type FileSystem struct {
	cfg    *Config
	logger *slog.Logger
}

// NewFileSystem creates a new FileSystem builtin.
func NewFileSystem(cfg *Config, logger *slog.Logger) *FileSystem {
	if logger == nil {
		logger = slog.Default()
	}
	return &FileSystem{cfg: cfg, logger: logger}
}

// Name returns the builtin name.
func (fs *FileSystem) Name() string {
	return "fs"
}

// Object returns the fs object.
func (fs *FileSystem) Object() map[string]any {
	return map[string]any{
		"readFile":     fs.ReadFile,
		"writeFile":    fs.WriteFile,
		"appendFile":   fs.AppendFile,
		"exists":       fs.Exists,
		"stat":         fs.Stat,
		"mkdir":        fs.Mkdir,
		"remove":       fs.Remove,
		"readdir":      fs.Readdir,
		"copy":         fs.Copy,
		"move":         fs.Move,
		"tempDir":      fs.TempDir,
		"join":         fs.Join,
		"basename":     fs.Basename,
		"dirname":      fs.Dirname,
		"extname":      fs.Extname,
		"isAbs":        fs.IsAbs,
	}
}

// ReadFile reads a file.
func (fs *FileSystem) ReadFile(path string) (string, error) {
	if !fs.cfg.AllowFileRead {
		return "", fmt.Errorf("file reading is not allowed")
	}

	absPath := fs.resolvePath(path)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes a file.
func (fs *FileSystem) WriteFile(path, content string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file writing is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.WriteFile(absPath, []byte(content), 0644)
}

// AppendFile appends to a file.
func (fs *FileSystem) AppendFile(path, content string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file writing is not allowed")
	}

	absPath := fs.resolvePath(path)
	f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// Exists checks if a file exists.
func (fs *FileSystem) Exists(path string) bool {
	absPath := fs.resolvePath(path)
	_, err := os.Stat(absPath)
	return err == nil
}

// Stat returns file information.
func (fs *FileSystem) Stat(path string) (map[string]any, error) {
	absPath := fs.resolvePath(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"name":    info.Name(),
		"size":    info.Size(),
		"isDir":   info.IsDir(),
		"modTime": info.ModTime().Unix(),
		"mode":    info.Mode().String(),
	}, nil
}

// Mkdir creates a directory.
func (fs *FileSystem) Mkdir(path string) error {
	if !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file writing is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.MkdirAll(absPath, 0755)
}

// Remove removes a file or directory.
func (fs *FileSystem) Remove(path string) error {
	if !fs.cfg.AllowFileDelete {
		return fmt.Errorf("file deletion is not allowed")
	}

	absPath := fs.resolvePath(path)
	return os.RemoveAll(absPath)
}

// Readdir reads directory contents.
func (fs *FileSystem) Readdir(path string) ([]map[string]any, error) {
	if !fs.cfg.AllowFileRead {
		return nil, fmt.Errorf("file reading is not allowed")
	}

	absPath := fs.resolvePath(path)
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		result = append(result, map[string]any{
			"name":  entry.Name(),
			"isDir": entry.IsDir(),
			"size":  info.Size(),
		})
	}
	return result, nil
}

// Copy copies a file.
func (fs *FileSystem) Copy(src, dst string) error {
	if !fs.cfg.AllowFileRead || !fs.cfg.AllowFileWrite {
		return fmt.Errorf("file operations not allowed")
	}

	srcPath := fs.resolvePath(src)
	dstPath := fs.resolvePath(dst)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Move moves a file.
func (fs *FileSystem) Move(src, dst string) error {
	if !fs.cfg.AllowFileWrite || !fs.cfg.AllowFileDelete {
		return fmt.Errorf("file operations not allowed")
	}

	srcPath := fs.resolvePath(src)
	dstPath := fs.resolvePath(dst)
	return os.Rename(srcPath, dstPath)
}

// TempDir returns the system temp directory.
func (fs *FileSystem) TempDir() string {
	return os.TempDir()
}

// Join joins path components.
func (fs *FileSystem) Join(paths ...string) string {
	return filepath.Join(paths...)
}

// Basename returns the base name of a path.
func (fs *FileSystem) Basename(path string) string {
	return filepath.Base(path)
}

// Dirname returns the directory name of a path.
func (fs *FileSystem) Dirname(path string) string {
	return filepath.Dir(path)
}

// Extname returns the extension of a path.
func (fs *FileSystem) Extname(path string) string {
	return filepath.Ext(path)
}

// IsAbs checks if a path is absolute.
func (fs *FileSystem) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// resolvePath resolves a path relative to workspace.
func (fs *FileSystem) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	// Security check: prevent path traversal
	resolved := filepath.Join(fs.cfg.Workspace, path)
	resolved = filepath.Clean(resolved)

	if !strings.HasPrefix(resolved, filepath.Clean(fs.cfg.Workspace)) {
		return "" // Path traversal attempt
	}

	return resolved
}