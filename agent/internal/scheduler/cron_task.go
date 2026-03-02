package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/adhocore/gronx"
	"github.com/icooclaw/icooclaw/internal/storage"
)

// CronParser Cron 表达式解析器
type CronParser struct {
	gronx *gronx.Gronx
}

// NewCronParser 创建 Cron 解析器
func NewCronParser() *CronParser {
	return &CronParser{
		gronx: &gronx.Gronx{},
	}
}

// IsValid 验证 Cron 表达式是否有效
func (p *CronParser) IsValid(expr string) bool {
	_, err := gronx.Segments(expr)
	return err == nil
}

// NextRun 计算下次执行时间
func (p *CronParser) NextRun(expr string, from time.Time) (time.Time, bool) {
	t, err := gronx.NextTickAfter(expr, from, false)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ShouldRun 检查是否应该执行
func (p *CronParser) ShouldRun(expr string, now time.Time) bool {
	next, err := gronx.NextTickAfter(expr, now, false)
	if err != nil {
		return false
	}
	// 如果下次执行时间在未来 60 秒内，认为应该执行
	return next.Sub(now) < 60*time.Second
}

/**
 * CronTaskRunner Cron 任务运行器
 * 负责：根据 Cron 表达式执行任务，管理自身状态
 */
type CronTaskRunner struct {
	name       string
	task       *storage.Task
	cronExpr   string
	logger     *slog.Logger
	cronParser *CronParser
	lastRun    time.Time
	nextRun    time.Time
	status     TaskRunStatus
	taskType   TaskType
	callback   TaskCallback
	mu         sync.Mutex

	// 重试配置
	retryConfig RetryConfig
	retryCount  int

	// 执行指标
	metrics TaskMetrics
}

func NewCronTaskRunner(name string, task *storage.Task, cronExpr string, logger *slog.Logger) (*CronTaskRunner, error) {
	now := time.Now()
	var nextRun time.Time

	parser := NewCronParser()

	// 验证 Cron 表达式
	isValid := parser.IsValid(task.CronExpr)
	if !isValid {
		return nil, ErrInvalidCronExpression
	}

	// 如果表达式有效，计算下次执行时间
	if isValid {
		nr, ok := parser.NextRun(task.CronExpr, now)
		if !ok {
			return nil, ErrInvalidCronExpression
		}

		nextRun = nr
	} else {
		logger.Warn("定时任务表达式校验失败", "task", task.Name, "cron", task.CronExpr)
	}

	// 如果上次运行时间存在，使用它
	lastRun := task.LastRunAt
	if lastRun.IsZero() {
		lastRun = now
	}

	runner := &CronTaskRunner{
		name:       name,
		task:       task,
		cronExpr:   cronExpr,
		logger:     logger,
		cronParser: parser,
		status:     TaskPending,
		taskType:   TaskTypeCron,
	}

	runner.lastRun = lastRun
	runner.nextRun = nextRun
	return runner, nil
}

/**
 * Run 执行任务
 * 只负责执行任务本身，不判断是否应该执行（由调度器负责）
 */
func (r *CronTaskRunner) Run(ctx context.Context) error {
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

	// 3. 检查是否应该运行（基于时间判断）
	if !r.cronParser.ShouldRun(r.cronExpr, time.Now()) {
		r.logger.Info("任务未到运行时间", "name", r.name, "next_run", r.nextRun)
		r.status = TaskCompleted
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	// 触发开始回调
	if r.callback != nil {
		r.callback.OnStart(ctx, r.task)
	}

	// 4. 执行任务逻辑
	// TODO: 执行任务逻辑

	// 5. 更新时间
	r.mu.Lock()
	now := time.Now()
	r.nextRun, _ = r.cronParser.NextRun(r.cronExpr, now)
	r.lastRun = now
	r.task.LastRunAt = now
	r.task.NextRunAt = r.nextRun
	r.status = TaskCompleted
	r.mu.Unlock()

	r.logger.Info("任务执行完成", "name", r.name)

	// 触发完成回调
	if r.callback != nil {
		r.callback.OnComplete(ctx, r.task, nil)
		r.callback.OnNextRunCalculated(r.task, r.nextRun)
	}

	return nil
}

// Stop 停止任务
func (r *CronTaskRunner) Stop(ctx context.Context) error {
	r.status = TaskTerminated
	r.logger.Info("任务已终止", "name", r.name)
	return nil
}

// GetStatus 获取任务运行状态
func (r *CronTaskRunner) GetStatus() TaskRunStatus {
	return r.status
}

// GetType 获取任务类型
func (r *CronTaskRunner) GetType() TaskType {
	return r.taskType
}

// GetName 获取任务名称
func (r *CronTaskRunner) GetName() string {
	return r.name
}

// GetInfo 获取任务信息
func (r *CronTaskRunner) GetInfo() *storage.Task {
	return r.task
}

// ShouldRun 是否应该运行
func (r *CronTaskRunner) ShouldRun(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cronParser.ShouldRun(r.cronExpr, now)
}

/**
 * SetCallback 设置任务回调
 */
func (r *CronTaskRunner) SetCallback(callback TaskCallback) {
	r.callback = callback
}

/**
 * GetMetrics 获取任务执行指标
 */
func (r *CronTaskRunner) GetMetrics() *TaskMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return &r.metrics
}

/**
 * SetRetryConfig 设置重试配置
 */
func (r *CronTaskRunner) SetRetryConfig(config RetryConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryConfig = config
}
