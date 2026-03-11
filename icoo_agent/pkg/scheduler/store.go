// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"icooclaw/pkg/storage"
)

// TaskExecutionRecord represents a task execution record.
type TaskExecutionRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    string    `gorm:"index;not null" json:"task_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Success   bool      `json:"success"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName returns the table name.
func (TaskExecutionRecord) TableName() string {
	return "task_executions"
}

// PersistentScheduler is a scheduler with database persistence.
type PersistentScheduler struct {
	*Scheduler
	store *storage.TaskStorage
}

// NewPersistentScheduler creates a new persistent scheduler.
func NewPersistentScheduler(store *storage.TaskStorage, logger *slog.Logger) *PersistentScheduler {
	return &PersistentScheduler{
		Scheduler: NewScheduler(logger),
		store:     store,
	}
}

// LoadTasks loads tasks from database.
func (s *PersistentScheduler) LoadTasks(handlers map[string]func(ctx context.Context) error) error {
	tasks, err := s.store.GetEnabled()
	if err != nil {
		return err
	}

	for _, record := range tasks {
		handler, ok := handlers[record.Name]
		if !ok {
			handler, ok = handlers[record.ID]
		}
		if !ok {
			s.logger.Warn("未找到任务处理器", "task_id", record.ID, "name", record.Name)
			continue
		}

		task := &Task{
			ID:          record.ID,
			Name:        record.Name,
			Schedule:    record.CronExpr,
			Description: record.Description,
			Handler:     handler,
			Enabled:     record.Enabled,
		}

		// Parse time strings
		if record.LastRunAt != "" {
			if t, err := time.Parse(time.RFC3339, record.LastRunAt); err == nil {
				task.LastRun = t
			}
		}
		if record.NextRunAt != "" {
			if t, err := time.Parse(time.RFC3339, record.NextRunAt); err == nil {
				task.NextRun = t
			}
		}

		if err := s.Scheduler.AddTask(task); err != nil {
			s.logger.Error("加载任务失败", "task_id", record.ID, "error", err)
		}
	}

	return nil
}

// AddTask adds a task and persists it.
func (s *PersistentScheduler) AddTask(task *Task) error {
	if err := s.Scheduler.AddTask(task); err != nil {
		return err
	}

	record := &storage.Task{
		Name:        task.Name,
		Description: task.Description,
		CronExpr:    task.Schedule,
		Enabled:     task.Enabled,
	}

	if !task.LastRun.IsZero() {
		record.LastRunAt = task.LastRun.Format(time.RFC3339)
	}
	if !task.NextRun.IsZero() {
		record.NextRunAt = task.NextRun.Format(time.RFC3339)
	}

	return s.store.CreateOrUpdate(record)
}

// RemoveTask removes a task and deletes it from database.
func (s *PersistentScheduler) RemoveTask(id string) error {
	if err := s.Scheduler.RemoveTask(id); err != nil {
		return err
	}
	return s.store.Delete(id)
}

// EnableTask enables a task and updates it in database.
func (s *PersistentScheduler) EnableTask(id string) error {
	if err := s.Scheduler.EnableTask(id); err != nil {
		return err
	}
	return s.store.ToggleEnabled(id)
}

// DisableTask disables a task and updates it in database.
func (s *PersistentScheduler) DisableTask(id string) error {
	if err := s.Scheduler.DisableTask(id); err != nil {
		return err
	}
	return s.store.ToggleEnabled(id)
}

// executeTask executes a task and records the result.
func (s *PersistentScheduler) executeTask(task *Task) {
	s.Scheduler.executeTask(task)

	// Update task record
	record, err := s.store.GetByID(task.ID)
	if err == nil {
		record.LastRunAt = task.LastRun.Format(time.RFC3339)
		record.NextRunAt = task.NextRun.Format(time.RFC3339)
		s.store.Update(record)
	}
}

// GetTask gets a task by ID from database.
func (s *PersistentScheduler) GetTask(id string) (*storage.Task, error) {
	return s.store.GetByID(id)
}

// ListTasksFromDB lists all tasks from database.
func (s *PersistentScheduler) ListTasksFromDB() ([]storage.Task, error) {
	return s.store.GetAll()
}