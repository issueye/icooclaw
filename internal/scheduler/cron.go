package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/adhocore/gronx"
	"github.com/icooclaw/icooclaw/internal/bus"
	"github.com/icooclaw/icooclaw/internal/config"
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

// Scheduler 定时任务调度器
type Scheduler struct {
	bus        *bus.MessageBus
	storage    *storage.Storage
	config     *config.Config
	logger     *slog.Logger
	tasks      map[string]*TaskRunner
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
	mu         sync.RWMutex
	cronParser *CronParser
}

// TaskRunner 任务运行器
type TaskRunner struct {
	name       string
	task       *storage.Task
	cronExpr   string
	logger     *slog.Logger
	cronParser *CronParser
	lastRun    time.Time
	nextRun    time.Time
}

// NewScheduler 创建调度器
func NewScheduler(bus *bus.MessageBus, storage *storage.Storage, cfg *config.Config, logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		bus:        bus,
		storage:    storage,
		config:     cfg,
		logger:     logger,
		tasks:      make(map[string]*TaskRunner),
		ctx:        ctx,
		cancel:     cancel,
		cronParser: NewCronParser(),
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
	for i := range tasks {
		task := &tasks[i]
		runner := s.createTaskRunner(task)
		s.tasks[task.Name] = runner
		s.logger.Info("Task loaded", "name", task.Name, "cron", task.CronExpr)
	}

	// 启动 goroutine 检查任务
	s.wg.Add(1)
	go s.run()

	s.running = true
	s.logger.Info("Scheduler started", "task_count", len(s.tasks))
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

	for name := range s.tasks {
		delete(s.tasks, name)
	}

	s.running = false
	s.logger.Info("Scheduler stopped")
	return nil
}

// IsRunning 检查是否运行
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// createTaskRunner 创建任务运行器
func (s *Scheduler) createTaskRunner(task *storage.Task) *TaskRunner {
	now := time.Now()
	var nextRun time.Time

	// 验证 Cron 表达式
	if s.cronParser.IsValid(task.CronExpr) {
		nextRun, _ = s.cronParser.NextRun(task.CronExpr, now)
	} else {
		s.logger.Warn("Invalid cron expression", "task", task.Name, "cron", task.CronExpr)
	}

	// 如果上次运行时间存在，使用它
	lastRun := task.LastRunAt
	if lastRun.IsZero() {
		lastRun = now
	}

	return &TaskRunner{
		name:       task.Name,
		task:       task,
		cronExpr:   task.CronExpr,
		logger:     s.logger,
		cronParser: s.cronParser,
		lastRun:    lastRun,
		nextRun:    nextRun,
	}
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

// checkTasks 检查任务
func (s *Scheduler) checkTasks() {
	now := time.Now()

	for _, runner := range s.tasks {
		if !runner.task.Enabled {
			continue
		}

		// 使用 gronx 检查是否应该运行
		if runner.cronParser.ShouldRun(runner.cronExpr, now) {
			// 检查是否已经运行过（避免重复运行）
			if now.Sub(runner.lastRun) > 30*time.Second {
				s.executeTask(runner)
			}
		}
	}
}

// executeTask 执行任务
func (s *Scheduler) executeTask(runner *TaskRunner) {
	s.logger.Info("Executing task", "name", runner.name, "cron", runner.cronExpr)

	// 更新运行时间
	now := time.Now()
	runner.lastRun = now

	// 计算下次运行时间
	nextRun, _ := runner.cronParser.NextRun(runner.cronExpr, now)
	runner.nextRun = nextRun

	// 更新存储中的最后执行时间
	runner.task.LastRunAt = now
	_ = s.storage.UpdateTask(runner.task)

	// 发送消息到消息总线
	msg := bus.InboundMessage{
		ID:        runner.task.Name,
		Channel:   runner.task.Channel,
		ChatID:    runner.task.ChatID,
		Content:   runner.task.Message,
		Timestamp: now,
		Metadata: map[string]interface{}{
			"task_name": runner.name,
			"cron_expr": runner.cronExpr,
			"next_run":  nextRun,
		},
	}

	if err := s.bus.PublishInbound(s.ctx, msg); err != nil {
		s.logger.Error("Failed to publish task message", "name", runner.name, "error", err)
	}
}

// AddTask 添加任务
func (s *Scheduler) AddTask(task *storage.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证 Cron 表达式
	if !s.cronParser.IsValid(task.CronExpr) {
		return ErrInvalidCronExpression
	}

	runner := s.createTaskRunner(task)
	s.tasks[task.Name] = runner
	return s.storage.CreateTask(task)
}

// RemoveTask 移除任务
func (s *Scheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.tasks[name]; ok {
		delete(s.tasks, name)
		return s.storage.DeleteTask(runner.task.ID)
	}
	return nil
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tasks))
	for name := range s.tasks {
		names = append(names, name)
	}
	return names
}

// GetTask 获取任务
func (s *Scheduler) GetTask(name string) *storage.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.tasks[name]; ok {
		return runner.task
	}
	return nil
}

// EnableTask 启用任务
func (s *Scheduler) EnableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.tasks[name]; ok {
		runner.task.Enabled = true
		return s.storage.UpdateTask(runner.task)
	}
	return nil
}

// DisableTask 禁用任务
func (s *Scheduler) DisableTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if runner, ok := s.tasks[name]; ok {
		runner.task.Enabled = false
		return s.storage.UpdateTask(runner.task)
	}
	return nil
}

// GetTaskNextRun 获取任务下次运行时间
func (s *Scheduler) GetTaskNextRun(name string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if runner, ok := s.tasks[name]; ok {
		return runner.nextRun, !runner.nextRun.IsZero()
	}
	return time.Time{}, false
}

// ValidateCronExpression 验证 Cron 表达式
func (s *Scheduler) ValidateCronExpression(expr string) bool {
	return s.cronParser.IsValid(expr)
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
