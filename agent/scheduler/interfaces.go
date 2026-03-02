package scheduler

import (
	"time"
)

/**
 * TaskType 任务类型
 */
type TaskType int

const (
	TaskTypeCron     TaskType = iota // Cron 任务
	TaskTypeInterval TaskType = iota // 间隔任务
	TaskTypeOnce     TaskType = iota // 一次性任务 立即执行
)

/**
 * TaskInfo 任务信息
 * 定义任务的基本属性，与 storage 解耦
 */
type TaskInfo struct {
	ID          uint
	Name        string
	Description string
	Type        TaskType
	CronExpr    string
	Interval    int
	Message     string
	Channel     string
	ChatID      string
	Enabled     bool
	NextRunAt   time.Time
	LastRunAt   time.Time
}

/**
 * TaskStorage 任务存储接口
 * 定义任务数据的存储和查询操作
 */
type TaskStorage interface {
	// GetEnabledTasks 获取所有启用的任务
	GetEnabledTasks() ([]TaskInfo, error)
	// GetTaskByName 根据名称获取任务
	GetTaskByName(name string) (*TaskInfo, error)
	// UpdateTask 更新任务信息
	UpdateTask(task *TaskInfo) error
}

/**
 * MessageBus 消息总线接口
 * 定义事件发布机制
 */
type MessageBus interface {
	// Publish 发布事件
	Publish(event interface{}) error
	// Subscribe 订阅事件
	Subscribe(handler interface{}) error
}

/**
 * Logger 日志接口
 */
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

/**
 * SchedulerConfig 调度器配置接口
 */
type SchedulerConfig interface {
	// GetTaskTimeout 获取任务超时时间
	GetTaskTimeout() time.Duration
	// GetQueueSize 获取任务队列大小
	GetQueueSize() int
	// GetCheckInterval 获取任务检查间隔
	GetCheckInterval() time.Duration
	// IsEnabled 检查调度器是否启用
	IsEnabled() bool
	// GetHeartbeatInterval 获取心跳间隔
	GetHeartbeatInterval() time.Duration
	// IsHeartbeatEnabled 检查心跳是否启用
	IsHeartbeatEnabled() bool
	// GetWorkspace 获取工作空间路径
	GetWorkspace() string
}

/**
 * DefaultSchedulerConfig 默认配置实现
 */
type DefaultSchedulerConfig struct {
	TaskTimeout       time.Duration
	QueueSize         int
	CheckInterval     time.Duration
	Enabled           bool
	HeartbeatInterval time.Duration
	HeartbeatEnabled  bool
	Workspace         string
}

func (c *DefaultSchedulerConfig) GetTaskTimeout() time.Duration {
	if c.TaskTimeout <= 0 {
		return 30 * time.Minute
	}
	return c.TaskTimeout
}

func (c *DefaultSchedulerConfig) GetQueueSize() int {
	if c.QueueSize <= 0 {
		return 100
	}
	return c.QueueSize
}

func (c *DefaultSchedulerConfig) GetCheckInterval() time.Duration {
	if c.CheckInterval <= 0 {
		return 60 * time.Second
	}
	return c.CheckInterval
}

func (c *DefaultSchedulerConfig) IsEnabled() bool {
	return c.Enabled
}

func (c *DefaultSchedulerConfig) GetHeartbeatInterval() time.Duration {
	if c.HeartbeatInterval <= 0 {
		return 30 * time.Minute
	}
	return c.HeartbeatInterval
}

func (c *DefaultSchedulerConfig) IsHeartbeatEnabled() bool {
	return c.HeartbeatEnabled
}

func (c *DefaultSchedulerConfig) GetWorkspace() string {
	return c.Workspace
}

/**
 * NewDefaultSchedulerConfig 创建默认配置
 */
func NewDefaultSchedulerConfig() *DefaultSchedulerConfig {
	return &DefaultSchedulerConfig{
		TaskTimeout:       30 * time.Minute,
		QueueSize:         100,
		CheckInterval:     60 * time.Second,
		Enabled:           true,
		HeartbeatInterval: 30 * time.Minute,
		HeartbeatEnabled:  true,
		Workspace:         "",
	}
}
