package agent

import (
	"context"
	"os"

	"icooclaw.ai/provider"
	"icooclaw.core/storage"
)

// ============ 核心接口定义 ============

// MemoryStoreInterface 记忆存储接口
// 封装长期记忆的加载和管理
type MemoryStoreInterface interface {
	Load(ctx context.Context) error
	GetAll() ([]storage.Memory, error)
	Add(memType, key, content string) error
	Consolidate(session *storage.Session) error
	RememberHistory(key, content string) error
}

// ============ Agent 上下文接口 ============

// ContextBuilderInterface 上下文构建器接口
type ContextBuilderInterface interface {
	Build(ctx context.Context) ([]provider.Message, string, error)
}

// ============ Agent 运行器接口 ============

// AgentRunner Agent 运行器接口
// 定义 Agent 的核心运行能力
type AgentRunner interface {
	Run(ctx context.Context, messages []provider.Message, systemPrompt string) (string, string, []provider.ToolCall, error)
}

// ============ 文件系统接口 ============

// FileSystemInterface 文件系统接口
// 封装文件操作，便于测试和替换实现
type FileSystemInterface interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
}

// OsFileSystem 操作系统文件系统实现
type OsFileSystem struct{}

// ReadFile 读取文件
func (OsFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile 写入文件
func (OsFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat 获取文件信息
func (OsFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// 确保 OsFileSystem 实现 FileSystemInterface
var _ FileSystemInterface = OsFileSystem{}
