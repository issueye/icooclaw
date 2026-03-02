package scheduler

import (
	"context"
	"time"

	"github.com/icooclaw/icooclaw/internal/storage"
)

// TaskRunStatus 任务运行状态
type TaskRunStatus int

const (
	TaskPending    TaskRunStatus = iota // 待运行
	TaskRunning    TaskRunStatus = iota // 运行中
	TaskCompleted  TaskRunStatus = iota // 已完成
	TaskTerminated TaskRunStatus = iota // 已终止
	TaskFailed     TaskRunStatus = iota // 运行失败
	TaskRetrying   TaskRunStatus = iota // 重试中
)

/**
 * RetryConfig 重试配置
 */
type RetryConfig struct {
	MaxRetries  int           // 最大重试次数
	Interval    time.Duration // 重试间隔
	Backoff     time.Duration // 退避因子
	MaxInterval time.Duration // 最大重试间隔
}

/**
 * TaskMetrics 任务执行指标
 */
type TaskMetrics struct {
	TotalRuns     int64         // 总执行次数
	SuccessRuns   int64         // 成功次数
	FailedRuns    int64         // 失败次数
	RetryCount    int64         // 重试次数
	AvgDuration   time.Duration // 平均执行时长
	MaxDuration   time.Duration // 最大执行时长
	LastRunAt     time.Time     // 最后执行时间
	LastSuccessAt time.Time     // 最后成功时间
	LastError     string        // 最后错误信息
}

// TaskType 任务类型
type TaskType int

const (
	TaskTypeCron     TaskType = iota // Cron 任务
	TaskTypeInterval TaskType = iota // 间隔任务
	TaskTypeOnce     TaskType = iota // 一次性任务 立即执行
)

/**
 * TaskCallback 定义任务执行回调接口
 * 用于任务生命周期事件的外部处理
 */
type TaskCallback interface {
	OnStart(ctx context.Context, task *storage.Task)
	OnComplete(ctx context.Context, task *storage.Task, err error)
	OnNextRunCalculated(task *storage.Task, nextRun time.Time)
}

/**
 * TaskRunner 任务运行器接口
 * 职责：执行任务、管理自身状态、提供任务信息
 * 不负责判断是否应该执行（由调度器负责）
 */
type TaskRunner interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
	GetStatus() TaskRunStatus
	GetType() TaskType
	GetName() string
	GetInfo() *storage.Task
	ShouldRun(now time.Time) bool
	SetCallback(callback TaskCallback)
	GetMetrics() *TaskMetrics
	SetRetryConfig(config RetryConfig)
}

/**
 * TaskExecutor 任务执行器接口
 * 定义实际执行任务的逻辑
 */
type TaskExecutor interface {
	Execute(ctx context.Context) error
}
