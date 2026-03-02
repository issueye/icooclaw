package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
	"github.com/icooclaw/icooclaw/internal/storage"
)

/**
 * Scheduler 定时任务调度器
 * 职责：加载任务、调度执行、管理生命周期
 */
type Scheduler struct {
	bus          *bus.MessageBus
	storage      *storage.Storage
	config       *config.Config
	logger       *slog.Logger
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	TaskRunners  map[string]TaskRunner
	running      bool
	taskCh       chan TaskRunner
	taskCallback TaskCallback
}

/**
 * SchedulerConfig 调度器配置
 */
type SchedulerConfig struct {
	TaskTimeout   time.Duration
	QueueSize     int
	CheckInterval time.Duration
}

// NewScheduler 创建调度器
func NewScheduler(bus *bus.MessageBus, storage *storage.Storage, cfg *config.Config, logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		bus:         bus,
		storage:     storage,
		config:      cfg,
		logger:      logger,
		wg:          sync.WaitGroup{},
		ctx:         ctx,
		cancel:      cancel,
		TaskRunners: make(map[string]TaskRunner),
		taskCh:      make(chan TaskRunner, 100),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// 加载任务
	tasks, err := s.storage.GetEnabledTasks()
	if err != nil {
		s.logger.Warn("Failed to load tasks", "error", err)
	}

	// 创建任务运行器
	for _, task := range tasks {
		var runner TaskRunner
		switch task.Type {
		case storage.TaskTypeImmediate:
			runner, err = NewOnceTaskRunner(task.Name, &task, s.logger)
		case storage.TaskTypeCron:
			runner, err = NewCronTaskRunner(task.Name, &task, task.CronExpr, s.logger)
		}

		if err != nil {
			s.logger.Error("任务加载失败", "name", task.Name, "cron", task.CronExpr, "error", err)
			continue
		}

		// 注册任务回调
		runner.SetCallback(s)

		s.TaskRunners[task.Name] = runner
		s.logger.Info("任务加载成功", "name", task.Name, "cron", task.CronExpr)
	}

	// 启动 goroutine 检查任务（2个：一个检查任务，一个执行任务）
	s.wg.Add(2)
	go s.run()
	go s.runExecutor()

	s.running = true
	s.logger.Info("调度器启动成功", "task_count", len(s.TaskRunners))
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.cancel()
	s.wg.Wait()

	for name := range s.TaskRunners {
		if err := s.TaskRunners[name].Stop(s.ctx); err != nil {
			s.logger.Warn("终止任务失败", "name", name, "error", err)
		}
		delete(s.TaskRunners, name)
	}

	s.running = false
	s.logger.Info("调度器已停止")
	return nil
}

// IsRunning 检查是否运行
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// run 运行调度器
func (s *Scheduler) run() {
	defer s.wg.Done()

	// 每分钟检查一次
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkTasks()
		}
	}
}

/**
 * checkTasks 检查任务
 * 在锁保护下遍历任务列表，判断是否应该执行
 */
func (s *Scheduler) checkTasks() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	for _, runner := range s.TaskRunners {
		if !runner.GetInfo().Enabled {
			continue
		}

		// 使用 Runner 自己的 ShouldRun 方法判断是否应该运行
		if runner.ShouldRun(now) {
			// 发送到执行队列，避免阻塞
			select {
			case s.taskCh <- runner:
			default:
				s.logger.Warn("任务队列已满", "name", runner.GetInfo().Name)
			}
		}
	}
}

/**
 * runExecutor 任务执行器
 * 从队列中取出任务并执行
 */
func (s *Scheduler) runExecutor() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case runner := <-s.taskCh:
			s.doExecuteTask(runner)
		}
	}
}

/**
 * doExecuteTask 执行任务（内部实现）
 * 包含超时控制和重试机制
 */
