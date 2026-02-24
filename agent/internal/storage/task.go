package storage

import (
	"time"

	"gorm.io/gorm"
)

// Task 定时任务模型
type Task struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;uniqueIndex" json:"name"`
	Description string    `gorm:"size:500" json:"description"`
	CronExpr    string    `gorm:"size:100" json:"cron_expr"` // Cron表达式
	Interval    int       `gorm:"default:0" json:"interval"` // 固定间隔(秒)
	Message     string    `gorm:"type:text" json:"message"`  // 触发消息
	Channel     string    `gorm:"size:50" json:"channel"`    // 投递通道
	ChatID      string    `gorm:"size:255" json:"chat_id"`   // 投递目标
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	NextRunAt   time.Time `gorm:"index" json:"next_run_at"` // 下次执行时间
	LastRunAt   time.Time `json:"last_run_at"`              // 上次执行时间
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 表名
func (Task) TableName() string {
	return "tasks"
}

// Create 创建任务
func (t *Task) Create() error {
	return DB.Create(t).Error
}

// Update 更新任务
func (t *Task) Update() error {
	return DB.Save(t).Error
}

// Delete 删除任务
func (t *Task) Delete() error {
	return DB.Delete(t).Error
}

// GetByID 通过ID获取任务
func GetTaskByID(id uint) (*Task, error) {
	var task Task
	err := DB.First(&task, id).Error
	return &task, err
}

// GetByName 通过名称获取任务
func GetTaskByName(name string) (*Task, error) {
	var task Task
	err := DB.Where("name = ?", name).First(&task).Error
	return &task, err
}

// GetEnabledTasks 获取所有启用的任务
func GetEnabledTasks() ([]Task, error) {
	var tasks []Task
	err := DB.Where("enabled = ?", true).Find(&tasks).Error
	return tasks, err
}

// GetDueTasks 获取到期的任务
func GetDueTasks() ([]Task, error) {
	var tasks []Task
	err := DB.Where("enabled = ? AND (next_run_at IS NULL OR next_run_at <= ?)", true, time.Now()).Find(&tasks).Error
	return tasks, err
}

// UpdateNextRun 更新下次执行时间
func (t *Task) UpdateNextRun(nextRun time.Time) error {
	return DB.Model(t).Update("next_run_at", nextRun).Error
}

// UpdateLastRun 更新上次执行时间
func (t *Task) UpdateLastRun() error {
	return DB.Model(t).Update("last_run_at", time.Now()).Error
}

// ToggleEnabled 切换启用状态
func (t *Task) ToggleEnabled() error {
	return DB.Model(t).Update("enabled", gorm.Expr("NOT enabled")).Error
}
