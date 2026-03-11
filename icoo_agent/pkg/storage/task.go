package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Task represents a scheduled task.
type Task struct {
	Model
	Name        string `gorm:"column:name;type:varchar(100);not null;comment:任务名称" json:"name"`
	Description string `gorm:"column:description;type:text;comment:任务描述" json:"description"`
	CronExpr    string `gorm:"column:cron_expr;type:varchar(100);comment:Cron表达式" json:"cron_expr"`
	Handler     string `gorm:"column:handler;type:varchar(100);not null;comment:处理器名称" json:"handler"`
	Params      string `gorm:"column:params;type:text;comment:参数(JSON格式)" json:"params"`
	Enabled     bool   `gorm:"column:enabled;type:tinyint(1);default:true;comment:是否启用" json:"enabled"`
	LastRunAt   string `gorm:"column:last_run_at;type:datetime;comment:最后执行时间" json:"last_run_at"`
	NextRunAt   string `gorm:"column:next_run_at;type:datetime;comment:下次执行时间" json:"next_run_at"`
}

// TableName returns the table name for Task.
func (Task) TableName() string {
	return tableNamePrefix + "tasks"
}

type QueryTask struct {
	Page    Page   `json:"page"`
	KeyWord string `json:"key_word"`
	Enabled *bool  `json:"enabled"`
}

type ResQueryTask struct {
	Page    Page   `json:"page"`
	Records []Task `json:"records"`
}

type TaskStorage struct {
	db *gorm.DB
}

func NewTaskStorage(db *gorm.DB) *TaskStorage {
	return &TaskStorage{db: db}
}

// Create creates a new task.
func (s *TaskStorage) Create(t *Task) error {
	return s.db.Create(t).Error
}

// Update updates a task.
func (s *TaskStorage) Update(t *Task) error {
	return s.db.Save(t).Error
}

// CreateOrUpdate creates or updates a task.
func (s *TaskStorage) CreateOrUpdate(t *Task) error {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"description", "cron_expr", "handler", "params", "enabled", "last_run_at", "next_run_at"}),
	}).Create(t)
	return result.Error
}

// Delete deletes a task by ID.
func (s *TaskStorage) Delete(id string) error {
	result := s.db.Where("id = ?", id).Delete(&Task{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete task: %w", result.Error)
	}
	return nil
}

// GetByID gets a task by ID.
func (s *TaskStorage) GetByID(id string) (*Task, error) {
	var t Task
	result := s.db.Where("id = ?", id).First(&t)
	if result.Error == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("task not found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get task: %w", result.Error)
	}
	return &t, nil
}

// GetAll gets all tasks.
func (s *TaskStorage) GetAll() ([]Task, error) {
	var tasks []Task
	result := s.db.Order("name").Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", result.Error)
	}
	return tasks, nil
}

// GetEnabled gets all enabled tasks.
func (s *TaskStorage) GetEnabled() ([]Task, error) {
	var tasks []Task
	result := s.db.Where("enabled = ?", true).Order("name").Find(&tasks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get enabled tasks: %w", result.Error)
	}
	return tasks, nil
}

// ToggleEnabled toggles a task's enabled status.
func (s *TaskStorage) ToggleEnabled(id string) error {
	var t Task
	result := s.db.Where("id = ?", id).First(&t)
	if result.Error != nil {
		return fmt.Errorf("failed to get task: %w", result.Error)
	}
	t.Enabled = !t.Enabled
	return s.db.Save(&t).Error
}

// Page gets tasks with pagination.
func (s *TaskStorage) Page(query *QueryTask) (*ResQueryTask, error) {
	var res ResQueryTask

	qry := s.db.Model(&Task{})

	if query.KeyWord != "" {
		qry = qry.Where("name LIKE ? OR description LIKE ?", "%"+query.KeyWord+"%", "%"+query.KeyWord+"%")
	}

	if query.Enabled != nil {
		qry = qry.Where("enabled = ?", *query.Enabled)
	}

	qry = qry.Order("name")

	result := qry.Count(&res.Page.Total)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", result.Error)
	}

	if query.Page.Page == 0 || query.Page.Size == 0 {
		result = qry.Find(&res.Records)
	} else {
		result = qry.Limit(query.Page.Size).
			Offset((query.Page.Page - 1) * query.Page.Size).
			Find(&res.Records)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", result.Error)
	}

	return &res, nil
}