func (s *Scheduler) doExecuteTask(runner TaskRunner) {
	info := runner.GetInfo()
	metrics := runner.GetMetrics()

	// 记录开始时间
	startTime := time.Now()

	s.logger.Info("执行任务", "name", info.Name, "cron", info.CronExpr)

	// 创建带超时的上下文
	defaultTimeout := 30 * time.Minute
	ctx, cancel := context.WithTimeout(s.ctx, defaultTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, "task_name", info.Name)

	var err error
	retryConfig := RetryConfig{
		MaxRetries:  3,
		Interval:    1 * time.Second,
		Backoff:     2.0,
		MaxInterval: 30 * time.Second,
	}

	// 重试循环
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		// 更新指标
		metrics.TotalRuns++

		// 执行任务
		err = runner.Run(ctx)

		if err == nil {
			// 成功
			metrics.SuccessRuns++
			metrics.LastSuccessAt = time.Now()
			s.logger.Info("任务执行成功", "name", info.Name, "attempt", attempt+1)
			break
		}

		// 失败
		metrics.FailedRuns++
		metrics.LastError = err.Error()
		s.logger.Warn("任务执行失败", "name", info.Name, "error", err, "attempt", attempt+1)

		// 如果还有重试次数，等待后重试
		if attempt < retryConfig.MaxRetries {
			metrics.RetryCount++
			s.logger.Info("任务即将重试", "name", info.Name, "next_attempt", attempt+2)

			// 计算退避间隔
			sleepTime := retryConfig.Interval
			for i := 0; i < attempt; i++ {
				sleepTime = time.Duration(float64(sleepTime) * float64(retryConfig.Backoff))
				if sleepTime > retryConfig.MaxInterval {
					sleepTime = retryConfig.MaxInterval
					break
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepTime):
			}
		}
	}

	// 更新执行时长指标
	duration := time.Since(startTime)
	metrics.TotalRuns++
	if metrics.TotalRuns == 1 {
		metrics.AvgDuration = duration
	} else {
		metrics.AvgDuration = (metrics.AvgDuration + duration) / 2
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
	metrics.LastRunAt = startTime

	// 更新存储中的任务信息
	_ = s.storage.UpdateTask(info)

	if err != nil {
		s.logger.Error("任务执行最终失败", "name", info.Name, "error", err)
		// info.Status = TaskFailed
	} else {
		s.logger.Info("任务执行完成", "name", info.Name)
		// info.Status = TaskCompleted
	}

	// 发送消息到消息总线
	s.publishTaskMessage(info, err)
}

/**
 * publishTaskMessage 发布任务消息到消息总线
 */
func (s *Scheduler) publishTaskMessage(info *storage.Task, taskErr error) {
	msg := bus.InboundMessage{
		ID:        info.Name,
		Channel:   info.Channel,
		ChatID:    info.ChatID,
		Content:   info.Message,
		Timestamp: info.LastRunAt,
		Metadata: map[string]interface{}{
			"task_name": info.Name,
			"cron_expr": info.CronExpr,
			"next_run":  info.NextRunAt,
		},
	}

	if taskErr != nil {
		msg.Metadata["error"] = taskErr.Error()
	}

	if err := s.bus.PublishInbound(s.ctx, msg); err != nil {
		s.logger.Error("推送任务消息失败", "name", info.Name, "error", err)
	}
}

// RunTask 运行任务
func (s *Scheduler) RunTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取任务运行器
	runner, ok := s.TaskRunners[name]
	if !ok {
		return fmt.Errorf("任务 %s 不存在", name)
	}

	// 运行任务
	ctx := context.WithValue(s.ctx, "task_name", runner.GetInfo().Name)
	err := runner.Run(ctx)
	if err != nil {
		return fmt.Errorf("任务 %s 运行失败: %w", name, err)
	}

	return nil
}

// AddTask 添加任务
func (s *Scheduler) AddTask(task *storage.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		runner TaskRunner
		err    error
	)
	switch task.Type {
	case storage.TaskTypeImmediate:
		runner, err = NewOnceTaskRunner(task.Name, task, s.logger)
	case storage.TaskTypeCron:
		runner, err = NewCronTaskRunner(task.Name, task, task.CronExpr, s.logger)
	}

	if err != nil {
		return fmt.Errorf("任务 %s 创建失败: %w", task.Name, err)
	}

	// 注册任务回调
	runner.SetCallback(s)

	s.TaskRunners[task.Name] = runner
	return s.storage.CreateTask(task)
}

// RemoveTask 移除任务
func (s *Scheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.TaskRunners[name]; ok {
		delete(s.TaskRunners, name)
		return s.storage.DeleteTask(runner.GetInfo().ID)
	}
	return nil
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.TaskRunners))
	for name := range s.TaskRunners {
		names = append(names, name)
	}
	return names
}

// GetTask 获取任务
func (s *Scheduler) GetTask(name string) *storage.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.TaskRunners[name]; ok {
		return runner.GetInfo()
	}
	return nil
}

// EnableTask 启用任务
func (s *Scheduler) EnableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.TaskRunners[name]; ok {
		runner.GetInfo().Enabled = true
		return s.storage.UpdateTask(runner.GetInfo())
	}
	return nil
}

