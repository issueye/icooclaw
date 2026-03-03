package agent

import (
	"context"
	"os"

	"icooclaw.ai/provider"
	"icooclaw.ai/skill"
	"icooclaw.ai/tools"
	"icooclaw.core/config"
	"icooclaw.core/storage"

	"log/slog"

	icooclawbus "icooclaw.core/bus"
)

// ============ 核心接口定义 ============

// StorageInterface 存储接口
// 封装会话和消息的持久化操作
type StorageInterface interface {
	GetOrCreateSession(channel, chatID, userID string) (*storage.Session, error)
	AddMessage(sessionID uint, role, content, toolCalls, toolCallID, toolName, reasoningContent string) (*storage.Message, error)
	GetSession(sessionID uint) (*storage.Session, error)
	UpdateSessionMetadata(sessionID uint, metadata string) error
	// Task operations
	CreateTask(task *storage.Task) error
	GetTaskByName(name string) (*storage.Task, error)
	GetAllTasks() ([]storage.Task, error)
	DeleteTask(id uint) error
}

// 确保 *storage.Storage 实现该接口
// var _ StorageInterface = (*storage.Storage)(nil)

// ToolRegistryInterface 工具注册表接口
// 封装工具的注册、查询和执行
type ToolRegistryInterface interface {
	Register(tool tools.Tool)
	Get(name string) (tools.Tool, error)
	ToDefinitions() []tools.ToolDefinition
	Execute(ctx context.Context, call interface{}) tools.ToolResult
}

// 确保 *tools.Registry 实现该接口
var _ ToolRegistryInterface = (*tools.Registry)(nil)

// MemoryStoreInterface 记忆存储接口
// 封装长期记忆的加载和管理
type MemoryStoreInterface interface {
	Load(ctx context.Context) error
	GetAll() ([]storage.Memory, error)
	Add(memType, key, content string) error
	Consolidate(session *storage.Session) error
	RememberHistory(key, content string) error
}

// 确保 *memory.MemoryStore 实现该接口
// var _ MemoryStoreInterface = (*memory.MemoryStore)(nil)

// SkillLoaderInterface 技能加载接口
// 封装技能的加载和管理
type SkillLoaderInterface interface {
	Load(ctx context.Context) error
	GetLoaded() []skill.Skill
	GetByName(name string) *skill.Skill
}

// 确保 *skill.Loader 实现该接口
var _ SkillLoaderInterface = (*skill.Loader)(nil)

// MessageBusInterface 消息总线接口
// 封装消息的输入和输出
type MessageBusInterface interface {
	ConsumeInbound(ctx context.Context) (icooclawbus.InboundMessage, error)
	PublishOutbound(ctx context.Context, msg icooclawbus.OutboundMessage) error
}

// 确保 *icooclawbus.MessageBus 实现该接口
var _ MessageBusInterface = (*icooclawbus.MessageBus)(nil)

// ============ Agent 上下文接口 ============

// AgentContext Agent 上下文接口
// 为 ContextBuilder 提供最小化的 Agent 能力抽象
type AgentContext interface {
	Logger() *slog.Logger
	Workspace() string
	Skills() SkillLoaderInterface
	Memory() MemoryStoreInterface
	Config() config.AgentSettings
	GetSessionRolePrompt(sessionID uint) (string, error)
	GetSystemPrompt() string
}

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
