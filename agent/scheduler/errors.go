package scheduler

import (
	"errors"
)

var (
	// ErrInvalidCronExpression 无效的 Cron 表达式
	ErrInvalidCronExpression = errors.New("invalid cron expression")

	// ErrSchedulerNotRunning 调度器未运行
	ErrSchedulerNotRunning = errors.New("scheduler is not running")

	// ErrTaskNotFound 任务未找到
	ErrTaskNotFound = errors.New("task not found")

	// ErrTaskAlreadyRunning 任务已在运行
	ErrTaskAlreadyRunning = errors.New("task is already running")
)
