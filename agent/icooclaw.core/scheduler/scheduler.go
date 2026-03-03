package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

/**
 * Scheduler 定时任务调度器
 * 职责：加载任务、调度执行、管理生命周期
 * 使用依赖注入接口，与具体实现解耦
 */
type Scheduler struct {
	storage     TaskStorage
	config      SchedulerConfig
	logger      Logger
	slogLogger  *slog.Logger
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	TaskRunners map[string]TaskRunner
	running     bool
	taskCh      chan TaskRunner
	taskCallback TaskCallback
}

/**
 * SchedulerConfig 调度器配置
 */
type SchedulerOptions struct {
	TaskTimeout   time.Duration
	QueueSize     int
	CheckInterval time.Duration
}

// NewScheduler 创建调度器（使用接口）
func NewScheduler(storage TaskStorage, config SchedulerConfig, logger Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	slogLogger := slog.Default()
	if logger != nil {
		// 如果传入了自定义 Logger，尝试转换为 slog
		if l, ok := logger.(*slog.Logger); ok {
			slogLogger = l
		}
	}
	
	queueSize := config.GetQueueSize()
	if queueSize <= 0 {
		queueSize = 100
	}
	
	return &Scheduler{
		storage:     storage,
		config:      config,
		logger:      logger,
		slogLogger:  slogLogger,
		wg:          sync.WaitGroup{},
		ctx:         ctx,
		cancel:      cancel,
		TaskRunners: make(map[string]TaskRunner),
		taskCh:      make(chan TaskRunner, queueSize),
	}
}

// NewSchedulerWithSlog 使用 slog.Logger 创建调度器
func NewSchedulerWithSlog(storage TaskStorage, config SchedulerConfig, slogLogger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	queueSize := config.GetQueueSize()
	if queueSize <= 0 {
		queueSize = 100
	}
	
	return &Scheduler{
		storage:     storage,
		config:      config,
		logger:      slogLogger,
		slogLogger:  slogLogger,
		wg:          sync.WaitGroup{},
		ctx:         ctx,
		cancel:      cancel,
		TaskRunners: make(map[string]TaskRunner),
		taskCh:      make(chan TaskRunner, queueSize),
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
		s.slogLogger.Warn("Failed to load tasks", "error", err)
	}

	// 创建任务运行器
	for _, task := range tasks {
		var runner TaskRunner
		var err error
		
		switch task.Type {
		case TaskTypeOnce:
			runner, err = NewOnceTaskRunner(task.Name, &task, s.slogLogger)
		case TaskTypeCron:
			runner, err = NewCronTaskRunner(task.Name, &task, task.CronExpr, s.slogLogger)
		default:
			s.slogLogger.Warn("Unknown task type", "name", task.Name, "type", task.Type)
			continue
		}

		if err != nil {
			s.slogLogger.Error("任务加载失败", "name", task.Name, "cron", task.CronExpr, "error", err)
			continue
		}

		// 注册任务回调
		runner.SetCallback(s)

		s.TaskRunners[task.Name] = runner
		s.slogLogger.Info("任务加载成功", "name", task.Name, "cron", task.CronExpr)
	}

	// 启动 goroutine 检查任务（2个：一个检查任务，一个执行任务）
	s.wg.Add(2)
	go s.run()
	go s.runExecutor()

	s.running = true
	s.slogLogger.Info("调度器启动成功", "task_count", len(s.TaskRunners))
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
			s.slogLogger.Warn("终止任务失败", "name", name, "error", err)
		}
		delete(s.TaskRunners, name)
	}

	s.running = false
	s.slogLogger.Info("调度器已停止")
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

	// 使用配置中的检查间隔
	checkInterval := s.config.GetCheckInterval()
	if checkInterval <= 0 {
		checkInterval = 60 * time.Second
	}
	
	ticker := time.NewTicker(checkInterval)
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
		info := runner.GetInfo()
		if !info.Enabled {
			continue
		}

		// 使用 Runner 自己的 ShouldRun 方法判断是否应该运行
		if runner.ShouldRun(now) {
			// 发送到执行队列，避免阻塞
			select {
			case s.taskCh <- runner:
			default:
				s.slogLogger.Warn("任务队列已满", "name", runner.GetInfo().Name)
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

	s.slogLogger.Info("执行任务", "name", info.Name, "cron", info.CronExpr)

	// 创建带超时的上下文
	defaultTimeout := s.config.GetTaskTimeout()
	if defaultTimeout <= 0 {
		defaultTimeout = 30 * time.Minute
	}
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
			s.slogLogger.Info("任务执行成功", "name", info.Name, "attempt", attempt+1)
			break
		}

		// 失败
		metrics.FailedRuns++
		metrics.LastError = err.Error()
		s.slogLogger.Warn("任务执行失败", "name", info.Name, "error", err, "attempt", attempt+1)

		// 如果还有重试次数，等待后重试
		if attempt < retryConfig.MaxRetries {
			metrics.RetryCount++
			s.slogLogger.Info("任务即将重试", "name", info.Name, "next_attempt", attempt+2)

			// 计算退避间隔
			sleepTime := retryConfig.Interval
			for i := 0; i < attempt; i++ {
				sleepTime = time.Duration(float64(sleepTime) * retryConfig.Backoff)
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
}

/**
 * OnStart 任务开始回调
 */
func (s *Scheduler) OnStart(ctx context.Context, task *TaskInfo) {
	s.slogLogger.Debug("Task started", "name", task.Name)
}

/**
 * OnComplete 任务完成回调
 */
func (s *Scheduler) OnComplete(ctx context.Context, task *TaskInfo, err error) {
	if err != nil {
		s.slogLogger.Warn("Task completed with error", "name", task.Name, "error", err)
	} else {
		s.slogLogger.Debug("Task completed", "name", task.Name)
	}
	
	// 更新任务状态到存储
	if s.storage != nil {
		go func() {
			if updateErr := s.storage.UpdateTask(task); updateErr != nil {
				s.slogLogger.Error("Failed to update task", "name", task.Name, "error", updateErr)
			}
		}()
	}
}

/**
 * OnNextRunCalculated 下次运行时间计算回调
 */
func (s *Scheduler) OnNextRunCalculated(task *TaskInfo, nextRun time.Time) {
	s.slogLogger.Debug("Next run calculated", "name", task.Name, "next_run", nextRun)
}

/**
 * GetTaskRunner 获取任务运行器
 */
func (s *Scheduler) GetTaskRunner(name string) (TaskRunner, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runner, ok := s.TaskRunners[name]
	return runner, ok
}

/**
 * AddTaskRunner 添加任务运行器（动态添加）
 */
func (s *Scheduler) AddTaskRunner(runner TaskRunner) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		// 调度器运行中，需要检查任务是否应该执行
		runner.SetCallback(s)
		s.TaskRunners[runner.GetName()] = runner
		s.slogLogger.Info("任务添加成功", "name", runner.GetName())
		return nil
	}

	return ErrSchedulerNotRunning
}

/**
 * RemoveTaskRunner 移除任务运行器
 */
func (s *Scheduler) RemoveTaskRunner(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrSchedulerNotRunning
	}

	runner, ok := s.TaskRunners[name]
	if !ok {
		return ErrTaskNotFound
	}

	if err := runner.Stop(s.ctx); err != nil {
		s.slogLogger.Warn("停止任务失败", "name", name, "error", err)
	}

	delete(s.TaskRunners, name)
	s.slogLogger.Info("任务移除成功", "name", name)
	return nil
}
