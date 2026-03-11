// Package scheduler provides task scheduling for icooclaw.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// TaskRecord represents a scheduled task record in database.
type TaskRecord struct {
	ID          uint      `gorm:"column:id;primaryKey:true" json:"id"`
	TaskID      string    `gorm:"column:task_id;uniqueIndex;not null" json:"task_id"`
	Name        string    `gorm:"column:name;not null" json:"name"`
	Schedule    string    `gorm:"column:schedule;not null" json:"schedule"`
	Description string    `gorm:"column:description;" json:"description"`
	Enabled     bool      `gorm:"column:enabled;default:true" json:"enabled"`
	LastRun     time.Time `gorm:"column:last_run;" json:"last_run"`
	NextRun     time.Time `gorm:"column:next_run;" json:"next_run"`
	CreatedAt   time.Time `gorm:"column:created_at;" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;" json:"updated_at"`
}

// TableName returns the table name.
func (TaskRecord) TableName() string {
	return "scheduled_tasks"
}

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

// TaskStore provides database storage for tasks.
type TaskStore struct {
	db *gorm.DB
}

// NewTaskStore creates a new task store.
func NewTaskStore(db *gorm.DB) (*TaskStore, error) {
	// Auto migrate
	if err := db.AutoMigrate(&TaskRecord{}, &TaskExecutionRecord{}); err != nil {
		return nil, err
	}
	return &TaskStore{db: db}, nil
}

// Save saves a task record.
func (s *TaskStore) Save(task *TaskRecord) error {
	return s.db.Save(task).Error
}

// Get gets a task by ID.
func (s *TaskStore) Get(taskID string) (*TaskRecord, error) {
	var task TaskRecord
	err := s.db.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List lists all tasks.
func (s *TaskStore) List() ([]*TaskRecord, error) {
	var tasks []*TaskRecord
	err := s.db.Find(&tasks).Error
	return tasks, err
}

// ListEnabled lists all enabled tasks.
func (s *TaskStore) ListEnabled() ([]*TaskRecord, error) {
	var tasks []*TaskRecord
	err := s.db.Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// Delete deletes a task.
func (s *TaskStore) Delete(taskID string) error {
	return s.db.Where("task_id = ?", taskID).Delete(&TaskRecord{}).Error
}

// SaveExecution saves an execution record.
func (s *TaskStore) SaveExecution(exec *TaskExecutionRecord) error {
	return s.db.Create(exec).Error
}

// GetExecutions gets execution history for a task.
func (s *TaskStore) GetExecutions(taskID string, limit int) ([]*TaskExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	var executions []*TaskExecutionRecord
	err := s.db.Where("task_id = ?", taskID).
		Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error
	return executions, err
}

// PersistentScheduler is a scheduler with database persistence.
type PersistentScheduler struct {
	*Scheduler
	store *TaskStore
}

// NewPersistentScheduler creates a new persistent scheduler.
func NewPersistentScheduler(store *TaskStore, logger *slog.Logger) *PersistentScheduler {
	return &PersistentScheduler{
		Scheduler: NewScheduler(logger),
		store:     store,
	}
}

// LoadTasks loads tasks from database.
func (s *PersistentScheduler) LoadTasks(handlers map[string]func(ctx context.Context) error) error {
	tasks, err := s.store.ListEnabled()
	if err != nil {
		return err
	}

	for _, record := range tasks {
		handler, ok := handlers[record.TaskID]
		if !ok {
			s.logger.Warn("handler not found for task", "task_id", record.TaskID)
			continue
		}

		task := &Task{
			ID:          record.TaskID,
			Name:        record.Name,
			Schedule:    record.Schedule,
			Description: record.Description,
			Handler:     handler,
			Enabled:     record.Enabled,
			LastRun:     record.LastRun,
		}

		if err := s.Scheduler.AddTask(task); err != nil {
			s.logger.Error("failed to load task", "task_id", record.TaskID, "error", err)
		}
	}

	return nil
}

// AddTask adds a task and persists it.
func (s *PersistentScheduler) AddTask(task *Task) error {
	if err := s.Scheduler.AddTask(task); err != nil {
		return err
	}

	record := &TaskRecord{
		TaskID:      task.ID,
		Name:        task.Name,
		Schedule:    task.Schedule,
		Description: task.Description,
		Enabled:     task.Enabled,
	}

	return s.store.Save(record)
}

// RemoveTask removes a task and deletes it from database.
func (s *PersistentScheduler) RemoveTask(id string) error {
	if err := s.Scheduler.RemoveTask(id); err != nil {
		return err
	}
	return s.store.Delete(id)
}

// executeTask executes a task and records the result.
func (s *PersistentScheduler) executeTask(task *Task) {
	s.Scheduler.executeTask(task)

	// Update task record
	record, err := s.store.Get(task.ID)
	if err == nil {
		record.LastRun = task.LastRun
		record.NextRun = task.NextRun
		s.store.Save(record)
	}
}

// SaveExecution saves an execution record.
func (s *PersistentScheduler) SaveExecution(result TaskResult) {
	exec := &TaskExecutionRecord{
		TaskID:    result.TaskID,
		StartTime: result.StartTime,
		EndTime:   result.EndTime,
		Success:   result.Error == nil,
	}
	if result.Error != nil {
		exec.Error = result.Error.Error()
	}
	s.store.SaveExecution(exec)
}

// GetExecutionHistory gets execution history for a task.
func (s *PersistentScheduler) GetExecutionHistory(taskID string, limit int) ([]*TaskExecutionRecord, error) {
	return s.store.GetExecutions(taskID, limit)
}
