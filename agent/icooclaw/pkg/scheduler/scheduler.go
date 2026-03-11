// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Task represents a scheduled task.
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

// TaskResult represents the result of a task execution.
type TaskResult struct {
	TaskID    string
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

// Scheduler manages scheduled tasks.
type Scheduler struct {
	cron    *cron.Cron
	tasks   map[string]*Task
	results chan TaskResult
	logger  *slog.Logger
	mu      sync.RWMutex
	running bool
}

// NewScheduler creates a new scheduler.
func NewScheduler(logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		cron:    cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC)),
		tasks:   make(map[string]*Task),
		results: make(chan TaskResult, 100),
		logger:  logger,
	}
}

// AddTask adds a new scheduled task.
func (s *Scheduler) AddTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	// Parse schedule
	schedule, err := cron.ParseStandard(task.Schedule)
	if err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	// Create cron job
	entryID := s.cron.Schedule(schedule, cron.FuncJob(func() {
		s.executeTask(task)
	}))

	task.EntryID = entryID
	task.NextRun = s.cron.Entry(entryID).Next
	s.tasks[task.ID] = task

	s.logger.Info("task added", "id", task.ID, "name", task.Name, "schedule", task.Schedule)
	return nil
}

// RemoveTask removes a scheduled task.
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	s.cron.Remove(task.EntryID)
	delete(s.tasks, id)

	s.logger.Info("task removed", "id", id)
	return nil
}

// EnableTask enables a task.
func (s *Scheduler) EnableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	task.Enabled = true
	s.logger.Info("task enabled", "id", id)
	return nil
}

// DisableTask disables a task.
func (s *Scheduler) DisableTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	task.Enabled = false
	s.logger.Info("task disabled", "id", id)
	return nil
}

// GetTask gets a task by ID.
func (s *Scheduler) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}

	// Update next run time
	entry := s.cron.Entry(task.EntryID)
	if entry.ID != 0 {
		task.NextRun = entry.Next
	}

	return task, nil
}

// ListTasks lists all tasks.
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

// RunTask executes a task immediately.
func (s *Scheduler) RunTask(id string) error {
	s.mu.RLock()
	task, exists := s.tasks[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}

	go s.executeTask(task)
	return nil
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.cron.Start()
	s.running = true
	s.logger.Info("scheduler started")
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.logger.Info("scheduler stopped")
}

// Results returns the results channel.
func (s *Scheduler) Results() <-chan TaskResult {
	return s.results
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// executeTask executes a task.
func (s *Scheduler) executeTask(task *Task) {
	if !task.Enabled {
		return
	}

	startTime := time.Now()
	s.logger.Debug("executing task", "id", task.ID, "name", task.Name)

	ctx := context.Background()
	err := task.Handler(ctx)

	endTime := time.Now()
	task.LastRun = endTime

	// Update next run time
	entry := s.cron.Entry(task.EntryID)
	if entry.ID != 0 {
		task.NextRun = entry.Next
	}

	// Send result
	result := TaskResult{
		TaskID:    task.ID,
		StartTime: startTime,
		EndTime:   endTime,
		Error:     err,
	}

	select {
	case s.results <- result:
	default:
		s.logger.Warn("result channel full, dropping result", "task_id", task.ID)
	}

	if err != nil {
		s.logger.Error("task execution failed", "id", task.ID, "error", err)
	} else {
		s.logger.Debug("task executed successfully", "id", task.ID, "duration", endTime.Sub(startTime))
	}
}

// Common task schedules
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

// ParseDuration parses a duration string and returns a cron schedule.
func ParseDuration(d string) (string, error) {
	duration, err := time.ParseDuration(d)
	if err != nil {
		return "", err
	}

	// Convert to cron schedule
	switch {
	case duration < time.Minute:
		return "", fmt.Errorf("minimum duration is 1 minute")
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
		return "", fmt.Errorf("unsupported duration: %s", d)
	}
}
