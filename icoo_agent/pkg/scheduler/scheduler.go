// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"fmt"
	"icooclaw/pkg/bus"
	"icooclaw/pkg/channels/consts"
	"icooclaw/pkg/storage"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Task 定时任务.
type Task struct {
	ID          string
	Name        string
	Schedule    string
	Description string
	Handler     func(ctx context.Context) error
	Enabled     bool
	LastRun     time.Time
	NextRun     time.Time
	EntryID     cron.EntryID
}

// TaskResult 任务执行结果。
type TaskResult struct {
	TaskID    string
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

// Scheduler 定时任务调度器.
type Scheduler struct {
	cron    *cron.Cron
	tasks   map[string]*Task
	results chan TaskResult
	logger  *slog.Logger
	mu      sync.RWMutex
	storage *storage.TaskStorage
	bus     *bus.MessageBus
	running bool
}

// NewScheduler 创建定时任务调度器.
func NewScheduler(storage *storage.TaskStorage, bus *bus.MessageBus, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		cron:    cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC)),
		tasks:   make(map[string]*Task),
		results: make(chan TaskResult, 100),
		logger:  logger,
		storage: storage,
		bus:     bus,
	}
}

// AddTask 添加定时任务.
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("任务ID %s 已存在", task.ID)
	}

	// Parse schedule
	schedule, err := cron.ParseStandard(task.Schedule)
	if err != nil {
		return fmt.Errorf("无效的调度表达式: %w", err)
	}

	// Create cron job
	entryID := s.cron.Schedule(schedule, cron.FuncJob(func() {
		s.executeTask(task)
	}))

	task.EntryID = entryID
	task.NextRun = s.cron.Entry(entryID).Next
	s.tasks[task.ID] = task

	s.logger.Info("任务已添加", "id", task.ID, "name", task.Name, "schedule", task.Schedule)
	return nil
}

// RemoveTask 删除定时任务.
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	s.cron.Remove(task.EntryID)
	delete(s.tasks, id)

	s.logger.Info("任务已移除", "id", id)
	return nil
}

// EnableTask 启用定时任务.
func (s *Scheduler) EnableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	task.Enabled = true
	s.logger.Info("任务已启用", "id", id)
	return nil
}

// DisableTask 禁用定时任务.
func (s *Scheduler) DisableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	task.Enabled = false
	s.logger.Info("任务已禁用", "id", id)
	return nil
}

// GetTask 获取定时任务详情.
func (s *Scheduler) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("任务ID %s 未找到", id)
	}

	// Update next run time
	entry := s.cron.Entry(task.EntryID)
	if entry.ID != 0 {
		task.NextRun = entry.Next
	}

	return task, nil
}

// ListTasks 列出所有定时任务.
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		// Update next run time
		entry := s.cron.Entry(task.EntryID)
		if entry.ID != 0 {
			task.NextRun = entry.Next
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// RunTask 立即执行定时任务.
func (s *Scheduler) RunTask(id string) error {
	s.mu.RLock()
	task, exists := s.tasks[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("任务ID %s 未找到", id)
	}

	go s.executeTask(task)
	return nil
}

func (s *Scheduler) LoadTasks() error {
	tasks, err := s.storage.GetAll()
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	for _, task := range tasks {
		s.tasks[task.ID] = &Task{
			ID:          task.ID,
			Name:        task.Name,
			Schedule:    task.CronExpr,
			Description: task.Description,
			Enabled:     task.Enabled,
			Handler: func(ctx context.Context) error {
				s.logger.Info("执行任务", "task_id", task.ID, "task_name", task.Name)
				// 发送一条 outbound 消息
				msg := bus.InboundMessage{
					Channel:   consts.WEBSOCKET,
					SessionID: "",
					Text:      task.Description,
					Timestamp: time.Now(),
					Metadata: map[string]any{
						"task_id":   task.ID,
						"task_name": task.Name,
					},
				}
				s.bus.PublishInbound(context.Background(), msg)
				return nil
			},
		}
	}
	return nil
}

// Start 启动定时任务调度器.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.cron.Start()
	s.running = true
	s.logger.Info("调度器已启动")
}

// Stop 停止定时任务调度器.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.logger.Info("调度器已停止")
}

// Results 返回任务执行结果通道.
func (s *Scheduler) Results() <-chan TaskResult {
	return s.results
}

// IsRunning 是否正在运行.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// executeTask 执行定时任务.
func (s *Scheduler) executeTask(task *Task) {
	if !task.Enabled {
		return
	}

	startTime := time.Now()
	s.logger.Debug("正在执行任务", "id", task.ID, "name", task.Name)

	ctx := context.Background()
	err := task.Handler(ctx)

	endTime := time.Now()
	task.LastRun = endTime

	// Update next run time
	entry := s.cron.Entry(task.EntryID)
	if entry.ID != 0 {
		task.NextRun = entry.Next
	}

	// Send result to channel
	result := TaskResult{
		TaskID:    task.ID,
		StartTime: startTime,
		EndTime:   endTime,
		Error:     err,
	}

	select {
	case s.results <- result:
	default:
		s.logger.Warn("结果通道已满，丢弃结果", "task_id", task.ID)
	}

	if err != nil {
		s.logger.Error("任务执行失败", "id", task.ID, "error", err)
	} else {
		s.logger.Debug("任务执行成功", "id", task.ID, "duration", endTime.Sub(startTime))
	}
}

// Common 定时任务调度.
const (
	EveryMinute    = "* * * * *"    // 每分钟执行一次
	Every5Minutes  = "*/5 * * * *"  // 每5分钟执行一次
	Every15Minutes = "*/15 * * * *" // 每15分钟执行一次
	Every30Minutes = "*/30 * * * *" // 每30分钟执行一次
	EveryHour      = "0 * * * *"    // 每小时执行一次
	Every2Hours    = "0 */2 * * *"  // 每2小时执行一次
	Every6Hours    = "0 */6 * * *"  // 每6小时执行一次
	Every12Hours   = "0 */12 * * *" // 每12小时执行一次
	EveryDay       = "0 0 * * *"    // 每天执行一次
	EveryWeek      = "0 0 * * 0"    // 每周执行一次（周日）
	EveryMonth     = "0 0 1 * *"    // 每月1号执行一次
)

// ParseDuration 解析持续时间字符串并返回定时任务调度表达式.
func ParseDuration(d string) (string, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return "", err
	}

	// Convert to cron schedule
	switch {
	case duration < time.Minute:
		return "", fmt.Errorf("最小持续时间为 1 分钟")
	case duration == time.Minute:
		return EveryMinute, nil
	case duration%time.Hour == 0:
		hours := int(duration / time.Hour)
		if hours == 1 {
			return EveryHour, nil
		}
		return fmt.Sprintf("0 */%d * * *", hours), nil
	case duration%time.Minute == 0:
		minutes := int(duration / time.Minute)
		if minutes == 1 {
			return EveryMinute, nil
		}
		return fmt.Sprintf("*/%d * * * *", minutes), nil
	default:
		return "", fmt.Errorf("不支持的持续时间: %s", d)
	}
}
