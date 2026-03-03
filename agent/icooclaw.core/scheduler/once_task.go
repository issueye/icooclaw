package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

/**
 * OnceTaskRunner 立即执行任务运行器
 */
type OnceTaskRunner struct {
	name     string
	task     *TaskInfo
	logger   *slog.Logger
	status   TaskRunStatus
	taskType TaskType
	callback TaskCallback
	mu       sync.Mutex

	// 重试配置
	retryConfig RetryConfig
	retryCount  int

	// 执行指标
	metrics TaskMetrics

	// 消息发送器
	messageSender TaskMessageSender
}

func NewOnceTaskRunner(name string, task *TaskInfo, logger *slog.Logger) (*OnceTaskRunner, error) {
	return &OnceTaskRunner{
		name:     name,
		task:     task,
		logger:   logger,
		status:   TaskPending,
		taskType: TaskTypeOnce,
	}, nil
}

/**
 * Run 运行任务
 * 只负责执行任务本身，不判断是否应该执行（由调度器负责）
 */
func (r *OnceTaskRunner) Run(ctx context.Context) error {
	r.mu.Lock()
	// 1. 检查当前是否在运行中
	if r.status == TaskRunning {
		r.mu.Unlock()
		r.logger.Info("任务已在运行中", "name", r.name)
		return nil
	}

	// 2. 设置状态为运行中
	r.status = TaskRunning
	r.logger.Info("任务开始执行", "name", r.name)
	r.mu.Unlock()

	// 触发开始回调
	if r.callback != nil {
		r.callback.OnStart(ctx, r.task)
	}

	// 3. 执行任务逻辑 - 发送消息到指定通道
	var execErr error
	if r.messageSender != nil && r.task.Message != "" {
		channel := r.task.Channel
		if channel == "" {
			channel = "websocket" // 默认通道
		}
		chatID := r.task.ChatID
		if chatID == "" {
			chatID = "default" // 默认会话
		}

		r.logger.Info("发送任务消息", "name", r.name, "channel", channel, "chat_id", chatID)
		if err := r.messageSender.SendTaskMessage(ctx, channel, chatID, r.task.Message); err != nil {
			execErr = err
			r.logger.Error("发送任务消息失败", "name", r.name, "error", err)
		}
	} else if r.messageSender == nil {
		r.logger.Warn("消息发送器未设置，跳过消息发送", "name", r.name)
	} else {
		r.logger.Debug("任务消息为空，跳过发送", "name", r.name)
	}

	// 4. 更新状态
	r.mu.Lock()
	if execErr != nil {
		r.status = TaskFailed
	} else {
		r.status = TaskCompleted
	}
	r.task.LastRunAt = time.Now()
	r.mu.Unlock()

	r.logger.Info("任务执行完成", "name", r.name)

	// 触发完成回调
	if r.callback != nil {
		r.callback.OnComplete(ctx, r.task, execErr)
	}

	return execErr
}

// Stop 停止任务
func (r *OnceTaskRunner) Stop(ctx context.Context) error {
	r.status = TaskTerminated
	r.logger.Info("任务已终止", "name", r.name)
	return nil
}

// GetStatus 获取任务运行状态
func (r *OnceTaskRunner) GetStatus() TaskRunStatus {
	return r.status
}

// GetType 获取任务类型
func (r *OnceTaskRunner) GetType() TaskType {
	return r.taskType
}

// GetName 获取任务名称
func (r *OnceTaskRunner) GetName() string {
	return r.name
}

// GetInfo 获取任务信息
func (r *OnceTaskRunner) GetInfo() *TaskInfo {
	return r.task
}

// ShouldRun 是否应该运行
func (r *OnceTaskRunner) ShouldRun(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.task.LastRunAt.IsZero() && r.status != TaskCompleted
}

/**
 * SetCallback 设置任务回调
 */
func (r *OnceTaskRunner) SetCallback(callback TaskCallback) {
	r.callback = callback
}

/**
 * GetMetrics 获取任务执行指标
 */
func (r *OnceTaskRunner) GetMetrics() *TaskMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &r.metrics
}

/**
 * SetRetryConfig 设置重试配置
 */
func (r *OnceTaskRunner) SetRetryConfig(config RetryConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryConfig = config
}

/**
 * SetMessageSender 设置消息发送器
 */
func (r *OnceTaskRunner) SetMessageSender(sender TaskMessageSender) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messageSender = sender
}