// DisableTask 禁用任务
func (s *Scheduler) DisableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.TaskRunners[name]; ok {
		runner.GetInfo().Enabled = false
		return s.storage.UpdateTask(runner.GetInfo())
	}
	return nil
}

// GetTaskNextRun 获取任务下次运行时间
func (s *Scheduler) GetTaskNextRun(name string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.TaskRunners[name]; ok {
		return runner.GetInfo().NextRunAt, !runner.GetInfo().NextRunAt.IsZero()
	}
	return time.Time{}, false
}

/**
 * GetTaskMetrics 获取任务执行指标
 */
func (s *Scheduler) GetTaskMetrics(name string) *TaskMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.TaskRunners[name]; ok {
		return runner.GetMetrics()
	}
	return nil
}

/**
 * GetAllMetrics 获取所有任务的执行指标
 */
func (s *Scheduler) GetAllMetrics() map[string]*TaskMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*TaskMetrics)
	for name, runner := range s.TaskRunners {
		result[name] = runner.GetMetrics()
	}
	return result
}

/**
 * UpdateTask 更新任务配置
 */
func (s *Scheduler) UpdateTask(task *storage.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, ok := s.TaskRunners[task.Name]
	if !ok {
		return ErrTaskNotFound
	}

	// 更新任务配置
	runner.GetInfo().Channel = task.Channel
	runner.GetInfo().ChatID = task.ChatID
	runner.GetInfo().Message = task.Message
	runner.GetInfo().Enabled = task.Enabled
	runner.GetInfo().CronExpr = task.CronExpr

	return s.storage.UpdateTask(runner.GetInfo())
}

/**
 * ReloadTask 重新加载任务
 */
func (s *Scheduler) ReloadTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	runner, ok := s.TaskRunners[name]
	if !ok {
		return ErrTaskNotFound
	}

	// 停止旧任务
	if err := runner.Stop(s.ctx); err != nil {
		s.logger.Warn("停止任务失败", "name", name, "error", err)
	}

	// 从存储重新加载任务
	task, err := storage.GetTaskByID(runner.GetInfo().ID)
	if err != nil {
		return fmt.Errorf("重新加载任务失败: %w", err)
	}

	// 重新创建运行器
	var newRunner TaskRunner
	switch task.Type {
	case storage.TaskTypeImmediate:
		newRunner, err = NewOnceTaskRunner(task.Name, task, s.logger)
	case storage.TaskTypeCron:
		newRunner, err = NewCronTaskRunner(task.Name, task, task.CronExpr, s.logger)
	}

	if err != nil {
		return fmt.Errorf("重新创建任务运行器失败: %w", err)
	}

	// 注册回调
	newRunner.SetCallback(s)

	s.TaskRunners[name] = newRunner
	s.logger.Info("任务已重新加载", "name", name)
	return nil
}

/**
 * GetTaskStatus 获取任务状态
 */
func (s *Scheduler) GetTaskStatus(name string) (TaskRunStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.TaskRunners[name]; ok {
		return runner.GetStatus(), true
	}
	return TaskPending, false
}

// 错误定义
var (
	ErrTaskNotFound          = &SchedulerError{"task not found"}
	ErrTaskAlreadyExists     = &SchedulerError{"task already exists"}
	ErrInvalidCronExpression = &SchedulerError{"invalid cron expression"}
)

type SchedulerError struct {
	msg string
}

func (e *SchedulerError) Error() string {
	return e.msg
}

/**
 * OnStart 实现 TaskCallback 接口 - 任务开始回调
 */
func (s *Scheduler) OnStart(ctx context.Context, task *storage.Task) {
	s.logger.Info("任务开始执行", "name", task.Name)
}

/**
 * OnComplete 实现 TaskCallback 接口 - 任务完成回调
 */
func (s *Scheduler) OnComplete(ctx context.Context, task *storage.Task, err error) {
	if err != nil {
		s.logger.Error("任务执行失败", "name", task.Name, "error", err)
	} else {
		s.logger.Info("任务执行完成", "name", task.Name)
	}
}

/**
 * OnNextRunCalculated 实现 TaskCallback 接口 - 下次运行时间计算回调
 */
func (s *Scheduler) OnNextRunCalculated(task *storage.Task, nextRun time.Time) {
	s.logger.Debug("下次运行时间已计算", "name", task.Name, "next_run", nextRun)
}
